/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/dnsrecords"
	"github.com/pier-oliviert/sequencer/pkg/providers"
	core "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const kDNSRecordFinalizer = "dns.sequencer.io/provider-finalizer"

// DNSRecordReconciler reconciles a DNSRecord object
type DNSRecordReconciler struct {
	DefaultProvider providers.Provider

	record.EventRecorder
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=se.quencer.io,resources=dnsrecords,verbs=get;list;watch;
// +kubebuilder:rbac:groups=se.quencer.io,resources=dnsrecords/status,verbs=get;patch
// +kubebuilder:rbac:groups=se.quencer.io,resources=dnsrecords/finalizers,verbs=update
//
// Operate on the DNSRecord custom resource that are present on the system. The reconciler isn't in charge
// of creating the DNSRecord, it only monitors them and make sure the providers are in sync with what the system has.
func (r *DNSRecordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var record sequencer.DNSRecord
	if err := r.Get(ctx, req.NamespacedName, &record); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("E#5001: Couldn't retrieve the DNSRecord (%s) -- %w", req.NamespacedName, err)
	}

	if record.CurrentPhase() == dnsrecords.PhaseInitializing {
		err := r.createRecord(ctx, &record)
		if err != nil {
			r.Event(&record, core.EventTypeWarning, string(dnsrecords.ProviderCondition), err.Error())
		}

		return ctrl.Result{}, err
	}

	if record.CurrentPhase() == dnsrecords.PhaseTerminating {
		err := r.deleteRecord(ctx, &record)
		if err != nil {
			r.Event(&record, core.EventTypeWarning, string(dnsrecords.ProviderCondition), err.Error())
		}

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DNSRecordReconciler) createRecord(ctx context.Context, record *sequencer.DNSRecord) error {
	conditions.SetCondition(&record.Status.Conditions, conditions.Condition{
		Type:   dnsrecords.ProviderCondition,
		Status: conditions.ConditionLocked,
		Reason: "Resource locked",
	})

	if err := r.Client.Status().Patch(ctx, record, client.Merge); err != nil {
		return err
	}

	if controllerutil.AddFinalizer(record, kDNSRecordFinalizer) {
		if err := r.Update(ctx, record); err != nil {
			return err
		}
	}

	if err := r.DefaultProvider.Create(ctx, record); err != nil {
		conditions.SetCondition(&record.Status.Conditions, conditions.Condition{
			Type:   dnsrecords.ProviderCondition,
			Status: conditions.ConditionError,
			Reason: err.Error(),
		})

		if updateErr := r.Client.Status().Patch(ctx, record, client.Merge); updateErr != nil {
			return fmt.Errorf("%w -- %w", updateErr, err)
		}
		return err
	}

	conditions.SetCondition(&record.Status.Conditions, conditions.Condition{
		Type:   dnsrecords.ProviderCondition,
		Status: conditions.ConditionCreated,
		Reason: "Provider created the DNS record",
	})

	return r.Client.Status().Patch(ctx, record, client.Merge)
}

func (r *DNSRecordReconciler) deleteRecord(ctx context.Context, record *sequencer.DNSRecord) error {
	condition := conditions.FindCondition(record.Status.Conditions, dnsrecords.ProviderCondition)
	if condition.Status == conditions.ConditionTerminated && controllerutil.ContainsFinalizer(record, kDNSRecordFinalizer) {
		controllerutil.RemoveFinalizer(record, kDNSRecordFinalizer)
		return r.Update(ctx, record)
	}

	conditions.SetCondition(&record.Status.Conditions, conditions.Condition{
		Type:   dnsrecords.ProviderCondition,
		Status: conditions.ConditionLocked,
		Reason: "Resource locked",
	})

	if err := r.Client.Status().Patch(ctx, record, client.Merge); err != nil {
		return err
	}

	if err := r.DefaultProvider.Delete(ctx, record); err != nil {
		conditions.SetCondition(&record.Status.Conditions, conditions.Condition{
			Type:   dnsrecords.ProviderCondition,
			Status: conditions.ConditionError,
			Reason: err.Error(),
		})

		if updateErr := r.Status().Patch(ctx, record, client.Merge); updateErr != nil {
			return fmt.Errorf("%w -- %w", updateErr, err)
		}
		return err
	}

	conditions.SetCondition(&record.Status.Conditions, conditions.Condition{
		Type:   dnsrecords.ProviderCondition,
		Status: conditions.ConditionTerminated,
		Reason: "Provider deleted the DNS record",
	})

	return r.Status().Patch(ctx, record, client.Merge)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DNSRecordReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sequencer.DNSRecord{}).
		Complete(r)
}
