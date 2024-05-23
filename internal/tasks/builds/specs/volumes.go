package specs

import (
	"errors"

	core "k8s.io/api/core/v1"
	"k8s.io/utils/env"
	sequencer "se.quencer.io/api/v1alpha1"
	buildConfig "se.quencer.io/api/v1alpha1/builds/config"
)

const (
	kBuildkitSocketName = "buildkit-socket"
	kBuildkitSocketPath = "/run/buildkit"

	kBuildkitTLSName = "buildkit-tls"
	kBuildkitTLSPath = "/srv/certs"

	kBuildkitConfigName = "buildkit-config"
	kBuildkitConfigPath = "/etc/buildkit"

	kBuildSecretsName = "build-secrets"
	kBuildSecretsPath = "/var/build/secrets"

	kBuildArgumentsName = "build-arguments"
	kBuildArgumentsPath = "/var/build/arguments"

	kBuildRegistriesName = "build-registries"
	kBuildRegistriesPath = "/var/build/registries"

	kBuildImportsName = "build-imports"
	kBuildImportsPath = "/var/build/imports"
)

func BuildkitSharedVolumesVolumes() []core.Volume {
	return []core.Volume{
		{
			Name: kBuildkitSocketName,
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		},
		{
			Name: kBuildkitTLSName,
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName: env.GetString("DISTRIBUTION_SECRET_NAME", "distribution-cert"),
				},
			},
		},
		{
			Name: kBuildkitConfigName,
			VolumeSource: core.VolumeSource{
				ConfigMap: &core.ConfigMapVolumeSource{
					LocalObjectReference: core.LocalObjectReference{
						Name: env.GetString("BUILDKITD_CONFIG_NAME", "sequencer-buildkitd"),
					},
				},
			},
		},
	}
}

func VolumesForContainer(container *core.Container, build *sequencer.Build) ([]core.Volume, error) {
	var volumes []core.Volume

	if secrets := build.Spec.Secrets; secrets != nil {
		volume, volumeMount, err := generateVolumeMapping(kBuildSecretsName, kBuildSecretsPath, secrets)
		if err != nil {
			return nil, err
		}

		container.VolumeMounts = append(container.VolumeMounts, *volumeMount)
		volumes = append(volumes, *volume)
	}

	if args := build.Spec.Args; args != nil {
		volume, volumeMount, err := generateVolumeMapping(kBuildArgumentsName, kBuildArgumentsPath, args)
		if err != nil {
			return nil, err
		}

		container.VolumeMounts = append(container.VolumeMounts, *volumeMount)
		volumes = append(volumes, *volume)
	}

	return volumes, nil
}

func generateVolumeMapping(name, path string, secrets *buildConfig.DynamicValues) (*core.Volume, *core.VolumeMount, error) {
	mount := &core.VolumeMount{
		Name:      name,
		MountPath: path,
		ReadOnly:  true,
	}

	volume := &core.Volume{
		Name: name,
	}

	switch {
	case secrets.ValuesFrom.SecretRef != nil:
		volume.VolumeSource = core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				SecretName: secrets.ValuesFrom.SecretRef.Name,
			},
		}

		for _, item := range secrets.Items {
			volume.VolumeSource.Secret.Items = append(volume.VolumeSource.Secret.Items, core.KeyToPath{
				Key:  item.Key,
				Path: *item.Path,
			})
		}

	case secrets.ValuesFrom.ConfigMapRef != nil:
		volume.VolumeSource = core.VolumeSource{
			ConfigMap: &core.ConfigMapVolumeSource{
				LocalObjectReference: core.LocalObjectReference{
					Name: secrets.ValuesFrom.ConfigMapRef.Name,
				},
			},
		}

		for _, item := range secrets.Items {
			volume.VolumeSource.ConfigMap.Items = append(volume.VolumeSource.ConfigMap.Items, core.KeyToPath{
				Key:  item.Key,
				Path: *item.Path,
			})
		}

	default:
		return nil, nil, errors.New("E#1002: requires one value to be set as `ValuesFrom`, none was set")
	}

	return volume, mount, nil
}
