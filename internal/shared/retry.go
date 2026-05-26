package shared

import (
	"context"

	"k8s.io/client-go/util/retry"
)

func RetryOnConflict(ctx context.Context, fn func() error) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return fn()
	})
}
