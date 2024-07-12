package components

import (
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/utils"
)

type Phase string

const (
	PhaseInitializing Phase = "Initializing"
	PhaseDeploying    Phase = "Deploying"
	PhaseHealthy      Phase = "Healthy"
	PhaseError        Phase = "Error"
	PhaseTerminating  Phase = "Terminating"
)

// +kubebuilder:object:generate=true
type Status struct {
	// +kubebuilder:validation:Enum=Initializing;Deploying;Healthy;Error;Terminating
	Phase Phase `json:"phase"`

	Conditions  []conditions.Condition `json:"conditions,omitempty"`
	BuildRefs   []utils.Reference      `json:"buildRefs,omitempty"`
	IngressRefs []utils.Reference      `json:"ingressRefs,omitempty"`

	Variables []GeneratedVariable `json:"variables,omitempty"`
}

type GeneratedVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (s *Status) Default() {
	s.Phase = PhaseInitializing
}
