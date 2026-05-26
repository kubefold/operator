package database

import (
	"slices"
	"testing"

	downloaderTypes "github.com/kubefold/downloader/pkg/types"

	datav1 "github.com/kubefold/operator/api/v1"
)

func TestDatasetEnumeratorAll(t *testing.T) {
	enumerator := NewDatasetEnumerator()
	all := enumerator.All()
	if len(all) != 9 {
		t.Fatalf("expected 9 datasets, got %d", len(all))
	}
}

func TestDatasetEnumeratorFromSpec(t *testing.T) {
	enumerator := NewDatasetEnumerator()
	database := &datav1.ProteinDatabase{
		Spec: datav1.ProteinDatabaseSpec{
			Datasets: datav1.ProteinDatabaseDatasetSelection{
				BFD: true, UniRef90: true, RFam: true,
			},
		},
	}
	got := enumerator.FromSpec(database)
	wantContains := []downloaderTypes.Dataset{
		downloaderTypes.DatasetBFD,
		downloaderTypes.DatasetUniRef90,
		downloaderTypes.DatasetRFam,
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 datasets, got %d", len(got))
	}
	for _, expected := range wantContains {
		if !slices.Contains(got, expected) {
			t.Fatalf("expected dataset %v in result", expected)
		}
	}
}

func TestDatasetEnumeratorEmpty(t *testing.T) {
	enumerator := NewDatasetEnumerator()
	got := enumerator.FromSpec(&datav1.ProteinDatabase{})
	if len(got) != 0 {
		t.Fatalf("expected empty result for empty spec, got %d", len(got))
	}
}
