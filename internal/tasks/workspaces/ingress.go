package workspaces

import (
	"context"
	"errors"
	"fmt"

	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	sequencer "se.quencer.io/api/v1alpha1"
	"se.quencer.io/api/v1alpha1/components"
	"se.quencer.io/api/v1alpha1/conditions"
	"se.quencer.io/api/v1alpha1/workspaces"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var ErrServiceNotYetReady = errors.New("internal: Waiting on Service to come online")

type IngressReconciler struct {
	client.Client
	record.EventRecorder
}

func (i *IngressReconciler) Reconcile(ctx context.Context, workspace *sequencer.Workspace) (*ctrl.Result, error) {
	if workspace.Spec.Networking.Ingress == nil {
		return nil, nil
	}

	logger := log.FromContext(ctx)

	condition := conditions.FindStatusCondition(workspace.Status.Conditions, workspaces.IngressCondition)

	if condition == nil {
		logger.Info("Workspace has an ingress defined. Initializing condition")
		i.Event(workspace, core.EventTypeNormal, "Conditions", "Initializing Ingress for workspace")

		conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
			Type:   workspaces.IngressCondition,
			Status: conditions.ConditionInitialized,
			Reason: "Workspace requires an Ingress",
		})

		return &ctrl.Result{}, i.Status().Update(ctx, workspace)
	}

	// Only allow these conditions
	if !(condition.Status == conditions.ConditionInitialized || condition.Status == conditions.ConditionWaiting) {
		return nil, nil
	}

	spec := workspace.Spec.Networking.Ingress

	services, err := i.findServices(ctx, workspace, spec.Rules)
	if errors.Is(err, ErrServiceNotYetReady) {
		i.EventRecorder.Event(workspace, core.EventTypeNormal, string(workspaces.IngressCondition), "Waiting for components to be ready")
		if condition.Status != conditions.ConditionWaiting {
			conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
				Type:   workspaces.IngressCondition,
				Status: conditions.ConditionWaiting,
				Reason: "Waiting on component to be ready",
			})
			return &ctrl.Result{}, i.Status().Update(ctx, workspace)
		}

		return &ctrl.Result{}, nil
	}

	// Future-proofing for when findServices can return a different error
	if err != nil {
		conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
			Type:   workspaces.IngressCondition,
			Status: conditions.ConditionError,
			Reason: err.Error(),
		})
		return nil, err
	}

	i.EventRecorder.Event(workspace, core.EventTypeNormal, string(workspaces.IngressCondition), "Networks configured, creating the ingress")
	conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
		Type:   workspaces.IngressCondition,
		Status: conditions.ConditionLocked,
		Reason: "Locked to create resources",
	})

	ingress := networking.Ingress{
		ObjectMeta: meta.ObjectMeta{
			OwnerReferences: []meta.OwnerReference{
				{
					Name:       workspace.Name,
					Kind:       workspace.Kind,
					APIVersion: workspace.APIVersion,
					UID:        workspace.UID,
				},
			},
			GenerateName: fmt.Sprintf("%s-", workspace.Name),
			Namespace:    workspace.Namespace,
			Labels: map[string]string{
				workspaces.InstanceLabel: workspace.Name,
			},
		},
		Spec: networking.IngressSpec{
			IngressClassName: spec.ClassName,
		},
	}

	ingress.Spec.Rules = i.ingressRules(spec.Rules, services, workspace.Status.DNS.Hostname)
	if err := i.Create(ctx, &ingress); err != nil {
		conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
			Type:   workspaces.IngressCondition,
			Status: conditions.ConditionError,
			Reason: err.Error(),
		})

		return nil, err
	}

	conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
		Type:   workspaces.IngressCondition,
		Status: conditions.ConditionCompleted,
		Reason: "Ingress is created",
	})

	return &ctrl.Result{}, i.Status().Update(ctx, workspace)
}

func (i *IngressReconciler) ingressRules(specs []workspaces.RuleSpec, services []*core.Service, hostname string) []networking.IngressRule {
	rules := []networking.IngressRule{}
	for _, spec := range specs {
		rule := networking.IngressRule{
			Host: hostname,
		}

		if spec.Subdomain != nil {
			rule.Host = fmt.Sprintf("%s.%s", *spec.Subdomain, hostname)
		}

		if spec.Path != nil {
			path := networking.HTTPIngressPath{
				Path:     *spec.Path,
				PathType: new(networking.PathType),
			}
			*path.PathType = networking.PathTypePrefix

			// The services were already checked to see if they match the rule spec, for this reason, there's no error here as
			// the service will exist. However, the indirection of this methoid and availableServices isn't really great for readability and maintenance.
			// It's expected this will be refactored to be easier to reason about.
			for _, s := range services {
				if s.Labels[components.NameLabel] == spec.ComponentName && s.Labels[components.NetworkLabel] == spec.NetworkName {
					path.Backend = networking.IngressBackend{
						Service: &networking.IngressServiceBackend{
							Name: s.Name,
							Port: networking.ServiceBackendPort{
								Number: s.Spec.Ports[0].Port,
							},
						},
					}
				}
			}

			rule.HTTP = &networking.HTTPIngressRuleValue{
				Paths: []networking.HTTPIngressPath{path},
			}
		}

		rules = append(rules, rule)
	}

	return rules
}

func (i *IngressReconciler) findServices(ctx context.Context, workspace *sequencer.Workspace, ruleSpecs []workspaces.RuleSpec) ([]*core.Service, error) {
	var list core.ServiceList
	var services []*core.Service

	selector, err := labels.Parse(fmt.Sprintf("%s=%s", workspaces.InstanceLabel, workspace.Name))
	if err != nil {
		return nil, err
	}

	err = i.List(ctx, &list, &client.ListOptions{
		Namespace:     workspace.Namespace,
		LabelSelector: selector,
	})

	if err != nil {
		i.Event(workspace, core.EventTypeWarning, "Fetching Services", err.Error())
		return nil, err
	}

	for _, ruleSpec := range ruleSpecs {
		found := false
		for i := range list.Items {
			s := &list.Items[i]
			if s.Labels[components.NameLabel] == ruleSpec.ComponentName && s.Labels[components.NetworkLabel] == ruleSpec.NetworkName {
				services = append(services, s)
				found = true
				break
			}
		}

		if !found {
			return nil, ErrServiceNotYetReady
		}
	}

	return services, err
}
