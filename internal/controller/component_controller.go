/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	core "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/builds"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/components"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	tasks "github.com/pier-oliviert/sequencer/internal/tasks/components"
)

// ComponentReconciler reconciles a Component object
type ComponentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	record.EventRecorder
}

//+kubebuilder:rbac:groups=se.quencer.io,resources=components,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=se.quencer.io,resources=components/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=se.quencer.io,resources=components/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=services;pods,verbs=get;watch;list;create;delete

func (r *ComponentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var component sequencer.Component

	if err := r.Client.Get(ctx, req.NamespacedName, &component); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("E#5001: Couldn't retrieve the build (%s) -- %w", req.NamespacedName, err)
	}

	if component.Status.Phase == "" {
		component.Status.Default()
		if err := r.Client.Status().Update(ctx, &component); err != nil {
			return r.componentFailed(ctx, ctrl.Result{}, &component, err)
		}
		return ctrl.Result{}, nil
	}

	// If the component already is in an errored phase, there's nothing
	// else to do.
	if component.Status.Phase == components.PhaseError {
		return ctrl.Result{}, nil
	}

	if result, err := (&tasks.NetworkReconciler{
		Client:        r.Client,
		EventRecorder: r.EventRecorder,
	}).Reconcile(ctx, &component); err != nil {
		return r.componentFailed(ctx, ctrl.Result{}, &component, fmt.Errorf("networking failed: %w", err))
	} else if result != nil {
		return *result, nil
	}

	if result, err := (&tasks.BuildReconciler{
		Client:        r.Client,
		EventRecorder: r.EventRecorder,
	}).Reconcile(ctx, &component); err != nil {
		return r.componentFailed(ctx, ctrl.Result{}, &component, fmt.Errorf("build failed: %w", err))
	} else if result != nil {
		return *result, nil
	}

	if result, err := (&tasks.DependenciesReconciler{
		Client:        r.Client,
		EventRecorder: r.EventRecorder,
	}).Reconcile(ctx, &component); err != nil {
		return r.componentFailed(ctx, ctrl.Result{}, &component, fmt.Errorf("dependencies failed: %w", err))
	} else if result != nil {
		return *result, nil
	}

	if result, err := (&tasks.VariablesReconciler{
		Client:        r.Client,
		EventRecorder: r.EventRecorder,
	}).Reconcile(ctx, &component); err != nil {
		return r.componentFailed(ctx, ctrl.Result{}, &component, fmt.Errorf("variables failed: %w", err))
	} else if result != nil {
		return *result, nil
	}

	if conditions.IsStatusConditionPresentAndEqual(component.Status.Conditions, components.DependenciesCondition, conditions.ConditionCompleted) {
		nr := tasks.PodReconciler{Client: r.Client, EventRecorder: r.EventRecorder}
		result, err := nr.Reconcile(ctx, &component)
		if err != nil {
			return r.componentFailed(ctx, ctrl.Result{}, &component, fmt.Errorf("pod failed: %w", err))
		} else if result != nil {
			return *result, nil
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ComponentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sequencer.Component{}).
		Watches(
			&sequencer.Build{},
			handler.EnqueueRequestsFromMapFunc(r.reconcileForBuildFunc),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&core.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.reconcileForPodFunc),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *ComponentReconciler) reconcileForBuildFunc(ctx context.Context, buildObj client.Object) []reconcile.Request {
	requests := []reconcile.Request{}
	build := buildObj.(*sequencer.Build)

	if !(build.Status.Phase == builds.PhaseError || build.Status.Phase == builds.PhaseSuccess) {
		// We're only interested in builds that are finished
		return requests
	}

	for _, owner := range build.GetOwnerReferences() {
		if owner.Kind == "Component" && owner.APIVersion == "se.quencer.io/v1alpha1" {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      owner.Name,
					Namespace: build.GetNamespace(),
				},
			})
		}
	}

	return requests
}

func (r *ComponentReconciler) reconcileForPodFunc(ctx context.Context, pod client.Object) []reconcile.Request {
	requests := []reconcile.Request{}
	for _, owner := range pod.GetOwnerReferences() {
		if owner.Kind == "Component" && owner.APIVersion == "se.quencer.io/v1alpha1" {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      owner.Name,
					Namespace: pod.GetNamespace(),
				},
			})
		}
	}
	return requests
}

func (r *ComponentReconciler) componentFailed(ctx context.Context, result ctrl.Result, component *sequencer.Component, err error) (ctrl.Result, error) {
	// Ignore 409, log and error on everything else
	if k8sErrors.IsConflict(err) {
		result.Requeue = true
		return result, nil
	}

	r.EventRecorder.Event(component, core.EventTypeWarning, string(components.PhaseError), err.Error())
	component.Status.Phase = components.PhaseError
	return result, r.Status().Update(ctx, component)
}
