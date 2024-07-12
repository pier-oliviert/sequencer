package specs

import (
	"fmt"

	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/builds"
	core "k8s.io/api/core/v1"
	"k8s.io/utils/env"
)

func BuilderContainerFor(build *sequencer.Build) core.Container {
	image := build.Spec.Runtime.Image
	if image == nil {
		image = new(string)
		*image = env.GetString("BUILDER_IMAGE", "builder:dev")
	}

	return core.Container{
		Name:  "build",
		Image: *image,
		Env: []core.EnvVar{
			{
				Name:  "BUILD_REFERENCE",
				Value: build.GetReference().String(),
			},
			{
				Name:  "BUILD_SECRETS_PATH",
				Value: kBuildSecretsPath,
			},
			{
				Name:  "BUILD_ARGUMENTS_PATH",
				Value: kBuildArgumentsPath,
			},
			{
				Name:  "BUILD_IMPORT_CREDENTIALS_PATH",
				Value: kBuildImportsPath,
			},
			{
				Name:  "BUILD_OCI_CREDENTIALS_PATH",
				Value: kBuildRegistriesPath,
			},
			{
				Name:  "BUILD_CACHE_URL",
				Value: fmt.Sprintf("%s.%s.svc.cluster.local", env.GetString("BUILD_CACHE_SVC", "sequencer-build-cache"), build.Namespace),
			},
		},
		VolumeMounts: []core.VolumeMount{
			{
				Name:      kBuildkitSocketName,
				MountPath: kBuildkitSocketPath,
			}, {
				Name:      kBuildkitTLSName,
				MountPath: kBuildkitTLSPath,
				ReadOnly:  true,
			},
		},
	}
}

func BackendContainerFor(build *sequencer.Build) core.Container {
	privileged := true
	return core.Container{
		Name:      "buildkitd",
		Image:     fmt.Sprintf("moby/buildkit:%s", env.GetString("BUILDKIT_VERSION", "v0.12.5")),
		Resources: resourcesForBuild(build),
		SecurityContext: &core.SecurityContext{
			Privileged: &privileged,
		},
		Env: []core.EnvVar{},
		LivenessProbe: &core.Probe{
			ProbeHandler: core.ProbeHandler{
				Exec: &core.ExecAction{
					Command: []string{
						"buildctl",
						"debug",
						"workers",
					},
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       30,
		},
		VolumeMounts: []core.VolumeMount{
			{
				Name:      kBuildkitSocketName,
				MountPath: kBuildkitSocketPath,
			}, {
				Name:      kBuildkitTLSName,
				MountPath: kBuildkitTLSPath,
			}, {
				Name:      kBuildkitConfigName,
				MountPath: kBuildkitConfigPath,
			},
		},
	}
}

func resourcesForBuild(build *sequencer.Build) core.ResourceRequirements {
	if build.Spec.Runtime.Resources != nil {
		return *build.Spec.Runtime.Resources
	}

	return *builds.BuildDefaultResourceRequirements
}
