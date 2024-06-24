package tunneling

import "se.quencer.io/api/v1alpha1/utils"

type CloudflareConnector string

var (
	TunnelingConnectorCloudflared CloudflareConnector = "cloudflared"
)

// +kubebuilder:object:generate=true
type CloudflareTunnelSpec struct {
	SecretKeyRef utils.SecretKeyRef  `json:"secretKeyRef"`
	Connector    CloudflareConnector `json:"connector"`
	Route        CloudflareRouteSpec `json:"route"`
	AccountID    string              `json:"accountId"`
}

// +kubebuilder:object:generate=true
type CloudflareRouteSpec struct {
	Path          string `json:"path,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
	ComponentName string `json:"component"`
	NetworkName   string `json:"network"`
}
