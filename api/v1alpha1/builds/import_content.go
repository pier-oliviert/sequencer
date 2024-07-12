package builds

import (
	"github.com/pier-oliviert/sequencer/api/v1alpha1/builds/config"
)

// +kubebuilder:object:generate=true
type ImportContent struct {
	// Path is a relative path the content
	Path        string       `json:"path,omitempty"`
	ContentFrom ImportSource `json:"contentFrom"`

	// Credentials needs to specify where to find the
	// the credentials for this Container Registry.
	Credentials *config.Credentials `json:"credentials,omitempty"`
}

func (ic ImportContent) IsPrivate() bool {
	return ic.Credentials != nil
}

func (ic *ImportContent) Default() {
	if ic.Credentials != nil {
		ic.Credentials.Default()
	}
}

// +kubebuilder:object:generate=true
type ImportSource struct {
	Git *GitSource `json:"git,omitempty"`
}

// +kubebuilder:object:generate=true
type GitSource struct {
	Ref   string  `json:"ref"`
	URL   string  `json:"url"`
	Depth *string `json:"depth,omitempty"`
}
