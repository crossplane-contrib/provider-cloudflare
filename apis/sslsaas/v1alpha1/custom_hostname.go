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

	// +kubebuilder:default="on"
	HTTP2 *string `json:"http2,omitempty"`

	// +kubebuilder:default="on"
	TLS13 *string `json:"tls_1_3,omitempty"`

	// +kubebuilder:default="1.2"
	MinTLSVersion *string `json:"min_tls_version,omitempty"`

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

	// +kubebuilder:default="http"
	Method *string `json:"method,omitempty"`

	// +kubebuilder:default="dv"
	Type *string `json:"type,omitempty"`

	Settings CustomHostnameSSLSettings `json:"settings,omitempty"`

	// +kubebuilder:default=false
	Wildcard *bool `json:"wildcard,omitempty"`

	// +kubebuilder:default=""
	CustomCertificate *string `json:"custom_certificate,omitempty"`

	// +kubebuilder:default=""
	CustomKey *string `json:"custom_key,omitempty"`

	// Following fields are in the API but not supported in go library yet
	// BundleMethod      *string                   `json:"bundle_method,omitempty"`

}

// CustomHostnameSSLObserved represents the Observed SSL section in a given custom hostname.
type CustomHostnameSSLObserved struct {
	Status               string                                         `json:"status,omitempty"`
	HTTPUrl              string                                         `json:"http_url,omitempty"`
	HTTPBody             string                                         `json:"http_body,omitempty"`
	ValidationErrors     []cloudflare.CustomHostnameSSLValidationErrors `json:"validation_errors,omitempty"`
	CertificateAuthority string                                         `json:"certificate_authority,omitempty"`
	CnameName            string                                         `json:"cname,omitempty"`
	CnameTarget          string                                         `json:"cname_target,omitempty"`

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
	// Hostname for the custom hostnam.
	// +immutable
	Hostname *string `json:"hostname,omitempty"`

	// SSL Settings for a Custom Hostname
	// +optional
	SSL CustomHostnameSSL `json:"ssl,omitempty"`

	// CustomOriginServer for a Custom Hostname
	CustomOriginServer *string `json:"custom_origin_server,omitempty"`

	// ZoneID this custom hostname is for.
	// +immutable
	// +optional
	Zone *string `json:"zone,omitempty"`

	// ZoneRef references the zone object this custom hostnam is for.
	// +immutable
	// +optional
	ZoneRef *xpv1.Reference `json:"zoneRef,omitempty"`

	// ZoneSelector selects the zone object this custom hostnam is for.
	// +immutable
	// +optional
	ZoneSelector *xpv1.Selector `json:"zoneSelector,omitempty"`
}

// CustomHostnameObservation are the observable fields of a custom hostnam.
type CustomHostnameObservation struct {
	Status             cloudflare.CustomHostnameStatus `json:"status,omitempty"`
	VerificationErrors []string                        `json:"verification_errors,omitempty"`
	SSL                CustomHostnameSSLObserved       `json:"ssl,omitempty"`
}

// A CustomHostnameSpec defines the desired state of a custom hostnam.
type CustomHostnameSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       CustomHostnameParameters `json:"forProvider"`
}

// A CustomHostnameStatus represents the observed state of a custom hostnam.
type CustomHostnameStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          CustomHostnameObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A CustomHostname is a custom hostnam required to use SSL for SaaS.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.bindingPhase"
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
