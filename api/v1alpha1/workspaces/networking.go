package workspaces

import (
	"se.quencer.io/api/v1alpha1/providers"
)

// NetworkingSpec is what binds the internal resources (components, builds, etc.) to the
// outside world. The Spec directly integrates with the cloud provider of your choosing.
// Needs exactly one of those provider to be properly configured for a NetworkingSpec to
// be valid.
//
// +kubebuilder:object:generate=true
type NetworkingSpec struct {
	Cloudflare *providers.CloudflareSpec `json:"cloudflare,omitempty"`
	AWS        *providers.AWSSpec        `json:"aws,omitempty"`

	// IngressSpec includes all the configuration to customize the networking section
	// of a workspace to work with an ingress controller
	//
	// At the moment, this ingress spec is required for networking to work.
	// It will become an optional field when the Gateway API is added to the NetworkingSpec
	Ingress *IngressSpec `json:"ingress,omitempty"`
}
