/*
Copyright 2024.

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

package v1alpha1

import (
	"github.com/pier-oliviert/sequencer/api/v1alpha1/workspaces"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkspaceSpec defines the desired state of Workspace
type WorkspaceSpec struct {
	Components []ComponentSpec           `json:"components"`
	Networking workspaces.NetworkingSpec `json:"networking"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Hostname",type=string,JSONPath=`.status.dns.hostname`
type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceSpec     `json:"spec,omitempty"`
	Status workspaces.Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WorkspaceList contains a list of Workspace
type WorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workspace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workspace{}, &WorkspaceList{})
}
