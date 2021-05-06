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

package fake

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
)

// A MockClient acts as a testable representation of the Cloudflare API.
type MockClient struct {
	MockCreateFirewallRules func(ctx context.Context, zoneID string, rr []cloudflare.FirewallRule) ([]cloudflare.FirewallRule, error)
	MockUpdateFirewallRule  func(ctx context.Context, zoneID string, rr cloudflare.FirewallRule) (cloudflare.FirewallRule, error)
	MockFirewallRule        func(ctx context.Context, zoneID, ruleID string) (cloudflare.FirewallRule, error)
	MockDeleteFirewallRule  func(ctx context.Context, zoneID, ruleID string) error
}

// CreateFirewallRules mocks the CreateFirewallRules method of the Cloudflare API.
func (m MockClient) CreateFirewallRules(ctx context.Context, zoneID string, rr []cloudflare.FirewallRule) ([]cloudflare.FirewallRule, error) {
	return m.MockCreateFirewallRules(ctx, zoneID, rr)
}

// UpdateFirewallRule mocks the UpdateFirewallRule method of the Cloudflare API.
func (m MockClient) UpdateFirewallRule(ctx context.Context, zoneID string, rr cloudflare.FirewallRule) (cloudflare.FirewallRule, error) {
	return m.MockUpdateFirewallRule(ctx, zoneID, rr)
}

// FirewallRule mocks the FirewallRule method of the Cloudflare API.
func (m MockClient) FirewallRule(ctx context.Context, zoneID, ruleID string) (cloudflare.FirewallRule, error) {
	return m.MockFirewallRule(ctx, zoneID, ruleID)
}

// DeleteFirewallRule mocks the DeleteFirewallRule method of the Cloudflare API.
func (m MockClient) DeleteFirewallRule(ctx context.Context, zoneID, ruleID string) error {
	return m.MockDeleteFirewallRule(ctx, zoneID, ruleID)
}
