package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProteinConformationPredictionProtein struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=10000
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Sequence string `json:"sequence"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	ID []string `json:"id"`
}

type ProteinConformationPredictionModelWeights struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=2048
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
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=63
	Bucket string `json:"bucket"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Region string `json:"region"`
}

type ProteinConformationPredictionDestination struct {
	S3 ProteinConformationPredictionDestinationS3 `json:"s3"`
}

type ProteinConformationPredictionNotifications struct {
	SMS    []string `json:"sms,omitempty"`
	Region string   `json:"region"`
}

type ProteinConformationPredictionProfile string

const (
	ProteinConformationPredictionProfileSmall  ProteinConformationPredictionProfile = "small"
	ProteinConformationPredictionProfileMedium ProteinConformationPredictionProfile = "medium"
	ProteinConformationPredictionProfileLarge  ProteinConformationPredictionProfile = "large"
)

type ProteinConformationPredictionJob struct {
	SearchNodeSelector     v1.NodeSelector                      `json:"searchNodeSelector,omitempty"`
	PredictionNodeSelector v1.NodeSelector                      `json:"predictionNodeSelector,omitempty"`
	Profile                ProteinConformationPredictionProfile `json:"profile,omitempty"`
}

type ProteinConformationPredictionSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	Protein ProteinConformationPredictionProtein `json:"protein"`
	// +kubebuilder:validation:Required
	Model ProteinConformationPredictionModel `json:"model"`
	// +kubebuilder:validation:Required
	Destination   ProteinConformationPredictionDestination   `json:"destination"`
	Notifications ProteinConformationPredictionNotifications `json:"notify,omitempty"`
	Job           ProteinConformationPredictionJob           `json:"job,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Database     string `json:"database"`
	StorageClass string `json:"storageClass,omitempty"`
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
	// Deprecated: use SearchRetryCount, PredictRetryCount or UploadRetryCount.
	RetryCount         int32        `json:"retryCount,omitempty"`
	SearchRetryCount   int32        `json:"searchRetryCount,omitempty"`
	PredictRetryCount  int32        `json:"predictRetryCount,omitempty"`
	UploadRetryCount   int32        `json:"uploadRetryCount,omitempty"`
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
	// +listType=map
	// +listMapKey=type
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Sequence",type=string,JSONPath=`.status.sequencePrefix`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
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
