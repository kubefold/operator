package database

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/shared"
)

type Reconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	volume    VolumeReconciler
	jobs      JobReconciler
	finalizer FinalizerReconciler
}

func NewReconciler(c client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) *Reconciler {
	enumerator := NewDatasetEnumerator()
	sizer := NewSizer(enumerator)
	return &Reconciler{
		Client:    c,
		Scheme:    scheme,
		Recorder:  recorder,
		volume:    NewVolumeReconciler(c, scheme, sizer),
		jobs:      NewJobReconciler(c, scheme, enumerator),
		finalizer: NewFinalizerReconciler(c),
	}
}

// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteindatabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteindatabases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteindatabases/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods/log,verbs=get

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	database := &datav1.ProteinDatabase{}
	if err := r.Get(ctx, req.NamespacedName, database); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ProteinDatabase")
		return ctrl.Result{}, err
	}

	if !database.DeletionTimestamp.IsZero() {
		return r.finalizer.HandleDeletion(ctx, database)
	}

	if result, err := r.finalizer.Ensure(ctx, database); err != nil || !result.IsZero() {
		return result, err
	}

	pvc, result, err := r.volume.Ensure(ctx, database)
	if err != nil {
		return ctrl.Result{}, err
	}
	if result != nil {
		return *result, nil
	}

	if database.Status.VolumeName != pvc.Spec.VolumeName {
		if err := r.updateVolumeName(ctx, database, pvc); err != nil {
			log.Error(err, "Failed to update ProteinDatabase status")
			return ctrl.Result{}, err
		}
	}

	if err := r.jobs.EnsureAll(ctx, database, pvc); err != nil {
		log.Error(err, "Failed to ensure downloader jobs")
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation completed successfully")
	return ctrl.Result{}, nil
}

func (r *Reconciler) updateVolumeName(ctx context.Context, database *datav1.ProteinDatabase, pvc *corev1.PersistentVolumeClaim) error {
	return shared.RetryOnConflict(ctx, func() error {
		latest := &datav1.ProteinDatabase{}
		if err := r.Get(ctx, types.NamespacedName{Name: database.Name, Namespace: database.Namespace}, latest); err != nil {
			return err
		}
		latest.Status.VolumeName = pvc.Spec.VolumeName
		return r.Status().Update(ctx, latest)
	})
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("proteindatabase-controller")
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&datav1.ProteinDatabase{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&batchv1.Job{}).
		Named("proteindatabase").
		Complete(r)
}
