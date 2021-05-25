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

	"github.com/benagricola/provider-cloudflare/apis/sslsaas/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
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
	errStr := err.Error()
	return strings.Contains(errStr, errCustomHostnameNotFound)
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

// UpToDate checks if the remote resource is up to date with the
// requested resource parameters.
func UpToDate(spec *v1alpha1.CustomHostnameParameters, o cloudflare.CustomHostname) bool { //nolint:gocyclo
	// NOTE(bagricola): The complexity here is simply repeated
	// if statements checking for updated fields. You should think
	// before adding further complexity to this method, but adding
	// more field checks is not an issue.
	if spec == nil {
		return true
	}

	// Check if mutable fields are up to date with resource
	if spec.CustomOriginServer != nil && o.CustomOriginServer != "" &&
		*spec.CustomOriginServer != o.CustomOriginServer {
		return false
	}

	// SSL
	if spec.SSL.Method != nil && o.SSL.Method != "" && *spec.SSL.Method != o.SSL.Method {
		return false
	}

	if spec.SSL.Type != nil && o.SSL.Type != "" && *spec.SSL.Type != o.SSL.Type {
		return false
	}

	if spec.SSL.Wildcard != nil && o.SSL.Wildcard != nil && *spec.SSL.Wildcard != *o.SSL.Wildcard {
		return false
	}

	if spec.SSL.CustomCertificate != nil && o.SSL.CustomCertificate != "" && *spec.SSL.CustomCertificate != o.SSL.CustomCertificate {
		return false
	}

	if spec.SSL.CustomKey != nil && o.SSL.CustomKey != "" && *spec.SSL.CustomKey != o.SSL.CustomKey {
		return false
	}

	return true
}

// UpdateCustomHostname updates mutable values on a Fallback Origin
func UpdateCustomHostname(ctx context.Context, client Client, chID string, spec *v1alpha1.CustomHostnameParameters) error {

	sslSettings := cloudflare.CustomHostnameSSLSettings{
		Ciphers: spec.SSL.Settings.Ciphers,
	}

	// Check the SSL Settings Config
	if spec.SSL.Settings.HTTP2 != nil {
		sslSettings.HTTP2 = *spec.SSL.Settings.HTTP2
	}

	if spec.SSL.Settings.TLS13 != nil {
		sslSettings.TLS13 = *spec.SSL.Settings.TLS13
	}

	if spec.SSL.Settings.MinTLSVersion != nil {
		sslSettings.MinTLSVersion = *spec.SSL.Settings.MinTLSVersion
	}

	ssl := cloudflare.CustomHostnameSSL{
		Wildcard: spec.SSL.Wildcard,
		Settings: sslSettings,
	}

	// Check the SSL Config
	if spec.SSL.Method != nil {
		ssl.Method = *spec.SSL.Method
	}

	if spec.SSL.Type != nil {
		ssl.Type = *spec.SSL.Type
	}

	if spec.SSL.CustomCertificate != nil {
		ssl.CustomCertificate = *spec.SSL.CustomCertificate
	}

	if spec.SSL.CustomKey != nil {
		ssl.CustomKey = *spec.SSL.CustomKey
	}

	ch := cloudflare.CustomHostname{
		Hostname: spec.Hostname,
		SSL:      ssl,
	}
	_, er := client.UpdateCustomHostname(ctx, *spec.Zone, chID, ch)

	return er

}
