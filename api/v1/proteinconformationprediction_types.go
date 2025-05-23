package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProteinConformationPredictionProtein struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Sequence string `json:"sequence"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	ID []string `json:"id"`
}

type ProteinConformationPredictionModelWeights struct {
	HTTP string `json:"http"`
}

type ProteinConformationPredictionModelVolume struct {
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty" protobuf:"bytes,5,opt,name=storageClassName"`
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,4,opt,name=selector"`
}

type ProteinConformationPredictionModel struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Seeds   []int                                     `json:"seeds,omitempty"`
	Weights ProteinConformationPredictionModelWeights `json:"weights"`
	Volume  ProteinConformationPredictionModelVolume  `json:"volume,omitempty"`
}

type ProteinConformationPredictionDestinationS3 struct {
	Bucket string `json:"bucket"`
	Region string `json:"region"`
}

type ProteinConformationPredictionDestination struct {
	S3 ProteinConformationPredictionDestinationS3 `json:"s3"`
}

type ProteinConformationPredictionNotifications struct {
	SMS    []string `json:"sms,omitempty"`
	Region string   `json:"region"`
}

type ProteinConformationPredictionJob struct {
	SearchNodeSelector     v1.NodeSelector `json:"searchNodeSelector,omitempty"`
	PredictionNodeSelector v1.NodeSelector `json:"predictionNodeSelector,omitempty"`
}

type ProteinConformationPredictionSpec struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Protein       ProteinConformationPredictionProtein       `json:"protein"`
	Model         ProteinConformationPredictionModel         `json:"model,omitempty"`
	Destination   ProteinConformationPredictionDestination   `json:"destination"`
	Notifications ProteinConformationPredictionNotifications `json:"notify,omitempty"`
	Job           ProteinConformationPredictionJob           `json:"job,omitempty"`
	Database      string                                     `json:"database"`
	StorageClass  string                                     `json:"storageClass,omitempty"`
}

type ProteinConformationPredictionStatusPhase string

const (
	ProteinConformationPredictionStatusPhaseNotStarted         ProteinConformationPredictionStatusPhase = "NotStarted"
	ProteinConformationPredictionStatusPhaseAligning           ProteinConformationPredictionStatusPhase = "Aligning"
	ProteinConformationPredictionStatusPhasePredicting         ProteinConformationPredictionStatusPhase = "Predicting"
	ProteinConformationPredictionStatusPhaseUploadingArtifacts ProteinConformationPredictionStatusPhase = "UploadingArtifacts"
	ProteinConformationPredictionStatusPhaseCompleted          ProteinConformationPredictionStatusPhase = "Completed"
	ProteinConformationPredictionStatusPhaseFailed             ProteinConformationPredictionStatusPhase = "Failed"
)

type ProteinConformationPredictionStatus struct {
	Phase          ProteinConformationPredictionStatusPhase `json:"phase,omitempty"`
	SequencePrefix string                                   `json:"sequencePrefix,omitempty"`
	Error          string                                   `json:"error,omitempty"`
	RetryCount     int32                                    `json:"retryCount,omitempty"`
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
