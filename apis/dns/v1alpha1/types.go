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
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/benagricola/provider-cloudflare/apis/zone/v1alpha1"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reference"
	"github.com/pkg/errors"
)

// RecordParameters are the configurable fields of a DNS Record.
type RecordParameters struct {
	// Type is the type of DNS Record.
	// +kubebuilder:validation:Enum=A;AAAA;CAA;CNAME;TXT;SRV;LOC;MX;NS;SPF;CERT;DNSKEY;DS;NAPTR;SMIMEA;SSHFP;TLSA;URI
	// +kubebuilder:default=A
	// +immutable
	// +optional
	Type *string `json:"type,omitempty"`

	// Name of the DNS Record.
	// +kubebuilder:validation:MaxLength=255
	Name string `json:"name"`

	// Content of the DNS Record
	Content string `json:"content"`

	// TTL of the DNS Record.
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// +optional
	TTL *int `json:"ttl,omitempty"`

	// Proxied enables or disables proxying traffic via Cloudflare.
	// +optional
	Proxied *bool `json:"proxied,omitempty"`

	// Priority of a record.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Priority *uint16 `json:"priority,omitempty"`

	// ZoneID this DNS Record is managed on.
	// +immutable
	// +optional
	Zone *string `json:"zone,omitempty"`

	// ZoneRef references the Zone object this DNS Record is managed on.
	// +immutable
	// +optional
	ZoneRef *xpv1.Reference `json:"zoneRef,omitempty"`

	// ZoneSelector selects the Zone object this DNS Record is managed on.
	// +immutable
	// +optional
	ZoneSelector *xpv1.Selector `json:"zoneSelector,omitempty"`
}

// RecordObservation is the observable fields of a DNS Record.
type RecordObservation struct {
	// Proxiable indicates whether this record _can be_ proxied
	// via Cloudflare.
	Proxiable bool `json:"proxiable,omitempty"`

	// FQDN contains the full FQDN of the created record
	// (Record Name + Zone).
	FQDN string `json:"fqdn,omitempty"`

	// Zone contains the name of the Zone this record
	// is managed on.
	Zone string `json:"zone,omitempty"`

	// Locked indicates if this record is locked or not.
	Locked bool `json:"locked,omitempty"`

	// CreatedOn indicates when this record was created
	// on Cloudflare.
	CreatedOn *metav1.Time `json:"createdOn,omitempty"`

	// ModifiedOn indicates when this record was modified
	// on Cloudflare.
	ModifiedOn *metav1.Time `json:"modifiedOn,omitempty"`
}

// A RecordSpec defines the desired state of a DNS Record.
type RecordSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RecordParameters `json:"forProvider"`
}

// A RecordStatus represents the observed state of a DNS Record.
type RecordStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RecordObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Record represents a single DNS Record managed on a Zone.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="FQDN",type="string",JSONPath=".status.atProvider.fqdn"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,cloudflare}
type Record struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RecordSpec   `json:"spec"`
	Status RecordStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RecordList contains a list of DNS Record objects
type RecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Record `json:"items"`
}

// ResolveReferences resolves references to the Zone that this DNS Record
// is managed on.
func (dr *Record) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, dr)

	// Resolve spec.forProvider.zone
	rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(dr.Spec.ForProvider.Zone),
		Reference:    dr.Spec.ForProvider.ZoneRef,
		Selector:     dr.Spec.ForProvider.ZoneSelector,
		To:           reference.To{Managed: &v1alpha1.Zone{}, List: &v1alpha1.ZoneList{}},
		Extract:      reference.ExternalName(),
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.zone")
	}
	dr.Spec.ForProvider.Zone = reference.ToPtrValue(rsp.ResolvedValue)
	dr.Spec.ForProvider.ZoneRef = rsp.ResolvedReference

	return nil
}
