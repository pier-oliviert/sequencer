package providers

import "se.quencer.io/api/v1alpha1/utils"

type CloudflareConnector string

var (
	TunnelingConnectorCloudflared CloudflareConnector = "cloudflared"
)

// +kubebuilder:object:generate=true
type CloudflareSpec struct {
	SecretKeyRef utils.SecretKeyRef    `json:"secretKeyRef"`
	Tunnel       *CloudflareTunnelSpec `json:"tunnel,omitempty"`
	DNS          *CloudflareDNSSpec    `json:"dns,omitempty"`
}

// +kubebuilder:object:generate=true
type CloudflareDNSSpec struct {
	ZoneID   string `json:"zoneId"`
	ZoneName string `json:"zoneName"`
}

// +kubebuilder:object:generate=true
type CloudflareTunnelSpec struct {
	Connector CloudflareConnector `json:"connector"`
	Route     CloudflareRouteSpec `json:"route"`
	AccountID string              `json:"accountId"`
}

// +kubebuilder:object:generate=true
type CloudflareRouteSpec struct {
	Path          string `json:"path,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
	ComponentName string `json:"component"`
	NetworkName   string `json:"network"`
}
