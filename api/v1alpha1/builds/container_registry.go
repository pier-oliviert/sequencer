package builds

import "se.quencer.io/api/v1alpha1/builds/config"

// +kubebuilder:object:generate=true
type ContainerRegistry struct {
	URL string `json:"url"`

	// Tags must represent a list of valid tags for the ContainerRegistry.
	// For example, for AWS ECR, a fully qualified URL is required.
	Tags []string `json:"tags"`

	// Credentials needs to specify where to find the
	// the credentials for this Container Registry.
	Credentials config.Credentials `json:"credentials"`
}

func (cr *ContainerRegistry) Default() {
	cr.Credentials.Default()
}
