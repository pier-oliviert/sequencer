package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	builds "github.com/pier-oliviert/sequencer/api/v1alpha1/builds"
	"github.com/pier-oliviert/sequencer/internal/builder/buildkit"
	"github.com/pier-oliviert/sequencer/internal/builder/k8s"
	"github.com/pier-oliviert/sequencer/internal/builder/oci"
	"github.com/pier-oliviert/sequencer/internal/builder/secrets"
	"github.com/pier-oliviert/sequencer/internal/builder/source"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	ps "github.com/mitchellh/go-ps"
)

const kSrcPath = "/src/%s"

func main() {
	log.SetLogger(zap.New(zap.UseDevMode(true)))

	ctx := context.Background()
	logger := log.FromContext(ctx)
	err := sequencer.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}

	client, err := k8s.NewClient(ctx, &sequencer.GroupVersion)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	build := client.MustGetBuild(ctx, strings.Split(os.Getenv("BUILD_REFERENCE"), "/"))

	var tags []string
	for _, registry := range build.Spec.ContainerRegistries {
		tags = append(tags, registry.Tags...)
	}

	buildkitOpts := []buildkit.BuildOption{
		buildkit.WithContext(fmt.Sprintf(kSrcPath, build.Spec.Context)),
		buildkit.WithDockerfile(build.Spec.Dockerfile),
		buildkit.WithCacheTags(tags...),
	}

	if build.Spec.Target != nil {
		buildkitOpts = append(buildkitOpts, buildkit.WithTarget(*build.Spec.Target))
	}

	client.StageCondition(build, builds.BackendConfiguredCondition).Do(ctx, func(t k8s.Tracker) error {
		return buildkit.ConnectRemoteDriver(ctx)
	})

	client.StageCondition(build, builds.SecretsCondition).Do(ctx, func(t k8s.Tracker) error {
		if build.Spec.Secrets != nil {
			s, err := secrets.ReadKeyValueFromDir(ctx, os.Getenv("BUILD_SECRETS_PATH"))
			if err != nil {
				return err
			}

			buildkitOpts = append(buildkitOpts, buildkit.WithSecrets(s))
		}

		if build.Spec.Args != nil {
			arguments, err := secrets.ReadKeyValueFromDir(ctx, os.Getenv("BUILD_ARGUMENTS_PATH"))
			if err != nil {
				return err
			}

			buildkitOpts = append(buildkitOpts, buildkit.WithArguments(arguments))
		}

		return nil
	})

	client.StageCondition(build, builds.ImportDirectoriesCondition).Do(ctx, func(t k8s.Tracker) error {
		for _, content := range build.Spec.ImportContent {
			git := content.ContentFrom.Git

			opts := []source.RepositoryOption{
				source.WithPath(fmt.Sprintf(kSrcPath, content.Path)),
				source.WithRef(git.Ref),
				source.WithURL(git.URL),
			}

			if content.Credentials != nil {
				opts = append(opts, source.WithAuth(os.Getenv("BUILD_IMPORT_CREDENTIALS_PATH"), content.Credentials))
			}

			_, err = source.Git(ctx, opts...)
			if err != nil {
				return fmt.Errorf("E#1012: Could not pull repository(URL: %s) -- %w", git.URL, err)
			}
		}

		return nil
	})

	var registries []*oci.Registry

	client.StageCondition(build, builds.ContainerRegistriesCondition).Do(ctx, func(t k8s.Tracker) error {
		keychain := oci.Keychain{}
		for _, containerRegistry := range build.Spec.ContainerRegistries {
			credentials, err := secrets.ReadCredentialsFromDir(os.Getenv("BUILD_OCI_CREDENTIALS_PATH"), &containerRegistry.Credentials)
			if err != nil {
				return err
			}

			logger.Info("Setting up credentials for the registries")
			keychain.AddCredential(containerRegistry.URL, credentials)

			logger.Info("Configuring container registry for upload")

			registry, err := oci.NewRegistry(
				containerRegistry.URL,
				oci.WithKeyChain(&keychain),
				oci.WithTags(containerRegistry.Tags),
			)
			if err != nil {
				return err
			}
			registries = append(registries, registry)
		}

		return nil
	})

	var imageIndex v1.ImageIndex
	client.StageCondition(build, builds.ImageCondition).Do(ctx, func(t k8s.Tracker) error {
		builder, err := buildkit.NewBuilder(buildkitOpts...)
		if err != nil {
			return err
		}

		imageIndex, err = builder.Execute(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	client.StageCondition(build, builds.UploadCondition).Do(ctx, func(t k8s.Tracker) error {
		// Eventually this should become a WaitGroup or something similar.

		for _, registry := range registries {
			image, err := registry.Upload(ctx, imageIndex)
			if err != nil {
				return err
			}
			build.Status.Images = append(build.Status.Images, image)
		}

		return err
	})

	// All done, let's tell buildkitd it can shut down now.
	// TODO: This is pretty rough, it should be possible to send a signal to buildkitd to safely shutdown through the socket.
	list, err := ps.Processes()
	if err != nil {
		logger.Info("Error listing processes for termination", "Error", err)
		return
	}

	for _, p := range list {
		if strings.HasPrefix(p.Executable(), "buildkitd") {
			process, err := os.FindProcess(p.Pid())
			if err != nil {
				logger.Error(err, "Error listing processes for termination")
				return
			}

			err = process.Signal(os.Kill)
			if err != nil {
				logger.Error(err, "Error listing processes for termination")
				return
			}
		}
	}
}
