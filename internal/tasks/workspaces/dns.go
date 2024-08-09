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
)

type DNSReconciler struct {
	client.Client
	record.EventRecorder
}

func (r *DNSReconciler) Reconcile(ctx context.Context, workspace *sequencer.Workspace) (*ctrl.Result, error) {
	if conditions.IsStatusConditionPresentAndEqual(workspace.Status.Conditions, workspaces.DNSCondition, conditions.ConditionError) {
		// An error for this conditions is fatal. Exit this reconciliation.
		return nil, nil
	}

	if conditions.IsStatusConditionPresentAndEqual(workspace.Status.Conditions, workspaces.DNSCondition, conditions.ConditionCompleted) {
		// This workspace reconciliation loop is completed, but it's possible that the DNSRecord that were created here failed.
		// If any of the DNSRecord created has an error, mark this condition as errored by fetching the conditions' error on the DNS record
		// and copying it to this condition.
		selector, err := labels.Parse(fmt.Sprintf("%s=%s", workspaces.InstanceLabel, workspace.Name))
		if err != nil {
			return nil, fmt.Errorf("E#3001: failed to parse the label selector -- %w", err)
		}

		var list sequencer.DNSRecordList
		err = r.List(ctx, &list, &client.ListOptions{
			LabelSelector: selector,
			Namespace:     workspace.Namespace,
		})

		if err != nil {
			return nil, fmt.Errorf("E#5002: failed to retrieve the list of DNS Records -- %w", err)
		}

		for _, record := range list.Items {
			if err := record.ConditionError(); err != nil {
				conditions.SetCondition(&workspace.Status.Conditions, conditions.Condition{
					Type:   workspaces.DNSCondition,
					Status: conditions.ConditionError,
					Reason: err.Error(),
				})

				// It's possible that more than one DNS record has an error, but it would massively increase the
				// complexity of surfacing those errors back to the user through conditions. Instead, the reconciliation
				// process will only surface the first error it finds and push it to the global condition.
				return &ctrl.Result{}, r.Status().Patch(ctx, &record, client.Merge)
			}
		}

		// The condition is completed and all DNS Records entries are healthy as far as this reconciliation loop is concerned.
		return nil, nil
	}

	for _, expected := range workspace.Status.DNS {
		record := &sequencer.DNSRecord{
			ObjectMeta: meta.ObjectMeta{
				Labels: map[string]string{
					workspaces.DNSLabel:      expected.Name,
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
			Spec: sequencer.DNSRecordSpec{
				RecordType: expected.RecordType,
				Name:       expected.Name,
				Target:     expected.Target,
				Properties: expected.Properties,
				Zone:       workspace.Spec.Networking.DNS.Zone,
			},
		}

		if err := r.Create(ctx, record); err != nil {
			conditions.SetCondition(&workspace.Status.Conditions, conditions.Condition{
				Type:   workspaces.DNSCondition,
				Status: conditions.ConditionError,
				Reason: err.Error(),
			})

			return nil, fmt.Errorf("E#3003: Could not create a DNS Record: %w", err)
		}

	}

	conditions.SetCondition(&workspace.Status.Conditions, conditions.Condition{
		Type:   workspaces.DNSCondition,
		Status: conditions.ConditionCompleted,
		Reason: "DNS Hostname configured",
	})

	return &ctrl.Result{}, r.Status().Update(ctx, workspace)
}
