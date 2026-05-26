package database

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/shared"
)

const Finalizer = "data.kubefold.io/finalizer"

type FinalizerReconciler interface {
	HandleDeletion(ctx context.Context, database *datav1.ProteinDatabase) (ctrl.Result, error)
	Ensure(ctx context.Context, database *datav1.ProteinDatabase) (ctrl.Result, error)
}

type finalizerReconciler struct {
	client client.Client
}

func NewFinalizerReconciler(c client.Client) FinalizerReconciler {
	return &finalizerReconciler{client: c}
}

func (f *finalizerReconciler) HandleDeletion(ctx context.Context, database *datav1.ProteinDatabase) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	if !controllerutil.ContainsFinalizer(database, Finalizer) {
		return ctrl.Result{}, nil
	}
	if err := f.deleteJobs(ctx, database); err != nil {
		log.Error(err, "Failed to clean up downloader jobs")
		return ctrl.Result{}, err
	}
	if err := f.deletePVC(ctx, database); err != nil {
		log.Error(err, "Failed to clean up PVC")
		return ctrl.Result{}, err
	}
	controllerutil.RemoveFinalizer(database, Finalizer)
	if err := f.client.Update(ctx, database); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (f *finalizerReconciler) Ensure(ctx context.Context, database *datav1.ProteinDatabase) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	if controllerutil.ContainsFinalizer(database, Finalizer) {
		return ctrl.Result{}, nil
	}
	controllerutil.AddFinalizer(database, Finalizer)
	if err := f.client.Update(ctx, database); err != nil {
		log.Error(err, "Failed to add finalizer")
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (f *finalizerReconciler) deleteJobs(ctx context.Context, database *datav1.ProteinDatabase) error {
	jobs := &batchv1.JobList{}
	if err := f.client.List(ctx, jobs,
		client.InNamespace(database.Namespace),
		client.MatchingLabels{shared.LabelDatabase: database.Name},
	); err != nil {
		return err
	}
	for i := range jobs.Items {
		if err := shared.DeleteInBackground(ctx, f.client, &jobs.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

func (f *finalizerReconciler) deletePVC(ctx context.Context, database *datav1.ProteinDatabase) error {
	if database.Spec.Volume.Selector != nil {
		return nil
	}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := f.client.Get(ctx, types.NamespacedName{Name: shared.DatabasePVCName(database.Name), Namespace: database.Namespace}, pvc); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return shared.DeleteInBackground(ctx, f.client, pvc)
}
