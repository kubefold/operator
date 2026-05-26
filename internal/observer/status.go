package observer

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/shared"
	"github.com/kubefold/operator/internal/util"
)

type StatusPublisher interface {
	Publish(ctx context.Context, database *datav1.ProteinDatabase, newStatus *datav1.ProteinDatabaseStatus) error
}

type statusPublisher struct {
	client     client.Client
	aggregator MetricsAggregator
}

func NewStatusPublisher(c client.Client, aggregator MetricsAggregator) StatusPublisher {
	return &statusPublisher{client: c, aggregator: aggregator}
}

func (p *statusPublisher) Publish(ctx context.Context, database *datav1.ProteinDatabase, newStatus *datav1.ProteinDatabaseStatus) error {
	return shared.RetryOnConflict(ctx, func() error {
		latest := &datav1.ProteinDatabase{}
		if err := p.client.Get(ctx, types.NamespacedName{Name: database.Name, Namespace: database.Namespace}, latest); err != nil {
			return fmt.Errorf("failed to get latest ProteinDatabase: %w", err)
		}
		p.aggregator.Aggregate(newStatus)
		volumeName := latest.Status.VolumeName
		latest.Status = *newStatus
		latest.Status.VolumeName = volumeName
		latest.Status.LastUpdate = util.GetNow()
		return p.client.Status().Update(ctx, latest)
	})
}
