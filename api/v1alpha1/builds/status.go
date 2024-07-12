package builds

import (
	"encoding/json"

	gcr "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/utils"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:generate=true
type Status struct {
	Phase      Phase                  `json:"phase,omitempty"`
	Conditions []conditions.Condition `json:"conditions"`
	PodRef     *utils.Reference       `json:"pod,omitempty"`
	Images     []*Image               `json:"images,omitempty"`
}

func (r *Status) Default() {
	r.Phase = PhaseInitialized
	r.Conditions = []conditions.Condition{
		{
			Type:               PodScheduledCondition,
			Status:             conditions.ConditionUnknown,
			Reason:             ConditionReasonInitialized,
			LastTransitionTime: meta.Now(),
		},
		{
			Type:               ImageCondition,
			Status:             conditions.ConditionUnknown,
			Reason:             ConditionReasonInitialized,
			LastTransitionTime: meta.Now(),
		},
		{
			Type:               UploadCondition,
			Status:             conditions.ConditionUnknown,
			Reason:             ConditionReasonInitialized,
			LastTransitionTime: meta.Now(),
		},
	}
}

// +kubebuilder:object:generate=true
type Image struct {
	URL string `json:"url"`
	// Unfortunately, gcr.IndexManifest includes a Hash type
	// that does custom marshalling which isn't supported by kubebuilder.
	// Until it is supported, a serialized string will have to do.
	// TODO: It might be possible to define an OpenAPI definition
	// as meta.Time does it: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
	IndexManifestStr string `json:"indexManifest"`
}

func (i Image) ParseIndexManifest() (*gcr.IndexManifest, error) {
	var indexManifest gcr.IndexManifest
	err := json.Unmarshal([]byte(i.IndexManifestStr), &indexManifest)
	return &indexManifest, err
}
