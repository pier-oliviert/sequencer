package oci

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	gcr "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/builds"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RegistryOption func(*Registry) error

type Registry struct {
	keychain  *Keychain
	reference name.Reference
	transport *http.Transport
	tags      []string
}

func NewRegistry(url string, opts ...RegistryOption) (*Registry, error) {
	ref, err := name.ParseReference(url)
	if err != nil {
		return nil, err
	}

	// Copying http.Transport settings from remote.DefaultTransport
	registry := &Registry{
		reference: ref,
		transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   50,
			TLSClientConfig:       &tls.Config{},
		},
	}

	for _, opt := range opts {
		if err := opt(registry); err != nil {
			return nil, err
		}
	}

	return registry, nil
}

func WithKeyChain(keychain *Keychain) RegistryOption {
	return func(r *Registry) error {
		r.keychain = keychain
		return nil
	}
}

func WithRootCert(filePath string) RegistryOption {
	return func(r *Registry) error {
		r.transport.TLSClientConfig.RootCAs = x509.NewCertPool()
		if caCertPEM, err := os.ReadFile(filePath); err != nil {
			return err
		} else if ok := r.transport.TLSClientConfig.RootCAs.AppendCertsFromPEM(caCertPEM); !ok {
			return err
		}
		return nil
	}
}

func WithTags(tags []string) RegistryOption {
	return func(r *Registry) error {
		r.tags = tags
		return nil
	}
}

func (r *Registry) Reference() name.Reference {
	return r.reference
}

// Upload the given imageIndex to the registy at `url`
func (r *Registry) Upload(ctx context.Context, index gcr.ImageIndex) (*builds.Image, error) {
	logger := log.FromContext(ctx)

	logger.Info("Uploading the index", "reference", r.reference)

	options := []remote.Option{
		remote.WithTransport(r.transport),
	}

	if r.keychain != nil {
		options = append(options, remote.WithAuthFromKeychain(r.keychain))
	}

	if err := remote.WriteIndex(r.reference, index, options...); err != nil {
		return nil, err
	}

	for _, t := range r.tags {
		err := remote.Tag(r.reference.Context().Tag(t), index, options...)
		if err != nil {
			return nil, err
		}
	}

	manifest, err := index.IndexManifest()
	if err != nil {
		return nil, err
	}
	logger.Info("Index written", "Manifest", manifest)

	payload, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	return &builds.Image{
		URL:              r.reference.String(),
		IndexManifestStr: string(payload),
	}, nil
}
