package integrations

import (
	"context"

	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/workspaces"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ProviderController is the interface that will be passed to a 3rd party provider
// to interact with the reconciler loop and all the operator's resource. The goal of this controller
// is to provide an abstraction to allow 3rd party integrations to safely make changes to any of the records.
// The idea is not to be restrictive about what an intergration can do but rather allow those to not have to
// think about all the gotchas of using a reconciler loop.
type ProviderController interface {

	// Guard should be used whenever a condition needs to mutate an external resource (cloud features likes DNS, Tunneling, etc.)
	// Using a Guard makes has 2 purposes, locking a condition to avoid creating duplicate of resources, and a safety trigger by
	// making sure the underlying resource is not outdated.
	//
	// Reconciliation in a distributed system is always a little tricky as it uses eventual consistency. This means that
	// the resource the reconciler is operating on could be outdated. When that happens, the operator only knows about it when
	// it tries to mutate the object, in our case, the status subresource. By locking the specific condition, the operator will
	// know if the resource has a conflict before starting. This helps avoiding conflict errors when trying to update the status
	// with information about newly created cloud resources.
	//
	// Eventually, this Guard might not be as needed when the operator uses patches instead of update for these kind of critical
	// updates to the Status subresource.
	Guard(ctx context.Context, reason string, task Task) error

	// UpdateCondition is a helper function to only update the status of the condition attached to this controller. This is
	// useful when a condition needs to change but nothing else about the status has changed.
	UpdateCondition(ctx context.Context, status conditions.ConditionStatus, reason string) error

	// Update the status of the attached workspace. An integration might need to update one of the status field.
	Update(context.Context, workspaces.Status) error

	Create(context.Context, client.Object, ...client.CreateOption) error
	Delete(context.Context, client.Object, ...client.DeleteOption) error

	SetFinalizer(ctx context.Context, finalizer string) error
	RemoveFinalizer(ctx context.Context, finalizer string) error

	// Retrieve the associated workspace for this instance
	Workspace() *sequencer.Workspace

	// Return the condition that is currently being processed by this controller
	// The condition object can change, but the Type of the condition should always be the same
	// throughout the controller's lifetime.
	Condition() conditions.Condition

	// Namespace of the Workspace. This is the same as calling Workspace().namespace
	Namespace() string

	// Proxy method to the underlying reconciler's eventRecorder. The object is always set
	// to the attached Workspace.
	Event(eventtype string, reason string, message string)
	Eventf(eventtype string, reason string, messageFmt string, args ...interface{})

	// Allows the integration to read any K8s objects it needs to operate.
	client.Reader
}

type Task func() (status conditions.ConditionStatus, reason string, err error)

type controller struct {
	workspace *sequencer.Workspace
	condition conditions.Condition

	Reconciler
}

func (c *controller) Namespace() string {
	return c.workspace.Namespace
}

func (c *controller) Condition() conditions.Condition {
	return c.condition
}

func (c *controller) Workspace() *sequencer.Workspace {
	return c.workspace
}

func (c *controller) Create(ctx context.Context, o client.Object, opts ...client.CreateOption) error {
	o.SetOwnerReferences([]meta.OwnerReference{{
		APIVersion: c.workspace.APIVersion,
		Name:       c.workspace.Name,
		Kind:       c.workspace.Kind,
		UID:        c.workspace.UID,
	}})

	o.SetNamespace(c.workspace.Namespace)

	labels := o.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[workspaces.InstanceLabel] = c.workspace.Name
	o.SetLabels(labels)

	return c.Reconciler.Create(ctx, o, opts...)
}

func (c *controller) Delete(ctx context.Context, o client.Object, opts ...client.DeleteOption) error {
	return c.Reconciler.Delete(ctx, o, opts...)
}

func (c *controller) SetFinalizer(ctx context.Context, finalizer string) error {
	if controllerutil.AddFinalizer(c.workspace, finalizer) {
		return c.Reconciler.Update(ctx, c.workspace)
	}
	return nil
}

func (c *controller) RemoveFinalizer(ctx context.Context, finalizer string) error {
	if controllerutil.RemoveFinalizer(c.workspace, finalizer) {
		return c.Reconciler.Update(ctx, c.workspace)
	}

	return nil
}

func (c *controller) UpdateCondition(ctx context.Context, status conditions.ConditionStatus, reason string) error {
	condition := c.Condition()
	condition.LastTransitionTime = meta.Now()
	condition.Status = status
	condition.Reason = reason

	conditions.SetCondition(&c.workspace.Status.Conditions, condition)
	c.condition = condition

	return c.Status().Update(ctx, c.workspace)
}

func (c *controller) Guard(ctx context.Context, reason string, task Task) error {
	if err := c.UpdateCondition(ctx, conditions.ConditionLocked, reason); err != nil {
		return err
	}

	status, reason, err := task()
	if err != nil {
		return err
	}

	return c.UpdateCondition(ctx, status, reason)
}

func (c *controller) Update(ctx context.Context, status workspaces.Status) error {
	c.workspace.Status = status

	return c.Status().Update(ctx, c.workspace)
}

func (c *controller) Event(eventtype, reason, message string) {
	c.Reconciler.Event(c.workspace, eventtype, reason, message)
}
func (c *controller) Eventf(eventtype, reason, messageFmt string, args ...interface{}) {
	c.Reconciler.Eventf(c.workspace, eventtype, reason, messageFmt, args...)
}

type Provider interface {
	Reconcile(context.Context) (*ctrl.Result, error)
	Terminate(context.Context) (*ctrl.Result, error)
}

type ProviderConfig struct {
	Workspace  *sequencer.Workspace
	Controller ProviderController
}

type Reconciler interface {
	client.Writer
	client.StatusClient
	client.Reader
	record.EventRecorder
}

func NewController(workspace *sequencer.Workspace, condition conditions.Condition, reconciler Reconciler) ProviderController {
	return &controller{workspace: workspace, condition: condition, Reconciler: reconciler}
}
