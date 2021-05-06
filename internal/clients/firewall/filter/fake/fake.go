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
	MockCreateFilters func(ctx context.Context, zoneID string, firewallFilters []cloudflare.Filter) ([]cloudflare.Filter, error)
	MockUpdateFilter  func(ctx context.Context, zoneID string, firewallFilter cloudflare.Filter) (cloudflare.Filter, error)
	MockDeleteFilter  func(ctx context.Context, zoneID, firewallFilterID string) error
	MockFilter        func(ctx context.Context, zoneID, filterID string) (cloudflare.Filter, error)
}

// CreateFilters mocks the CreateFilters method of the Cloudflare API.
func (m MockClient) CreateFilters(ctx context.Context, zoneID string, rr []cloudflare.Filter) ([]cloudflare.Filter, error) {
	return m.MockCreateFilters(ctx, zoneID, rr)
}

// UpdateFilter mocks the UpdateFilter method of the Cloudflare API.
func (m MockClient) UpdateFilter(ctx context.Context, zoneID string, rr cloudflare.Filter) (cloudflare.Filter, error) {
	return m.MockUpdateFilter(ctx, zoneID, rr)
}

// Filter mocks the Filter method of the Cloudflare API.
func (m MockClient) Filter(ctx context.Context, zoneID, filterID string) (cloudflare.Filter, error) {
	return m.MockFilter(ctx, zoneID, filterID)
}

// DeleteFilter mocks the DeleteFilter method of the Cloudflare API.
func (m MockClient) DeleteFilter(ctx context.Context, zoneID, filterID string) error {
	return m.MockDeleteFilter(ctx, zoneID, filterID)
}
