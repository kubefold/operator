package prediction

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/shared"
)

type StatusWriter interface {
	Update(ctx context.Context, prediction *datav1.ProteinConformationPrediction, mutate func(*datav1.ProteinConformationPrediction)) error
}

type statusWriter struct {
	client client.Client
}

func NewStatusWriter(c client.Client) StatusWriter {
	return &statusWriter{client: c}
}

func (w *statusWriter) Update(ctx context.Context, prediction *datav1.ProteinConformationPrediction, mutate func(*datav1.ProteinConformationPrediction)) error {
	return shared.RetryOnConflict(ctx, func() error {
		latest := &datav1.ProteinConformationPrediction{}
		if err := w.client.Get(ctx, types.NamespacedName{Name: prediction.Name, Namespace: prediction.Namespace}, latest); err != nil {
			return err
		}
		mutate(latest)
		if err := w.client.Status().Update(ctx, latest); err != nil {
			return err
		}
		prediction.Status = latest.Status
		prediction.ResourceVersion = latest.ResourceVersion
		return nil
	})
}
