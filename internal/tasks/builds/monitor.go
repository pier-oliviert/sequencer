package builds

import (
	"context"
	"errors"
	"fmt"

	core "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	sequencer "se.quencer.io/api/v1alpha1"
	builds "se.quencer.io/api/v1alpha1/builds"
	"se.quencer.io/api/v1alpha1/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrUnexpectedPodFailure = errors.New("#E1009 pod had an unexpected failure")
)

type MonitorReconciler struct {
	client.Client
	record.EventRecorder
}

func (r *MonitorReconciler) Reconcile(ctx context.Context, build *sequencer.Build) (*ctrl.Result, error) {
	// Let's first see if one of the existing condition has failed since that would be unrecoverable
	// and the reconcilation can be done with this build.
	if conditions.IsAnyConditionWithStatus(build.Status.Conditions, conditions.ConditionError) {
		if build.Status.Phase != builds.PhaseError {
			if build.Status.PodRef == nil {
				return nil, errors.New("E#1008: No pod dispatched for the build, does the operator have the right permission?")
			}

			podDescriptor := build.Status.PodRef.NamespacedName()
			build.Status.Phase = builds.PhaseError
			r.EventRecorder.Event(build, "Normal", string(build.Status.Phase), fmt.Sprintf("Build had an error, logs are located in pod(%s/%s)", podDescriptor.Namespace, podDescriptor.Name))

			if err := r.Client.Status().Update(ctx, build); err != nil {
				return nil, err
			}
		}

		return &ctrl.Result{}, nil
	}

	if conditions.AreAllConditionsWithStatus(build.Status.Conditions, conditions.ConditionCompleted) {
		if build.Status.Phase == builds.PhaseSuccess {
			// The task already processed this phase, nothing else to do at this point.
			return nil, nil
		}

		build.Status.Phase = builds.PhaseSuccess
		podDescriptor := build.Status.PodRef.NamespacedName()
		r.EventRecorder.Event(build, "Normal", string(build.Status.Phase), fmt.Sprintf("Build finished in pod(%s/%s)", podDescriptor.Namespace, podDescriptor.Name))

		if err := r.Client.Status().Update(ctx, build); err != nil {
			return nil, err
		}

		return &ctrl.Result{}, nil
	}

	if conditions.IsStatusConditionPresentAndEqual(build.Status.Conditions, builds.PodScheduledCondition, conditions.ConditionCompleted) {
		// Monitor the health of the pod to make sure it hasn't
		// been terminated without changing the status of the build.
		var pod core.Pod
		err := r.Get(ctx, build.Status.PodRef.NamespacedName(), &pod)
		if err != nil {
			return nil, err
		}

		// Monitor the health of the pod to make sure it hasn't
		// Are pod still running?
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Terminated != nil {
				if cs.State.Terminated.ExitCode != 0 {
					return nil, ErrUnexpectedPodFailure
				}
			}
		}
		return nil, nil
	}

	return nil, nil
}
