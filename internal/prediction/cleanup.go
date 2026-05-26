package prediction

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubefold/operator/internal/shared"
)

type ResourceCleaner interface {
	DeleteJobsForPrediction(ctx context.Context, name, namespace string) error
	DeletePVCForPrediction(ctx context.Context, name, namespace string) error
	DeleteCompletedJobs(ctx context.Context, name, namespace string) error
}

type resourceCleaner struct {
	client client.Client
}

func NewResourceCleaner(c client.Client) ResourceCleaner {
	return &resourceCleaner{client: c}
}

func (r *resourceCleaner) DeleteJobsForPrediction(ctx context.Context, name, namespace string) error {
	jobs, err := r.listJobs(ctx, name, namespace)
	if err != nil {
		return err
	}
	for i := range jobs.Items {
		if err := shared.DeleteInBackground(ctx, r.client, &jobs.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *resourceCleaner) DeletePVCForPrediction(ctx context.Context, name, namespace string) error {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: shared.PredictionDataPVCName(name), Namespace: namespace}, pvc); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return shared.DeleteInBackground(ctx, r.client, pvc)
}

func (r *resourceCleaner) DeleteCompletedJobs(ctx context.Context, name, namespace string) error {
	jobs, err := r.listJobs(ctx, name, namespace)
	if err != nil {
		return err
	}
	for i := range jobs.Items {
		if jobs.Items[i].Status.Succeeded > 0 {
			if err := shared.DeleteInBackground(ctx, r.client, &jobs.Items[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *resourceCleaner) listJobs(ctx context.Context, name, namespace string) (*batchv1.JobList, error) {
	jobs := &batchv1.JobList{}
	if err := r.client.List(ctx, jobs,
		client.InNamespace(namespace),
		client.MatchingLabels{shared.LabelPrediction: name},
	); err != nil {
		return nil, err
	}
	return jobs, nil
}
