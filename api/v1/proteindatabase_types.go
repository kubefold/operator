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

// ProteinDatabaseSpec defines the desired state of ProteinDatabase.
type ProteinDatabaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ProteinDatabase. Edit proteindatabase_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// ProteinDatabaseStatus defines the observed state of ProteinDatabase.
type ProteinDatabaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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
