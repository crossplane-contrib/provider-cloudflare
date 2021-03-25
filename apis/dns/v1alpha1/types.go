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

// DNSRecordParameters are the configurable fields of a DNS Record.
type DNSRecordParameters struct {
	// Type is the type of DNS record.
	// +kubebuilder:validation:Enum=A;AAAA;CAA;CNAME;TXT;SRV;LOC;MX;NS;SPF;CERT;DNSKEY;DS;NAPTR;SMIMEA;SSHFP;TLSA;URI
	// +kubebuilder:default=A
	// +immutable
	// +optional
	Type *string `json:"type,omitempty"`

	// Name of the DNS record.
	// +kubebuilder:validation:MaxLength=255
	Name string `json:"name"`

	// Content of the DNS Record
	Content string `json:"content"`

	// TTL of the DNS record
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// +optional
	TTL *int `json:"ttl,omitempty"`

	// Proxied enables or disables proxying traffic via Cloudflare
	// +optional
	Proxied *bool `json:"proxied,omitempty"`

	// Priority of a record
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Priority *uint16 `json:"priority,omitempty"`

	// ZoneID this DNS record is for.
	// +immutable
	// +optional
	Zone *string `json:"zone,omitempty"`

	// ZoneRef references the zone object this DNS Record is for.
	// +immutable
	// +optional
	ZoneRef *xpv1.Reference `json:"zoneRef,omitempty"`

	// ZoneSelector selects the zone object this DNS Record is for.
	// +immutable
	// +optional
	ZoneSelector *xpv1.Selector `json:"zoneSelector,omitempty"`
}

// DNSRecordObservation are the observable fields of a DNSRecord.
type DNSRecordObservation struct {
	Proxiable  bool         `json:"proxiable,omitempty"`
	Zone       string       `json:"zone,omitempty"`
	Locked     bool         `json:"locked,omitempty"`
	CreatedOn  *metav1.Time `json:"createdOn,omitempty"`
	ModifiedOn *metav1.Time `json:"modifiedOn,omitempty"`
}

// A DNSRecordSpec defines the desired state of a DNSRecord.
type DNSRecordSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       DNSRecordParameters `json:"forProvider"`
}

// A DNSRecordStatus represents the observed state of a DNSRecord.
type DNSRecordStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DNSRecordObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A DNSRecord is a set of DNS Records.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.bindingPhase"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="CLASS",type="string",JSONPath=".spec.classRef.name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,cloudflare}
type DNSRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSRecordSpec   `json:"spec"`
	Status DNSRecordStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DNSRecordList contains a list of DNSRecord
type DNSRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSRecord `json:"items"`
}

// ResolveReferences of this DNS Record
func (dr *DNSRecord) ResolveReferences(ctx context.Context, c client.Reader) error {
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
