package workspaces

import (
	"context"

	core "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	sequencer "se.quencer.io/api/v1alpha1"
	"se.quencer.io/api/v1alpha1/conditions"
	"se.quencer.io/api/v1alpha1/workspaces"
	"se.quencer.io/internal/integrations"
	"se.quencer.io/internal/nameservers"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DNSReconciler struct {
	client.Client
	record.EventRecorder
}

func (r *DNSReconciler) Reconcile(ctx context.Context, workspace *sequencer.Workspace) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)
	spec := workspace.Spec.Networking
	condition := conditions.FindStatusCondition(workspace.Status.Conditions, workspaces.DNSCondition)
	if condition == nil {
		// If the spec doesn't include a DNS spec, let's skip this task.
		if !nameservers.IncludesDNSSpec(spec) {
			return nil, nil
		}

		logger.Info("Workspace requires a dns entry. Initializing condition")
		r.Event(workspace, core.EventTypeNormal, "Conditions", "Initializing DNS for workspace")

		conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
			Type:   workspaces.DNSCondition,
			Status: conditions.ConditionInitialized,
			Reason: "Workspace requires DNS",
		})

		return &ctrl.Result{}, r.Status().Update(ctx, workspace)
	}

	// Find the tunnel provider
	provider, err := nameservers.NewProvider(ctx, integrations.NewController(workspace, *condition, r))
	if err != nil {
		return nil, r.reconciliationError(ctx, workspace, err)
	}

	if workspace.Status.Phase == workspaces.PhaseTerminating {
		// Workspace marked for deletion.
		return provider.Terminate(ctx)
	}

	// Handing off reconciliation of this condition to the tunnel provider
	return provider.Reconcile(ctx)
}

func (r *DNSReconciler) reconciliationError(ctx context.Context, workspace *sequencer.Workspace, err error) error {
	conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
		Type:   workspaces.TunnelingCondition,
		Status: conditions.ConditionError,
		Reason: err.Error(),
	})

	if updateErr := r.Client.Status().Update(ctx, workspace); updateErr != nil {
		return updateErr
	}

	return err
}
