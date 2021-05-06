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
	MockUpdateCustomHostnameFallbackOrigin func(ctx context.Context, zoneID string, chfo cloudflare.CustomHostnameFallbackOrigin) (*cloudflare.CustomHostnameFallbackOriginResponse, error)
	MockDeleteCustomHostnameFallbackOrigin func(ctx context.Context, zoneID string) error
	MockCustomHostnameFallbackOrigin       func(ctx context.Context, zoneID string) (cloudflare.CustomHostnameFallbackOrigin, error)
}

// UpdateCustomHostnameFallbackOrigin mocks the UpdateCustomHostnameFallbackOrigin method of the Cloudflare API.
func (m MockClient) UpdateCustomHostnameFallbackOrigin(ctx context.Context, zoneID string, chfo cloudflare.CustomHostnameFallbackOrigin) (*cloudflare.CustomHostnameFallbackOriginResponse, error) {
	return m.MockUpdateCustomHostnameFallbackOrigin(ctx, zoneID, chfo)
}

// DeleteCustomHostnameFallbackOrigin mocks the DeleteCustomHostnameFallbackOrigin method of the Cloudflare API.
func (m MockClient) DeleteCustomHostnameFallbackOrigin(ctx context.Context, zoneID string) error {
	return m.MockDeleteCustomHostnameFallbackOrigin(ctx, zoneID)
}

// CustomHostnameFallbackOrigin mocks the CustomHostnameFallbackOrigin method of the Cloudflare API.
func (m MockClient) CustomHostnameFallbackOrigin(ctx context.Context, zoneID string) (cloudflare.CustomHostnameFallbackOrigin, error) {
	return m.MockCustomHostnameFallbackOrigin(ctx, zoneID)
}
