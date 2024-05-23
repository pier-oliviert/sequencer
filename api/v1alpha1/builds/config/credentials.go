package config

import (
	"fmt"

	core "k8s.io/api/core/v1"
	"se.quencer.io/api/v1alpha1/utils"
)

type AuthScheme string

// +kubebuilder:object:generate=true
const (
	// The secret referenced needs to have the following key/value defined:
	// 	- privateKey
	SingleToken AuthScheme = "token"

	// The secret referenced needs to have two pair of key/values defined:
	//	- accessKey
	//	- secretToken
	KeyPair AuthScheme = "keyPair"
)

// +kubebuilder:object:generate=true
type Credentials struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=token;keyPair
	AuthScheme AuthScheme           `json:"authScheme"`
	SecretRef  LocalObjectReference `json:"secretRef"`

	// +optional
	Name *string `json:"path,omitempty"`
}

func (c *Credentials) Default() {
	if c.Name == nil {
		c.Name = new(string)
		*c.Name = fmt.Sprintf("credentials-%s", utils.RandomValue(utils.KPFilenameLength))
	}
}

func (c *Credentials) IsValidForSecret(secret *core.Secret) bool {
	switch c.AuthScheme {
	case SingleToken:
		_, ok := secret.Data["privateKey"]
		return ok
	case KeyPair:
		_, keyOk := secret.Data["accessKey"]
		_, secretOk := secret.Data["secretToken"]

		return keyOk && secretOk
	}

	return false
}
