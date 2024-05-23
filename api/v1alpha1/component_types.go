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
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"se.quencer.io/api/v1alpha1/components"
	"se.quencer.io/api/v1alpha1/conditions"
)

type ComponentSpec struct {
	Name     string        `json:"name"`
	Networks []NetworkSpec `json:"networks"`

	// If a Build is not present, a template with a valid image needs to exist.
	// +optional
	Build *BuildSpec `json:"build,omitempty"`

	// Pod Template that will be used for the pod
	Pod core.PodSpec `json:"template"`

	// Block a component to deploy a pod until each of the dependencies are
	// running and healthy.
	// +optional
	DependsOn []Dependency `json:"dependsOn,omitempty"`
}

// +kubebuilder:object:generate=true
type Dependency struct {
	ComponentName   string                     `json:"componentName"`
	ConditionType   conditions.ConditionType   `json:"conditionType"`
	ConditionStatus conditions.ConditionStatus `json:"conditionStatus"`
}

type NetworkSpec struct {
	Ingress          *networking.IngressSpec `json:"ingress,omitempty"`
	core.ServicePort `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// Component is the Schema for the components API
type Component struct {
	meta.TypeMeta   `json:",inline"`
	meta.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComponentSpec     `json:"spec,omitempty"`
	Status components.Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ComponentList contains a list of Component
type ComponentList struct {
	meta.TypeMeta `json:",inline"`
	meta.ListMeta `json:"metadata,omitempty"`
	Items         []Component `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Component{}, &ComponentList{})
}
