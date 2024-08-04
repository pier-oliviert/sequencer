package dnsrecords

import "github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"

// Defines the observed state of a DNSRecord
// +kubebuilder:object:generate=true
type Status struct {
	// Conditions representing the current state of the DNSRecord
	Conditions []conditions.Condition `json:"conditions,omitempty"`

	// Name of the provider that was used to create this record.
	Provider string `json:"provider,omitempty"`
	// RemoteID is the ID, if available for the record that was created
	RemoteID *string `json:"remoteID,omitempty"`
}
