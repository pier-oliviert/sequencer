package workspaces

// +kubebuilder:object:generate=true
type IngressSpec struct {
	// Rules represent each Ingress Rule that you want to define to make
	// and endpoint accessible through the Ingress.
	//+kubebuilder:validation:MinItems:1

	Rules []RuleSpec `json:"rules"`

	// Ingress Class name, will default to nginx if it's not set.
	//+kubebuilder:default:=nginx
	ClassName *string `json:"className,omitempty"`
}

// +kubebuilder:object:generate=true
type RuleSpec struct {
	// Name of the rule.
	Name string `json:"name"`

	Subdomain *string `json:"subdomain,omitempty"`

	// +kubebuilder:validation:Pattern=`^/.*$`
	Path          *string `json:"path,omitempty"`
	ComponentName string  `json:"component,omitempty"`
	NetworkName   string  `json:"network,omitempty"`
}
