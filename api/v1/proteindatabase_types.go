/*
Copyright 2025 Mateusz Woźniak <wozniakmat@student.agh.edu.pl>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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

type ProteinDatabaseDatasets struct {
	MHYClusters bool `json:"mgy_clusters_2022_05"`
	BFD         bool `json:"bfd-first_non_consensus_sequences"`
	UniRef90    bool `json:"uniref90_2022_05"`
	PDB         bool `json:"pdb_2022_09_28_mmcif_files"`
	PDBSeqReq   bool `json:"pdb_seqres_2022_09_28"`
	RNACentral  bool `json:"rnacentral_active_seq_id_90_cov_80_linclust"`
	NT          bool `json:"nt_rna_2023_02_23_clust_seq_id_90_cov_80_rep_seq"`
	RFam        bool `json:"rfam_14_9_clust_seq_id_90_cov_80_rep_seq"`
}

// ProteinDatabaseSpec defines the desired state of ProteinDatabase.
type ProteinDatabaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ProteinDatabase. Edit proteindatabase_types.go to remove/update
	Foo string `json:"foo,omitempty"`

	Datasets ProteinDatabaseDatasets `json:"datasets,omitempty"`
	Volume   ProteinDatabaseVolume   `json:"volume,omitempty"`
}

type ProteinDatabaseDownloadStatus string

const (
	ProteinDatabaseDownloadStatusNotStarted ProteinDatabaseDownloadStatus = "NotStarted"
	ProteinDatabaseDownloadStatusInProgress ProteinDatabaseDownloadStatus = "InProgress"
	ProteinDatabaseDownloadStatusCompleted  ProteinDatabaseDownloadStatus = "Completed"
	ProteinDatabaseDownloadStatusFailed     ProteinDatabaseDownloadStatus = "Failed"
)

// ProteinDatabaseStatus defines the observed state of ProteinDatabase.
type ProteinDatabaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	DownloadStatus ProteinDatabaseDownloadStatus `json:"downloadStatus,omitempty"`
	DownloadSpeed  string                        `json:"downloadSpeed,omitempty"`
	VolumeName     string                        `json:"volumeName"`
	Progress       string                        `json:"progress,omitempty"`
	Size           string                        `json:"size,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.spec.downloadStatus`
// +kubebuilder:printcolumn:name="Progress",type=string,JSONPath=`.spec.progress`
// +kubebuilder:printcolumn:name="Size",type=string,JSONPath=`.spec.size`
// +kubebuilder:printcolumn:name="Download Speed",type=string,JSONPath=`.spec.downloadSpeed`
// +kubebuilder:printcolumn:name="Volume",type=string,JSONPath=`.status.volumeName`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ProteinDatabase is the Schema for the proteindatabases API.
type ProteinDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProteinDatabaseSpec   `json:"spec,omitempty"`
	Status ProteinDatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProteinDatabaseList contains a list of ProteinDatabase.
type ProteinDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProteinDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProteinDatabase{}, &ProteinDatabaseList{})
}
