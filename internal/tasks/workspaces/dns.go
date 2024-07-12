package workspaces

import (
	"context"
	"fmt"

	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/workspaces"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/external-dns/endpoint"
)

type DNSReconciler struct {
	client.Client
	record.EventRecorder
}

func (r *DNSReconciler) Reconcile(ctx context.Context, workspace *sequencer.Workspace) (*ctrl.Result, error) {
	if workspace.Status.DNS == nil {
		workspace.Status.DNS = &workspaces.DNS{
			Hostname: fmt.Sprintf("%s.%s", workspace.Name, workspace.Spec.Networking.DNS.Zone),
		}

		conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
			Type:   workspaces.DNSCondition,
			Status: conditions.ConditionInitialized,
			Reason: "DNS Initialized",
		})
		return &ctrl.Result{}, r.Status().Update(ctx, workspace)
	}

	if len(workspace.Status.DNS.Records) == 0 {
		return nil, nil
	}

	conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
		Type:   workspaces.DNSCondition,
		Status: conditions.ConditionLocked,
		Reason: "Locked to create resources",
	})

	createdRecords := false

	// For each records present, the reconciler will try to retrieve a DNSEndpoint
	// if it doesn't exist, it will create a new one.
	for _, record := range workspace.Status.DNS.Records {
		exists, err := r.DNSEndpointExists(ctx, workspace, &record)
		if err != nil {
			return nil, err
		}

		// Not updating records as of now. If it exists, it means it's configured
		if exists {
			continue
		}

		dnsEndpoint := &endpoint.DNSEndpoint{
			ObjectMeta: meta.ObjectMeta{
				Labels: map[string]string{
					workspaces.DNSLabel:      record.Name,
					workspaces.InstanceLabel: workspace.Name,
				},
				OwnerReferences: []meta.OwnerReference{{
					APIVersion: workspace.APIVersion,
					Name:       workspace.Name,
					Kind:       workspace.Kind,
					UID:        workspace.UID,
				}},
				GenerateName: fmt.Sprintf("%s-", workspace.Name),
				Namespace:    workspace.Namespace,
			},
			Spec: endpoint.DNSEndpointSpec{
				Endpoints: []*endpoint.Endpoint{{
					DNSName:    record.Name,
					RecordType: record.Type,
					Targets:    []string{record.Target},
				}},
			},
		}

		for key, value := range record.Properties {
			ep := dnsEndpoint.Spec.Endpoints[0]
			ep.ProviderSpecific = append(ep.ProviderSpecific, endpoint.ProviderSpecificProperty{
				Name:  key,
				Value: value,
			})
		}

		if err := r.Create(ctx, dnsEndpoint); err != nil {
			return nil, fmt.Errorf("E#3003: Could not create a DNS Endpoint for external-dns: %w", err)
		}

		createdRecords = true
	}

	conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
		Type:   workspaces.DNSCondition,
		Status: conditions.ConditionCreated,
		Reason: "All DNS Records were created",
	})

	if createdRecords {
		return &ctrl.Result{}, r.Status().Update(ctx, workspace)
	}

	return nil, r.Status().Update(ctx, workspace)
}

func (r *DNSReconciler) DNSEndpointExists(ctx context.Context, workspace *sequencer.Workspace, record *workspaces.DNSRecord) (bool, error) {
	var list endpoint.DNSEndpointList
	selector, err := labels.Parse(fmt.Sprintf("%s=%s,%s=%s", workspaces.InstanceLabel, workspace.Name, workspaces.DNSLabel, record.Name))
	if err != nil {
		return false, fmt.Errorf("E#3001: Failed to parsed the label -- %w", err)
	}

	err = r.List(ctx, &list, &client.ListOptions{
		Namespace:     workspace.Namespace,
		LabelSelector: selector,
	})

	if err != nil {
		return false, fmt.Errorf("E#5002: Could not retrieve the list of DNS Endpoints from external-dns -- %w", err)
	}

	return len(list.Items) > 0, nil
}

func (r *DNSReconciler) record(ctx context.Context, workspace *sequencer.Workspace) {
}
