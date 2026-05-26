package shared

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeleteInBackground(ctx context.Context, c client.Client, object client.Object) error {
	propagation := metav1.DeletePropagationBackground
	if err := c.Delete(ctx, object, &client.DeleteOptions{PropagationPolicy: &propagation}); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}
