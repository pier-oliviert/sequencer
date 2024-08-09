package components

import (
	"context"
	"fmt"

	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	builds "github.com/pier-oliviert/sequencer/api/v1alpha1/builds"
	components "github.com/pier-oliviert/sequencer/api/v1alpha1/components"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	utils "github.com/pier-oliviert/sequencer/api/v1alpha1/utils"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BuildReconciler struct {
	client.Client
	record.EventRecorder
}

func (r *BuildReconciler) Reconcile(ctx context.Context, component *sequencer.Component) (*ctrl.Result, error) {
	condition := conditions.FindCondition(component.Status.Conditions, components.BuildCondition)
	if condition == nil {
		condition = &conditions.Condition{
			Type:   components.BuildCondition,
			Status: conditions.ConditionUnknown,
			Reason: components.ConditionReasonInitialized,
		}
	}

	switch condition.Status {
	case conditions.ConditionUnknown:
		return r.launchBuild(ctx, component)
	case conditions.ConditionInProgress:
		return r.monitorBuild(ctx, component)
	case conditions.ConditionCompleted:
		return nil, nil
	}

	// Might want to log the reconciler loop got here, the switch should account for all of the outcome
	return &ctrl.Result{}, nil
}

func (r *BuildReconciler) launchBuild(ctx context.Context, component *sequencer.Component) (*ctrl.Result, error) {
	conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
		Type:   components.BuildCondition,
		Status: conditions.ConditionInProgress,
		Reason: components.ConditionReasonProcessing,
	})
	if err := r.Client.Status().Update(ctx, component); err != nil {
		return nil, err
	}

	// It's possible a component doesn't have a build and only uses an existing image, if that's the case,
	// we'll skip the build process
	if component.Spec.Build == nil {
		conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
			Type:   components.BuildCondition,
			Status: conditions.ConditionCompleted,
			Reason: components.ConditionReasonSkipped,
		})

		return &ctrl.Result{}, r.Client.Status().Update(ctx, component)
	}

	// The Build resource is created with owner references set to the component which means there is a direct depndency
	// between a component and the associated build.
	build := sequencer.Build{
		ObjectMeta: meta.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", component.Spec.Build.Name),
			Namespace:    component.GetNamespace(),
			Labels: map[string]string{
				components.NameLabel: component.Name,
				builds.LabelName:     component.Spec.Build.Name,
			},
			OwnerReferences: []meta.OwnerReference{{
				APIVersion: component.APIVersion,
				Kind:       component.Kind,
				Name:       component.Name,
				UID:        component.UID,
			}},
		},
		Spec: *component.Spec.Build,
	}

	if err := r.Client.Create(ctx, &build); err != nil {
		conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
			Type:   components.BuildCondition,
			Status: conditions.ConditionError,
			Reason: err.Error(),
		})

		if err := r.Client.Status().Update(ctx, component); err != nil {
			return &ctrl.Result{}, err
		}

		return nil, err
	}

	r.EventRecorder.Eventf(component, core.EventTypeNormal, string(components.BuildCondition), "Dispatched builder (%s)", build.Name)
	component.Status.BuildRefs = append(component.Status.BuildRefs, *utils.NewReference(&build))
	return &ctrl.Result{}, r.Client.Status().Update(ctx, component)
}

func (r *BuildReconciler) monitorBuild(ctx context.Context, component *sequencer.Component) (*ctrl.Result, error) {
	// Expecting only 1 build ref at this point
	if len(component.Status.BuildRefs) != 1 {
		return &ctrl.Result{}, fmt.Errorf("E#1010: Expected only 1 build reference, found %d", len(component.Status.BuildRefs))
	}

	// Reaching here means there's a Build resource already dispatched and we're monitoring the state of the build
	var build sequencer.Build
	if err := r.Client.Get(ctx, component.Status.BuildRefs[0].NamespacedName(), &build); err != nil {
		conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
			Type:   components.BuildCondition,
			Status: conditions.ConditionError,
			Reason: err.Error(),
		})

		return &ctrl.Result{}, r.Client.Status().Update(ctx, component)
	}

	switch build.Status.Phase {
	case builds.PhaseSuccess:
		conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
			Type:   components.BuildCondition,
			Status: conditions.ConditionCompleted,
			Reason: components.ConditionReasonSuccessful,
		})

		return &ctrl.Result{}, r.Client.Status().Update(ctx, component)

	case builds.PhaseError:
		conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
			Type:   components.BuildCondition,
			Status: conditions.ConditionError,
			Reason: "Build had a failure",
		})

		if err := r.Client.Status().Update(ctx, component); err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("E#1011: Build (%s) had an error, logs are attached to the build", build.Name)
	}

	return &ctrl.Result{}, nil
}
