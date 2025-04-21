package controller

import (
	"context"
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
)


type VolumeHandler struct {
	client client.Client
	scheme *runtime.Scheme
}


func (v *VolumeHandler) ensurePVC(ctx context.Context, pd *datav1.ProteinDatabase) (*corev1.PersistentVolumeClaim, *ctrl.Result, error) {
	log := logf.FromContext(ctx)
	pvcName := pd.Name + PersistentVolumeClaimNameSuffix

	pvc := &corev1.PersistentVolumeClaim{}
	err := v.client.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: pd.Namespace}, pvc)

	if err != nil && errors.IsNotFound(err) {
		pvc, err = v.createPVC(ctx, pd)
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


func (v *VolumeHandler) createPVC(ctx context.Context, pd *datav1.ProteinDatabase) (*corev1.PersistentVolumeClaim, error) {
	pvcName := pd.Name + PersistentVolumeClaimNameSuffix

	labels := pd.Spec.Volume.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app.kubernetes.io/name"] = "proteindatabase"
	labels["app.kubernetes.io/instance"] = pd.Name
	labels["app.kubernetes.io/managed-by"] = "kubefold-operator"

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: pd.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			StorageClassName: pd.Spec.Volume.StorageClassName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(PersistentVolumeClaimSize),
				},
			},
		},
	}

	if pd.Spec.Volume.Selector != nil {
		pvc.Spec.Selector = pd.Spec.Volume.Selector
	}

	if err := controllerutil.SetControllerReference(pd, pvc, v.scheme); err != nil {
		return nil, err
	}

	if err := v.client.Create(ctx, pvc); err != nil {
		return nil, err
	}

	return pvc, nil
}
