package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProteinDatabaseVolume struct {
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,12,rep,name=annotations"`

	// +optional
	StorageClassName *string `json:"storageClassName,omitempty" protobuf:"bytes,5,opt,name=storageClassName"`
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,4,opt,name=selector"`
}

type ProteinDatabaseDatasetSelection struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf || (self == true && oldSelf == false)",message="Dataset can only be enabled. Deletion of dataset is not supported yet"
	MGYClusters bool `json:"mgyclusters"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf || (self == true && oldSelf == false)",message="Dataset can only be enabled. Deletion of dataset is not supported yet"
	BFD bool `json:"bfd"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf || (self == true && oldSelf == false)",message="Dataset can only be enabled. Deletion of dataset is not supported yet"
	UniRef90 bool `json:"uniref90"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf || (self == true && oldSelf == false)",message="Dataset can only be enabled. Deletion of dataset is not supported yet"
	UniProt bool `json:"uniprot"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf || (self == true && oldSelf == false)",message="Dataset can only be enabled. Deletion of dataset is not supported yet"
	PDB bool `json:"pdb"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf || (self == true && oldSelf == false)",message="Dataset can only be enabled. Deletion of dataset is not supported yet"
	PDBSeqReq bool `json:"pdbseqreq"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf || (self == true && oldSelf == false)",message="Dataset can only be enabled. Deletion of dataset is not supported yet"
	RNACentral bool `json:"rnacentral"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf || (self == true && oldSelf == false)",message="Dataset can only be enabled. Deletion of dataset is not supported yet"
	NT bool `json:"nt"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf || (self == true && oldSelf == false)",message="Dataset can only be enabled. Deletion of dataset is not supported yet"
	RFam bool `json:"rfam"`
}

type ProteinDatabaseSpec struct {
	Datasets ProteinDatabaseDatasetSelection `json:"datasets,omitempty"`
	Volume   ProteinDatabaseVolume           `json:"volume,omitempty"`
}

type ProteinDatabaseDownloadStatus string

const (
	ProteinDatabaseDownloadStatusNotStarted  ProteinDatabaseDownloadStatus = "NotStarted"
	ProteinDatabaseDownloadStatusDownloading ProteinDatabaseDownloadStatus = "Downloading"
	ProteinDatabaseDownloadStatusCompleted   ProteinDatabaseDownloadStatus = "Completed"
	ProteinDatabaseDownloadStatusFailed      ProteinDatabaseDownloadStatus = "Failed"
)

type ProteinDatabaseDatasetDownloadProgress struct {
	DownloadStatus ProteinDatabaseDownloadStatus `json:"downloadStatus,omitempty"`
	DownloadSpeed  string                        `json:"downloadSpeed,omitempty"`
	Progress       string                        `json:"progress,omitempty"`
	Size           int64                         `json:"size,omitempty"`
	TotalSize      int64                         `json:"totalSize,omitempty"`
	LastUpdate     *metav1.Time                  `json:"lastUpdate,omitempty"`
	Delta          int64                         `json:"delta,omitempty"`
	DeltaDuration  *metav1.Duration              `json:"deltaDuration,omitempty"`
}

type ProteinDatabaseDatasetDownloadStatus struct {
	MGYClusters ProteinDatabaseDatasetDownloadProgress `json:"mgyclusters,omitempty"`
	BFD         ProteinDatabaseDatasetDownloadProgress `json:"bfd,omitempty"`
	UniRef90    ProteinDatabaseDatasetDownloadProgress `json:"uniref90,omitempty"`
	UniProt     ProteinDatabaseDatasetDownloadProgress `json:"uniprot,omitempty"`
	PDB         ProteinDatabaseDatasetDownloadProgress `json:"pdb,omitempty"`
	PDBSeqReq   ProteinDatabaseDatasetDownloadProgress `json:"pdbseqreq,omitempty"`
	RNACentral  ProteinDatabaseDatasetDownloadProgress `json:"rnacentral,omitempty"`
	NT          ProteinDatabaseDatasetDownloadProgress `json:"nt,omitempty"`
	RFam        ProteinDatabaseDatasetDownloadProgress `json:"rfam,omitempty"`
}

type ProteinDatabaseStatus struct {
	DownloadStatus ProteinDatabaseDownloadStatus        `json:"downloadStatus,omitempty"`
	DownloadSpeed  string                               `json:"downloadSpeed,omitempty"`
	VolumeName     string                               `json:"volumeName"`
	Progress       string                               `json:"progress,omitempty"`
	Size           string                               `json:"size,omitempty"`
	TotalSize      string                               `json:"totalSize,omitempty"`
	LastUpdate     *metav1.Time                         `json:"lastUpdate,omitempty"`
	Datasets       ProteinDatabaseDatasetDownloadStatus `json:"datasets,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.downloadStatus`
// +kubebuilder:printcolumn:name="Progress",type=string,JSONPath=`.status.progress`
// +kubebuilder:printcolumn:name="Size",type=string,JSONPath=`.status.size`
// +kubebuilder:printcolumn:name="Total Size",type=string,JSONPath=`.status.totalSize`
// +kubebuilder:printcolumn:name="Download Speed",type=string,JSONPath=`.status.downloadSpeed`
// +kubebuilder:printcolumn:name="Volume",type=string,JSONPath=`.status.volumeName`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type ProteinDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProteinDatabaseSpec   `json:"spec,omitempty"`
	Status ProteinDatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ProteinDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProteinDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProteinDatabase{}, &ProteinDatabaseList{})
}
