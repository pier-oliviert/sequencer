package conditions

import (
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string
type ConditionStatus string

const (
	ConditionInitialized ConditionStatus = "Initialized"
	ConditionHealthy     ConditionStatus = "Healthy"
	ConditionNotHealthy  ConditionStatus = "Not Healthy"
	ConditionInProgress  ConditionStatus = "In Progress"
	ConditionCompleted   ConditionStatus = "Completed"
	ConditionCreated     ConditionStatus = "Created"
	ConditionTerminated  ConditionStatus = "Terminated"
	ConditionWaiting     ConditionStatus = "Waiting"
	ConditionUnknown     ConditionStatus = "Unknown"
	ConditionError       ConditionStatus = "Error"
	ConditionLocked      ConditionStatus = "Locked"
)

// +kubebuilder:object:generate=true
type Condition struct {
	// type of condition in CamelCase or in foo.example.com/CamelCase.
	// ---
	// Many .condition.type values are consistent across resources like Available, but because arbitrary conditions can be
	// useful (see .node.status.conditions), the ability to deconflict is important.
	// The regex it matches is (dns1123SubdomainFmt/)?(qualifiedNameFmt)
	// +required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/)?(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])$`
	// +kubebuilder:validation:MaxLength=316
	Type ConditionType `json:"type" protobuf:"bytes,1,opt,name=type"`
	// status of the condition.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Initialized;Created;Terminated;In Progress;Waiting;Completed;Error;Unknown;Healthy;Not Healthy;Locked
	Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status"`
	// observedGeneration represents the .metadata.generation that the condition was set based upon.
	// For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
	// with respect to the current state of the instance.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,3,opt,name=observedGeneration"`
	// lastTransitionTime is the last time the condition transitioned from one status to another.
	// This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format=date-time
	LastTransitionTime meta.Time `json:"lastTransitionTime" protobuf:"bytes,4,opt,name=lastTransitionTime"`
	// reason is a human readable message indicating details about the transition.
	// This field may not be empty.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:MinLength=1
	Reason string `json:"reason" protobuf:"bytes,5,opt,name=reason"`
}
