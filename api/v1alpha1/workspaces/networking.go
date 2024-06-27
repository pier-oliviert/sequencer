package workspaces

import (
	"se.quencer.io/api/v1alpha1/tunneling"
)

// NetworkingSpec is what binds the internal resources (components, builds, etc.) to the
// outside world. The Spec directly integrates with the cloud provider of your choosing.
// Needs exactly one of those provider to be properly configured for a NetworkingSpec to
// be valid.
//
// +kubebuilder:object:generate=true
type NetworkingSpec struct {
	// The DNSSpec includes the basic information needed for external-dns to generate
	// DNS entries. The spec is going to be used with the configured spec (tunnel, ingress, etc.)
	DNS DNSSpec `json:"dns"`

	// Tunneling is used when you need to create a tunnel for an application to be accessible
	Tunnel *TunnelSpec `json:"tunnel,omitempty"`

	// IngressSpec includes all the configuration to customize the networking section
	// of a workspace to work with an ingress controller
	//
	// At the moment, this ingress spec is required for networking to work.
	// It will become an optional field when the Gateway API is added to the NetworkingSpec
	Ingress *IngressSpec `json:"ingress,omitempty"`
}

// +kubebuilder:object:generate=true
type TunnelSpec struct {
	Cloudflare *tunneling.CloudflareTunnelSpec `json:"cloudflare,omitempty"`
}

// +kubebuilder:object:generate=true
type DNSSpec struct {
	Zone string `json:"zone"`
}
