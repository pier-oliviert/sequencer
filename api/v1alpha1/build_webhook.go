package v1alpha1

import (
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"se.quencer.io/api/v1alpha1/builds/validators"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var buildlog = logf.Log.WithName("build-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Build) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-sequencer-se-quencer-io-v1alpha1-build,mutating=true,failurePolicy=fail,sideEffects=None,groups=se.quencer.io,resources=builds,verbs=create,versions=v1alpha1,name=mbuild.se.quencer.io,admissionReviewVersions=v1
var _ webhook.Defaulter = &Build{}

// +kubebuilder:webhook:verbs=create,path=/validate-sequencer-se-quencer-io-v1alpha1-build,mutating=false,failurePolicy=fail,groups=se.quencer.io,resources=builds,versions=v1,name=vbuild.se.quencer.io,sideEffects=None,admissionReviewVersions=v1
var _ webhook.Validator = &Build{}

var ErrContentFromMissing = field.Error{
	Type:   field.ErrorTypeInvalid,
	Field:  "ContentFrom",
	Detail: "E#TODO: ContentFrom is nil, only Git sources are supported at the moment",
}

var ErrContainerRegistryEmpty = field.Error{
	Type:   field.ErrorTypeInvalid,
	Field:  "containerRegistries",
	Detail: "E#TODO: containerRegistries is empty, need to export the build to at least one container registry",
}

func (b *Build) ValidateCreate() (admission.Warnings, error) {
	var errors field.ErrorList

	if len(b.Spec.ContainerRegistries) == 0 {
		errors = append(errors, &ErrContainerRegistryEmpty)
	}

	for _, id := range b.Spec.ImportContent {
		if id.ContentFrom.Git == nil {
			errors = append(errors, &ErrContentFromMissing)
			continue
		}

		if err := validators.ValidateGit(id.ContentFrom.Git); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: "se.quencer.io", Kind: "Build"},
			b.Name, errors)
	}
	return nil, nil
}

func (b *Build) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	return nil, errors.New("E#TODO: builds are immutable")
}

func (b *Build) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}
