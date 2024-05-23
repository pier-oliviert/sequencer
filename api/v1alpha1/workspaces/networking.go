package workspaces

import (
	"se.quencer.io/api/v1alpha1/providers"
)

// +kubebuilder:object:generate=true
type NetworkingSpec struct {
	Cloudflare *providers.CloudflareSpec `json:"cloudflare,omitempty"`

	// Rules are provider agnostic and are Ingress Rules that are
	// going to be used by the ingress controller to map to the proper services.
	Rules []RuleSpec `json:"rules,omitempty"`
}
