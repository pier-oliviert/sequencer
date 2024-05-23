package components

import (
	"context"
	"fmt"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	sequencer "se.quencer.io/api/v1alpha1"
	"se.quencer.io/api/v1alpha1/components"
	"se.quencer.io/api/v1alpha1/conditions"
	"se.quencer.io/api/v1alpha1/workspaces"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NetworkReconciler struct {
	client.Client
	record.EventRecorder
}

func (n *NetworkReconciler) Reconcile(ctx context.Context, component *sequencer.Component) (*ctrl.Result, error) {
	condition := conditions.FindStatusCondition(component.Status.Conditions, components.NetworkCondition)
	if condition == nil {
		condition = &conditions.Condition{
			Type:   components.NetworkCondition,
			Status: conditions.ConditionUnknown,
			Reason: components.ConditionReasonProcessing,
		}
	}

	// Skipping this reconciliation loop if the condition is not ConditionUnknown
	if condition.Status != conditions.ConditionUnknown {
		return nil, nil
	}

	conditions.SetStatusCondition(&component.Status.Conditions, conditions.Condition{
		Type:   components.NetworkCondition,
		Status: conditions.ConditionInProgress,
		Reason: components.ConditionReasonProcessing,
	})
	if err := n.Client.Status().Update(ctx, component); err != nil {
		return nil, err
	}

	for _, ns := range component.Spec.Networks {
		svc := &core.Service{
			ObjectMeta: meta.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-", component.Spec.Name),
				Namespace:    component.GetNamespace(),
				Labels: map[string]string{
					components.NameLabel:     component.Spec.Name,
					components.InstanceLabel: component.Name,
					components.NetworkLabel:  ns.Name,
				},
				OwnerReferences: []meta.OwnerReference{{
					APIVersion: component.APIVersion,
					Kind:       component.Kind,
					Name:       component.Name,
					UID:        component.UID,
				}},
			},
			Spec: core.ServiceSpec{
				Selector: make(map[string]string),
			},
		}

		if label, ok := component.Labels[workspaces.InstanceLabel]; ok {
			svc.Labels[workspaces.InstanceLabel] = label
		}

		svc.Spec.Selector[components.InstanceLabel] = component.Name

		svc.Spec.Ports = append(svc.Spec.Ports, ns.ServicePort)
		if err := n.Client.Create(ctx, svc); err != nil {
			return nil, err
		}
	}

	conditions.SetStatusCondition(&component.Status.Conditions, conditions.Condition{
		Type:   components.NetworkCondition,
		Status: conditions.ConditionCompleted,
		Reason: components.ConditionReasonCompleted,
	})

	return &ctrl.Result{}, n.Client.Status().Update(ctx, component)
}
