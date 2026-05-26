package prediction

import (
	"context"
	"os"
	"slices"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	datav1 "github.com/kubefold/operator/api/v1"
)

const (
	Finalizer              = "proteinconformationprediction.data.kubefold.io/finalizer"
	WeightsAllowedHostsEnv = "KUBEFOLD_WEIGHTS_ALLOWED_HOSTS"
)

type Reconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	validator SpecValidator
	cleaner   ResourceCleaner
	status    StatusWriter
	router    PhaseRouter
}

func NewReconciler(c client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) *Reconciler {
	allowedHosts := parseAllowedHosts(os.Getenv(WeightsAllowedHostsEnv))
	validator := NewSpecValidator(allowedHosts)
	pvcBuilder := NewPVCBuilder()
	nodeSelector := NewNodeSelectorTranslator()
	jobBuilder := NewJobBuilder(nodeSelector)
	input := NewInputEncoder()
	retry := NewRetryPolicy(MaxRetries)
	timeout := NewTimeoutChecker()
	cleaner := NewResourceCleaner(c)
	conditions := NewConditionsManager()
	status := NewStatusWriter(c)
	router := NewPhaseRouter(c, scheme, recorder, pvcBuilder, jobBuilder, input, retry, timeout, cleaner, conditions, status)

	return &Reconciler{
		Client:    c,
		Scheme:    scheme,
		Recorder:  recorder,
		validator: validator,
		cleaner:   cleaner,
		status:    status,
		router:    router,
	}
}

// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteinconformationpredictions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteinconformationpredictions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteinconformationpredictions/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	prediction := &datav1.ProteinConformationPrediction{}
	if err := r.Get(ctx, req.NamespacedName, prediction); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ProteinConformationPrediction")
		return ctrl.Result{}, err
	}

	if !prediction.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, prediction)
	}

	if !controllerutil.ContainsFinalizer(prediction, Finalizer) {
		controllerutil.AddFinalizer(prediction, Finalizer)
		if err := r.Update(ctx, prediction); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if err := r.validator.Validate(prediction); err != nil {
		log.Error(err, "Invalid spec")
		return r.failValidation(ctx, prediction, err.Error())
	}

	if prediction.Status.Phase == "" {
		return r.initializeStatus(ctx, prediction)
	}

	return r.router.Handle(ctx, prediction)
}

func (r *Reconciler) handleDeletion(ctx context.Context, prediction *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	if !controllerutil.ContainsFinalizer(prediction, Finalizer) {
		return ctrl.Result{}, nil
	}
	if err := r.cleaner.DeleteJobsForPrediction(ctx, prediction.Name, prediction.Namespace); err != nil {
		log.Error(err, "Failed to delete prediction jobs")
		return ctrl.Result{}, err
	}
	if err := r.cleaner.DeletePVCForPrediction(ctx, prediction.Name, prediction.Namespace); err != nil {
		log.Error(err, "Failed to delete prediction PVC")
		return ctrl.Result{}, err
	}
	controllerutil.RemoveFinalizer(prediction, Finalizer)
	if err := r.Update(ctx, prediction); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) failValidation(ctx context.Context, prediction *datav1.ProteinConformationPrediction, message string) (ctrl.Result, error) {
	conditions := NewConditionsManager()
	if err := r.status.Update(ctx, prediction, func(p *datav1.ProteinConformationPrediction) {
		p.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseFailed
		p.Status.Error = message
		conditions.Set(&p.Status, metav1.Condition{
			Type:    ConditionTypeValidationFailed,
			Status:  metav1.ConditionTrue,
			Reason:  ReasonValidationFailed,
			Message: message,
		})
		conditions.Set(&p.Status, metav1.Condition{
			Type:    ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonValidationFailed,
			Message: message,
		})
	}); err != nil {
		return ctrl.Result{}, err
	}
	r.Recorder.Event(prediction, corev1.EventTypeWarning, ReasonValidationFailed, message)
	return ctrl.Result{}, nil
}

func (r *Reconciler) initializeStatus(ctx context.Context, prediction *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	prefix := r.validator.SequencePrefix(prediction.Spec.Protein.Sequence)
	if err := r.status.Update(ctx, prediction, func(p *datav1.ProteinConformationPrediction) {
		p.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseNotStarted
		p.Status.SequencePrefix = prefix
		p.Status.SearchRetryCount = 0
		p.Status.PredictRetryCount = 0
		p.Status.UploadRetryCount = 0
	}); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("proteinconformationprediction-controller")
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&datav1.ProteinConformationPrediction{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&batchv1.Job{}).
		Named("proteinconformationprediction").
		Complete(r)
}

func parseAllowedHosts(raw string) []string {
	if raw == "" || raw == "*" {
		return nil
	}
	hosts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' })
	for i, host := range hosts {
		hosts[i] = strings.TrimSpace(host)
	}
	return slices.DeleteFunc(hosts, func(host string) bool { return host == "" })
}
