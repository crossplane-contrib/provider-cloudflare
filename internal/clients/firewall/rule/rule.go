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

package rule

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/go-cmp/cmp"

	"github.com/pkg/errors"

	"github.com/cloudflare/cloudflare-go"

	"github.com/benagricola/provider-cloudflare/apis/firewall/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
)

const (
	errUpdateRule = "error updating firewall rule"
	errCreateRule = "error creating firewall rule"
	errSpecNil    = "rule spec is empty"
)

// Client is a Cloudflare API client that implements methods for working
// with Firewall rules.
type Client interface {
	// Note there is no singular CreateRule in cloudflare-go
	CreateFirewallRules(ctx context.Context, zoneID string, firewallRules []cloudflare.FirewallRule) ([]cloudflare.FirewallRule, error)
	UpdateFirewallRule(ctx context.Context, zoneID string, firewallRule cloudflare.FirewallRule) (cloudflare.FirewallRule, error)
	DeleteFirewallRule(ctx context.Context, zoneID, firewallRuleID string) error
	FirewallRule(ctx context.Context, zoneID, firewallRuleID string) (cloudflare.FirewallRule, error)
}

// NewClient returns a new Cloudflare API client for working with Firewall rules.
func NewClient(cfg clients.Config, hc *http.Client) (Client, error) {
	return clients.NewClient(cfg, hc)
}

// IsRuleNotFound returns true if the passed error indicates
// a Rule was not found.
func IsRuleNotFound(err error) bool {
	return strings.Contains(err.Error(), "HTTP status 404")
}

// GenerateObservation creates an observation of a cloudflare Rule
func GenerateObservation(in cloudflare.FirewallRule) v1alpha1.RuleObservation {
	return v1alpha1.RuleObservation{}
}

func productsToBypassProducts(products []string) []v1alpha1.RuleBypassProduct {
	bpp := make([]v1alpha1.RuleBypassProduct, len(products))
	for i, v := range products {
		bpp[i] = v1alpha1.RuleBypassProduct(v)
	}
	return bpp
}

func bypassProductsToProducts(bypassProducts []v1alpha1.RuleBypassProduct) []string {
	p := make([]string, len(bypassProducts))
	for i, v := range bypassProducts {
		p[i] = string(v)
	}
	return p
}

// LateInitialize initializes RuleParameters based on the remote resource
func LateInitialize(spec *v1alpha1.RuleParameters, r cloudflare.FirewallRule) bool { //nolint:gocyclo
	// NOTE: Gocyclo ignored here because this method has to check each field.

	if spec == nil {
		return false
	}

	li := false
	if len(spec.BypassProducts) == 0 && len(r.Products) > 0 {
		spec.BypassProducts = productsToBypassProducts(r.Products)
		li = true
	}

	if spec.Paused == nil {
		spec.Paused = &r.Paused
		li = true
	}

	if spec.Description == nil && len(r.Description) > 0 {
		spec.Description = &r.Description
		li = true
	}

	// Note that the cloudflare field itself can be a float, but
	// we represent it in the Kubernetes API as an int32.
	// We think this gives users adequate ability to control
	// priority without resorting to decimals.
	if spec.Priority == nil {
		// Priority should be a whole number
		if p, ok := r.Priority.(float64); ok {
			in := int32(p)
			spec.Priority = &in
			li = true
		}
	}

	return li
}

// UpToDate checks if the remote resource is up to date with the
// requested resource parameters.
func UpToDate(spec *v1alpha1.RuleParameters, r cloudflare.FirewallRule) bool { //nolint:gocyclo
	// If we don't have a spec, we _must_ be up to date.
	if spec == nil {
		return true
	}

	// Check if mutable fields are up to date with resource
	if spec.Action != r.Action {
		return false
	}

	cbp := productsToBypassProducts(r.Products)

	// IF bypassProducts IS NOT a nil slice AND is not equal to current products
	// OR if bypassProducts IS a nil slice AND there is more than 0 current products.
	if (spec.BypassProducts != nil && !cmp.Equal(spec.BypassProducts, cbp)) ||
		(spec.BypassProducts == nil && len(cbp) > 0) {
		return false
	}

	if spec.Description != nil && *spec.Description != r.Description {
		return false
	}

	if spec.Filter != nil && *spec.Filter != r.Filter.ID {
		return false
	}

	if spec.Paused != nil && *spec.Paused != r.Paused {
		return false
	}

	if spec.Priority != nil {
		if p, ok := r.Priority.(float64); ok {
			if int32(p) != *spec.Priority {
				return false
			}
		} else {
			// Remote value is unset but requested value is set
			return false
		}
	}

	return true
}

// CreateRule creates a new Rule
func CreateRule(ctx context.Context, client Client, spec *v1alpha1.RuleParameters) (*cloudflare.FirewallRule, error) {

	if spec == nil {
		return nil, errors.New(errSpecNil)
	}

	r := cloudflare.FirewallRule{
		Action: spec.Action,
		Filter: cloudflare.Filter{
			ID: *spec.Filter,
		},
		Products: bypassProductsToProducts(spec.BypassProducts),
	}

	if spec.Description != nil {
		r.Description = *spec.Description
	}
	if spec.Paused != nil {
		r.Paused = *spec.Paused
	}
	if spec.Priority != nil {
		r.Priority = *spec.Priority
	}

	res, err := client.CreateFirewallRules(
		ctx,
		*spec.Zone,
		[]cloudflare.FirewallRule{r},
	)

	if err != nil || len(res) != 1 {
		return nil, errors.Wrap(err, errCreateRule)
	}

	return &res[0], nil
}

// UpdateRule updates mutable values on a Rule
func UpdateRule(ctx context.Context, client Client, ruleID string, spec *v1alpha1.RuleParameters) error { //nolint:gocyclo
	// Get current firewall rule status
	r, err := client.FirewallRule(ctx, *spec.Zone, ruleID)
	if err != nil {
		return errors.Wrap(err, errUpdateRule)
	}

	r.Action = spec.Action
	r.Products = bypassProductsToProducts(spec.BypassProducts)

	if spec.Description != nil {
		r.Description = *spec.Description
	}

	if spec.Filter != nil {
		r.Filter.ID = *spec.Filter
	}

	if spec.Paused != nil {
		r.Paused = *spec.Paused
	}

	if spec.Priority != nil {
		r.Priority = *spec.Priority
	} else {
		r.Priority = nil
	}

	// Update firewall rule
	_, err = client.UpdateFirewallRule(ctx, *spec.Zone, r)
	return errors.Wrap(err, errUpdateRule)
}
