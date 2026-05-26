package prediction

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/shared"
)

const (
	ConditionTypeReady            = "Ready"
	ConditionTypeSearchSucceeded  = "SearchSucceeded"
	ConditionTypePredictSucceeded = "PredictSucceeded"
	ConditionTypeUploadSucceeded  = "UploadSucceeded"
	ConditionTypeValidationFailed = "ValidationFailed"
)

const (
	ReasonPhaseProgressed  = "PhaseProgressed"
	ReasonPhaseFailed      = "PhaseFailed"
	ReasonValidationFailed = "ValidationFailed"
	ReasonJobTimeout       = "JobTimeout"
	ReasonRetriesExhausted = "RetriesExhausted"
	ReasonCompleted        = "Completed"
)

type ConditionsManager interface {
	Set(status *datav1.ProteinConformationPredictionStatus, condition metav1.Condition)
	IsTrue(status *datav1.ProteinConformationPredictionStatus, conditionType string) bool
}

type conditionsManager struct{}

func NewConditionsManager() ConditionsManager {
	return &conditionsManager{}
}

func (m *conditionsManager) Set(status *datav1.ProteinConformationPredictionStatus, condition metav1.Condition) {
	shared.SetCondition(&status.Conditions, condition)
	if updated := shared.FindCondition(status.Conditions, condition.Type); updated != nil {
		stamp := updated.LastTransitionTime
		status.LastTransitionTime = &stamp
	}
}

func (m *conditionsManager) IsTrue(status *datav1.ProteinConformationPredictionStatus, conditionType string) bool {
	return shared.IsConditionTrue(status.Conditions, conditionType)
}
