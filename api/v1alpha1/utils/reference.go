package utils

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:object:generate=true
type SecretKeyRef struct {
	Key       string `json:"key"`
	SecretRef `json:""`
}

// +kubebuilder:object:generate=true
type SecretRef struct {
	Name      string  `json:"name"`
	Namespace *string `json:"namespace,omitempty"`
}

// +kubebuilder:object:generate=true
type ConfigMapRef struct {
	Name      string  `json:"name"`
	Namespace *string `json:"namespace,omitempty"`
}

// Reference is used to create untyped references to different object
// that needs to be tracked inside of Custom Resources.
// Examples can be found in Workspace & Build where for workspace,
// it needs to reference a build or a pod and uses this struct as a way
// to serialize the labels of the underlying resource.
// +kubebuilder:object:generate=true
type Reference struct {
	// `namespace` is the namespace of the resource.
	// Required
	Namespace string `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`
	// `name` is the name of the resourec.
	// Required
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
}

// Returns a new Reference object, the client.Object interface
// is global and any core resource defined by K8s(Pod, services, etc) as well
// as CRD (Build, Workspace, etc) implements this interface.
//
// NOTE: Since this reference is untyped, different type could, in theory, share the same
// namespace/name and could cause issues. This is why it's important to use generatedName()
// when creating resources internally.
func NewReference(obj client.Object) *Reference {
	return &Reference{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

func (r Reference) String() string {
	return fmt.Sprintf("%s/%s", r.Namespace, r.Name)
}

func (r Reference) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      r.Name,
		Namespace: r.Namespace,
	}
}
