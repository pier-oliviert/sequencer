package workspaces

import (
	"context"

	"k8s.io/client-go/tools/record"
	sequencer "se.quencer.io/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RulesReconciler struct {
	client.Client
	record.EventRecorder
}

func (r *RulesReconciler) Reconcile(ctx context.Context, workspace *sequencer.Workspace) (*ctrl.Result, error) {
	return nil, nil
}
