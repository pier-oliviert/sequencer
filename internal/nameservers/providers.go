package nameservers

import (
	"context"
	"errors"

	"se.quencer.io/api/v1alpha1/workspaces"
	"se.quencer.io/internal/integrations"
)

func NewProvider(ctx context.Context, controller integrations.ProviderController) (integrations.Provider, error) {
	spec := controller.Workspace().Spec.Networking
	if spec.Cloudflare != nil {
		return newCloudflareProvider(ctx, controller)
	}

	return nil, errors.New("E#3003: The DNS spec doesn't include a valid provider")
}

func IncludesDNSSpec(spec workspaces.NetworkingSpec) bool {
	if spec.Cloudflare != nil && spec.Cloudflare.DNS != nil {
		return true
	}

	return false
}
