package components

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/components"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/workspaces"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var resolverParser = regexp.MustCompile(`^\${([a-zA-Z]+)::((?:[a-zA-Z]+\.?)+)}`)
var ErrNotAResolvableValue = errors.New("E#2006: Variable couldn't be parsed")

type VariablesReconciler struct {
	client.Client
	record.EventRecorder
}

func (r *VariablesReconciler) Reconcile(ctx context.Context, component *sequencer.Component) (*ctrl.Result, error) {
	_ = log.FromContext(ctx)
	condition := conditions.FindCondition(component.Status.Conditions, components.VariablesCondition)
	if condition == nil {
		condition = &conditions.Condition{
			Type:   components.VariablesCondition,
			Status: conditions.ConditionUnknown,
			Reason: components.ConditionReasonInitialized,
		}
	}

	// Skipping this reconciliation loop if the condition is not unknown
	if condition.Status != conditions.ConditionUnknown {
		return nil, nil
	}

	conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
		Type:   components.VariablesCondition,
		Status: conditions.ConditionInProgress,
		Reason: components.ConditionReasonProcessing,
	})

	if err := r.Status().Update(ctx, component); err != nil {
		return nil, err
	}

	// If the component is not part of a workspace, this reconciliation loop can be skipped.
	if _, ok := component.Labels[workspaces.InstanceLabel]; !ok {
		conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
			Type:   components.VariablesCondition,
			Status: conditions.ConditionCompleted,
			Reason: "Skipped, component is not part of a workspace",
		})
		return &ctrl.Result{}, r.Status().Update(ctx, component)
	}

	for _, container := range component.Spec.Pod.Containers {
		if strings.HasPrefix(container.Image, components.InterpolationDelimStart) {
			variable, err := r.VariableFrom(ctx, container.Image, component)
			if err != nil {
				conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
					Type:   components.VariablesCondition,
					Status: conditions.ConditionError,
					Reason: err.Error(),
				})

				return nil, err
			}

			component.Status.Variables = append(component.Status.Variables, *variable)
		}

		for _, env := range container.Env {
			if !strings.HasPrefix(env.Value, components.InterpolationDelimStart) {
				continue
			}

			variable, err := r.VariableFrom(ctx, env.Value, component)
			if err != nil {
				conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
					Type:   components.VariablesCondition,
					Status: conditions.ConditionError,
					Reason: err.Error(),
				})

				return nil, err
			}

			component.Status.Variables = append(component.Status.Variables, *variable)
		}
	}

	conditions.SetCondition(&component.Status.Conditions, conditions.Condition{
		Type:   components.VariablesCondition,
		Status: conditions.ConditionCompleted,
		Reason: "All variables processed",
	})

	return &ctrl.Result{}, r.Client.Status().Update(ctx, component)
}

func (r *VariablesReconciler) VariableFrom(ctx context.Context, rawPath string, component *sequencer.Component) (*components.GeneratedVariable, error) {
	resolver, err := newResolver(rawPath, component)
	if err != nil {
		return nil, err
	}

	value, err := resolver.Value(ctx, r.Client)
	if err != nil {
		return nil, err
	}

	return &components.GeneratedVariable{
		Name:  rawPath,
		Value: value,
	}, nil
}

func newResolver(content string, component *sequencer.Component) (variableResolver, error) {
	parsed := resolverParser.FindSubmatch([]byte(content))
	if parsed == nil {
		return nil, ErrNotAResolvableValue
	}

	switch t := string(parsed[1]); t {
	case "build":
		return &buildResolver{
			namespace:     component.Namespace,
			componentName: component.Name,
			buildName:     string(parsed[2]),
			content:       content,
		}, nil

	case "components":
		return &serviceResolver{
			namespace:     component.Namespace,
			workspaceName: component.Labels[workspaces.InstanceLabel],
			params:        strings.Split(string(parsed[2]), "."),
			content:       content,
		}, nil

	case "ingress":
		return &ingressResolver{
			namespace:     component.Namespace,
			workspaceName: component.Labels[workspaces.InstanceLabel],
			params:        strings.Split(string(parsed[2]), "."),
			content:       content,
		}, nil
	}

	return nil, fmt.Errorf("E#2007: no resolver exists for (%s)", string(parsed[1]))
}

type variableResolver interface {
	Value(context.Context, client.Client) (string, error)
}

type buildResolver struct {
	namespace     string
	componentName string
	buildName     string
	content       string
}

