package tunneling

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/components"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	tunneling "github.com/pier-oliviert/sequencer/api/v1alpha1/tunneling"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/utils"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/workspaces"
	"github.com/pier-oliviert/sequencer/internal/integrations"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const kCloudflareTunnelFinalizer = "tunnel.se.quencer.io/cloudflare"
const kCloudflareTunnelTokenEnvName = "CLOUDFLARE_TUNNEL_TOKEN"
const kCloudflareTunnelIDKey = "TunnelID"
const kCloudflareTunnelTokenKey = "TunnelToken"
const kCloudflareConnectorLabel = "tunneling.se.quencer.io/connector"
const kConnectorGracefulTermination = time.Second * 5

var cfTunnelDNSContentFmt = "%s.cfargotunnel.com"

type cf struct {
	api       *cloudflare.API
	accountID string

	integrations.ProviderController
}

func newCloudflareProvider(ctx context.Context, controller integrations.ProviderController) (integrations.Provider, error) {
	spec := controller.Workspace().Spec.Networking.Tunnel.Cloudflare

	var secret core.Secret
	namespacedName := types.NamespacedName{
		Name: spec.SecretKeyRef.Name,
	}

	if spec.SecretKeyRef.Namespace != nil {
		namespacedName.Namespace = *spec.SecretKeyRef.Namespace
	} else {
		namespacedName.Namespace = controller.Namespace()
	}

	if err := controller.Get(ctx, namespacedName, &secret); err != nil {
		return nil, err
	}

	token, ok := secret.Data[spec.SecretKeyRef.Key]
	if !ok {
		return nil, fmt.Errorf("E#3004: secret %s doesn't include a value at key %s", secret.Name, spec.SecretKeyRef.Key)
	}

	api, err := cloudflare.NewWithAPIToken(string(token))
	if err != nil {
		return nil, fmt.Errorf("E#3007: Could not create new Cloudflare Client -- %w", err)
	}

	return &cf{
		api:                api,
		accountID:          spec.AccountID,
		ProviderController: controller,
	}, nil
}

func (c cf) Reconcile(ctx context.Context) (_ *ctrl.Result, err error) {
	tunnelSpec := c.Workspace().Spec.Networking.Tunnel.Cloudflare
	condition := c.Condition()

	if err := c.SetFinalizer(ctx, kCloudflareTunnelFinalizer); err != nil {
		return nil, err
	}

	if condition.Status == conditions.ConditionInitialized {
		return &ctrl.Result{}, c.Guard(ctx, "Configuring Tunnel on Cloudflare", func() (conditions.ConditionStatus, string, error) {
			status := &c.Workspace().Status
			status.Host = fmt.Sprintf("%s.%s", c.Workspace().Name, c.Workspace().Spec.Networking.DNS.Zone)

			token, tunnel, err := c.createTunnel(ctx, c.Workspace())
			if err != nil {
				return "", "", err
			}

			status.Ingress = fmt.Sprintf(cfTunnelDNSContentFmt, tunnel.ID)
			status.DNS = append(status.DNS, workspaces.DNS{
				Name:       status.Host,
				RecordType: "CNAME",
				Target:     status.Ingress,
				Properties: map[string]string{
					"proxied": "true",
				},
			})

			status.Tunnel = &workspaces.Tunnel{
				RemoteID: tunnel.ID,
				ProviderMeta: map[string]string{
					kCloudflareConnectorLabel: "cloudflared",
					kCloudflareTunnelIDKey:    tunnel.ID,
					kCloudflareTunnelTokenKey: token,
				},
			}
			return conditions.ConditionCreated, "Tunnel created on Cloudflare", c.Update(ctx, *status)
		})
	}

	if condition.Status == conditions.ConditionCreated {
		var service *core.Service
		service, err = c.getServiceFor(ctx, c.Workspace(), tunnelSpec.Route)
		if err != nil {
			return nil, err
		}

		if service == nil {
			c.Eventf(core.EventTypeNormal, "Tunneling", "Waiting for (%s:%s) service", tunnelSpec.Route.ComponentName, tunnelSpec.Route.NetworkName)
			return nil, nil
		}

		return &ctrl.Result{}, c.Guard(ctx, "Configuring DNS with the tunnel", func() (conditions.ConditionStatus, string, error) {
			status := &c.Workspace().Status
			if err := c.attachTunnelToDNSRecord(ctx, c.Workspace(), service); err != nil {
				return "", "", fmt.Errorf("E#3008: Could not attach tunnel to the DNS record -- %w", err)
			}

			c.Eventf(core.EventTypeNormal, "Tunneling", "Tunnel now pointing to service (%s)", service.Name)
			pod, err := c.deployConnector(ctx, status.Tunnel.ProviderMeta[kCloudflareTunnelTokenKey])
			if err != nil {
				return "", "", err
			}

			c.Eventf(core.EventTypeNormal, "Tunneling", "Connector deployed (%s)", pod.Name)
			return conditions.ConditionCompleted, "Tunnel ready to use", nil
		})
	}

	return nil, nil
}

