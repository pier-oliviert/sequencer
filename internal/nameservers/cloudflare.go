package nameservers

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"se.quencer.io/api/v1alpha1/conditions"
	"se.quencer.io/api/v1alpha1/workspaces"
	"se.quencer.io/internal/integrations"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const kCloudflareRecordIDKey = "RecordID"
const kCloudflareRecordTypeKey = "RecordType"
const kCloudflareDNSFinalizer = "dns.se.quencer.io/cloudflare"

type cf struct {
	api    *cloudflare.API
	zoneID string

	integrations.ProviderController
}

func newCloudflareProvider(ctx context.Context, controller integrations.ProviderController) (integrations.Provider, error) {
	spec := controller.Workspace().Spec.Networking.Cloudflare
	secretKeyRef := spec.SecretKeyRef
	var secret core.Secret
	namespacedName := types.NamespacedName{
		Name: spec.SecretKeyRef.Name,
	}

	if secretKeyRef.Namespace != nil {
		namespacedName.Namespace = *secretKeyRef.Namespace
	} else {
		namespacedName.Namespace = controller.Namespace()
	}

	if err := controller.Get(ctx, namespacedName, &secret); err != nil {
		return nil, err
	}

	token, ok := secret.Data[secretKeyRef.Key]
	if !ok {
		return nil, fmt.Errorf("E#3004: secret %s doesn't have a value at key %s", secret.Name, secretKeyRef.Key)
	}

	api, err := cloudflare.NewWithAPIToken(string(token))
	if err != nil {
		return nil, fmt.Errorf("E#3007: Could not create new Cloudflare Client -- %w", err)
	}

	return &cf{
		api:                api,
		zoneID:             spec.DNS.ZoneID,
		ProviderController: controller,
	}, nil
}

func (c cf) Reconcile(ctx context.Context) (_ *ctrl.Result, err error) {
	status := c.Workspace().Status
	condition := c.Condition()

	if err := c.SetFinalizer(ctx, kCloudflareDNSFinalizer); err != nil {
		return nil, err
	}

	if status.Tunnel == nil {
		c.Event(core.EventTypeNormal, string(conditions.ConditionWaiting), "Waiting for the tunnel to be configured")
		if condition.Status != conditions.ConditionWaiting {
			return &ctrl.Result{}, c.UpdateCondition(ctx, conditions.ConditionWaiting, "Waiting for the tunnel to be configured")
		}
		return nil, nil
	}

	if condition.Status == conditions.ConditionInitialized || condition.Status == conditions.ConditionWaiting {
		return &ctrl.Result{}, c.Guard(ctx, "Creating DNS Record on Cloudflare", func() (conditions.ConditionStatus, string, error) {
			record, err := c.createRecordFromTunnel(ctx, status.Tunnel)
			if err != nil {
				return "", "", fmt.Errorf("E#3005: Could not create a Cloudflare DNS record -- %w", err)
			}

			status.DNS = &workspaces.DNS{
				Provider: "cloudflare",
				Hostname: record.Name,
				ProviderMeta: map[string]string{
					kCloudflareRecordIDKey:   record.ID,
					kCloudflareRecordTypeKey: record.Type,
				},
			}

			if err := c.Update(ctx, status); err != nil {
				return "", "", err
			}

			return conditions.ConditionCreated, "DNS Entry created for tunnel", nil
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
	if workspace.Status.DNS == nil || workspace.Status.DNS.ProviderMeta == nil {
		return nil, c.RemoveFinalizer(ctx, kCloudflareDNSFinalizer)
	}
	providerMeta := workspace.Status.DNS.ProviderMeta

	return nil, c.Guard(ctx, "Deleting DNS records from Cloudflare", func() (status conditions.ConditionStatus, reason string, err error) {
		err = c.api.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(c.zoneID), providerMeta[kCloudflareRecordIDKey])
		if err != nil {
			logger.Error(err, "E#3006: Could not delete the DNS record", "ID", providerMeta[kCloudflareRecordIDKey])
			c.Eventf(core.EventTypeWarning, string(c.Condition().Type), "E#3006: Could not delete the DNS record (ID: %s) -- %s", providerMeta[kCloudflareRecordIDKey], err.Error())
			return "", "", err
		}
		return conditions.ConditionTerminated, "DNS deleted on Cloudflare", c.RemoveFinalizer(ctx, kCloudflareDNSFinalizer)
	})
}

func (c cf) createRecordFromTunnel(ctx context.Context, tunnel *workspaces.Tunnel) (*cloudflare.DNSRecord, error) {
	workspaceName := c.Workspace().Name
	dnsParams := cloudflare.CreateDNSRecordParams{
		ZoneID:  c.zoneID,
		Type:    "CNAME",
		Name:    workspaceName,
		Content: tunnel.Hostname,
		Proxied: new(bool),
		Comment: fmt.Sprintf("Record managed by sequencer for %s", workspaceName),
	}
	*dnsParams.Proxied = true

	record, err := c.api.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(c.zoneID), dnsParams)
	if err != nil {
		return nil, err
	}

	return &record, nil
}
