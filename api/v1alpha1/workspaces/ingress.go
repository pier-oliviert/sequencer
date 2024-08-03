package workspaces

import "github.com/pier-oliviert/sequencer/api/v1alpha1/utils"

// +kubebuilder:object:generate=true
type IngressSpec struct {
	// Rules represent each Ingress Rule that you want to define to make
	// and endpoint accessible through the Ingress.
	//+kubebuilder:validation:MinItems:1

	Rules []RuleSpec `json:"rules"`

	// Ingress Class name, will default to nginx if it's not set.
	//+kubebuilder:default:=nginx
	ClassName *string `json:"className,omitempty"`

	// Reference to the load balancer that is attached to the ingress
	// class name. This service represents the entry point for the external traffic,
	// through the cloud provider and into the cluster. Eventually this could become a
	// runtime settings configurable through a configmap, which would be ideal to avoid repetition and
	// also to only have a central point to configure this. Even in the case a global value is configurable,
	// this field would always exists as an optional one to allow overwriting the default settings.
	//
	// While it is assume this load balancer is public, it is not required to be. A private load balancer
	// would work the same, but any workspace configured with a private load balancer would only be accessible
	// to those who can reach this private load balancer.
	//
	// TLS connection is also supported with private load balancer as the DNS01 Challenge doesn't require to
	// reach any services
	LoadBalancerRef utils.Reference `json:"loadBalancerRef"`
}

// +kubebuilder:object:generate=true
type RuleSpec struct {
	// Name of the rule.
	Name string `json:"name"`

	Subdomain *string `json:"subdomain,omitempty"`

	// +kubebuilder:validation:Pattern=`^/.*$`
	Path          *string `json:"path,omitempty"`
	ComponentName string  `json:"component"`
	NetworkName   string  `json:"network"`
}
