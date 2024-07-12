package buildkit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	gcr "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"k8s.io/utils/env"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/pier-oliviert/sequencer/internal/builder/secrets"
)

var ImagePath = fmt.Sprintf("%s/%s", os.TempDir(), "image")
var MetadataPath = fmt.Sprintf("%s/%s", os.TempDir(), "metadata.json")

// Exposing this as a dependency injection for testing purposes.
var CommandExecutor = exec.CommandContext

type Builder struct {
	context    string
	dockerfile string
	cacheTags  []string
	target     *string

	arguments []secrets.KeyValue
	secrets   []secrets.KeyValue

	// store files that are used as a mount point. Buildkit support
	// some arguments to be stored in a file and be referenced as mount point
	// for run command. SSH Keys and Secrets can be passed that way so the images aren't
	// storing secrets which could be leaked.
	// Files are stored here because they are temporary files and need to be removed after the build is
	// completed
	files map[secrets.KeyValue]*os.File
}

// Create a new Build ready to be executed. The reference is the registry reference that was used to configure the registry.
// It will be used to store the cache layers over to distribution.
// See the different BuildOption below to learn about the different options.
// Returns an error if any of the option passed fails to configure the Builder.
func NewBuilder(opts ...BuildOption) (*Builder, error) {
	builder := &Builder{
		files: make(map[secrets.KeyValue]*os.File),
	}

	for _, opt := range opts {
		if err := opt(builder); err != nil {
			return nil, err
		}
	}

	return builder, nil
}

// Build the repository into an ImageIndex (OCI Standard)
// The context is set around the repository which means it needs to be
// present in the filesystem.
//
// The build execute buildkit as a system command directly and
// pipes both STDOUT and STDERR to their respective file descriptor.
//
// The error that returns from Build is any error that is returned from the buildkit
// process.
//
// The ImageIndex is generated from go-containerregistry and is a valid
// OCI ImageIndex that can be exported to any container registry.
//
// Metadata from the build is return a valid JSON as a byteslice.
func (b *Builder) Execute(ctx context.Context) (gcr.ImageIndex, error) {
	logger := log.FromContext(ctx)
	logger.Info("Starting a build from a Repo", "Path", b.context)

	cacheURL := env.GetString("BUILD_CACHE_URL", "sequencer-build-cache.sequencer-system.svc.cluster.local")
	cmd := CommandExecutor(ctx, "buildx", "build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Args = append(cmd.Args, "--file", fmt.Sprintf("%s/%s", b.context, b.dockerfile))
	cmd.Args = append(cmd.Args, "--output", fmt.Sprintf("type=oci,dest=%s,tar=false", ImagePath))

	if b.target != nil {
		cmd.Args = append(cmd.Args, "--target", *b.target)
	}
	// Set cache export settings to point to the distribution deployment
	// The extra options (image-manifest, oci-mediatypes) seems to be required based on an issue in
	// distribution(https://github.com/distribution/distribution/issues/3863#issuecomment-1519734071). Buildkit seems to have
	// an issue with how it packs the image manifest(https://github.com/moby/buildkit/pull/3724/files)
	for _, tag := range b.cacheTags {
		// Same rule as the previous cache entry (image-manifest, oci-mediatypes).
		cmd.Args = append(cmd.Args, "--cache-to", fmt.Sprintf("type=registry,mode=max,image-manifest=true,oci-mediatypes=true,ref=%s/%s", cacheURL, tag))
		cmd.Args = append(cmd.Args, "--cache-from", fmt.Sprintf("type=registry,ref=%s/%s", cacheURL, tag))
	}

	// Need to export the metadata locally as we'll store the metadata in the build's status when we're done.
	cmd.Args = append(cmd.Args, "--metadata-file", MetadataPath)

	for _, arg := range b.arguments {
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("%s=%s", arg.Key, arg.Value))
	}

	for _, secret := range b.secrets {
		file := b.files[secret]
		if file == nil {
			return nil, errors.New("expected a secret to store its content in a temporary file")
		}

		logger.Info("Using secret's temporary path", "Path", file.Name())
		cmd.Args = append(cmd.Args, "--secret", fmt.Sprintf("id=%s,src=%s", secret.Key, file.Name()))
	}

	// This is the context for buildx.
	cmd.Args = append(cmd.Args, b.context)

	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	imageIndex, err := layout.ImageIndexFromPath(ImagePath)

	// Let's clean up secret's file so those secrets aren't lingering around.
	// It's not a huge deal if they were since the pod will terminate and eventually the filesystem will
	// be torn down, but it's good to be precautious with secrets.
	if err == nil {
		for pair, file := range b.files {
			err := os.Remove(file.Name())
			if err != nil {
				// This is not a critical error, let's just log the error and move on.
				logger.Info("Couldn't remove file", "Name", pair.Key, "File", file.Name, "Error", err)
			}
		}
	}

	return imageIndex, err
}

// /////////////////////////////////////////////////////////////////////////
// Build Options
type BuildOption func(*Builder) error

// Pair of Key/Value that will be stored on disk and be passed to
// BuildKit so that the Dockerfile can mount those secrets.
// https://docs.docker.com/build/building/secrets/
func WithSecrets(secrets []secrets.KeyValue) BuildOption {
	return func(b *Builder) (err error) {
		b.secrets = secrets
		for _, s := range b.secrets {
			b.files[s], err = os.CreateTemp(os.TempDir(), fmt.Sprintf("%s-*", s.Key))
			if err != nil {
				return err
			}
		}

		return nil
	}
}

// Pair of Key/Value that will be passed as build arguments
// when building the image.
func WithArguments(arguments []secrets.KeyValue) BuildOption {
	return func(b *Builder) error {
		b.arguments = arguments

		return nil
	}
}

// List of tags to use for caching all layers with.
func WithCacheTags(tags ...string) BuildOption {
	return func(b *Builder) error {
		b.cacheTags = tags

		return nil
	}
}

// Specify the context
func WithContext(context string) BuildOption {
	return func(b *Builder) error {
		b.context = context

		return nil
	}
}

// Specify the dockerfile
func WithDockerfile(dockerfile string) BuildOption {
	return func(b *Builder) error {
		b.dockerfile = dockerfile

		return nil
	}
}

// Specify the target
func WithTarget(target string) BuildOption {
	return func(b *Builder) error {
		b.target = &target

		return nil
	}
}
