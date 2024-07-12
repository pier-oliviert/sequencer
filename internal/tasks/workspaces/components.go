package workspaces

import (
	"context"
	"fmt"

	core "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/components"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/workspaces"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ComponentsReconciler struct {
	client.Client
	record.EventRecorder
}

func (r *ComponentsReconciler) ReconcileComponentHealth(ctx context.Context, workspace *sequencer.Workspace) (*ctrl.Result, error) {
	selector, err := labels.Parse(fmt.Sprintf("%s=%s", workspaces.InstanceLabel, workspace.Name))
	if err != nil {
		return nil, fmt.Errorf("E#3001: failed to parse the label selector -- %w", err)
	}

	// Let's load all the components for this
	var list sequencer.ComponentList
	err = r.Client.List(ctx, &list, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     workspace.Namespace,
	})

	if err != nil {
		return nil, err
	}

	var componentsHealthy []*sequencer.Component

	for _, component := range list.Items {
		if component.Status.Phase == components.PhaseHealthy {
			componentsHealthy = append(componentsHealthy, &component)
			continue
		}

		if component.Status.Phase == components.PhaseError {
			conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
				Type:   workspaces.ComponentCondition,
				Status: conditions.ConditionError,
				Reason: fmt.Sprintf("Error in component (%s)", component.Name),
			})
			workspace.Status.Phase = workspaces.PhaseError

			r.Eventf(workspace, core.EventTypeWarning, "Components", "Component (%s) has failed", component.Name)

			return &ctrl.Result{}, r.Status().Update(ctx, workspace)
		}
	}

	if len(componentsHealthy) == len(workspace.Spec.Components) && workspace.Status.Phase != workspaces.PhaseHealthy {
		conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
			Type:   workspaces.ComponentCondition,
			Status: conditions.ConditionHealthy,
			Reason: "All components are healthy",
		})
		workspace.Status.Phase = workspaces.PhaseHealthy

		r.Event(workspace, core.EventTypeNormal, "Components", "All components are healthy")
		return &ctrl.Result{}, r.Status().Update(ctx, workspace)
	}

	return nil, nil
}

func (r *ComponentsReconciler) Reconcile(ctx context.Context, workspace *sequencer.Workspace) (*ctrl.Result, error) {
	_ = log.FromContext(ctx)

	if workspace.Status.Phase == workspaces.PhaseTerminating {
		return nil, nil
	}

	condition := conditions.FindStatusCondition(workspace.Status.Conditions, workspaces.ComponentCondition)
	if condition == nil {
		condition = &conditions.Condition{
			Type:   workspaces.ComponentCondition,
			Status: conditions.ConditionUnknown,
			Reason: workspaces.ConditionReasonInitialized,
		}
	}

	if condition.Status != conditions.ConditionUnknown {
		return r.ReconcileComponentHealth(ctx, workspace)
	}

	conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
		Type:   workspaces.ComponentCondition,
		Status: conditions.ConditionInProgress,
		Reason: "Deploying components",
	})
	if err := r.Status().Update(ctx, workspace); err != nil {
		return nil, err
	}

	for _, component := range workspace.Spec.Components {
		err := r.createComponent(ctx, workspace, &component)
		if err != nil {
			conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
				Type:   workspaces.ComponentCondition,
				Status: conditions.ConditionError,
				Reason: fmt.Sprintf("Error creating component (%s)", component.Name),
			})
			if err := r.Status().Update(ctx, workspace); err != nil {
				return nil, err
			}

			return nil, fmt.Errorf("E#3002: Error creating component (%s) -- %w", component.Name, err)
		}
	}

	conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
		Type:   workspaces.ComponentCondition,
		Status: conditions.ConditionCreated,
		Reason: workspaces.ConditionReasonDeploying,
	})
	if err := r.Status().Update(ctx, workspace); err != nil {
		return nil, err
	}

	return &ctrl.Result{}, nil
}

func (r *ComponentsReconciler) createComponent(ctx context.Context, workspace *sequencer.Workspace, spec *sequencer.ComponentSpec) error {
	component := sequencer.Component{
		ObjectMeta: meta.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", spec.Name),
			Namespace:    workspace.Namespace,
			Labels: map[string]string{
				workspaces.InstanceLabel: workspace.Name,
				components.NameLabel:     spec.Name,
			},
			OwnerReferences: []meta.OwnerReference{
				{
					Name:       workspace.Name,
					Kind:       workspace.Kind,
					APIVersion: workspace.APIVersion,
					UID:        workspace.UID,
				},
			},
		},
		Spec: *spec,
	}

	return r.Create(ctx, &component)
}
