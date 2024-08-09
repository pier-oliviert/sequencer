package workspaces

import (
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
)

// +kubebuilder:object:generate=true
type Status struct {
	Phase      Phase                  `json:"phase"`
	Conditions []conditions.Condition `json:"conditions,omitempty"`
	Tunnel     *Tunnel                `json:"tunnel,omitempty"`
	Ingress    string                 `json:"ingress,omitempty"`

	// Host is the root URL where the application will point to
	Host string `json:"host,omitempty"`

	// Any components or task that needs a DNS entry to work
	// needs to set it up here. Sequencer will take those dns records here
	// and create a DNSRecord that represent each of those entries.
	// It's better for the DNSRecord to be created by the workspace as its own step
	// as it decouples the creating of the DNS with the requests from each of the subtasks
	// that represent a workspace.
	DNS []DNS `json:"dns,omitempty"`
}

// +kubebuilder:object:generate=true
type Tunnel struct {
	RemoteID string `json:"remoteId"`

	// Any key/value pair that needs to be used by the provider
	// These values are going to be stored as-is and won't be secret.
	// If a secret value needs to be provided, a secret can be created by
	// the provider and the reference to the secret can be added to this field.
	ProviderMeta map[string]string `json:"meta"`
}

// +kubebuilder:object:generate=true
type DNS struct {
	RecordType string            `json:"recordType"`
	Name       string            `json:"name"`
	Target     string            `json:"target"`
	Properties map[string]string `json:"properties,omitempty"`
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
