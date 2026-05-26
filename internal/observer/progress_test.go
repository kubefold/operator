package observer

import (
	"testing"

	downloaderTypes "github.com/kubefold/downloader/pkg/types"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/database"
)

func TestDispatchTableCoversAllDatasets(t *testing.T) {
	enumerator := database.NewDatasetEnumerator()
	for _, dataset := range enumerator.All() {
		if _, ok := datasetSlots[dataset]; !ok {
			t.Fatalf("dispatch table missing slot for dataset %q", dataset)
		}
	}
	if len(datasetSlots) != len(enumerator.All()) {
		t.Fatalf("dispatch table has %d entries, enumerator declares %d", len(datasetSlots), len(enumerator.All()))
	}
}

func TestProgressUpdaterIgnoresUnknownDataset(t *testing.T) {
	updater := NewProgressUpdater()
	status := &datav1.ProteinDatabaseStatus{}
	updater.Update(status, LogEntry{DatasetName: "unknown_dataset", Size: 100})
}

func TestProgressUpdaterEmptyDatasetName(t *testing.T) {
	updater := NewProgressUpdater()
	status := &datav1.ProteinDatabaseStatus{}
	updater.Update(status, LogEntry{})
}

func TestProgressUpdaterClassifiesStatus(t *testing.T) {
	updater := NewProgressUpdater()
	status := &datav1.ProteinDatabaseStatus{}
	updater.Update(status, LogEntry{
		DatasetName: string(downloaderTypes.DatasetBFD),
		Size:        0,
	})
	if status.Datasets.BFD.DownloadStatus != datav1.ProteinDatabaseDownloadStatusNotStarted {
		t.Fatalf("expected NotStarted, got %q", status.Datasets.BFD.DownloadStatus)
	}
	updater.Update(status, LogEntry{
		DatasetName: string(downloaderTypes.DatasetBFD),
		Size:        downloaderTypes.DatasetBFD.Size() / 2,
	})
	if status.Datasets.BFD.DownloadStatus != datav1.ProteinDatabaseDownloadStatusDownloading {
		t.Fatalf("expected Downloading, got %q", status.Datasets.BFD.DownloadStatus)
	}
	updater.Update(status, LogEntry{
		DatasetName: string(downloaderTypes.DatasetBFD),
		Size:        downloaderTypes.DatasetBFD.Size(),
	})
	if status.Datasets.BFD.DownloadStatus != datav1.ProteinDatabaseDownloadStatusCompleted {
		t.Fatalf("expected Completed, got %q", status.Datasets.BFD.DownloadStatus)
	}
}
