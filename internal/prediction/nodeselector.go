package prediction

import (
	corev1 "k8s.io/api/core/v1"
)

type NodeSelectorTranslator interface {
	ToAffinity(selector *corev1.NodeSelector) *corev1.Affinity
}

type nodeSelectorTranslator struct{}

func NewNodeSelectorTranslator() NodeSelectorTranslator {
	return &nodeSelectorTranslator{}
}

func (t *nodeSelectorTranslator) ToAffinity(selector *corev1.NodeSelector) *corev1.Affinity {
	if selector == nil || len(selector.NodeSelectorTerms) == 0 {
		return nil
	}
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: selector.DeepCopy(),
		},
	}
}
