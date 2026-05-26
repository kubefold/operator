package prediction

import (
	"testing"

	datav1 "github.com/kubefold/operator/api/v1"
)

func TestRetryPolicyPerPhaseIndependence(t *testing.T) {
	policy := NewRetryPolicy(3)
	status := &datav1.ProteinConformationPredictionStatus{}

	policy.Increment(status, PhaseSearch)
	policy.Increment(status, PhaseSearch)
	if status.SearchRetryCount != 2 {
		t.Fatalf("SearchRetryCount = %d, want 2", status.SearchRetryCount)
	}
	if status.PredictRetryCount != 0 || status.UploadRetryCount != 0 {
		t.Fatal("Predict/Upload counters should not be touched by Search increment")
	}

	policy.Increment(status, PhasePredict)
	if status.PredictRetryCount != 1 {
		t.Fatalf("PredictRetryCount = %d, want 1", status.PredictRetryCount)
	}
}

func TestRetryPolicyAtLimit(t *testing.T) {
	policy := NewRetryPolicy(2)
	status := &datav1.ProteinConformationPredictionStatus{}

	if policy.AtLimit(status, PhaseUpload) {
		t.Fatal("should not be at limit initially")
	}
	policy.Increment(status, PhaseUpload)
	if policy.AtLimit(status, PhaseUpload) {
		t.Fatal("should not be at limit after 1 retry with max=2")
	}
	policy.Increment(status, PhaseUpload)
	if !policy.AtLimit(status, PhaseUpload) {
		t.Fatal("should be at limit after 2 retries with max=2")
	}
}

func TestRetryPolicyCounter(t *testing.T) {
	policy := NewRetryPolicy(5)
	status := &datav1.ProteinConformationPredictionStatus{
		SearchRetryCount:  3,
		PredictRetryCount: 1,
		UploadRetryCount:  0,
	}
	if got := policy.Counter(status, PhaseSearch); got != 3 {
		t.Fatalf("Counter(Search) = %d, want 3", got)
	}
	if got := policy.Counter(status, PhasePredict); got != 1 {
		t.Fatalf("Counter(Predict) = %d, want 1", got)
	}
	if got := policy.Counter(status, PhaseUpload); got != 0 {
		t.Fatalf("Counter(Upload) = %d, want 0", got)
	}
}
