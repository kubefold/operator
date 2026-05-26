package database

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/shared"
)

const ReconcileInterval = 10 * time.Second

type VolumeReconciler interface {
	Ensure(ctx context.Context, database *datav1.ProteinDatabase) (*corev1.PersistentVolumeClaim, *ctrl.Result, error)
}

type volumeReconciler struct {
	client client.Client
	scheme *runtime.Scheme
	sizer  Sizer
}

func NewVolumeReconciler(c client.Client, scheme *runtime.Scheme, sizer Sizer) VolumeReconciler {
	return &volumeReconciler{client: c, scheme: scheme, sizer: sizer}
}

func (v *volumeReconciler) Ensure(ctx context.Context, database *datav1.ProteinDatabase) (*corev1.PersistentVolumeClaim, *ctrl.Result, error) {
	log := logf.FromContext(ctx)
	pvcName := shared.DatabasePVCName(database.Name)

	pvc := &corev1.PersistentVolumeClaim{}
	err := v.client.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: database.Namespace}, pvc)
	if errors.IsNotFound(err) {
		pvc, err = v.create(ctx, database, pvcName)
		if err != nil {
			log.Error(err, "Failed to create PVC")
			return nil, nil, err
		}
		log.Info("Created new PVC", "pvcName", pvc.Name)
	} else if err != nil {
		log.Error(err, "Failed to get PVC")
		return nil, nil, err
	}

	if pvc.Status.Phase != corev1.ClaimBound {
		log.Info("PVC is not bound yet", "pvcName", pvc.Name, "phase", pvc.Status.Phase)
		result := ctrl.Result{Requeue: true, RequeueAfter: ReconcileInterval}
		return pvc, &result, nil
	}
	return pvc, nil, nil
}

func (v *volumeReconciler) create(ctx context.Context, database *datav1.ProteinDatabase, pvcName string) (*corev1.PersistentVolumeClaim, error) {
	labels := shared.MergeLabels(
		database.Spec.Volume.Labels,
		shared.DatabaseLabels(database.Name, "proteindatabase"),
	)

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pvcName,
			Namespace:   database.Namespace,
			Labels:      labels,
			Annotations: database.Spec.Volume.Annotations,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			StorageClassName: database.Spec.Volume.StorageClassName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", v.sizer.RequestedGigabytes(database))),
				},
			},
		},
	}
	if database.Spec.Volume.Selector != nil {
		pvc.Spec.Selector = database.Spec.Volume.Selector
	}
	if err := controllerutil.SetControllerReference(database, pvc, v.scheme); err != nil {
		return nil, err
	}
	if err := v.client.Create(ctx, pvc); err != nil {
		return nil, err
	}
	return pvc, nil
}
