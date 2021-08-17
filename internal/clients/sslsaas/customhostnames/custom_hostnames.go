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

package customhostnames

import (
	"context"
	"net/http"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/crossplane-contrib/provider-cloudflare/apis/sslsaas/v1alpha1"
	clients "github.com/crossplane-contrib/provider-cloudflare/internal/clients"
)

const (
	// Cloudflare returns this code when a custom hostname isnt found
	errCustomHostnameNotFound = "1436"
)

// Client is a Cloudflare API client that implements methods for working
// with Fallback Origins.
type Client interface {
	UpdateCustomHostnameSSL(ctx context.Context, zoneID string, customHostnameID string, ssl cloudflare.CustomHostnameSSL) (*cloudflare.CustomHostnameResponse, error)
	UpdateCustomHostname(ctx context.Context, zoneID string, customHostnameID string, ch cloudflare.CustomHostname) (*cloudflare.CustomHostnameResponse, error)
	DeleteCustomHostname(ctx context.Context, zoneID string, customHostnameID string) error
	CreateCustomHostname(ctx context.Context, zoneID string, ch cloudflare.CustomHostname) (*cloudflare.CustomHostnameResponse, error)
	CustomHostname(ctx context.Context, zoneID string, customHostnameID string) (cloudflare.CustomHostname, error)
}

// NewClient returns a new Cloudflare API client for working with Custom Hostnames.
func NewClient(cfg clients.Config, hc *http.Client) (Client, error) {
	return clients.NewClient(cfg, hc)
}

// IsCustomHostnameNotFound returns true if the passed error indicates
// that the CustomHostname is not found (been deleted or not set at all).
func IsCustomHostnameNotFound(err error) bool {
	return strings.Contains(err.Error(), errCustomHostnameNotFound)
}

// GenerateObservation creates an observation of a cloudflare Custom Hostname
func GenerateObservation(in cloudflare.CustomHostname) v1alpha1.CustomHostnameObservation {

	ssl := v1alpha1.CustomHostnameSSLObserved{
		Status:               in.SSL.Status,
		HTTPUrl:              in.SSL.HTTPUrl,
		HTTPBody:             in.SSL.HTTPBody,
		CnameName:            in.SSL.CnameName,
		CnameTarget:          in.SSL.CnameTarget,
		CertificateAuthority: in.SSL.CertificateAuthority,
		ValidationErrors:     in.SSL.ValidationErrors,
	}

	// Cloudflare API does not capitalise DNS record type in this field.
	// We capitalise it ourselves so it's consistent with other usage
	// on this provider.
	ovdnst := strings.ToUpper(in.OwnershipVerification.Type)

	return v1alpha1.CustomHostnameObservation{
		Status: in.Status,
		OwnershipVerification: v1alpha1.CustomHostnameOwnershipVerification{
			DNSRecord: &v1alpha1.CustomHostnameOwnershipVerificationDNS{
				Type:  &ovdnst,
				Name:  &in.OwnershipVerification.Name,
				Value: &in.OwnershipVerification.Value,
			},
			HTTPFile: &v1alpha1.CustomHostnameOwnershipVerificationHTTP{
				URL:  &in.OwnershipVerificationHTTP.HTTPUrl,
				Body: &in.OwnershipVerificationHTTP.HTTPBody,
			},
		},
		VerificationErrors: in.VerificationErrors,
		SSL:                ssl,
	}
}

// CustomHostnameToParameters returns a CustomHostnameParameters representation of
// a Cloudflare Custom Hostname.
func CustomHostnameToParameters(in cloudflare.CustomHostname) v1alpha1.CustomHostnameParameters {
	return v1alpha1.CustomHostnameParameters{
		Hostname:           in.Hostname,
		CustomOriginServer: clients.ToOptionalString(in.CustomOriginServer),
		SSL: v1alpha1.CustomHostnameSSL{
			// These fields are not optional in our API calls but are
			// defaulted by us.
			Method: clients.ToOptionalString(in.SSL.Method),
			Type:   clients.ToOptionalString(in.SSL.Type),
			Settings: v1alpha1.CustomHostnameSSLSettings{
				HTTP2:         clients.ToOptionalString(in.SSL.Settings.HTTP2),
				TLS13:         clients.ToOptionalString(in.SSL.Settings.TLS13),
				MinTLSVersion: clients.ToOptionalString(in.SSL.Settings.MinTLSVersion),
				Ciphers:       clients.ToStringSlice(in.SSL.Settings.Ciphers),
			},
			Wildcard:          clients.ToBool(in.SSL.Wildcard),
			CustomCertificate: clients.ToOptionalString(in.SSL.CustomCertificate),
			CustomKey:         clients.ToOptionalString(in.SSL.CustomKey),
		},
	}
}

// ParametersToCustomHostname returns a Cloudflare API representation of a Custom
// Hostname from our CustomHostnameParameters.
func ParametersToCustomHostname(in v1alpha1.CustomHostnameParameters) cloudflare.CustomHostname {
	return cloudflare.CustomHostname{
		Hostname: in.Hostname,
		SSL: cloudflare.CustomHostnameSSL{
			Method: *in.SSL.Method,
			Type:   *in.SSL.Type,
			Settings: cloudflare.CustomHostnameSSLSettings{
				HTTP2:         *clients.ToOptionalString(in.SSL.Settings.HTTP2),
				TLS13:         *clients.ToOptionalString(in.SSL.Settings.TLS13),
				MinTLSVersion: *clients.ToOptionalString(in.SSL.Settings.MinTLSVersion),
				Ciphers:       in.SSL.Settings.Ciphers,
			},
			Wildcard:          in.SSL.Wildcard,
			CustomCertificate: *clients.ToOptionalString(in.SSL.CustomCertificate),
			CustomKey:         *clients.ToOptionalString(in.SSL.CustomKey),
		},
	}
}

// UpToDate checks if the remote resource is up to date with the
// requested resource parameters.
func UpToDate(spec *v1alpha1.CustomHostnameParameters, o cloudflare.CustomHostname) bool {
	if spec == nil {
		return true
	}

	return cmp.Equal(*spec,
		CustomHostnameToParameters(o),
		cmpopts.EquateEmpty(),
		cmpopts.IgnoreTypes(&xpv1.Reference{}, &xpv1.Selector{}, []xpv1.Reference{}),
		cmpopts.IgnoreFields(v1alpha1.CustomHostnameParameters{}, "Zone"),
	)
}

// CreateCustomHostname creates a new Custom Hostname.
func CreateCustomHostname(ctx context.Context, client Client, spec v1alpha1.CustomHostnameParameters) (*cloudflare.CustomHostnameResponse, error) {
	return client.CreateCustomHostname(ctx, *spec.Zone, ParametersToCustomHostname(spec))
}

// UpdateCustomHostname updates mutable values on a Custom Hostname.
func UpdateCustomHostname(ctx context.Context, client Client, id string, spec v1alpha1.CustomHostnameParameters) error {
	_, err := client.UpdateCustomHostname(ctx, *spec.Zone, id, ParametersToCustomHostname(spec))
	return err
}
