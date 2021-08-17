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

package route

import (
	"context"
	"net/http"
	"strings"

	"github.com/cloudflare/cloudflare-go"

	"github.com/crossplane-contrib/provider-cloudflare/apis/workers/v1alpha1"
	clients "github.com/crossplane-contrib/provider-cloudflare/internal/clients"
)

const (
	// Cloudflare returns this code when a route isnt found.
	errRouteNotFound = "10007"
)

// Client is a Cloudflare API client that implements methods for working
// with Worker Routes.
type Client interface {
	CreateWorkerRoute(ctx context.Context, zoneID string, route cloudflare.WorkerRoute) (cloudflare.WorkerRouteResponse, error)
	UpdateWorkerRoute(ctx context.Context, zoneID string, routeID string, route cloudflare.WorkerRoute) (cloudflare.WorkerRouteResponse, error)
	GetWorkerRoute(ctx context.Context, zoneID string, routeID string) (cloudflare.WorkerRouteResponse, error)
	DeleteWorkerRoute(ctx context.Context, zoneID string, routeID string) (cloudflare.WorkerRouteResponse, error)
}

// NewClient returns a new Cloudflare API client for working with Worker Routes.
func NewClient(cfg clients.Config, hc *http.Client) (Client, error) {
	return clients.NewClient(cfg, hc)
}

// IsRouteNotFound returns true if the passed error indicates
// a Worker Route was not found.
func IsRouteNotFound(err error) bool {
	return strings.Contains(err.Error(), errRouteNotFound)
}

// UpToDate checks if the remote Route is up to date with the
// requested resource parameters.
func UpToDate(spec *v1alpha1.RouteParameters, o cloudflare.WorkerRoute) bool { //nolint:gocyclo
	// NOTE(bagricola): The complexity here is simply repeated
	// if statements checking for updated fields. You should think
	// before adding further complexity to this method, but adding
	// more field checks should not be an issue.
	if spec == nil {
		return true
	}

	// Check if mutable fields are up to date with resource
	if spec.Pattern != o.Pattern {
		return false
	}

	if spec.Script == nil && o.Script != "" {
		return false
	}

	if spec.Script != nil && *spec.Script != o.Script {
		return false
	}

	return true
}

// UpdateRoute updates mutable values on a Worker Route.
func UpdateRoute(ctx context.Context, client Client, routeID string, spec *v1alpha1.RouteParameters) error {
	r := cloudflare.WorkerRoute{
		Pattern: spec.Pattern,
	}

	if spec.Script != nil {
		r.Script = *spec.Script
	}

	_, err := client.UpdateWorkerRoute(ctx, *spec.Zone, routeID, r)

	return err

}
