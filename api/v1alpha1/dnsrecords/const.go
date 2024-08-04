package dnsrecords

import "github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"

type Phase string

const (
	PhaseInitializing Phase = "Initializing"
	PhaseCreated      Phase = "Created"
	PhaseError        Phase = "Error"
	PhaseTerminating  Phase = "Terminating"
)

const (
	ProviderCondition conditions.ConditionType = "Provider"
)
