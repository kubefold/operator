package prediction

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestToAffinityPreservesAllOperators(t *testing.T) {
	translator := NewNodeSelectorTranslator()

	selector := &corev1.NodeSelector{
		NodeSelectorTerms: []corev1.NodeSelectorTerm{
			{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{Key: "zone", Operator: corev1.NodeSelectorOpIn, Values: []string{"a", "b", "c"}},
					{Key: "role", Operator: corev1.NodeSelectorOpNotIn, Values: []string{"control-plane"}},
					{Key: "gpu", Operator: corev1.NodeSelectorOpExists},
					{Key: "spot", Operator: corev1.NodeSelectorOpDoesNotExist},
					{Key: "cpu", Operator: corev1.NodeSelectorOpGt, Values: []string{"4"}},
					{Key: "memory", Operator: corev1.NodeSelectorOpLt, Values: []string{"64"}},
				},
				MatchFields: []corev1.NodeSelectorRequirement{
					{Key: "metadata.name", Operator: corev1.NodeSelectorOpIn, Values: []string{"node-1"}},
				},
			},
		},
	}

	affinity := translator.ToAffinity(selector)
	if affinity == nil {
		t.Fatal("expected non-nil affinity")
	}
	if affinity.NodeAffinity == nil || affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		t.Fatal("expected RequiredDuringSchedulingIgnoredDuringExecution to be populated")
	}
	terms := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	if len(terms) != 1 {
		t.Fatalf("expected 1 term, got %d", len(terms))
	}
	if len(terms[0].MatchExpressions) != 6 {
		t.Fatalf("expected 6 match expressions, got %d", len(terms[0].MatchExpressions))
	}
	if got := terms[0].MatchExpressions[0].Values; len(got) != 3 {
		t.Fatalf("expected 3 values for In operator, got %d", len(got))
	}
	if len(terms[0].MatchFields) != 1 {
		t.Fatalf("expected 1 match field, got %d", len(terms[0].MatchFields))
	}
}

func TestToAffinityNilOrEmpty(t *testing.T) {
	translator := NewNodeSelectorTranslator()
	if affinity := translator.ToAffinity(nil); affinity != nil {
		t.Fatal("expected nil affinity for nil selector")
	}
	if affinity := translator.ToAffinity(&corev1.NodeSelector{}); affinity != nil {
		t.Fatal("expected nil affinity for empty terms")
	}
}
