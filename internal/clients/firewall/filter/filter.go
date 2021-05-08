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

package filter

import (
	"context"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/cloudflare/cloudflare-go"

	"github.com/benagricola/provider-cloudflare/apis/firewall/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
)

const (
	errUpdateFilter         = "error updating filter"
	errCreateFilterBadCount = "create returned wrong number of filters"
)

// Client is a Cloudflare API client that implements methods for working
// with Firewall rules.
type Client interface {
	// Note there is no singular CreateFilter in cloudflare-go
	CreateFilters(ctx context.Context, zoneID string, firewallFilters []cloudflare.Filter) ([]cloudflare.Filter, error)
	UpdateFilter(ctx context.Context, zoneID string, firewallFilter cloudflare.Filter) (cloudflare.Filter, error)
	DeleteFilter(ctx context.Context, zoneID, firewallFilterID string) error
	Filter(ctx context.Context, zoneID, filterID string) (cloudflare.Filter, error)
}

// NewClient returns a new Cloudflare API client for working with Firewall rules.
func NewClient(cfg clients.Config, hc *http.Client) (Client, error) {
	return clients.NewClient(cfg, hc)
}

// IsFilterNotFound returns true if the passed error indicates
// a Filter was not found.
func IsFilterNotFound(err error) bool {
	return strings.Contains(err.Error(), "HTTP status 404")
}

// GenerateObservation creates an observation of a cloudflare Filter
func GenerateObservation(in cloudflare.Filter) v1alpha1.FilterObservation {
	return v1alpha1.FilterObservation{}
}

// LateInitialize initializes FilterParameters based on the remote resource
func LateInitialize(spec *v1alpha1.FilterParameters, r cloudflare.Filter) bool {

	if spec == nil {
		return false
	}

	li := false

	if spec.Paused == nil {
		spec.Paused = &r.Paused
		li = true
	}

	return li
}

// UpToDate checks if the remote resource is up to date with the
// requested resource parameters.
func UpToDate(spec *v1alpha1.FilterParameters, f cloudflare.Filter) bool {
	// If we don't have a spec, we _must_ be up to date.
	if spec == nil {
		return true
	}

	// Check if mutable fields are up to date with resource
	if strings.TrimSpace(spec.Expression) != f.Expression {
		return false
	}

	if spec.Description != nil && *spec.Description != f.Description {
		return false
	}

	if spec.Paused != nil && *spec.Paused != f.Paused {
		return false
	}

	return true
}

// CreateFilter creates a new Filter
func CreateFilter(ctx context.Context, client Client, spec *v1alpha1.FilterParameters) (*cloudflare.Filter, error) {
	f := cloudflare.Filter{
		Expression: strings.TrimSpace(spec.Expression),
	}

	if spec.Description != nil {
		f.Description = *spec.Description
	}
	if spec.Paused != nil {
		f.Paused = *spec.Paused
	}

	res, err := client.CreateFilters(
		ctx,
		*spec.Zone,
		[]cloudflare.Filter{f},
	)

	if err != nil {
		return nil, err
	}

	// If creation worked then we should have _one_ filter
	// returned. We sanity check here for completeness
	// but we should NEVER return this error as it
	// indicates a problem in the Cloudflare API that
	// was not properly captured by err above.

	// NOTE: This WILL cause the creation to be seen as
	// failed, even though the CreateFilters call may
	// have created filters _on_ Cloudflare, so we rely
	// on repeated calls to CreateFilters not creating
	// duplicates for the same filter expressions (this
	// is the current filter behaviour).
	if len(res) != 1 {
		return nil, errors.New(errCreateFilterBadCount)
	}
	return &res[0], nil
}

// UpdateFilter updates mutable values on a Filter
func UpdateFilter(ctx context.Context, client Client, ruleID string, spec *v1alpha1.FilterParameters) error { //nolint:gocyclo
	// Get current firewall rule status
	f, err := client.Filter(ctx, *spec.Zone, ruleID)
	if err != nil {
		return errors.Wrap(err, errUpdateFilter)
	}

	f.Expression = strings.TrimSpace(spec.Expression)

	if spec.Description != nil {
		f.Description = *spec.Description
	}

	if spec.Paused != nil {
		f.Paused = *spec.Paused
	}

	// Update Filter
	_, err = client.UpdateFilter(ctx, *spec.Zone, f)
	return err
}
