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
	"time"

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

	sequencer "se.quencer.io/api/v1alpha1"
	"se.quencer.io/api/v1alpha1/workspaces"

	tasks "se.quencer.io/internal/tasks/workspaces"
)

// WorkspaceReconciler reconciles a Workspace object
type WorkspaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	record.EventRecorder
}

//+kubebuilder:rbac:groups=se.quencer.io,resources=workspaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=se.quencer.io,resources=workspaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=se.quencer.io,resources=workspaces/conditions,verbs=get;update;patch
//+kubebuilder:rbac:groups=se.quencer.io,resources=workspaces/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *WorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var workspace sequencer.Workspace

	if err := r.Client.Get(ctx, req.NamespacedName, &workspace); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("E#5001: Couldn't retrieve the build (%s) -- %w", req.NamespacedName, err)
	}

	if workspace.Status.Phase == "" {
		workspace.Status = workspaces.DefaultStatus()
		return ctrl.Result{}, r.Status().Update(ctx, &workspace)
	}

	if !workspace.ObjectMeta.DeletionTimestamp.IsZero() && workspace.Status.Phase != workspaces.PhaseTerminating {
		workspace.Status.Phase = workspaces.PhaseTerminating
		r.Eventf(&workspace, core.EventTypeNormal, string(workspaces.PhaseTerminating), "Waiting for external dependencies to be cleaned up")
		return ctrl.Result{}, r.Status().Update(ctx, &workspace)
	}

	if workspace.Status.Phase == workspaces.PhaseError {
		return ctrl.Result{}, nil
	}

	if result, err := (&tasks.TunnelingReconciler{
		Client:        r.Client,
		EventRecorder: r.EventRecorder,
	}).Reconcile(ctx, &workspace); err != nil {
		return r.workspaceFailed(ctx, ctrl.Result{}, &workspace, fmt.Errorf("Tunneling->%w", err))
	} else if result != nil {
		return *result, nil
	}

	if result, err := (&tasks.DNSReconciler{
		Client:        r.Client,
		EventRecorder: r.EventRecorder,
	}).Reconcile(ctx, &workspace); err != nil {
		return r.workspaceFailed(ctx, ctrl.Result{}, &workspace, fmt.Errorf("DNS->%w", err))
	} else if result != nil {
		return *result, nil
	}

	if result, err := (&tasks.RulesReconciler{
		Client:        r.Client,
		EventRecorder: r.EventRecorder,
	}).Reconcile(ctx, &workspace); err != nil {
		return r.workspaceFailed(ctx, ctrl.Result{}, &workspace, fmt.Errorf("Rules->%w", err))
	} else if result != nil {
		return *result, nil
	}

	if result, err := (&tasks.ComponentsReconciler{
		Client:        r.Client,
		EventRecorder: r.EventRecorder,
	}).Reconcile(ctx, &workspace); err != nil {
		return r.workspaceFailed(ctx, ctrl.Result{}, &workspace, fmt.Errorf("Components->%w", err))
	} else if result != nil {
		return *result, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkspaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sequencer.Workspace{}).
		Watches(
			&sequencer.Component{},
			handler.EnqueueRequestsFromMapFunc(r.reconcileForComponentFunc),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *WorkspaceReconciler) reconcileForComponentFunc(ctx context.Context, obj client.Object) []reconcile.Request {
	requests := []reconcile.Request{}
	component := obj.(*sequencer.Component)

	if label, ok := component.Labels[workspaces.InstanceLabel]; !ok {
		return requests
	} else {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      label,
				Namespace: component.Namespace,
			},
		})
	}
	return requests
}

func (r *WorkspaceReconciler) workspaceFailed(ctx context.Context, result ctrl.Result, workspace *sequencer.Workspace, err error) (ctrl.Result, error) {
	// Ignore 409, log and error on everything else
	if k8sErrors.IsConflict(err) {
		result.RequeueAfter = 1 * time.Second
		return result, nil
	}

	r.Event(workspace, core.EventTypeWarning, string(workspaces.PhaseError), err.Error())
	workspace.Status.Phase = workspaces.PhaseError
	return result, r.Status().Update(ctx, workspace)
}
