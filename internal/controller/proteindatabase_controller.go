package controller

import (
	"context"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	datav1 "github.com/kubefold/operator/api/v1"
)

type ProteinDatabaseReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	volumeHandler    *VolumeHandler
	jobHandler       *JobHandler
	finalizerHandler *FinalizerHandler
}

func (r *ProteinDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	pd := &datav1.ProteinDatabase{}
	if err := r.Get(ctx, req.NamespacedName, pd); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ProteinDatabase")
		return ctrl.Result{}, err
	}

	if r.volumeHandler == nil {
		r.volumeHandler = &VolumeHandler{client: r.Client, scheme: r.Scheme}
	}
	if r.jobHandler == nil {
		r.jobHandler = &JobHandler{client: r.Client, scheme: r.Scheme}
	}
	if r.finalizerHandler == nil {
		r.finalizerHandler = &FinalizerHandler{client: r.Client, scheme: r.Scheme}
	}

	if !pd.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.finalizerHandler.handleDeletion(ctx, pd)
	}

	if !controllerutil.ContainsFinalizer(pd, ProteinDatabaseFinalizer) {
		return r.finalizerHandler.ensureFinalizer(ctx, pd)
	}

	pvc, result, err := r.volumeHandler.ensurePVC(ctx, pd)
	if err != nil {
		return ctrl.Result{}, err
	}
	if result != nil {
		return *result, nil
	}

	if pd.Status.VolumeName != pvc.Spec.VolumeName {
		if err := r.updateStatus(ctx, pd, pvc); err != nil {
			log.Error(err, "Failed to update ProteinDatabase status")
			return ctrl.Result{}, err
		}
	}

	if err := r.jobHandler.ensureJobs(ctx, pd, pvc); err != nil {
		log.Error(err, "Failed to ensure downloader jobs")
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation completed successfully")
	//return ctrl.Result{Requeue: true, RequeueAfter: ReconcileInterval}, nil
	return ctrl.Result{}, nil
}

func (r *ProteinDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&datav1.ProteinDatabase{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&batchv1.Job{}).
		Named("proteindatabase").
		Complete(r)
}

func (r *ProteinDatabaseReconciler) updateStatus(ctx context.Context, pd *datav1.ProteinDatabase, pvc *corev1.PersistentVolumeClaim) error {
	pdCopy := pd.DeepCopy()
	pdCopy.Status.VolumeName = pvc.Spec.VolumeName
	return r.Status().Update(ctx, pdCopy)
}
