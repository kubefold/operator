package database

import (
	"testing"

	datav1 "github.com/kubefold/operator/api/v1"
)

func TestSizerAddsBuffer(t *testing.T) {
	sizer := NewSizer(NewDatasetEnumerator())
	empty := &datav1.ProteinDatabase{}
	if got := sizer.RequestedGigabytes(empty); got != sizeBufferGigabytes {
		t.Fatalf("expected buffer-only size %d for empty spec, got %d", sizeBufferGigabytes, got)
	}
}

func TestSizerSumsEnabledDatasets(t *testing.T) {
	sizer := NewSizer(NewDatasetEnumerator())
	database := &datav1.ProteinDatabase{
		Spec: datav1.ProteinDatabaseSpec{
			Datasets: datav1.ProteinDatabaseDatasetSelection{BFD: true},
		},
	}
	got := sizer.RequestedGigabytes(database)
	if got < sizeBufferGigabytes {
		t.Fatalf("size with one dataset must include buffer: %d", got)
	}
}
