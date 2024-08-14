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
	"k8s.io/apimachinery/pkg/fields"
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
	builds "github.com/pier-oliviert/sequencer/api/v1alpha1/builds"
	tasks "github.com/pier-oliviert/sequencer/internal/tasks/builds"
)

const (
	kPodStatusField = ".status.pod"
)

// BuildReconciler reconciles a Build object
type BuildReconciler struct {
	Scheme *runtime.Scheme
	client.Client
	record.EventRecorder
}

//+kubebuilder:rbac:groups=se.quencer.io,resources=builds,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups=se.quencer.io,resources=builds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=se.quencer.io,resources=builds/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=pods;secrets,verbs=get;watch;list;create;delete

func (r *BuildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var build sequencer.Build
	if err := r.Get(ctx, req.NamespacedName, &build); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("E#5001: Couldn't retrieve the build (%s) -- %w", req.NamespacedName, err)
	}

	if build.Status.Phase == "" {
		build.Status.Default()
		if err := r.Status().Update(ctx, &build); err != nil {
			return r.buildFailed(ctx, ctrl.Result{}, &build, err)
		}
		return ctrl.Result{}, nil
	}

	// If the component already is in an errored phase, there's nothing
	// else to do.
	if build.Status.Phase == builds.PhaseError {
		return ctrl.Result{}, nil
	}

	if result, err := (&tasks.PodReconciler{
		Client:        r.Client,
		EventRecorder: r.EventRecorder,
	}).Reconcile(ctx, &build); err != nil {
		return r.buildFailed(ctx, ctrl.Result{}, &build, err)
	} else if result != nil {
		return *result, nil
	}

	// Pod Reconciler has allowed to continue.
	if result, err := (&tasks.MonitorReconciler{
		Client:        r.Client,
		EventRecorder: r.EventRecorder,
	}).Reconcile(ctx, &build); err != nil {
		return r.buildFailed(ctx, ctrl.Result{}, &build, fmt.Errorf("monitoring failed: %w", err))
	} else if result != nil {
		return *result, nil
	}

	// Pod seems healthy, need to wait for the builder pod to finish
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &sequencer.Build{}, kPodStatusField, func(rawObj client.Object) []string {
		build := rawObj.(*sequencer.Build)
		ref := build.Status.PodRef
		if ref == nil {
			return nil
		}

		return []string{ref.Name}
	})

	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&sequencer.Build{}).
		Watches(
			&core.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueBuildReconcilerForOwnedPod),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *BuildReconciler) enqueueBuildReconcilerForOwnedPod(ctx context.Context, pod client.Object) []reconcile.Request {
	builds := &sequencer.BuildList{}

	err := r.List(ctx, builds, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(kPodStatusField, pod.GetName()),
		Namespace:     pod.GetNamespace(),
	})

	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(builds.Items))
	for i, item := range builds.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

func (r *BuildReconciler) buildFailed(ctx context.Context, result ctrl.Result, build *sequencer.Build, err error) (ctrl.Result, error) {
	if k8sErrors.IsConflict(err) {
		result.Requeue = true
		return result, nil
	}

	r.Event(build, "Warning", string(builds.PhaseError), err.Error())
	build.Status.Phase = builds.PhaseError
	return result, r.Status().Update(ctx, build)
}
