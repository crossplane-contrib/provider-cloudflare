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
	"strings"

	"github.com/google/go-cmp/cmp"

	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudflare/cloudflare-go"

	"github.com/benagricola/provider-cloudflare/apis/firewall/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
)

const (
	errUpdateFirewallRule = "error updating firewall rule"
)

// Client is a Cloudflare API client that implements methods for working
// with Firewall rules.
type Client interface {
	// Note there is no singular CreateFirewallRule in cloudflare-go
	CreateFirewallRules(ctx context.Context, zoneID string, firewallRules []cloudflare.FirewallRule) ([]cloudflare.FirewallRule, error)
	UpdateFirewallRule(ctx context.Context, zoneID string, firewallRule cloudflare.FirewallRule) (cloudflare.FirewallRule, error)
	DeleteFirewallRule(ctx context.Context, zoneID, firewallRuleID string) error
	FirewallRule(ctx context.Context, zoneID, firewallRuleID string) (cloudflare.FirewallRule, error)
}

// NewClient returns a new Cloudflare API client for working with Firewall rules.
func NewClient(cfg clients.Config) Client {
	return clients.NewClient(cfg)
}

// IsRuleNotFound returns true if the passed error indicates
// a Rule was not found.
func IsRuleNotFound(err error) bool {
	return strings.Contains(err.Error(), "HTTP status 404")
}

// GenerateObservation creates an observation of a cloudflare FirewallRule
func GenerateObservation(in cloudflare.FirewallRule) v1alpha1.FirewallRuleObservation {
	return v1alpha1.FirewallRuleObservation{
		CreatedOn:  &metav1.Time{Time: in.CreatedOn},
		ModifiedOn: &metav1.Time{Time: in.ModifiedOn},
	}
}

func productsToBypassProducts(products []string) []v1alpha1.FirewallRuleBypassProduct {
	bpp := make([]v1alpha1.FirewallRuleBypassProduct, len(products))
	for i, v := range products {
		bpp[i] = v1alpha1.FirewallRuleBypassProduct(v)
	}
	return bpp
}

func bypassProductsToProducts(bypassProducts []v1alpha1.FirewallRuleBypassProduct) []string {
	p := make([]string, len(bypassProducts))
	for i, v := range bypassProducts {
		p[i] = string(v)
	}
	return p
}

// LateInitialize initializes FirewallRuleParameters based on the remote resource
func LateInitialize(spec *v1alpha1.FirewallRuleParameters, r cloudflare.FirewallRule) bool {

	if spec == nil {
		return false
	}

	li := false
	if len(spec.BypassProducts) == 0 {
		spec.BypassProducts = productsToBypassProducts(r.Products)
		li = true
	}
	if spec.Paused == nil {
		spec.Paused = &r.Paused
		li = true
	}

	return li
}

// UpToDate checks if the remote resource is up to date with the
// requested resource parameters.
func UpToDate(spec *v1alpha1.FirewallRuleParameters, r cloudflare.FirewallRule) bool { //nolint:gocyclo
	// If we don't have a spec, we _must_ be up to date.
	if spec == nil {
		return true
	}

	// Check if mutable fields are up to date with resource
	if spec.Action != r.Action {
		return false
	}

	if !cmp.Equal(spec.BypassProducts, productsToBypassProducts(r.Products)) {
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

	if spec.Priority != nil && *spec.Priority != r.Priority {
		return false
	}

	return true
}

// CreateFirewallRule creates a new FirewallRule
func CreateFirewallRule(ctx context.Context, client Client, spec *v1alpha1.FirewallRuleParameters) (*cloudflare.FirewallRule, error) {
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

	if err != nil {
		return nil, err
	}

	if len(res) != 1 {
		return nil, err
	}
	return &res[0], nil
}

// UpdateFirewallRule updates mutable values on a FirewallRule
func UpdateFirewallRule(ctx context.Context, client Client, ruleID string, spec *v1alpha1.FirewallRuleParameters) error { //nolint:gocyclo
	// Get current firewall rule status
	r, err := client.FirewallRule(ctx, *spec.Zone, ruleID)
	if err != nil {
		return errors.Wrap(err, errUpdateFirewallRule)
	}

	u := false

	if spec.Action != r.Action {
		r.Action = spec.Action
		u = true
	}

	if !cmp.Equal(spec.BypassProducts, productsToBypassProducts(r.Products)) {
		r.Products = bypassProductsToProducts(spec.BypassProducts)
		u = true
	}

	if spec.Description != nil && *spec.Description != r.Description {
		r.Description = *spec.Description
		u = true
	}

	if spec.Filter != nil && *spec.Filter != r.Filter.ID {
		r.Filter.ID = *spec.Filter
		u = true
	}

	if spec.Paused != nil && *spec.Paused != r.Paused {
		r.Paused = *spec.Paused
		u = true
	}

	if spec.Priority != nil && *spec.Priority != r.Priority {
		r.Priority = *spec.Priority
		u = true
	}

	if !u {
		return nil
	}

	// Update firewall rule
	_, err = client.UpdateFirewallRule(ctx, *spec.Zone, r)
	return err
}
