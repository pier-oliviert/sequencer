package components

import (
	"context"
	"fmt"
	"strings"

	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/components"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/workspaces"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodReconciler struct {
	client.Client
	record.EventRecorder
}

func (p *PodReconciler) Reconcile(ctx context.Context, component *sequencer.Component) (*ctrl.Result, error) {
	condition := conditions.FindCondition(component.Status.Conditions, components.PodCondition)
	if condition == nil {
		condition = &conditions.Condition{
			Type:   components.PodCondition,
			Status: conditions.ConditionUnknown,
			Reason: components.ConditionReasonInitialized,
		}
	}

	if condition.Status == conditions.ConditionHealthy {
		err := p.monitorPod(ctx, component)
		if err != nil {
			conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
				Type:   components.PodCondition,
				Status: conditions.ConditionError,
				Reason: "Pod became unhealthy",
			})
		}
		return nil, err
	}

	// Do nothing more if the condition is not unknown
	if condition.Status != conditions.ConditionUnknown {
		return nil, nil
	}

	conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
		Type:   components.PodCondition,
		Status: conditions.ConditionInProgress,
		Reason: components.ConditionReasonProcessing,
	})
	if err := p.Status().Update(ctx, component); err != nil {
		return nil, err
	}

	if err := p.deployPod(ctx, component); err != nil {
		p.EventRecorder.Event(component, "Warning", string(components.PodCondition), err.Error())
		conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
			Type:   components.PodCondition,
			Status: conditions.ConditionError,
			Reason: "Pod deployment failed",
		})

		return nil, err
	}

	conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
		Type:   components.PodCondition,
		Status: conditions.ConditionHealthy,
		Reason: components.ConditionReasonCompleted,
	})

	return &ctrl.Result{}, p.Status().Update(ctx, component)
}

func (p *PodReconciler) deployPod(ctx context.Context, component *sequencer.Component) error {
	pod := &core.Pod{
		ObjectMeta: meta.ObjectMeta{
			Namespace: component.Namespace,
			Labels: map[string]string{
				components.NameLabel:     component.Spec.Name,
				components.InstanceLabel: component.Name,
			},
			GenerateName: fmt.Sprintf("component-%s-", component.Name),
			OwnerReferences: []meta.OwnerReference{
				{
					APIVersion: component.APIVersion,
					Kind:       component.Kind,
					Name:       component.Name,
					UID:        component.UID,
				},
			},
		},
		Spec: component.Spec.Pod,
	}

	if label, ok := component.Labels[workspaces.InstanceLabel]; ok {
		pod.Labels[workspaces.InstanceLabel] = label
	}

	pod.Spec.RestartPolicy = core.RestartPolicyNever
	if err := p.assignInterpolatedVariables(&pod.Spec, component); err != nil {
		return err
	}

	err := p.Client.Create(ctx, pod)
	if err != nil {
		return err
	}

	// Pod is created and as far as this reconciler loop is concerned, we're done.
	conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
		Type:   components.PodCondition,
		Status: conditions.ConditionHealthy,
		Reason: "Pod is deployed",
	})

	component.Status.Phase = components.PhaseHealthy

	return p.Client.Status().Update(ctx, component)
}

func (p *PodReconciler) assignInterpolatedVariables(spec *core.PodSpec, component *sequencer.Component) error {
	for i := range spec.Containers {
		container := &spec.Containers[i]
		if strings.HasPrefix(container.Image, components.InterpolationDelimStart) {
			p.Eventf(component, "Normal", string(components.PhaseInitializing), "Container uses a build, replacing the image %s", container.Image)
			found := false
			for _, v := range component.Status.Variables {
				if v.Name == container.Image {
					container.Image = v.Value
					found = true
				}
			}

			if !found {
				return fmt.Errorf("E#2001: Could not find a variable to interpolate for %s", container.Image)
			}
		}

		for j := range container.Env {
			env := &container.Env[j]
			if strings.HasPrefix(env.Value, components.InterpolationDelimStart) {
				found := false
				for _, v := range component.Status.Variables {
					if v.Name == env.Value {
						env.Value = v.Value
						found = true
					}
				}
				if !found {
					return fmt.Errorf("E#2001: Could not find a variable to interpolate for %s", env.Value)
				}
			}
		}
	}

	return nil
}

func (p *PodReconciler) monitorPod(ctx context.Context, component *sequencer.Component) error {
	list := &core.PodList{}

	selector, err := labels.Parse(fmt.Sprintf("%s=%s", components.InstanceLabel, component.Name))
	if err != nil {
		return fmt.Errorf("E#3001: failed to parse the label selector -- %w", err)
	}

	err = p.List(ctx, list, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     component.GetNamespace(),
	})

	if err != nil {
		return fmt.Errorf("E#5002: failure to retrieve a list of pods for component (%s) -- Label Selector: %s", component.Name, selector.String())
	}

	if len(list.Items) != 1 {
		if len(list.Items) == 0 {
			return fmt.Errorf("E#2003: no pod exists for component (%s)", component.Name)
		}

		return fmt.Errorf("E#2004: Expected to retrieve a single pod for component (%s), got %d. Label Selector: %s", component.Name, list.Size(), selector.String())
	}

	pod := list.Items[0]
	// Are pod still running?
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Terminated != nil {
			if cs.State.Terminated.ExitCode != 0 {
				return fmt.Errorf("E#2005: Pod (%s) had a failure in one of the container (%s) -- Reason: %s", pod.Name, cs.Name, cs.State.Terminated.Reason)
			}
		}
	}

	return nil
}
