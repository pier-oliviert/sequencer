package builds

import (
	"context"
	"fmt"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	sequencer "se.quencer.io/api/v1alpha1"
	builds "se.quencer.io/api/v1alpha1/builds"
	"se.quencer.io/api/v1alpha1/conditions"
	"se.quencer.io/api/v1alpha1/utils"
	"se.quencer.io/internal/tasks/builds/specs"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodReconciler struct {
	client.Client
	record.EventRecorder
}

func (r *PodReconciler) Reconcile(ctx context.Context, build *sequencer.Build) (*ctrl.Result, error) {
	condition := conditions.FindStatusCondition(build.Status.Conditions, builds.PodScheduledCondition)
	if condition == nil {
		condition = &conditions.Condition{
			Type:   builds.PodScheduledCondition,
			Status: conditions.ConditionUnknown,
			Reason: builds.ConditionReasonInitialized,
		}
	}

	// Passthrough
	if condition.Status != conditions.ConditionUnknown {
		return nil, nil
	}

	conditions.SetStatusCondition(&build.Status.Conditions, conditions.Condition{
		Type:   builds.PodScheduledCondition,
		Status: conditions.ConditionInProgress,
		Reason: builds.ConditionReasonProcessing,
	})

	if err := r.Status().Update(ctx, build); err != nil {
		return nil, err
	}

	// The Build is just initialized and nothing has been processed, yet. For the Build to actually start, a pod
	// needs to be scheduled with the right service account so that it can update the state of the Build has it goes
	// through each of the steps.
	pod := specs.PodFor(build)

	container := specs.BuilderContainerFor(build)
	volumes, err := specs.VolumesForContainer(&container, build)
	if err != nil {
		return &ctrl.Result{}, err
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, volumes...)
	pod.Spec.Volumes = append(pod.Spec.Volumes, specs.BuildkitSharedVolumesVolumes()...)

	volumes, mounts, err := r.configureContainerForCredentials(ctx, build)
	if err != nil {
		return nil, fmt.Errorf("E#TODO: Could not attach credentials to the build -- %w", err)
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, volumes...)
	container.VolumeMounts = append(container.VolumeMounts, mounts...)
	pod.Spec.Containers = []core.Container{container, specs.BackendContainerFor(build)}

	err = r.Create(ctx, pod)
	if err != nil {
		conditions.SetStatusCondition(&build.Status.Conditions, conditions.Condition{
			Type:   builds.PodScheduledCondition,
			Status: conditions.ConditionError,
			Reason: err.Error(),
		})

		return nil, fmt.Errorf("E#1001: Could not create the pod for the build -- %w", err)
	}

	// It's important to set the condition first before calling conditions.Phase() as otherwise it would
	// not include the state of this condition when deriving the value.
	build.Status.PodRef = utils.NewReference(pod)
	build.Status.Phase = builds.PhaseRunning
	conditions.SetStatusCondition(&build.Status.Conditions, conditions.Condition{
		Type:   builds.PodScheduledCondition,
		Status: conditions.ConditionCompleted,
		Reason: "Pod created",
	})

	if err := r.Status().Update(ctx, build); err != nil {
		return nil, err
	}

	return &ctrl.Result{}, nil
}

func (r *PodReconciler) configureContainerForCredentials(ctx context.Context, build *sequencer.Build) (volumes []core.Volume, mounts []core.VolumeMount, err error) {
	for _, cr := range build.Spec.ContainerRegistries {
		key := types.NamespacedName{
			Namespace: build.Namespace,
			Name:      cr.Credentials.SecretRef.Name,
		}

		var secret core.Secret

		if err := r.Get(ctx, key, &secret); err != nil {
			return nil, nil, err
		}

		if !cr.Credentials.IsValidForSecret(&secret) {
			return nil, nil, fmt.Errorf("E#TODO: Secret is not valid for the credential authScheme: %s", cr.Credentials.AuthScheme)
		}

		mounts = append(mounts, core.VolumeMount{
			Name:      *cr.Credentials.Name,
			MountPath: fmt.Sprintf("/var/build/registries/%s", *cr.Credentials.Name),
			ReadOnly:  true,
		})

		volumes = append(volumes, core.Volume{
			Name: *cr.Credentials.Name,
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName: cr.Credentials.SecretRef.Name,
				},
			},
		})
	}

	for _, ic := range build.Spec.ImportContent {
		if !ic.IsPrivate() {
			continue
		}

		key := types.NamespacedName{
			Namespace: build.Namespace,
			Name:      ic.Credentials.SecretRef.Name,
		}

		var secret core.Secret

		if err := r.Get(ctx, key, &secret); err != nil {
			return nil, nil, err
		}

		if !ic.Credentials.IsValidForSecret(&secret) {
			return nil, nil, fmt.Errorf("E#TODO: Secret is not valid for the credential authScheme: %s", ic.Credentials.AuthScheme)
		}

		mounts = append(mounts, core.VolumeMount{
			Name:      *ic.Credentials.Name,
			MountPath: fmt.Sprintf("/var/build/imports/%s", *ic.Credentials.Name),
			ReadOnly:  true,
		})

		volumes = append(volumes, core.Volume{
			Name: *ic.Credentials.Name,
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName: ic.Credentials.SecretRef.Name,
				},
			},
		})
	}

	return volumes, mounts, err
}
