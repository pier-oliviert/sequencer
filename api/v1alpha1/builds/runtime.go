package builds

import (
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// +kubebuilder:object:generate=true
type Runtime struct {
	// + optional
	Affinity *core.Affinity `json:"affinity,omitempty"`

	//+ optional
	Resources *core.ResourceRequirements `json:"resources,omitempty"`

	// If a specific image needs to be used for as a builder, the image can
	// be specified here. If no image is specified here, the operator
	// will use the vanilla builder for the version of operator currently running.
	// The vanilla version is set as an environment variable in the operator's
	// controller runtime.
	Image *string `json:"image,omitempty"`
}

var BuildDefaultResourceRequirements = &core.ResourceRequirements{
	Requests: core.ResourceList{
		core.ResourceMemory: resource.MustParse("1Gi"),
		core.ResourceCPU:    resource.MustParse("500m"),
	},
	Limits: core.ResourceList{
		core.ResourceMemory: resource.MustParse("4Gi"),
		core.ResourceCPU:    resource.MustParse("2000m"),
	},
}
