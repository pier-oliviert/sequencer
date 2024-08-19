package v1alpha1

import (
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	dnsrecords "github.com/pier-oliviert/sequencer/api/v1alpha1/dnsrecords"
)

// Represent a single DNS Record
// The record will be created and metadata about the record will
// be stored in the Status.
type DNSRecordSpec struct {
	Zone       string `json:"zone"`
	RecordType string `json:"recordType"`
	Name       string `json:"name"`
	Target     string `json:"target"`

	// Provider specific configuration settings that can be used
	// to configure a DNS Record in accordance to the provider used.
	// Each provider provides its own set of custom fields.
	Properties map[string]string `json:"properties,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DNSRecord is the Schema for the dnsrecords API
type DNSRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSRecordSpec     `json:"spec,omitempty"`
	Status dnsrecords.Status `json:"status,omitempty"`
}

func (r DNSRecord) IsInitializing() bool {
	return len(r.Status.Conditions) == 0
}

func (r DNSRecord) IsTerminating() bool {
	return !r.DeletionTimestamp.IsZero()
}

func (r DNSRecord) IsErrored() bool {
	return conditions.IsAnyConditionWithStatus(r.Status.Conditions, conditions.ConditionError)
}

// Finds the first condition that has an Error status and return
// the Reason as an error.
func (r DNSRecord) ConditionError() error {
	condition := conditions.FindStatusCondition(r.Status.Conditions, conditions.ConditionError)
	if condition != nil {
		return errors.New(condition.Reason)
	}

	return nil
}

// +kubebuilder:object:root=true

// DNSRecordList contains a list of DNSRecord
type DNSRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSRecord `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DNSRecord{}, &DNSRecordList{})
}
