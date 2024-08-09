package components

import (
	"context"
	"fmt"

	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/components"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/workspaces"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DependenciesReconciler struct {
	client.Client
	record.EventRecorder
}

func (r *DependenciesReconciler) Reconcile(ctx context.Context, component *sequencer.Component) (*ctrl.Result, error) {
	_ = log.FromContext(ctx)
	condition := conditions.FindCondition(component.Status.Conditions, components.DependenciesCondition)
	if condition == nil {
		condition = &conditions.Condition{
			Type:   components.DependenciesCondition,
			Status: conditions.ConditionUnknown,
			Reason: components.ConditionReasonInitialized,
		}
	}

	if condition.Status == conditions.ConditionUnknown {
		return r.setup(ctx, component)
	}

	if condition.Status == conditions.ConditionWaiting {
		return r.watch(ctx, component)
	}

	return nil, nil
}

func (r *DependenciesReconciler) setup(ctx context.Context, component *sequencer.Component) (*ctrl.Result, error) {
	conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
		Type:   components.DependenciesCondition,
		Status: conditions.ConditionInProgress,
		Reason: components.ConditionReasonProcessing,
	})

	if err := r.Status().Update(ctx, component); err != nil {
		return nil, err
	}

	// If the component is not part of a workspace, this reconciliation loop can be skipped.
	if _, ok := component.Labels[workspaces.InstanceLabel]; !ok {
		conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
			Type:   components.DependenciesCondition,
			Status: conditions.ConditionCompleted,
			Reason: "Skipped, component is not part of a workspace",
		})

		return &ctrl.Result{}, r.Status().Update(ctx, component)
	}

	return r.watch(ctx, component)
}

func (r *DependenciesReconciler) watch(ctx context.Context, component *sequencer.Component) (*ctrl.Result, error) {
	selector, err := labels.Parse(fmt.Sprintf("%s=%s", workspaces.InstanceLabel, component.Labels[workspaces.InstanceLabel]))
	if err != nil {
		return nil, fmt.Errorf("E#3001: failed to parse the label selector -- %w", err)
	}

	var list sequencer.ComponentList
	err = r.List(ctx, &list, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     component.Namespace,
	})
	if err != nil {
		conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
			Type:   components.DependenciesCondition,
			Status: conditions.ConditionError,
			Reason: "Error loading dependencies",
		})

		return nil, r.Status().Update(ctx, component)
	}

	conditions.SetCondition(&component.Status.Conditions, r.conditionForDependency(list.Items, component))

	return &ctrl.Result{}, r.Status().Update(ctx, component)
}

func (r *DependenciesReconciler) conditionForDependency(deps []sequencer.Component, self *sequencer.Component) conditions.Condition {
	condition := conditions.Condition{
		Type:   components.DependenciesCondition,
		Status: conditions.ConditionCompleted,
		Reason: "All dependencies are met",
	}

Chain:
	for _, component := range deps {
		for _, dep := range self.Spec.DependsOn {
			if dep.ComponentName == component.Spec.Name {
				c := conditions.FindCondition(component.Status.Conditions, dep.ConditionType)
				if c == nil || c.Status != dep.ConditionStatus {
					condition.Status = conditions.ConditionWaiting
					condition.Reason = fmt.Sprintf("Waiting on %s's Condition to be %s:%s", component.Name, dep.ConditionType, dep.ConditionStatus)
					break Chain
				}

				if c.Status == conditions.ConditionError {
					condition.Status = conditions.ConditionError
					condition.Reason = fmt.Sprintf("Component (%s) had an error", component.Name)
					break Chain
				}
			}
		}
	}

	return condition
}
