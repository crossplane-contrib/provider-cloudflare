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
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	rtv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/benagricola/provider-cloudflare/apis/workers/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	"github.com/benagricola/provider-cloudflare/internal/clients/workers/route"
)

const (
	errNotRoute = "managed resource is not a Route custom resource"

	errClientConfig = "error getting client config"

	errRouteLookup   = "cannot lookup Route"
	errRouteCreation = "cannot create Route"
	errRouteUpdate   = "cannot update Route"
	errRouteDeletion = "cannot delete Route"
	errRouteNoZone   = "no zone found"

	maxConcurrency = 5
)

// Setup adds a controller that reconciles Route managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.RouteGroupKind)

	o := controller.Options{
		RateLimiter:             ratelimiter.NewDefaultManagedRateLimiter(rl),
		MaxConcurrentReconciles: maxConcurrency,
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RouteGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:                  mgr.GetClient(),
			newCloudflareClientFn: route.NewClient}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithPollInterval(5*time.Minute),
		// Do not initialize external-name field.
		managed.WithInitializers(),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.Route{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                  client.Client
	newCloudflareClientFn func(cfg clients.Config) (route.Client, error)
}

// Connect produces a valid configuration for a Cloudflare API
// instance, and returns it as an external client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Route)
	if !ok {
		return nil, errors.New(errNotRoute)
	}

	// Get client configuration
	config, err := clients.GetConfig(ctx, c.kube, mg)
	if err != nil {
		return nil, errors.Wrap(err, errClientConfig)
	}

	client, err := c.newCloudflareClientFn(*config)
	if err != nil {
		return nil, err
	}

	return &external{client: client}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client route.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Route)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRoute)
	}

	// Route does not exist if we dont have an ID stored in external-name
	rid := meta.GetExternalName(cr)
	if rid == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalObservation{}, errors.New(errRouteNoZone)
	}

	r, err := e.client.GetWorkerRoute(ctx, *cr.Spec.ForProvider.Zone, rid)

	if err != nil {
		return managed.ExternalObservation{},
			errors.Wrap(resource.Ignore(route.IsRouteNotFound, err), errRouteLookup)
	}

	cr.Status.SetConditions(rtv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: route.UpToDate(&cr.Spec.ForProvider, r.WorkerRoute),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Route)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRoute)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalCreation{}, errors.Wrap(errors.New(errRouteNoZone), errRouteCreation)
	}

	r := cloudflare.WorkerRoute{
		Pattern: cr.Spec.ForProvider.Pattern,
	}
	if cr.Spec.ForProvider.Script != nil {
		r.Script = *cr.Spec.ForProvider.Script
	}

	nr, err := e.client.CreateWorkerRoute(ctx, *cr.Spec.ForProvider.Zone, r)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errRouteCreation)
	}

	// Update the external name with the ID of the new Route
	meta.SetExternalName(cr, nr.WorkerRoute.ID)

	return managed.ExternalCreation{ExternalNameAssigned: true}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Route)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRoute)
	}

	rid := meta.GetExternalName(cr)
	if rid == "" {
		return managed.ExternalUpdate{}, errors.New(errRouteUpdate)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalUpdate{}, errors.Wrap(errors.New(errRouteNoZone), errRouteUpdate)
	}

	return managed.ExternalUpdate{},
		errors.Wrap(
			route.UpdateRoute(ctx, e.client, meta.GetExternalName(cr), &cr.Spec.ForProvider),
			errRouteUpdate,
		)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Route)
	if !ok {
		return errors.New(errNotRoute)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return errors.Wrap(errors.New(errRouteNoZone), errRouteDeletion)
	}

	rid := meta.GetExternalName(cr)
	if rid == "" {
		return errors.New(errRouteDeletion)
	}

	_, err := e.client.DeleteWorkerRoute(ctx, *cr.Spec.ForProvider.Zone, meta.GetExternalName(cr))

	return errors.Wrap(err, errRouteDeletion)
}
