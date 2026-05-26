package shared

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeleteInBackground(ctx context.Context, c client.Client, object client.Object) error {
	propagation := metav1.DeletePropagationBackground
	return client.IgnoreNotFound(c.Delete(ctx, object, &client.DeleteOptions{PropagationPolicy: &propagation}))
}
