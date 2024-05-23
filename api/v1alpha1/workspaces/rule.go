package workspaces

// +kubebuilder:object:generate=true
type RuleSpec struct {
	Name       string   `json:"name"`
	Subdomains []string `json:"subdomains,omitempty"`
	Paths      []string `json:"paths,omitempty"`
	Selector   Selector `json:",omitempty"`
}

// +kubebuilder:object:generate=true
type Selector struct {
	ComponentRef string `json:"component,omitempty"`
	NetworkName  string `json:"network,omitempty"`
}
