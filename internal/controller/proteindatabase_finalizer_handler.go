package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	datav1 "github.com/kubefold/operator/api/v1"
)

type FinalizerHandler struct {
	client client.Client
	scheme *runtime.Scheme
}

func (f *FinalizerHandler) handleDeletion(ctx context.Context, pd *datav1.ProteinDatabase) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if controllerutil.ContainsFinalizer(pd, ProteinDatabaseFinalizer) {
		if err := f.cleanupResources(ctx, pd); err != nil {
			log.Error(err, "Failed to clean up resources")
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(pd, ProteinDatabaseFinalizer)
		if err := f.client.Update(ctx, pd); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (f *FinalizerHandler) ensureFinalizer(ctx context.Context, pd *datav1.ProteinDatabase) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	controllerutil.AddFinalizer(pd, ProteinDatabaseFinalizer)
	if err := f.client.Update(ctx, pd); err != nil {
		log.Error(err, "Failed to add finalizer")
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (f *FinalizerHandler) cleanupResources(ctx context.Context, pd *datav1.ProteinDatabase) error {
	return nil
}
