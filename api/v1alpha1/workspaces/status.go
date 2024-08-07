package workspaces

import (
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
)

// +kubebuilder:object:generate=true
type Status struct {
	Phase      Phase                  `json:"phase"`
	Conditions []conditions.Condition `json:"conditions,omitempty"`
	Tunnel     *Tunnel                `json:"tunnel,omitempty"`
	Host       string                 `json:"host"`
}

// +kubebuilder:object:generate=true
type Tunnel struct {
	Provider string `json:"provider"`
	Hostname string `json:"hostname"`

	// Any key/value pair that needs to be used by the provider
	// These values are going to be stored as-is and won't be secret.
	// If a secret value needs to be provided, a secret can be created by
	// the provider and the reference to the secret can be added to this field.
	ProviderMeta map[string]string `json:"meta"`
}

// +kubebuilder:validation:Enum=Deploying;Healthy;Error;Terminating
type Phase string

const (
	PhaseDeploying   Phase = "Deploying"
	PhaseHealthy     Phase = "Healthy"
	PhaseError       Phase = "Error"
	PhaseTerminating Phase = "Terminating"
)

func DefaultStatus() Status {
	return Status{
		Phase: PhaseDeploying,
	}
}
