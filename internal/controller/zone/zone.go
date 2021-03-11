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

package zone

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
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

	"github.com/cloudflare/cloudflare-go"

	apisv1alpha1 "github.com/benagricola/provider-cloudflare/apis/v1alpha1"
	"github.com/benagricola/provider-cloudflare/apis/zone/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	zones "github.com/benagricola/provider-cloudflare/internal/clients/zones"
)

const (
	errNotZone      = "managed resource is not a Zone custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"

	errNewClient = "cannot create new Cloudflare API client"

	errAccountLookup = "cannot lookup account details"
	errZoneLookup    = "cannot lookup zone"
	errZoneCreation  = "cannot create zone"
	errZoneUpdate    = "cannot update zone"
	errZoneDeletion  = "cannot delete zone"

	maxConcurrency = 5

	zoneStatusActive = "active"
)

// Setup adds a controller that reconciles Zone managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.ZoneGroupKind)

	o := controller.Options{
		RateLimiter:             ratelimiter.NewDefaultManagedRateLimiter(rl),
		MaxConcurrentReconciles: maxConcurrency,
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ZoneGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:                  mgr.GetClient(),
			usage:                 resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newCloudflareClientFn: clients.NewClient}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.Zone{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                  client.Client
	usage                 resource.Tracker
	newCloudflareClientFn func(cfg clients.Config) (*cloudflare.API, error)
}

// Connect typically produces an ExternalClient by:1
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Zone)
	if !ok {
		return nil, errors.New(errNotZone)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	config, err := clients.GetConfig(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	api, err := c.newCloudflareClientFn(*config)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{api: api}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	api *cloudflare.API
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Zone)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotZone)
	}

	z, err := zones.LookupZoneByIDOrName(ctx, *c.api, cr.Status.AtProvider.ID, meta.GetExternalName(cr))
	if err != nil {
		if zones.IsZoneNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{ResourceExists: false},
			errors.Wrap(err, errZoneLookup)
	}

	cr.Status.AtProvider = zones.GenerateObservation(z)

	if cr.Status.AtProvider.Status == zoneStatusActive {
		cr.Status.SetConditions(rtv1.Available())
	}

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceLateInitialized: zones.LateInitialize(&cr.Spec.ForProvider, cr.Status.AtProvider),
		ResourceUpToDate:        zones.UpToDate(&cr.Spec.ForProvider, cr.Status.AtProvider),
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Zone)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotZone)
	}

	var (
		account cloudflare.Account
		err     error
	)

	// Get account if user specified one
	if cr.Spec.ForProvider.AccountID != nil {
		account, _, err = c.api.Account(ctx, *cr.Spec.ForProvider.AccountID)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errAccountLookup)
		}
	}

	// This has a default set by CRD, so should not happen,
	// but we sanity check anyway to avoid a nil pointer
	// dereference calling CreateZone below.
	if cr.Spec.ForProvider.JumpStart == nil ||
		cr.Spec.ForProvider.Type == nil {
		return managed.ExternalCreation{}, errors.New(errZoneCreation)
	}

	cr.SetConditions(rtv1.Creating())

	zone, err := c.api.CreateZone(
		ctx,
		meta.GetExternalName(cr),
		*cr.Spec.ForProvider.JumpStart,
		account,
		*cr.Spec.ForProvider.Type,
	)

	// Cloudflare returns an empty Zone when it throws an error.
	// Generating an observation here will do nothing (but also
	// not error).
	cr.Status.AtProvider = zones.GenerateObservation(zone)

	return managed.ExternalCreation{}, errors.Wrap(err, errZoneCreation)
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Zone)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotZone)
	}

	return managed.ExternalUpdate{},
		errors.Wrap(
			zones.UpdateZone(ctx, c.api, &cr.Spec.ForProvider, &cr.Status.AtProvider),
			errZoneUpdate,
		)
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Zone)
	if !ok {
		return errors.New(errNotZone)
	}
	_, err := c.api.DeleteZone(ctx, cr.Status.AtProvider.ID)
	return errors.Wrap(err, errZoneDeletion)
}
