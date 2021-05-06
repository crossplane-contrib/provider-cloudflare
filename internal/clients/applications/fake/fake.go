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
	MockCreateSpectrumApplication func(ctx context.Context, zoneID string, appDetails cloudflare.SpectrumApplication) (cloudflare.SpectrumApplication, error)
	MockSpectrumApplication       func(ctx context.Context, zoneID string, applicationID string) (cloudflare.SpectrumApplication, error)
	MockUpdateSpectrumApplication func(ctx context.Context, zoneID, appID string, appDetails cloudflare.SpectrumApplication) (cloudflare.SpectrumApplication, error)
	MockDeleteSpectrumApplication func(ctx context.Context, zoneID string, applicationID string) error
}

// CreateSpectrumApplication mocks the CreateSpectrumApplication method of the Cloudflare API.
func (m MockClient) CreateSpectrumApplication(ctx context.Context, zoneID string, appDetails cloudflare.SpectrumApplication) (cloudflare.SpectrumApplication, error) {
	return m.MockCreateSpectrumApplication(ctx, zoneID, appDetails)
}

// SpectrumApplication mocks the SpectrumApplication method of the Cloudflare API.
func (m MockClient) SpectrumApplication(ctx context.Context, zoneID string, applicationID string) (cloudflare.SpectrumApplication, error) {
	return m.MockSpectrumApplication(ctx, zoneID, applicationID)
}

// UpdateSpectrumApplication mocks the UpdateSpectrumApplication method of the Cloudflare API.
func (m MockClient) UpdateSpectrumApplication(ctx context.Context, zoneID, appID string, appDetails cloudflare.SpectrumApplication) (cloudflare.SpectrumApplication, error) {
	return m.MockUpdateSpectrumApplication(ctx, zoneID, appID, appDetails)
}

// DeleteSpectrumApplication mocks the DeleteSpectrumApplication method of the Cloudflare API.
func (m MockClient) DeleteSpectrumApplication(ctx context.Context, zoneID string, applicationID string) error {
	return m.MockDeleteSpectrumApplication(ctx, zoneID, applicationID)
}
