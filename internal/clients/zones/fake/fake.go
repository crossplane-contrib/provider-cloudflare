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
	MockCreateZone         func(ctx context.Context, name string, jumpstart bool, account cloudflare.Account, zoneType string) (cloudflare.Zone, error)
	MockDeleteZone         func(ctx context.Context, zoneID string) (cloudflare.ZoneID, error)
	MockEditZone           func(ctx context.Context, zoneID string, zoneOpts cloudflare.ZoneOptions) (cloudflare.Zone, error)
	MockUpdateZoneSettings func(ctx context.Context, zoneID string, cs []cloudflare.ZoneSetting) (*cloudflare.ZoneSettingResponse, error)
	MockZoneDetails        func(ctx context.Context, zoneID string) (cloudflare.Zone, error)
	MockZoneIDByName       func(zoneName string) (string, error)
	MockZoneSetPlan        func(ctx context.Context, zoneID string, planType string) error
	MockZoneSettings       func(ctx context.Context, zoneID string) (*cloudflare.ZoneSettingResponse, error)
}

// CreateZone mocks the CreateZone method of the Cloudflare API.
func (m MockClient) CreateZone(ctx context.Context, name string, jumpstart bool, account cloudflare.Account, zoneType string) (cloudflare.Zone, error) {
	return m.MockCreateZone(ctx, name, jumpstart, account, zoneType)
}

// DeleteZone mocks the DeleteZone method of the Cloudflare API.
func (m MockClient) DeleteZone(ctx context.Context, zoneID string) (cloudflare.ZoneID, error) {
	return m.MockDeleteZone(ctx, zoneID)
}

// EditZone mocks the EditZone method of the Cloudflare API.
func (m MockClient) EditZone(ctx context.Context, zoneID string, zoneOpts cloudflare.ZoneOptions) (cloudflare.Zone, error) {
	return m.MockEditZone(ctx, zoneID, zoneOpts)
}

// UpdateZoneSettings mocks the UpdateZoneSettings method of the Cloudflare API.
func (m MockClient) UpdateZoneSettings(ctx context.Context, zoneID string, cs []cloudflare.ZoneSetting) (*cloudflare.ZoneSettingResponse, error) {
	return m.MockUpdateZoneSettings(ctx, zoneID, cs)
}

// ZoneDetails mocks the ZoneDetails method of the Cloudflare API.
func (m MockClient) ZoneDetails(ctx context.Context, zoneID string) (cloudflare.Zone, error) {
	return m.MockZoneDetails(ctx, zoneID)
}

// ZoneIDByName mocks the ZoneIDByName method of the Cloudflare API.
func (m MockClient) ZoneIDByName(zoneName string) (string, error) {
	return m.MockZoneIDByName(zoneName)
}

// ZoneSetPlan mocks the ZoneSetPlan method of the Cloudflare API.
func (m MockClient) ZoneSetPlan(ctx context.Context, zoneID string, planType string) error {
	return m.MockZoneSetPlan(ctx, zoneID, planType)
}

// ZoneSettings mocks the ZoneSettings method of the Cloudflare API.
func (m MockClient) ZoneSettings(ctx context.Context, zoneID string) (*cloudflare.ZoneSettingResponse, error) {
	return m.MockZoneSettings(ctx, zoneID)
}
