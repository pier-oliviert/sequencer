package tunneling

import (
	"context"
	"errors"

	"github.com/pier-oliviert/sequencer/api/v1alpha1/workspaces"
	"github.com/pier-oliviert/sequencer/internal/integrations"
)

func NewProvider(ctx context.Context, controller integrations.ProviderController) (integrations.Provider, error) {
	spec := controller.Workspace().Spec.Networking.Tunnel
	if spec.Cloudflare != nil {
		return newCloudflareProvider(ctx, controller)
	}

	return nil, errors.New("E#3012: The tunnel spec doesn't include a valid provider")
}

func IncludesTunnelSpec(spec workspaces.NetworkingSpec) bool {
	if spec.Tunnel == nil {
		return false
	}

	if spec.Tunnel.Cloudflare != nil {
		return true
	}

	return false
}
