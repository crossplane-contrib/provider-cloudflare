/*
Copyright 2021 The Crossplane Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// ZoneParameters are the configurable fields of a Zone.
type ZoneParameters struct {
	// AccountID is the account ID under which this Zone will be
	// created.
	// +immutable
	// +optional
	AccountID *string `json:"accountId,omitempty"`

	// JumpStart enables attempting to import existing DNS records
	// when a new Zone is created
	// +immutable
	// +optional
	JumpStart *bool `json:"jumpStart,omitempty"`

	// Paused indicates if the zone is only using Cloudflare DNS services.
	// +optional
	Paused *bool `json:"paused,omitempty"`

	// PlanID indicates the plan that this Zone will be subscribed
	// to.
	// +optional
	PlanID *string `json:"planId,omitempty"`

	// Type indicates the type of this zone - partial (partner-hosted
	// or CNAME only) or full.
	// +kubebuilder:validation:Enum=full;partial
	// +kubebuilder:default=full
	// +immutable
	// +optional
	Type *string `json:"type,omitempty"`

	// VanityNameServers lists an array of domains to use for custom
	// nameservers.
	// +optional
	VanityNameServers []string `json:"vanityNameServers,omitempty"`
}

// ZoneObservation are the observable fields of a Zone.
type ZoneObservation struct {
	AccountID         string   `json:"accountId,omitempty"`
	AccountName       string   `json:"accountName,omitempty"`
	ID                string   `json:"id,omitempty"`
	DevMode           int      `json:"developmentMode,omitempty"`
	OriginalNS        []string `json:"originalNameServers,omitempty"`
	OriginalRegistrar string   `json:"originalRegistrar,omitempty"`
	OriginalDNSHost   string   `json:"originalDNSHost,omitempty"`
	NameServers       []string `json:"nameServers,omitempty"`
	Paused            bool     `json:"paused,omitempty"`
	Permissions       []string `json:"permissions,omitempty"`
	PlanID            string   `json:"planId,omitempty"`
	Plan              string   `json:"plan,omitempty"`
	PlanPending       string   `json:"planPending,omitempty"`
	PlanPendingID     string   `json:"planPendingId,omitempty"`
	Status            string   `json:"status,omitempty"`
	Betas             []string `json:"betas,omitempty"`
	DeactReason       string   `json:"deactivationReason,omitempty"`
	VerificationKey   string   `json:"verificationKey,omitempty"`
	VanityNameServers []string `json:"vanityNameServers,omitempty"`
}

// A ZoneSpec defines the desired state of a Zone.
type ZoneSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ZoneParameters `json:"forProvider"`
}

// A ZoneStatus represents the observed state of a Zone.
type ZoneStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ZoneObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Zone is a set of common settings applied to one or more domains.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.bindingPhase"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="CLASS",type="string",JSONPath=".spec.classRef.name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,cloudflare}
type Zone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZoneSpec   `json:"spec"`
	Status ZoneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ZoneList contains a list of Zone
type ZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Zone `json:"items"`
}
