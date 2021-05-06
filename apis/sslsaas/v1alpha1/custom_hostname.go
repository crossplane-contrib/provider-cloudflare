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

	dns "github.com/benagricola/provider-cloudflare/apis/dns/v1alpha1"
	zone "github.com/benagricola/provider-cloudflare/apis/zone/v1alpha1"
	"github.com/cloudflare/cloudflare-go"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reference"
	"github.com/pkg/errors"
)

// CustomHostnameSSLValidationErrors represents errors that occurred during SSL validation.
type CustomHostnameSSLValidationErrors struct {
	Message string `json:"message,omitempty"`
}

// CustomHostnameSSLSettings represents the SSL settings for a custom hostname.
type CustomHostnameSSLSettings struct {

	// Whether or not HTTP2 is enabled for the Custom Hostname
	// +kubebuilder:validation:Enum=on;off
	// +kubebuilder:default="on"
	HTTP2 *string `json:"http2,omitempty"`

	// Whether or not TLS 1.3 is enabled for the Custom Hostname
	// +kubebuilder:validation:Enum=on;off
	// +kubebuilder:default="on"
	TLS13 *string `json:"tls13,omitempty"`

	// The minimum TLS version supported for the Custom Hostname
	// +kubebuilder:validation:Enum="1.0";"1.1";"1.2";"1.3"
	// +kubebuilder:default="1.2"
	MinTLSVersion *string `json:"minTLSVersion,omitempty"`

	// An allowlist of ciphers for TLS termination. These ciphers must be in the BoringSSL format.
	Ciphers []string `json:"ciphers,omitempty"`

	// Fields not supported in the go library yet
	// HTTP3         *string  `json:"http3,omitempty"`
}

// CustomHostnameOwnershipVerification represents ownership verification status of a given custom hostname.
type CustomHostnameOwnershipVerification struct {
	Type  *string `json:"type,omitempty"`
	Name  *string `json:"name,omitempty"`
	Value *string `json:"value,omitempty"`
}

// CustomHostnameSSL represents the SSL section in a given custom hostname.
type CustomHostnameSSL struct {

	// Domain control validation (DCV) method used for this custom hostname.
	// +kubebuilder:validation:Enum=http;txt;email
	// +kubebuilder:default="http"
	Method *string `json:"method,omitempty"`

	// Level of validation to be used for this custom hostname. Domain validation (dv) must be used.
	// +kubebuilder:validation:Enum=dv
	// +kubebuilder:default="dv"
	Type *string `json:"type,omitempty"`

	// CustomHostnameSSLSettings represents the SSL settings for a custom hostname.
	Settings CustomHostnameSSLSettings `json:"settings,omitempty"`

	// Indicates whether the certificate for the custom hostname covers a wildcard.
	// +kubebuilder:default=false
	Wildcard *bool `json:"wildcard,omitempty"`

	// Custom Certificate used for this Custom Hostname
	// If provided then Cloudflare will not attempt to generate an ACME certificate
	// +kubebuilder:default=""
	CustomCertificate *string `json:"customCertificate,omitempty"`

	// Custom Certificate Key used for this Custom Hostname
	// If provided then Cloudflare will not attempt to generate an ACME certificate
	// +kubebuilder:default=""
	CustomKey *string `json:"customKey,omitempty"`

	// Following fields are in the API but not supported in go library yet
	// BundleMethod      *string                   `json:"bundle_method,omitempty"`

}

// CustomHostnameSSLObserved represents the Observed SSL section in a given custom hostname.
type CustomHostnameSSLObserved struct {
	Status               string                                         `json:"status"`
	HTTPUrl              string                                         `json:"httpURL"`
	HTTPBody             string                                         `json:"httpBody"`
	ValidationErrors     []cloudflare.CustomHostnameSSLValidationErrors `json:"validationErrors,omitempty"`
	CertificateAuthority string                                         `json:"certificateAuthority"`
	CnameName            string                                         `json:"cname"`
	CnameTarget          string                                         `json:"cnameTarget"`

	// Following fields are in the API but not supported in go library yet
	// TxtName          string                              `json:"txt_name,omitempty"`
	// TxtValue         string                              `json:"txt_value,omitempty"`
	// UplaodedOn metav1.Time `json:"uploaded_on,omitempty"`
	// ExpiresOn  metav1.Time `json:"expires_on,omitempty"`

	// Waiting on 0.15 to release
	// Issuer           string                              `json:"issuer,omitempty"`
	// SerialNumber     string                              `json:"serial_number,omitempty"`
	// Signature        string                              `json:"signature,omitempty"`

}

