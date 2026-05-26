package observer

import (
	downloaderTypes "github.com/kubefold/downloader/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/util"
)

type datasetSlot func(*datav1.ProteinDatabaseStatus) *datav1.ProteinDatabaseDatasetDownloadProgress

var datasetSlots = map[downloaderTypes.Dataset]datasetSlot{
	downloaderTypes.DatasetMGYClusters: func(s *datav1.ProteinDatabaseStatus) *datav1.ProteinDatabaseDatasetDownloadProgress {
		return &s.Datasets.MGYClusters
	},
	downloaderTypes.DatasetBFD: func(s *datav1.ProteinDatabaseStatus) *datav1.ProteinDatabaseDatasetDownloadProgress {
		return &s.Datasets.BFD
	},
	downloaderTypes.DatasetUniRef90: func(s *datav1.ProteinDatabaseStatus) *datav1.ProteinDatabaseDatasetDownloadProgress {
		return &s.Datasets.UniRef90
	},
	downloaderTypes.DatasetUniProt: func(s *datav1.ProteinDatabaseStatus) *datav1.ProteinDatabaseDatasetDownloadProgress {
		return &s.Datasets.UniProt
	},
	downloaderTypes.DatasetPDB: func(s *datav1.ProteinDatabaseStatus) *datav1.ProteinDatabaseDatasetDownloadProgress {
		return &s.Datasets.PDB
	},
	downloaderTypes.DatasetPDBSeqReq: func(s *datav1.ProteinDatabaseStatus) *datav1.ProteinDatabaseDatasetDownloadProgress {
		return &s.Datasets.PDBSeqReq
	},
	downloaderTypes.DatasetRNACentral: func(s *datav1.ProteinDatabaseStatus) *datav1.ProteinDatabaseDatasetDownloadProgress {
		return &s.Datasets.RNACentral
	},
	downloaderTypes.DatasetNT: func(s *datav1.ProteinDatabaseStatus) *datav1.ProteinDatabaseDatasetDownloadProgress {
		return &s.Datasets.NT
	},
	downloaderTypes.DatasetRFam: func(s *datav1.ProteinDatabaseStatus) *datav1.ProteinDatabaseDatasetDownloadProgress {
		return &s.Datasets.RFam
	},
}

type ProgressUpdater interface {
	Update(status *datav1.ProteinDatabaseStatus, entry LogEntry)
}

type progressUpdater struct{}

func NewProgressUpdater() ProgressUpdater {
	return &progressUpdater{}
}

func (u *progressUpdater) Update(status *datav1.ProteinDatabaseStatus, entry LogEntry) {
	if entry.DatasetName == "" {
		return
	}
	slot, ok := datasetSlots[entry.Dataset()]
	if !ok {
		return
	}
	progress := computeProgress(slot(status), entry)
	*slot(status) = progress
}

func computeProgress(previous *datav1.ProteinDatabaseDatasetDownloadProgress, entry LogEntry) datav1.ProteinDatabaseDatasetDownloadProgress {
	totalSize := entry.Dataset().Size()
	progress := datav1.ProteinDatabaseDatasetDownloadProgress{
		DownloadStatus: classifyStatus(entry.Size, totalSize),
		Size:           entry.Size,
		TotalSize:      totalSize,
		Progress:       util.FormatPercentage(entry.Size, totalSize),
		LastUpdate:     &metav1.Time{Time: entry.Time},
	}
	if previous.LastUpdate != nil {
		progress.Delta = progress.Size - previous.Size
		progress.DeltaDuration = &metav1.Duration{Duration: progress.LastUpdate.Sub(previous.LastUpdate.Time)}
		progress.DownloadSpeed = util.FormatSpeed(util.CalculateDownloadSpeed(progress.Delta, progress.DeltaDuration.Duration))
	}
	if progress.DownloadStatus == datav1.ProteinDatabaseDownloadStatusCompleted {
		progress.Delta = 0
		progress.DeltaDuration = nil
		progress.DownloadSpeed = ""
	}
	return progress
}

func classifyStatus(size, totalSize int64) datav1.ProteinDatabaseDownloadStatus {
	switch {
	case size == 0:
		return datav1.ProteinDatabaseDownloadStatusNotStarted
	case size < totalSize:
		return datav1.ProteinDatabaseDownloadStatusDownloading
	default:
		return datav1.ProteinDatabaseDownloadStatusCompleted
	}
}
