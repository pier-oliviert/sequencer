package workspaces

import (
	"context"
	"time"

	core "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	sequencer "se.quencer.io/api/v1alpha1"
	"se.quencer.io/api/v1alpha1/conditions"
	"se.quencer.io/api/v1alpha1/workspaces"
	"se.quencer.io/internal/integrations"
	"se.quencer.io/internal/tunneling"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TunnelingReconciler struct {
	client.Client
	record.EventRecorder
}

func (r *TunnelingReconciler) Reconcile(ctx context.Context, workspace *sequencer.Workspace) (*ctrl.Result, error) {
	spec := workspace.Spec.Networking
	condition := conditions.FindStatusCondition(workspace.Status.Conditions, workspaces.TunnelingCondition)
	if condition == nil {
		// If the spec doesn't include a Tunnel spec, let's skip this task.
		if !tunneling.IncludesTunnelSpec(spec) {
			return nil, nil
		}

		r.Event(workspace, core.EventTypeNormal, "Conditions", "Initializing a tunnel for workspace")

		conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
			Type:   workspaces.TunnelingCondition,
			Status: conditions.ConditionInitialized,
			Reason: "Workspace requires a tunnel",
		})

		return &ctrl.Result{}, r.Client.Status().Update(ctx, workspace)
	}

	if condition.Status == conditions.ConditionLocked {
		return &ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}
	// Find the tunnel provider
	provider, err := tunneling.NewProvider(ctx, integrations.NewController(workspace, *condition, r))
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

func (r *TunnelingReconciler) reconciliationError(ctx context.Context, workspace *sequencer.Workspace, err error) error {
	conditions.SetStatusCondition(&workspace.Status.Conditions, conditions.Condition{
		Type:   workspaces.TunnelingCondition,
		Status: conditions.ConditionError,
		Reason: err.Error(),
	})

	if updateErr := r.Status().Update(ctx, workspace); updateErr != nil {
		return updateErr
	}

	return err
}
