package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProteinConformationPredictionProtein struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Sequence string `json:"sequence"`
}

type ProteinConformationPredictionModel struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Seeds []int `json:"seeds,omitempty"`
}

type ProteinConformationPredictionDestination struct {
	S3 string `json:"s3"`
}

type ProteinConformationPredictionNotifications struct {
	SMS []string `json:"sms,omitempty"`
}

type ProteinConformationPredictionJob struct {
	SearchNodeSelector     v1.NodeSelector `json:"searchNodeSelector,omitempty"`
	PredictionNodeSelector v1.NodeSelector `json:"predictionNodeSelector,omitempty"`
}

type ProteinConformationPredictionSpec struct {
	Protein       ProteinConformationPredictionProtein       `json:"protein"`
	Model         ProteinConformationPredictionModel         `json:"model,omitempty"`
	Destination   ProteinConformationPredictionDestination   `json:"destination"`
	Notifications ProteinConformationPredictionNotifications `json:"notify,omitempty"`
	Job           ProteinConformationPredictionJob           `json:"job,omitempty"`
	Database      string                                     `json:"database"`
}

type ProteinConformationPredictionStatusPhase string

const (
	ProteinConformationPredictionStatusPhaseNotStarted ProteinConformationPredictionStatusPhase = "NotStarted"
	ProteinConformationPredictionStatusPhaseAligning   ProteinConformationPredictionStatusPhase = "Aligning"
	ProteinConformationPredictionStatusPhasePredicting ProteinConformationPredictionStatusPhase = "Predicting"
	ProteinConformationPredictionStatusPhaseCompleted  ProteinConformationPredictionStatusPhase = "Completed"
	ProteinConformationPredictionStatusPhaseFailed     ProteinConformationPredictionStatusPhase = "Failed"
)

type ProteinConformationPredictionStatus struct {
	Phase          ProteinConformationPredictionStatusPhase `json:"phase,omitempty"`
	SequencePrefix string                                   `json:"sequencePrefix,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Sequence",type=string,JSONPath=`.status.sequencePrefix`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type ProteinConformationPrediction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProteinConformationPredictionSpec   `json:"spec,omitempty"`
	Status ProteinConformationPredictionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ProteinConformationPredictionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProteinConformationPrediction `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProteinConformationPrediction{}, &ProteinConformationPredictionList{})
}
