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

package fallbackorigins

import (
	"context"
	"errors"
	"strings"

	"github.com/benagricola/provider-cloudflare/apis/sslsaas/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	"github.com/cloudflare/cloudflare-go"
)

const (
	// Cloudflare returns this code when a fallback origin isnt found
	errFallbackOriginNotFound = "1551"
)

// ErrNotFound is an error type so that we can return it in the controller test mocks
type ErrNotFound struct{}

func (e *ErrNotFound) Error() string { return "Fallback origin not found" }

// Client is a Cloudflare API client that implements methods for working
// with Fallback Origins.
type Client interface {
	UpdateCustomHostnameFallbackOrigin(ctx context.Context, zoneID string, chfo cloudflare.CustomHostnameFallbackOrigin) (*cloudflare.CustomHostnameFallbackOriginResponse, error)
	DeleteCustomHostnameFallbackOrigin(ctx context.Context, zoneID string) error
	CustomHostnameFallbackOrigin(ctx context.Context, zoneID string) (cloudflare.CustomHostnameFallbackOrigin, error)
}

// NewClient returns a new Cloudflare API client for working with Fallback Origins.
func NewClient(cfg clients.Config) (Client, error) {
	return clients.NewClient(cfg)
}

// IsFallbackOriginNotFound returns true if the passed error indicates
// that the FallbackOrigin is not found (been deleted or not set at all).
func IsFallbackOriginNotFound(err error) bool {
	// We check for a custom error type here because we need to be able
	// to export something which can be used in the Mock for the controller tests.
	var notFoundError *ErrNotFound
	if errors.As(err, &notFoundError) {
		return true
	}

	// The actual Cloudflare API indicates a "not found" with this error code.
	errStr := err.Error()
	return strings.Contains(errStr, errFallbackOriginNotFound)
}

// GenerateObservation creates an observation of a cloudflare Fallback Origin
func GenerateObservation(in cloudflare.CustomHostnameFallbackOrigin) v1alpha1.FallbackOriginObservation {
	return v1alpha1.FallbackOriginObservation{
		Status: in.Status,
		Errors: in.Errors,
	}
}

// UpToDate checks if the remote resource is up to date with the
// requested resource parameters.
func UpToDate(spec *v1alpha1.FallbackOriginParameters, o cloudflare.CustomHostnameFallbackOrigin) bool { //nolint:gocyclo
	// NOTE(bagricola): The complexity here is simply repeated
	// if statements checking for updated fields. You should think
	// before adding further complexity to this method, but adding
	// more field checks is not an issue.
	if spec == nil {
		return true
	}

	// Check if mutable fields are up to date with resource
	if spec.Origin != nil && o.Origin != "" && *spec.Origin != o.Origin {
		return false
	}

	return true
}

// UpdateFallbackOrigin updates mutable values on a Fallback Origin
func UpdateFallbackOrigin(ctx context.Context, client Client, spec *v1alpha1.FallbackOriginParameters) error {

	fo := cloudflare.CustomHostnameFallbackOrigin{
		Origin: *spec.Origin,
	}

	_, er := client.UpdateCustomHostnameFallbackOrigin(ctx, *spec.Zone, fo)

	return er

}
