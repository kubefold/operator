package prediction

import (
	datav1 "github.com/kubefold/operator/api/v1"
)

const MaxRetries = int32(3)

type Phase string

const (
	PhaseSearch  Phase = "search"
	PhasePredict Phase = "predict"
	PhaseUpload  Phase = "upload"
)

type RetryPolicy interface {
	Counter(status *datav1.ProteinConformationPredictionStatus, phase Phase) int32
	Increment(status *datav1.ProteinConformationPredictionStatus, phase Phase)
	AtLimit(status *datav1.ProteinConformationPredictionStatus, phase Phase) bool
}

type retryPolicy struct {
	max int32
}

func NewRetryPolicy(max int32) RetryPolicy {
	return &retryPolicy{max: max}
}

func (p *retryPolicy) Counter(status *datav1.ProteinConformationPredictionStatus, phase Phase) int32 {
	switch phase {
	case PhaseSearch:
		return status.SearchRetryCount
	case PhasePredict:
		return status.PredictRetryCount
	case PhaseUpload:
		return status.UploadRetryCount
	}
	return 0
}

func (p *retryPolicy) Increment(status *datav1.ProteinConformationPredictionStatus, phase Phase) {
	switch phase {
	case PhaseSearch:
		status.SearchRetryCount++
	case PhasePredict:
		status.PredictRetryCount++
	case PhaseUpload:
		status.UploadRetryCount++
	}
}

func (p *retryPolicy) AtLimit(status *datav1.ProteinConformationPredictionStatus, phase Phase) bool {
	return p.Counter(status, phase) >= p.max
}