// CustomHostnameParameters represents the settings of a CustomHostname
type CustomHostnameParameters struct {

	// Hostname for the custom hostname.
	// +kubebuilder:validation:Format=hostname
	// +kubebuilder:validation:MaxLength=255
	// +immutable
	Hostname *string `json:"hostname,omitempty"`

	// SSL Settings for a Custom Hostname
	// +optional
	SSL CustomHostnameSSL `json:"ssl,omitempty"`

	// CustomOriginServer for a Custom Hostname
	// A valid hostname that’s been added to your DNS zone as an A, AAAA, or CNAME record.
	CustomOriginServer *string `json:"customOriginServer,omitempty"`

	// CustomOriginServerRef references the Record object that this Custom Hostname should point to.
	// +immutable
	// +optional
	CustomOriginServerRef *xpv1.Reference `json:"customOriginServerRef,omitempty"`

	// CustomOriginServerSelector selects the Record object that this Custom Hostname should point to.
	// +optional
	CustomOriginServerSelector *xpv1.Selector `json:"customOriginServerSelector,omitempty"`

	// ZoneID this custom hostname is for.
	// +immutable
	// +optional
	Zone *string `json:"zone,omitempty"`

	// ZoneRef references the zone object this custom hostname is for.
	// +immutable
	// +optional
	ZoneRef *xpv1.Reference `json:"zoneRef,omitempty"`

	// ZoneSelector selects the zone object this custom hostname is for.
	// +immutable
	// +optional
	ZoneSelector *xpv1.Selector `json:"zoneSelector,omitempty"`
}

// CustomHostnameObservation are the observable fields of a custom hostname.
type CustomHostnameObservation struct {
	Status             cloudflare.CustomHostnameStatus `json:"status"`
	VerificationErrors []string                        `json:"verification_errors,omitempty"`
	SSL                CustomHostnameSSLObserved       `json:"ssl"`
}

// A CustomHostnameSpec defines the desired state of a custom hostname.
type CustomHostnameSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       CustomHostnameParameters `json:"forProvider"`
}

// A CustomHostnameStatus represents the observed state of a custom hostname.
type CustomHostnameStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          CustomHostnameObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A CustomHostname is a custom hostname required to use SSL for SaaS.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,cloudflare}
type CustomHostname struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CustomHostnameSpec   `json:"spec"`
	Status CustomHostnameStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CustomHostnameList contains a list of CustomHostname
type CustomHostnameList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomHostname `json:"items"`
}

// ResolveReferences of this Custom Hostname
func (dr *CustomHostname) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, dr)

	// Resolve spec.forProvider.customOriginServer to FQDN from DNS Record
	rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(dr.Spec.ForProvider.CustomOriginServer),
		Reference:    dr.Spec.ForProvider.CustomOriginServerRef,
		Selector:     dr.Spec.ForProvider.CustomOriginServerSelector,
		To:           reference.To{Managed: &dns.Record{}, List: &dns.RecordList{}},
		Extract:      dns.RecordFQDN(),
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.customOriginServer")
	}
	dr.Spec.ForProvider.CustomOriginServer = reference.ToPtrValue(rsp.ResolvedValue)
	dr.Spec.ForProvider.CustomOriginServerRef = rsp.ResolvedReference

	// Resolve spec.forProvider.zone
	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(dr.Spec.ForProvider.Zone),
		Reference:    dr.Spec.ForProvider.ZoneRef,
		Selector:     dr.Spec.ForProvider.ZoneSelector,
		To:           reference.To{Managed: &zone.Zone{}, List: &zone.ZoneList{}},
		Extract:      reference.ExternalName(),
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.zone")
	}
	dr.Spec.ForProvider.Zone = reference.ToPtrValue(rsp.ResolvedValue)
	dr.Spec.ForProvider.ZoneRef = rsp.ResolvedReference

	return nil
}