func (br buildResolver) Value(ctx context.Context, c client.Client) (string, error) {
	list := &sequencer.BuildList{}

	selector, err := labels.Parse(fmt.Sprintf("%s=%s", components.NameLabel, br.componentName))
	if err != nil {
		return "", fmt.Errorf("E#3001: failed to parse the label selector -- %w", err)
	}

	err = c.List(ctx, list, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     br.namespace,
	})

	if err != nil {
		return "", fmt.Errorf("E#5002: failed to find a build using the reference stored in the component -- %w", err)
	}

	if len(list.Items) == 0 {
		return "", fmt.Errorf("E#2008: failed to find a build using the reference stored in the component (%s)", br.componentName)
	}

	var build *sequencer.Build
	for _, b := range list.Items {
		if b.Spec.Name == br.buildName {
			build = &b
			break
		}
	}

	if build == nil {
		return "", fmt.Errorf("E#2009: Failed to find a build within component(%s) that has the name (%s)", br.componentName, br.buildName)
	}

	var name, digest string
	for _, image := range build.Status.Images {
		indexManifest, err := image.ParseIndexManifest()

		if err != nil {
			return "", fmt.Errorf("E#2010: failed to decode the index manifest -- %w", err)
		}

		for _, manifest := range indexManifest.Manifests {
			digest = manifest.Digest.String()
			if digest != "" {
				break
			}
		}

		if digest != "" {
			name = image.URL
			break
		}
	}

	if name == "" || digest == "" {
		return "", errors.New("E#2011: Build referenced by the component don't include a valid image")
	}

	return fmt.Sprintf("%s@%s", name, digest), nil
}

type serviceResolver struct {
	namespace     string
	workspaceName string
	params        []string
	content       string
}

func (sr serviceResolver) Value(ctx context.Context, c client.Client) (string, error) {
	if len(sr.params) != 3 {
		return "", fmt.Errorf("E#2012: Unexpected value. Expected format `${components::componentName.section.serviceName}`, got %s", sr.content)
	}

	if sr.Section() != "networks" {
		return "", errors.New("E#2013: Invalid section, only supported sections are: networks")
	}

	var service *core.Service

	selector, err := labels.Parse(fmt.Sprintf("%s=%s", workspaces.InstanceLabel, sr.workspaceName))
	if err != nil {
		return "", fmt.Errorf("E#3001: failed to parse the label selector -- %w", err)
	}

	var list core.ServiceList
	err = c.List(ctx, &list, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     sr.namespace,
	})
	if err != nil {
		return "", err
	}

	for _, s := range list.Items {
		if s.Labels[components.NameLabel] == sr.ComponentName() && s.Labels[components.NetworkLabel] == sr.NetworkName() {
			service = &s
			break
		}
	}

	if service == nil {
		return "", fmt.Errorf("E#2008: couldn't find a service for matching %s=%s & %s=%s", components.InstanceLabel, sr.ComponentName(), components.NetworkLabel, sr.NetworkName())
	}

	return resolverParser.ReplaceAllString(sr.content, fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace)), nil
}

func (sr serviceResolver) ComponentName() string {
	return sr.params[0]
}

func (sr serviceResolver) Section() string {
	return sr.params[1]
}

func (sr serviceResolver) NetworkName() string {
	return sr.params[2]
}

type ingressResolver struct {
	namespace     string
	workspaceName string
	params        []string
	content       string
}

func (ir ingressResolver) Value(ctx context.Context, c client.Client) (string, error) {
	var ingress *networking.Ingress
	if len(ir.params) > 2 {
		return "", fmt.Errorf("E#2012: Unexpected value. Expected format `${ingress::subdomain?.ingressName}`, got `%s`", ir.content)
	}

	selector, err := labels.Parse(fmt.Sprintf("%s=%s", workspaces.InstanceLabel, ir.workspaceName))
	if err != nil {
		return "", fmt.Errorf("E#3001: failed to parse the label selector -- %w", err)
	}

	var list networking.IngressList
	err = c.List(ctx, &list, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     ir.namespace,
	})
	if err != nil {
		return "", err
	}

	for _, i := range list.Items {
		if i.Labels[workspaces.InstanceLabel] == ir.workspaceName && i.Labels[workspaces.IngressLabel] == ir.RuleName() {
			ingress = &i
			break
		}
	}

	if ingress == nil {
		return "", fmt.Errorf("E#2008: couldn't find an ingress for matching %s=%s & %s=%s", workspaces.InstanceLabel, ir.workspaceName, workspaces.IngressLabel, ir.RuleName())
	}

	if ir.HasSubdomain() {
		return fmt.Sprintf("%s.%s", ir.Subdomain(), ingress.Name), nil
	}
	return ingress.Name, nil
}

func (ir ingressResolver) RuleName() string {
	if len(ir.params) == 1 {
		return ir.params[0]
	}

	return ir.params[1]
}

func (ir ingressResolver) HasSubdomain() bool {
	return len(ir.params) == 2
}

func (ir ingressResolver) Subdomain() string {
	return ir.params[1]
}
