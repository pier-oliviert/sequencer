package validators

import (
	"github.com/pier-oliviert/sequencer/api/v1alpha1/builds"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateGit(git *builds.GitSource) *field.Error {
	return nil
}
