package observer

import (
	"time"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/database"
	"github.com/kubefold/operator/internal/util"
)

type MetricsAggregator interface {
	Aggregate(status *datav1.ProteinDatabaseStatus)
}

type metricsAggregator struct {
	enumerator database.DatasetEnumerator
}

func NewMetricsAggregator(enumerator database.DatasetEnumerator) MetricsAggregator {
	return &metricsAggregator{enumerator: enumerator}
}

func (a *metricsAggregator) Aggregate(status *datav1.ProteinDatabaseStatus) {
	var size, totalSize, delta int64
	for _, dataset := range a.enumerator.All() {
		slot, ok := datasetSlots[dataset]
		if !ok {
			continue
		}
		progress := slot(status)
		size += progress.Size
		totalSize += progress.TotalSize
		if progress.DeltaDuration != nil && progress.DeltaDuration.Duration > 0 {
			delta += progress.Delta
		}
	}
	status.Size = util.FormatSize(size)
	status.TotalSize = util.FormatSize(totalSize)
	status.Progress = util.FormatPercentage(size, totalSize)
	status.DownloadSpeed = util.FormatSpeed(util.CalculateDownloadSpeed(delta, time.Second))
	status.DownloadStatus = classifyStatus(size, totalSize)
}
