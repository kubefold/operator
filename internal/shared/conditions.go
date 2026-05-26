package shared

import (
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetCondition(conditions *[]metav1.Condition, condition metav1.Condition) {
	if condition.LastTransitionTime.IsZero() {
		condition.LastTransitionTime = metav1.Now()
	}
	apimeta.SetStatusCondition(conditions, condition)
}

func FindCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	return apimeta.FindStatusCondition(conditions, conditionType)
}
