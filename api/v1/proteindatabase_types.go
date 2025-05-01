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
	MHYClusters bool `json:"mgy_clusters_2022_05"`
	BFD         bool `json:"bfd-first_non_consensus_sequences"`
	UniRef90    bool `json:"uniref90_2022_05"`
	UniProt     bool `json:"uniprot_all_2021_04"`
	PDB         bool `json:"pdb_2022_09_28_mmcif_files"`
	PDBSeqReq   bool `json:"pdb_seqres_2022_09_28"`
	RNACentral  bool `json:"rnacentral_active_seq_id_90_cov_80_linclust"`
	NT          bool `json:"nt_rna_2023_02_23_clust_seq_id_90_cov_80_rep_seq"`
	RFam        bool `json:"rfam_14_9_clust_seq_id_90_cov_80_rep_seq"`
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
	Size           string                        `json:"size,omitempty"`
	TotalSize      string                        `json:"totalSize,omitempty"`
	LastUpdate     metav1.Time                   `json:"lastUpdate"`
}

type ProteinDatabaseDatasetDownloadStatus struct {
	MHYClusters ProteinDatabaseDatasetDownloadProgress `json:"mgy_clusters_2022_05,omitempty"`
	BFD         ProteinDatabaseDatasetDownloadProgress `json:"bfd-first_non_consensus_sequences,omitempty"`
	UniRef90    ProteinDatabaseDatasetDownloadProgress `json:"uniref90_2022_05,omitempty"`
	UniProt     ProteinDatabaseDatasetDownloadProgress `json:"uniprot_all_2021_04,omitempty"`
	PDB         ProteinDatabaseDatasetDownloadProgress `json:"pdb_2022_09_28_mmcif_files,omitempty"`
	PDBSeqReq   ProteinDatabaseDatasetDownloadProgress `json:"pdb_seqres_2022_09_28,omitempty"`
	RNACentral  ProteinDatabaseDatasetDownloadProgress `json:"rnacentral_active_seq_id_90_cov_80_linclust,omitempty"`
}

type ProteinDatabaseStatus struct {
	DownloadStatus ProteinDatabaseDownloadStatus `json:"downloadStatus,omitempty"`
	DownloadSpeed  string                        `json:"downloadSpeed,omitempty"`
	VolumeName     string                        `json:"volumeName"`
	Progress       string                        `json:"progress,omitempty"`
	Size           string                        `json:"size,omitempty"`
	TotalSize      string                        `json:"totalSize,omitempty"`
	LastUpdate     metav1.Time                   `json:"lastUpdate,omitempty"`
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
