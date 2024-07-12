package builds

import (
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
)

const (
	LabelName string = "github.com/pier-oliviert/sequencer/build"
)

// +kubebuilder:validation:Enum=Initialized;Running;Success;Error
// +kubebuilder:default=Initialized
type Phase string

const (
	PhaseUninitialized Phase = ""
	PhaseInitialized   Phase = "Initialized"
	PhaseRunning       Phase = "Running"
	PhaseSuccess       Phase = "Success"
	PhaseError         Phase = "Error"
)

const (
	PodScheduledCondition        conditions.ConditionType = "Pod.Scheduled"
	BackendConfiguredCondition   conditions.ConditionType = "Backend"
	SecretsCondition             conditions.ConditionType = "Secrets"
	ImportDirectoriesCondition   conditions.ConditionType = "ImportDirectories"
	ContainerRegistriesCondition conditions.ConditionType = "ContainerRegistries"
	ImageCondition               conditions.ConditionType = "Image"
	UploadCondition              conditions.ConditionType = "Upload"
)

const (
	ConditionReasonInitialized        string = "Initialized"
	ConditionReasonError              string = "Error"
	ConditionReasonProcessing         string = "Processing"
	ConditionReasonCompleted          string = "Completed"
	ConditionReasonPodErrorTerminated string = "TerminatedByError"
)
