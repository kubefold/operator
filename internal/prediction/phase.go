package prediction

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/shared"
)

const requeueAfterJobPoll = 10 * time.Second

type PhaseRouter interface {
	Handle(ctx context.Context, prediction *datav1.ProteinConformationPrediction) (ctrl.Result, error)
}

type phaseRouter struct {
	client     client.Client
	scheme     *runtime.Scheme
	recorder   record.EventRecorder
	pvc        PVCBuilder
	jobs       JobBuilder
	input      InputEncoder
	retry      RetryPolicy
	timeout    TimeoutChecker
	cleaner    ResourceCleaner
	conditions ConditionsManager
	status     StatusWriter
}

func NewPhaseRouter(
	c client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	pvc PVCBuilder,
	jobs JobBuilder,
	input InputEncoder,
	retry RetryPolicy,
	timeout TimeoutChecker,
	cleaner ResourceCleaner,
	conditions ConditionsManager,
	status StatusWriter,
) PhaseRouter {
	return &phaseRouter{
		client:     c,
		scheme:     scheme,
		recorder:   recorder,
		pvc:        pvc,
		jobs:       jobs,
		input:      input,
		retry:      retry,
		timeout:    timeout,
		cleaner:    cleaner,
		conditions: conditions,
		status:     status,
	}
}

func (r *phaseRouter) Handle(ctx context.Context, prediction *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	switch prediction.Status.Phase {
	case datav1.ProteinConformationPredictionStatusPhaseNotStarted:
		return r.handleNotStarted(ctx, prediction)
	case datav1.ProteinConformationPredictionStatusPhaseAligning:
		return r.handleAligning(ctx, prediction)
	case datav1.ProteinConformationPredictionStatusPhasePredicting:
		return r.handlePredicting(ctx, prediction)
	case datav1.ProteinConformationPredictionStatusPhaseUploadingArtifacts:
		return r.handleUploadingArtifacts(ctx, prediction)
	case datav1.ProteinConformationPredictionStatusPhaseCompleted, datav1.ProteinConformationPredictionStatusPhaseFailed:
		return ctrl.Result{}, nil
	}
	log.Info("Unknown phase", "Phase", prediction.Status.Phase)
	return ctrl.Result{}, nil
}

