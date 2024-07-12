package components

import (
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
)

const (
	NameLabel     string = "components.sequencer.io/name"
	InstanceLabel string = "components.sequencer.io/instance"
	NetworkLabel  string = "components.sequencer.io/network"

	InterpolationDelimStart string = "${"
)

const (
	BuildCondition        conditions.ConditionType = "Builds"
	NetworkCondition      conditions.ConditionType = "Network"
	PodCondition          conditions.ConditionType = "Pod"
	DependenciesCondition conditions.ConditionType = "Dependency"
	VariablesCondition    conditions.ConditionType = "Variables"

	ConditionReasonInitialized   string = "Initialized"
	ConditionReasonProcessing    string = "Processing"
	ConditionReasonCompleted     string = "Completed"
	ConditionReasonSuccessful    string = "Successful"
	ConditionReasonBuildError    string = "Build Error"
	ConditionReasonSkipped       string = "Build Skipped"
	ConditionReasonPodTerminated string = "Pod Terminated"
	ConditionReasonDependsOn     string = "Depends on other components"
)
