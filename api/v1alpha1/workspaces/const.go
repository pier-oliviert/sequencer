package workspaces

import (
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
)

const (
	InstanceLabel string = "workspaces.sequencer.io/instance"
	IngressLabel  string = "workspaces.sequencer.io/ingress"
)

const (
	DNSCondition       conditions.ConditionType = "DNS"
	IngressCondition   conditions.ConditionType = "Ingress"
	TunnelingCondition conditions.ConditionType = "Tunneling"
	ComponentCondition conditions.ConditionType = "Components"
)

const (
	ConditionReasonInitialized string = "Initialized"
	ConditionReasonError       string = "Error"
	ConditionReasonProcessing  string = "Processing"
	ConditionReasonCompleted   string = "Completed"
	ConditionReasonCreated     string = "Created"
	ConditionReasonDeploying   string = "Deploying"
)
