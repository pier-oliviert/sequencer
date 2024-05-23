package validators

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
	"se.quencer.io/api/v1alpha1/builds"
)

func ValidateGit(git *builds.GitSource) *field.Error {
	return nil
}
