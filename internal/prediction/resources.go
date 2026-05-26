package prediction

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	datav1 "github.com/kubefold/operator/api/v1"
)

type phaseResources struct {
	cpu    string
	memory string
}

const (
	phaseSearchKey  = "search"
	phasePredictKey = "predict"
	phaseUploadKey  = "upload"
)

var profileResourceMap = map[datav1.ProteinConformationPredictionProfile]map[string]phaseResources{
	datav1.ProteinConformationPredictionProfileSmall: {
		phaseSearchKey:  {cpu: "1500m", memory: "6Gi"},
		phasePredictKey: {cpu: "1", memory: "4Gi"},
		phaseUploadKey:  {cpu: "100m", memory: "256Mi"},
	},
	datav1.ProteinConformationPredictionProfileMedium: {
		phaseSearchKey:  {cpu: "3", memory: "12Gi"},
		phasePredictKey: {cpu: "1", memory: "8Gi"},
		phaseUploadKey:  {cpu: "100m", memory: "256Mi"},
	},
	datav1.ProteinConformationPredictionProfileLarge: {
		phaseSearchKey:  {cpu: "6", memory: "48Gi"},
		phasePredictKey: {cpu: "2", memory: "16Gi"},
		phaseUploadKey:  {cpu: "100m", memory: "256Mi"},
	},
}

func resourcesForPhase(profile datav1.ProteinConformationPredictionProfile, phaseKey string) corev1.ResourceRequirements {
	chosenProfile := profile
	if chosenProfile == "" {
		chosenProfile = datav1.ProteinConformationPredictionProfileMedium
	}
	phases, ok := profileResourceMap[chosenProfile]
	if !ok {
		phases = profileResourceMap[datav1.ProteinConformationPredictionProfileMedium]
	}
	requested := phases[phaseKey]
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(requested.cpu),
			corev1.ResourceMemory: resource.MustParse(requested.memory),
		},
	}
}
