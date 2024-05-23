package config

import (
	"fmt"

	"se.quencer.io/api/v1alpha1/utils"
)

// DynamicValues are values provided by the user and that will be passed down to
// the builder. These can range from simple key/value attributes to more secretive values
// like credentials. It's important for the user to recognize the threat level of the
// information it passes to the builder and need to use `SourceRef` accordingly.
// For example, if the information is secretive (like credentials), `SecretRef` should be
// used.
//
// Most of those DynamicValues are provided by the user but it's not the only scenario.
// ImportDirectories includes DynamicValues as an example where those values are short-lived
// credentials populated by the operator itself. You can read more about those short-lived credentials
// by looking up `ImportContent`.
//
// Those values will be mounted to the builder's container as volumes and each of the
// `Items` will be stored as readonly files in the container.
//
// +kubebuilder:object:generate=true
type DynamicValues struct {
	ValuesFrom SourceRef   `json:"valuesFrom"`
	Items      []KeyToPath `json:"items,omitempty"`
}

const kKPFilenameLength = 12

// +kubebuilder:object:generate=true
type KeyToPath struct {
	Key string `json:"key"`

	//+optional
	Path *string `json:"path,omitempty"`
}

func (kp *KeyToPath) Default() {
	if kp.Path == nil {
		kp.Path = new(string)
		*kp.Path = fmt.Sprintf("%s-%s", kp.Key, utils.RandomValue(kKPFilenameLength))
	}
}

func (upv *DynamicValues) SourceRefIsValid() bool {
	if upv.ValuesFrom.ConfigMapRef != nil {
		if upv.ValuesFrom.SecretRef != nil {
			return false
		}
	}

	return upv.ValuesFrom.SecretRef != nil
}

// +kubebuilder:object:generate=true
type SourceRef struct {
	ConfigMapRef *LocalObjectReference `json:"configMapRef,omitempty"`
	SecretRef    *LocalObjectReference `json:"secretRef,omitempty"`
}

// This is a copy of the core Kubernetes `LocalObjectReference` to make sure the Name is
// required for this structure to exists.
type LocalObjectReference struct {
	Name string `json:"name"`
}
