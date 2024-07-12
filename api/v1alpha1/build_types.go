/*
Copyright 2023.

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
	builds "github.com/pier-oliviert/sequencer/api/v1alpha1/builds"
	config "github.com/pier-oliviert/sequencer/api/v1alpha1/builds/config"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/utils"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BuildSpec struct {
	// Name represents the name specified for the container that needs to use the
	// image generated from this build. This value needs to be unique within a workspace.
	// Here's an example that might help see how it is all tied together:
	//// - name: click-mania
	////   template:
	////   	containers:
	////   		- name: click
	////   			image: my-custom-build  // This value needs to be the exact same as a build name
	////   build:
	////   	name: my-custom-build // This value will be searched for through all containers within a workspace.
	////   	dockerfile: Dockerfile
	//
	Name string `json:"name"`

	// Context represents the root path of where all the content for the build are located. Often, this value is set to `.` and
	// this is a valid value here too. This value needs to be a relative path as the path will be created under a random
	// temporary folder created for this build, inside the builder pod.
	// If the value is not provided, it will be set to "."
	// TODO: Create Validation Pattern to exclude absolute path and path starting with ..
	Context string `json:"context,omitempty"`

	// Dockerfile is the name of the Dockerfile to run for this build. It can include a
	// TODO: Create Validation Pattern to exclude absolute path and path starting with ..
	Dockerfile string `json:"dockerfile"`

	// Target is an optional field that can be set if a build needs to use a Docker target.
	Target *string `json:"target,omitempty"`

	// Args is an optional field to pass build arguments to buildkit.
	Args *config.DynamicValues `json:"args,omitempty"`

	// Secrets is an optional field to pass build secrets to buildkit.
	Secrets *config.DynamicValues `json:"secrets,omitempty"`

	// ContainerRegistries is a list of registries provided by the user, each ContainerRegistry is self contained and
	// includes all the information needed to push an image to it.
	ContainerRegistries []builds.ContainerRegistry `json:"containerRegistries,omitempty"`

	// ImportContent represents a list to be imported for the build. For most, an ImportContent
	// can be a Git repo that is imported for a build. Multiple git repositories can be imported for a single
	// build through different ImportContent
	ImportContent []builds.ImportContent `json:"importContent,omitempty"`

	// Runtime includes all the runtime values that can be used to tweak the build.
	// Many settings can be changed, ie. Node Affinity, Image for the builder, etc.
	//
	// More information can be found reading the documentation for +builds.Runtime+
	Runtime builds.Runtime `json:"runtime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

type Build struct {
	meta.TypeMeta   `json:",inline"`
	meta.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildSpec     `json:"spec,omitempty"`
	Status builds.Status `json:"status,omitempty"`
}

func (b *Build) GetReference() utils.Reference {
	return utils.Reference{
		Namespace: b.Namespace,
		Name:      b.Name,
	}
}

// +kubebuilder:object:root=true
type BuildList struct {
	meta.TypeMeta `json:",inline"`
	meta.ListMeta `json:"metadata,omitempty"`
	Items         []Build `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Build{}, &BuildList{})
}