func (c cf) Terminate(ctx context.Context) (_ *ctrl.Result, err error) {
	if c.Condition().Status == conditions.ConditionTerminated {
		return nil, nil
	}

	logger := log.FromContext(ctx)
	workspace := c.Workspace()
	if workspace.Status.Tunnel == nil {
		return nil, c.RemoveFinalizer(ctx, kCloudflareTunnelFinalizer)
	}

	// Need to tear down the connector first as the Tunnel cannot be deleted with an active connection.
	var pods core.PodList
	selector, err := labels.Parse(fmt.Sprintf("%s=%s,%s=%s", workspaces.InstanceLabel, workspace.Name, kCloudflareConnectorLabel, workspace.Status.Tunnel.ProviderMeta[kCloudflareConnectorLabel]))
	if err != nil {
		return nil, err
	}

	err = c.List(ctx, &pods, &client.ListOptions{
		Namespace:     workspace.Namespace,
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	shouldRequeue := false
	for _, pod := range pods.Items {
		if pod.Status.Phase == core.PodRunning {
			if err := c.Delete(ctx, &pod); err != nil {
				return nil, err
			}
			shouldRequeue = true
		}

		if pod.Status.Phase != core.PodSucceeded {
			shouldRequeue = true
		}
	}

	if shouldRequeue {
		return &ctrl.Result{RequeueAfter: kConnectorGracefulTermination}, nil
	}

	return nil, c.Guard(ctx, "Deleting tunnel on Cloudflare", func() (status conditions.ConditionStatus, reason string, err error) {
		tunnelID := workspace.Status.Tunnel.RemoteID
		err = c.api.DeleteTunnel(ctx, cloudflare.AccountIdentifier(c.accountID), tunnelID)
		if err != nil {
			logger.Error(err, "E#3009: Could not delete the Tunnel", "ID", tunnelID)
			c.Eventf(core.EventTypeWarning, string(c.Condition().Type), "E#3009: Could not delete the Tunnel (ID: %s) -- %s", tunnelID, err.Error())
			return "", "", err
		}

		return conditions.ConditionTerminated, "Tunnel deleted on Cloudflare", c.RemoveFinalizer(ctx, kCloudflareTunnelFinalizer)
	})
}

func (c cf) serviceEndpoint(routeSpec *tunneling.CloudflareRouteSpec, service *core.Service) string {
	protocol := routeSpec.Protocol
	if protocol == "" {
		protocol = "http"
	}

	return fmt.Sprintf("%s://%s:%d", protocol, service.Name, service.Spec.Ports[0].Port)
}

func (cf *cf) getServiceFor(ctx context.Context, workspace *sequencer.Workspace, routeSpec tunneling.CloudflareRouteSpec) (*core.Service, error) {
	var services core.ServiceList
	var service *core.Service

	selector, err := labels.Parse(fmt.Sprintf("%s=%s", workspaces.InstanceLabel, workspace.Name))
	if err != nil {
		return nil, err
	}

	err = cf.List(ctx, &services, &client.ListOptions{
		Namespace:     workspace.Namespace,
		LabelSelector: selector,
	})

	if err != nil {
		cf.Event(core.EventTypeWarning, "Fetching Services", err.Error())
		return nil, err
	}

	for i := range services.Items {
		s := &services.Items[i]
		if s.Labels[components.NameLabel] == routeSpec.ComponentName && s.Labels[components.NetworkLabel] == routeSpec.NetworkName {
			service = s
			break
		}
	}

	return service, err
}

func (c cf) createTunnel(ctx context.Context, workspace *sequencer.Workspace) (token string, _ *cloudflare.Tunnel, err error) {
	tunnel, err := c.api.CreateTunnel(ctx, cloudflare.AccountIdentifier(c.accountID), cloudflare.TunnelCreateParams{
		Name:   workspace.Name,
		Secret: utils.RandomValue(16),
	})

	if err != nil {
		return "", nil, fmt.Errorf("E#3010: Tunnel creation error -- %w", err)
	}

	token, err = c.api.GetTunnelToken(ctx, cloudflare.AccountIdentifier(c.accountID), tunnel.ID)
	if err != nil {
		return "", nil, fmt.Errorf("E#3010: Tunnel created, token retrieval error -- %w", err)
	}

	return token, &tunnel, nil
}

func (c cf) attachTunnelToDNSRecord(ctx context.Context, workspace *sequencer.Workspace, service *core.Service) error {
	spec := workspace.Spec.Networking.Tunnel.Cloudflare
	tunnel := workspace.Status.Tunnel
	_, err := c.api.UpdateTunnelConfiguration(ctx, cloudflare.AccountIdentifier(c.accountID), cloudflare.TunnelConfigurationParams{
		TunnelID: tunnel.ProviderMeta[kCloudflareTunnelIDKey],
		Config: cloudflare.TunnelConfiguration{
			Ingress: []cloudflare.UnvalidatedIngressRule{
				{
					Path:     spec.Route.Path,
					Hostname: workspace.Status.Host,
					Service:  c.serviceEndpoint(&spec.Route, service),
				},
				{
					Service: "http_status:404",
				},
			},
		},
	})

	return err
}

func (c cf) deployConnector(ctx context.Context, token string) (*core.Pod, error) {
	pod := &core.Pod{
		ObjectMeta: meta.ObjectMeta{
			GenerateName: "tunnel-cloudflare-",
			Labels: map[string]string{
				kCloudflareConnectorLabel: "cloudflared",
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{{
				Name:  "cloudflared",
				Image: "cloudflare/cloudflared:latest",
				Args: []string{
					"tunnel",
					"--no-autoupdate",
					"run",
					"--token",
					fmt.Sprintf("$(%s)", kCloudflareTunnelTokenEnvName),
				},
				Env: []core.EnvVar{{
					Name:  kCloudflareTunnelTokenEnvName,
					Value: token,
				}},
			}},
		}}

	return pod, c.Create(ctx, pod)
}