func (r *phaseRouter) handleNotStarted(ctx context.Context, prediction *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	database := &datav1.ProteinDatabase{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: prediction.Spec.Database, Namespace: prediction.Namespace}, database); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Waiting for ProteinDatabase to be created", "Database", prediction.Spec.Database)
			r.recorder.Event(prediction, corev1.EventTypeNormal, "DatabaseNotFound", fmt.Sprintf("Waiting for ProteinDatabase %s to be created", prediction.Spec.Database))
			return ctrl.Result{RequeueAfter: requeueAfterJobPoll}, nil
		}
		log.Error(err, "Failed to get ProteinDatabase")
		r.recorder.Event(prediction, corev1.EventTypeWarning, "DatabaseError", fmt.Sprintf("Failed to get ProteinDatabase: %v", err))
		return ctrl.Result{RequeueAfter: requeueAfterJobPoll}, nil
	}

	pvcName := shared.PredictionDataPVCName(prediction.Name)
	if result, err := r.ensurePVC(ctx, prediction, pvcName); err != nil || !result.IsZero() {
		return result, err
	}

	jobName := shared.SearchJobName(prediction.Name)
	job := &batchv1.Job{}
	err := r.client.Get(ctx, types.NamespacedName{Name: jobName, Namespace: prediction.Namespace}, job)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to get search job")
		r.recorder.Event(prediction, corev1.EventTypeWarning, "JobError", fmt.Sprintf("Failed to get search job: %v", err))
		return ctrl.Result{}, err
	}
	if errors.IsNotFound(err) {
		encodedInput, encodeErr := r.input.Encode(prediction, false)
		if encodeErr != nil {
			log.Error(encodeErr, "Failed to prepare FoldInput")
			r.recorder.Event(prediction, corev1.EventTypeWarning, "InputError", fmt.Sprintf("Failed to prepare FoldInput: %v", encodeErr))
			return ctrl.Result{}, encodeErr
		}
		newJob := r.jobs.BuildSearch(prediction, jobName, pvcName, encodedInput)
		if err := controllerutil.SetControllerReference(prediction, newJob, r.scheme); err != nil {
			log.Error(err, "Failed to set controller reference for search job")
			return ctrl.Result{}, err
		}
		if err := r.client.Create(ctx, newJob); err != nil && !errors.IsAlreadyExists(err) {
			log.Error(err, "Failed to create search job")
			r.recorder.Event(prediction, corev1.EventTypeWarning, "JobCreationError", fmt.Sprintf("Failed to create search job: %v", err))
			return ctrl.Result{}, err
		}
		r.recorder.Event(prediction, corev1.EventTypeNormal, "JobCreated", fmt.Sprintf("Created search job %s", jobName))
	}

	if err := r.status.Update(ctx, prediction, func(p *datav1.ProteinConformationPrediction) {
		p.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseAligning
		r.conditions.Set(&p.Status, metav1.Condition{
			Type:    ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonPhaseProgressed,
			Message: "Aligning phase started",
		})
	}); err != nil {
		log.Error(err, "Failed to update ProteinConformationPrediction status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *phaseRouter) ensurePVC(ctx context.Context, prediction *datav1.ProteinConformationPrediction, pvcName string) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.client.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: prediction.Namespace}, pvc)
	if err == nil {
		return ctrl.Result{}, nil
	}
	if !errors.IsNotFound(err) {
		log.Error(err, "Failed to get PVC")
		r.recorder.Event(prediction, corev1.EventTypeWarning, "PVCError", fmt.Sprintf("Failed to get PVC: %v", err))
		return ctrl.Result{}, err
	}

	newPVC := r.pvc.Build(prediction, pvcName)
	if err := controllerutil.SetControllerReference(prediction, newPVC, r.scheme); err != nil {
		log.Error(err, "Failed to set controller reference for PVC")
		return ctrl.Result{}, err
	}
	if err := r.client.Create(ctx, newPVC); err != nil && !errors.IsAlreadyExists(err) {
		log.Error(err, "Failed to create PVC")
		r.recorder.Event(prediction, corev1.EventTypeWarning, "PVCCreationError", fmt.Sprintf("Failed to create PVC: %v", err))
		return ctrl.Result{}, err
	}
	r.recorder.Event(prediction, corev1.EventTypeNormal, "PVCCreated", fmt.Sprintf("Created PVC %s", pvcName))
	return ctrl.Result{Requeue: true}, nil
}

func (r *phaseRouter) handleAligning(ctx context.Context, prediction *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	return r.observeJob(ctx, prediction, jobObservation{
		phase:       PhaseSearch,
		jobName:     shared.SearchJobName(prediction.Name),
		nextPhase:   datav1.ProteinConformationPredictionStatusPhasePredicting,
		condType:    ConditionTypeSearchSucceeded,
		condMessage: "Search job completed",
	})
}

func (r *phaseRouter) handlePredicting(ctx context.Context, prediction *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	jobName := shared.PredictJobName(prediction.Name)
	job := &batchv1.Job{}
	err := r.client.Get(ctx, types.NamespacedName{Name: jobName, Namespace: prediction.Namespace}, job)
	if errors.IsNotFound(err) {
		return r.createPredictionJob(ctx, prediction, jobName)
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	return r.observeExistingJob(ctx, prediction, job, jobObservation{
		phase:       PhasePredict,
		jobName:     jobName,
		nextPhase:   datav1.ProteinConformationPredictionStatusPhaseUploadingArtifacts,
		condType:    ConditionTypePredictSucceeded,
		condMessage: "Prediction job completed",
	})
}

func (r *phaseRouter) createPredictionJob(ctx context.Context, prediction *datav1.ProteinConformationPrediction, jobName string) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	encodedInput, err := r.input.Encode(prediction, true)
	if err != nil {
		log.Error(err, "Failed to prepare FoldInput")
		return ctrl.Result{}, err
	}
	pvcName := shared.PredictionDataPVCName(prediction.Name)
	job := r.jobs.BuildPredict(prediction, jobName, pvcName, encodedInput)
	if err := controllerutil.SetControllerReference(prediction, job, r.scheme); err != nil {
		log.Error(err, "Failed to set controller reference for prediction job")
		return ctrl.Result{}, err
	}
	if err := r.client.Create(ctx, job); err != nil && !errors.IsAlreadyExists(err) {
		log.Error(err, "Failed to create prediction job")
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *phaseRouter) handleUploadingArtifacts(ctx context.Context, prediction *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	jobName := shared.UploadJobName(prediction.Name)
	job := &batchv1.Job{}
	err := r.client.Get(ctx, types.NamespacedName{Name: jobName, Namespace: prediction.Namespace}, job)
	if errors.IsNotFound(err) {
		return r.createUploadJob(ctx, prediction, jobName)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	if job.Status.Succeeded > 0 {
		return r.completeUpload(ctx, prediction)
	}
	if r.timeout.IsTimedOut(job, DefaultJobTimeout) {
		return r.markFailedDueToTimeout(ctx, prediction, jobName)
	}
	if job.Status.Failed > 0 {
		return r.retryOrFail(ctx, prediction, job, PhaseUpload, jobName, "Upload job failed")
	}
	return ctrl.Result{RequeueAfter: requeueAfterJobPoll}, nil
}

func (r *phaseRouter) createUploadJob(ctx context.Context, prediction *datav1.ProteinConformationPrediction, jobName string) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	pvcName := shared.PredictionDataPVCName(prediction.Name)
	job := r.jobs.BuildUpload(prediction, jobName, pvcName)
	if err := controllerutil.SetControllerReference(prediction, job, r.scheme); err != nil {
		log.Error(err, "Failed to set controller reference for upload job")
		return ctrl.Result{}, err
	}
	if err := r.client.Create(ctx, job); err != nil && !errors.IsAlreadyExists(err) {
		log.Error(err, "Failed to create upload job")
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *phaseRouter) completeUpload(ctx context.Context, prediction *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	if err := r.status.Update(ctx, prediction, func(p *datav1.ProteinConformationPrediction) {
		p.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseCompleted
		r.conditions.Set(&p.Status, metav1.Condition{
			Type:    ConditionTypeUploadSucceeded,
			Status:  metav1.ConditionTrue,
			Reason:  ReasonCompleted,
			Message: "Upload job completed",
		})
		r.conditions.Set(&p.Status, metav1.Condition{
			Type:    ConditionTypeReady,
			Status:  metav1.ConditionTrue,
			Reason:  ReasonCompleted,
			Message: "Prediction artifacts uploaded",
		})
	}); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.cleaner.DeleteCompletedJobs(ctx, prediction.Name, prediction.Namespace); err != nil {
		log.Error(err, "Failed to cleanup completed jobs")
		return ctrl.Result{}, err
	}
	if err := r.cleaner.DeletePVCForPrediction(ctx, prediction.Name, prediction.Namespace); err != nil {
		log.Error(err, "Failed to delete PVC")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

type jobObservation struct {
	phase       Phase
	jobName     string
	nextPhase   datav1.ProteinConformationPredictionStatusPhase
	condType    string
	condMessage string
}

func (r *phaseRouter) observeJob(ctx context.Context, prediction *datav1.ProteinConformationPrediction, observation jobObservation) (ctrl.Result, error) {
	job := &batchv1.Job{}
	err := r.client.Get(ctx, types.NamespacedName{Name: observation.jobName, Namespace: prediction.Namespace}, job)
	if errors.IsNotFound(err) {
		return ctrl.Result{RequeueAfter: requeueAfterJobPoll}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	return r.observeExistingJob(ctx, prediction, job, observation)
}

func (r *phaseRouter) observeExistingJob(ctx context.Context, prediction *datav1.ProteinConformationPrediction, job *batchv1.Job, observation jobObservation) (ctrl.Result, error) {
	if job.Status.Succeeded > 0 {
		return r.advancePhase(ctx, prediction, observation)
	}
	if r.timeout.IsTimedOut(job, DefaultJobTimeout) {
		return r.markFailedDueToTimeout(ctx, prediction, observation.jobName)
	}
	if job.Status.Failed > 0 {
		return r.retryOrFail(ctx, prediction, job, observation.phase, observation.jobName, fmt.Sprintf("%s job failed", observation.phase))
	}
	return ctrl.Result{RequeueAfter: requeueAfterJobPoll}, nil
}

func (r *phaseRouter) advancePhase(ctx context.Context, prediction *datav1.ProteinConformationPrediction, observation jobObservation) (ctrl.Result, error) {
	if err := r.status.Update(ctx, prediction, func(p *datav1.ProteinConformationPrediction) {
		p.Status.Phase = observation.nextPhase
		r.conditions.Set(&p.Status, metav1.Condition{
			Type:    observation.condType,
			Status:  metav1.ConditionTrue,
			Reason:  ReasonPhaseProgressed,
			Message: observation.condMessage,
		})
		r.conditions.Set(&p.Status, metav1.Condition{
			Type:    ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonPhaseProgressed,
			Message: fmt.Sprintf("Phase transitioned to %s", observation.nextPhase),
		})
	}); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *phaseRouter) retryOrFail(ctx context.Context, prediction *datav1.ProteinConformationPrediction, job *batchv1.Job, phase Phase, jobName, message string) (ctrl.Result, error) {
	if r.retry.AtLimit(&prediction.Status, phase) {
		return r.markFailed(ctx, prediction, fmt.Sprintf("%s (job %s) after max retries", message, jobName), ReasonRetriesExhausted)
	}
	if err := r.status.Update(ctx, prediction, func(p *datav1.ProteinConformationPrediction) {
		r.retry.Increment(&p.Status, phase)
	}); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.client.Delete(ctx, job); err != nil && !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *phaseRouter) markFailedDueToTimeout(ctx context.Context, prediction *datav1.ProteinConformationPrediction, jobName string) (ctrl.Result, error) {
	if err := r.cleaner.DeleteJobsForPrediction(ctx, prediction.Name, prediction.Namespace); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.cleaner.DeletePVCForPrediction(ctx, prediction.Name, prediction.Namespace); err != nil {
		return ctrl.Result{}, err
	}
	return r.markFailed(ctx, prediction, fmt.Sprintf("Job %s timed out", jobName), ReasonJobTimeout)
}

func (r *phaseRouter) markFailed(ctx context.Context, prediction *datav1.ProteinConformationPrediction, message, reason string) (ctrl.Result, error) {
	if err := r.status.Update(ctx, prediction, func(p *datav1.ProteinConformationPrediction) {
		p.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseFailed
		p.Status.Error = message
		r.conditions.Set(&p.Status, metav1.Condition{
			Type:    ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  reason,
			Message: message,
		})
	}); err != nil {
		return ctrl.Result{}, err
	}
	r.recorder.Event(prediction, corev1.EventTypeWarning, reason, message)
	return ctrl.Result{}, nil
}
