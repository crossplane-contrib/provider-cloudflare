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
	MockCreateWorkerRoute func(ctx context.Context, zoneID string, route cloudflare.WorkerRoute) (cloudflare.WorkerRouteResponse, error)
	MockUpdateWorkerRoute func(ctx context.Context, zoneID string, routeID string, route cloudflare.WorkerRoute) (cloudflare.WorkerRouteResponse, error)
	MockGetWorkerRoute    func(ctx context.Context, zoneID string, routeID string) (cloudflare.WorkerRouteResponse, error)
	MockDeleteWorkerRoute func(ctx context.Context, zoneID string, routeID string) (cloudflare.WorkerRouteResponse, error)
}

// CreateWorkerRoute mocks the CreateWorkerRoute method of the Cloudflare API.
func (m MockClient) CreateWorkerRoute(ctx context.Context, zoneID string, route cloudflare.WorkerRoute) (cloudflare.WorkerRouteResponse, error) {
	return m.MockCreateWorkerRoute(ctx, zoneID, route)
}

// UpdateWorkerRoute mocks the UpdateWorkerRoute method of the Cloudflare API.
func (m MockClient) UpdateWorkerRoute(ctx context.Context, zoneID string, routeID string, route cloudflare.WorkerRoute) (cloudflare.WorkerRouteResponse, error) {
	return m.MockUpdateWorkerRoute(ctx, zoneID, routeID, route)
}

// GetWorkerRoute mocks the GetWorkerRoute method of the Cloudflare API.
func (m MockClient) GetWorkerRoute(ctx context.Context, zoneID string, routeID string) (cloudflare.WorkerRouteResponse, error) {
	return m.MockGetWorkerRoute(ctx, zoneID, routeID)
}

// DeleteWorkerRoute mocks the DeleteWorkerRoute method of the Cloudflare API.
func (m MockClient) DeleteWorkerRoute(ctx context.Context, zoneID string, routeID string) (cloudflare.WorkerRouteResponse, error) {
	return m.MockDeleteWorkerRoute(ctx, zoneID, routeID)
}
