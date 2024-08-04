package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Represent a single DNS Record
// The record will be created and metadata about the record will
// be stored in the Status.
type DNSRecordSpec struct {
	RecordType string `json:"recordType,omitempty"`
	Name       string `json:"name,omitempty"`
	Target     string `json:"target,omitempty"`

	// Provider specific configuration settings that can be used
	// to configure a DNS Record in accordance to the provider used.
	// Each provider provides its own set of custom fields.
	Settings map[string]string `json:"settings,omitempty"`
}

// DNSRecordStatus defines the observed state of DNSRecord
type DNSRecordStatus struct {
	// Name of the provider that was used to create this record.
	Provider string `json:"provider,omitempty"`

	// RemoteID is the ID, if available for the record that was created
	RemoteID *string `json:"remoteID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DNSRecord is the Schema for the dnsrecords API
type DNSRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSRecordSpec   `json:"spec,omitempty"`
	Status DNSRecordStatus `json:"status,omitempty"`
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
