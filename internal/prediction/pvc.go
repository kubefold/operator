package prediction

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/shared"
)

const (
	DefaultStorageClass = "fsx-sc"
	predictionPVCSize   = "10Gi"
)

type PVCBuilder interface {
	Build(prediction *datav1.ProteinConformationPrediction, pvcName string) *corev1.PersistentVolumeClaim
}

type pvcBuilder struct{}

func NewPVCBuilder() PVCBuilder {
	return &pvcBuilder{}
}

func (b *pvcBuilder) Build(prediction *datav1.ProteinConformationPrediction, pvcName string) *corev1.PersistentVolumeClaim {
	storageClass := resolveStorageClass(prediction)

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: prediction.Namespace,
			Labels:    shared.PredictionPVCLabels(prediction.Name),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(predictionPVCSize),
				},
			},
			StorageClassName: &storageClass,
		},
	}

	if prediction.Spec.Model.Volume.Selector != nil {
		pvc.Spec.Selector = prediction.Spec.Model.Volume.Selector
	}

	return pvc
}

func resolveStorageClass(prediction *datav1.ProteinConformationPrediction) string {
	if prediction.Spec.Model.Volume.StorageClassName != nil && *prediction.Spec.Model.Volume.StorageClassName != "" {
		return *prediction.Spec.Model.Volume.StorageClassName
	}
	if prediction.Spec.StorageClass != "" {
		return prediction.Spec.StorageClass
	}
	return DefaultStorageClass
}
