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
	MockCreateDNSRecord func(ctx context.Context, zoneID string, rr cloudflare.DNSRecord) (*cloudflare.DNSRecordResponse, error)
	MockUpdateDNSRecord func(ctx context.Context, zoneID, recordID string, rr cloudflare.DNSRecord) error
	MockDNSRecord       func(ctx context.Context, zoneID, recordID string) (cloudflare.DNSRecord, error)
	MockDeleteDNSRecord func(ctx context.Context, zoneID, recordID string) error
}

// CreateDNSRecord mocks the CreateDNSRecord method of the Cloudflare API.
func (m MockClient) CreateDNSRecord(ctx context.Context, zoneID string, rr cloudflare.DNSRecord) (*cloudflare.DNSRecordResponse, error) {
	return m.MockCreateDNSRecord(ctx, zoneID, rr)
}

// UpdateDNSRecord mocks the UpdateDNSRecord method of the Cloudflare API.
func (m MockClient) UpdateDNSRecord(ctx context.Context, zoneID, recordID string, rr cloudflare.DNSRecord) error {
	return m.MockUpdateDNSRecord(ctx, zoneID, recordID, rr)
}

// DNSRecord mocks the DNSRecord method of the Cloudflare API.
func (m MockClient) DNSRecord(ctx context.Context, zoneID, recordID string) (cloudflare.DNSRecord, error) {
	return m.MockDNSRecord(ctx, zoneID, recordID)
}

// DeleteDNSRecord mocks the DeleteDNSRecord method of the Cloudflare API.
func (m MockClient) DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	return m.MockDeleteDNSRecord(ctx, zoneID, recordID)
}
