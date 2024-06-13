package workspaces

import (
	"se.quencer.io/api/v1alpha1/providers"
)

// +kubebuilder:object:generate=true
type NetworkingSpec struct {
	Cloudflare *providers.CloudflareSpec `json:"cloudflare,omitempty"`

	Ingress *IngressSpec `json:"ingress,omitempty"`
}
