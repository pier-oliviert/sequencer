package workspaces

// +kubebuilder:object:generate=true
type IngressSpec struct {
	// Ingress Class name, will default to nginx if it's not set.
	//+kubebuilder:default:=nginx
	ClassName *string `json:"className,omitempty"`

	//+kubebuilder:validation:MinItems:1
	Rules []RuleSpec `json:"rules"`
}

// +kubebuilder:object:generate=true
type RuleSpec struct {
	Name      string  `json:"name"`
	Subdomain *string `json:"subdomain,omitempty"`

	// +kubebuilder:validation:Pattern=`^/.*$`
	Path          *string `json:"path,omitempty"`
	ComponentName string  `json:"component,omitempty"`
	NetworkName   string  `json:"network,omitempty"`
}
