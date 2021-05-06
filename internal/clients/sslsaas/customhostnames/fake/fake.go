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
	MockUpdateCustomHostnameSSL func(ctx context.Context, zoneID string, customHostnameID string, ssl cloudflare.CustomHostnameSSL) (*cloudflare.CustomHostnameResponse, error)
	MockUpdateCustomHostname    func(ctx context.Context, zoneID string, customHostnameID string, ch cloudflare.CustomHostname) (*cloudflare.CustomHostnameResponse, error)
	MockDeleteCustomHostname    func(ctx context.Context, zoneID string, customHostnameID string) error
	MockCreateCustomHostname    func(ctx context.Context, zoneID string, ch cloudflare.CustomHostname) (*cloudflare.CustomHostnameResponse, error)
	MockCustomHostname          func(ctx context.Context, zoneID string, customHostnameID string) (cloudflare.CustomHostname, error)
}

// UpdateCustomHostnameSSL mocks the UpdateCustomHostnameSSL method of the Cloudflare API.
func (m MockClient) UpdateCustomHostnameSSL(ctx context.Context, zoneID string, customHostnameID string, ssl cloudflare.CustomHostnameSSL) (*cloudflare.CustomHostnameResponse, error) {
	return m.MockUpdateCustomHostnameSSL(ctx, zoneID, customHostnameID, ssl)
}

// UpdateCustomHostname mocks the UpdateCustomHostname method of the Cloudflare API.
func (m MockClient) UpdateCustomHostname(ctx context.Context, zoneID string, customHostnameID string, ch cloudflare.CustomHostname) (*cloudflare.CustomHostnameResponse, error) {
	return m.MockUpdateCustomHostname(ctx, zoneID, customHostnameID, ch)
}

// DeleteCustomHostname mocks the DeleteCustomHostname method of the Cloudflare API.
func (m MockClient) DeleteCustomHostname(ctx context.Context, zoneID string, customHostnameID string) error {
	return m.MockDeleteCustomHostname(ctx, zoneID, customHostnameID)
}

// CreateCustomHostname mocks the CreateCustomHostname method of the Cloudflare API.
func (m MockClient) CreateCustomHostname(ctx context.Context, zoneID string, ch cloudflare.CustomHostname) (*cloudflare.CustomHostnameResponse, error) {
	return m.MockCreateCustomHostname(ctx, zoneID, ch)
}

// CustomHostname mocks the CustomHostname method of the Cloudflare API.
func (m MockClient) CustomHostname(ctx context.Context, zoneID string, customHostnameID string) (cloudflare.CustomHostname, error) {
	return m.MockCustomHostname(ctx, zoneID, customHostnameID)
}
