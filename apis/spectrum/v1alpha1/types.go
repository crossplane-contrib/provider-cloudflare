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

	"github.com/benagricola/provider-cloudflare/apis/zone/v1alpha1"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reference"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SpectrumApplicationDNS holds the external DNS configuration
// for a Spectrum Application.
type SpectrumApplicationDNS struct {
	// Type is the type of edge IP configuration specified
	// Only valid with CNAME DNS names
	// +kubebuilder:validation:Enum=CNAME;ADDRESS
	Type *string `json:"type"`

	// Name is the name of the DNS record associated with the application.
	// +kubebuilder:validation:Format=hostname
	Name *string `json:"name"`
}

// SpectrumApplicationOriginDNS holds the origin DNS configuration for a Spectrum
// Application.
type SpectrumApplicationOriginDNS struct {
	// Name is the name of the Origin DNS for the Spectrum Application
	// +kubebuilder:validation:Format=hostname
	Name *string `json:"name"`
}

// SpectrumApplicationOriginPort holds the origin ports for a Spectrum Application
type SpectrumApplicationOriginPort struct {
	// Port is a singular port for a Spectrum Application
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port *uint32 `json:"port,omitempty"`

	// Start is the start of a port range for a Spectrum Application
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Start *uint32 `json:"start,omitempty"`

	// End is the end of a port range for a Spectrum Application
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	End *uint32 `json:"end,omitempty"`
}

// SpectrumApplicationEdgeIPs holds the anycast edge IP configuration for the hostname of this application.
type SpectrumApplicationEdgeIPs struct {
	// Type is the type of edge IP configuration specified.
	// +kubebuilder:validation:Enum=dynamic;static
	// +optional
	Type *string `json:"type,omitempty"`

	// Connectivity is IP versions supported for inbound connections on Spectrum anycast IPs.
	// +kubebuilder:validation:Enum=all;ipv4;ipv6
	// +optional
	Connectivity *string `json:"connectivity,omitempty"`

	// IPs is a slice of customer owned IPs we broadcast via anycast for this hostname and application.
	// +optional
	IPs []string `json:"ips,omitempty"`
}

// ApplicationParameters are the configurable fields of a Spectrum Application.
type ApplicationParameters struct {
	// Protocol port configuration at Cloudflare’s edge.
	// +optional
	Protocol *string `json:"protocol,omitempty"`

	IPv4 *bool `json:"ipv4,omitempty"`

	// The name and type of DNS record for the Spectrum application.
	DNS SpectrumApplicationDNS `json:"dns,omitempty"`

	// OriginDirect is a list of destination addresses to the origin.
	// +optional
	OriginDirect []string `json:"originDirect,omitempty"`

	// OriginPort is the port range when using Origin DNS
	OriginPort *SpectrumApplicationOriginPort `json:"originPort,omitempty"`

	// OriginDNS is the DNS entry when using DNS Origins
	OriginDNS *SpectrumApplicationOriginDNS `json:"originDNS,omitempty"`

	// IPFirewall enables IP Access Rules for this application.
	// +optional
	IPFirewall *bool `json:"ipFirewall,omitempty"`

	// ProxyProtocol enables / sets the Proxy Protocol to the origin.
	// +kubebuilder:validation:Enum=off;v1;v2;simple
	// +optional
	ProxyProtocol *string `json:"proxyProtocol,omitempty"`

	// TLS is the type of TLS termination associated with the application.
	// +kubebuilder:validation:Enum=off;flexible;full;strict
	// +optional
	TLS *string `json:"tls,omitempty"`

	// TrafficType determines how data travels from the edge to the origin.
	// +kubebuilder:validation:Enum=direct;http;https
	// +optional
	TrafficType *string `json:"trafficType,omitempty"`

	// EdgeIPs is the anycast edge IP configuration for the hostname of this application.
	EdgeIPs *SpectrumApplicationEdgeIPs `json:"edgeIPs,omitempty"`

	// ArgoSmartRouting enables Argo Smart Routing for this application.
	// +optional
	ArgoSmartRouting *bool `json:"argoSmartRouting,omitempty"`

	// ZoneID this Spectrum Application is managed on.
	// +immutable
	// +optional
	Zone *string `json:"zone,omitempty"`

	// ZoneRef references the Zone object this Spectrum Application is managed on.
	// +optional
	ZoneRef *xpv1.Reference `json:"zoneRef,omitempty"`

	// ZoneSelector selects the Zone object this Spectrum Application is managed on.
	// +optional
	ZoneSelector *xpv1.Selector `json:"zoneSelector,omitempty"`
}

// ApplicationObservation are the observable fields of a Spectrum Application.
type ApplicationObservation struct {
	CreatedOn  *metav1.Time `json:"createdOn,omitempty"`
	ModifiedOn *metav1.Time `json:"modifiedOn,omitempty"`
}

// A ApplicationSpec defines the desired state of a Spectrum Application.
type ApplicationSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ApplicationParameters `json:"forProvider"`
}

// A ApplicationStatus represents the observed state of a Spectrum Application.
type ApplicationStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ApplicationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Application is a set of common settings applied to one or more domains.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,cloudflare}
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application objects.
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

// ResolveReferences resolves references to the Zone that this Spectrum Application
// is managed on.
func (dr *Application) ResolveReferences(ctx context.Context, c client.Reader) error {
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
