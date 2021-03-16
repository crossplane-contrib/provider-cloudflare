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

	"github.com/cloudflare/cloudflare-go"
	"github.com/google/go-cmp/cmp"
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

	"github.com/benagricola/provider-cloudflare/apis/zone/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	zones "github.com/benagricola/provider-cloudflare/internal/clients/zones"
)

const (
	errNotZone = "managed resource is not a Zone custom resource"

	errClientConfig = "error getting client config"

	errZoneLookup      = "cannot lookup zone"
	errZoneObservation = "cannot observe zone"
	errZoneCreation    = "cannot create zone"
	errZoneUpdate      = "cannot update zone"
	errZoneDeletion    = "cannot delete zone"

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
			newCloudflareClientFn: zones.NewClient,
		}),
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
	newCloudflareClientFn func(cfg clients.Config) zones.Client
}

// Connect produces a valid configuration for a Cloudflare API
// instance, and returns it as an external client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Zone)
	if !ok {
		return nil, errors.New(errNotZone)
	}

	// Get client configuration
	config, err := clients.GetConfig(ctx, c.kube, mg)
	if err != nil {
		return nil, errors.Wrap(err, errClientConfig)
	}

	return &external{client: c.newCloudflareClientFn(*config)}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client zones.Client
}

func (e *external) Observe(ctx context.Context,
	mg resource.Managed) (managed.ExternalObservation, error) {

	cr, ok := mg.(*v1alpha1.Zone)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotZone)
	}

	// Zone does not exist if we're not tracking an ID
	id := cr.Status.AtProvider.ZoneID
	if id == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	z, err := e.client.ZoneDetails(ctx, id)
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

	observedSettings, err := zones.LoadSettingsForZone(ctx, e.client, z.ID)

	if err != nil {
		return managed.ExternalObservation{ResourceExists: false},
			errors.Wrap(err, errZoneObservation)
	}

	desiredSettings := zones.ZoneToSettingsMap(&cr.Spec.ForProvider.Settings)

	return managed.ExternalObservation{
		ResourceExists: true,
		ResourceLateInitialized: zones.LateInitialize(&cr.Spec.ForProvider, z,
			observedSettings, desiredSettings),
		ResourceUpToDate: zones.UpToDate(&cr.Spec.ForProvider, z) &&
			cmp.Equal(observedSettings, desiredSettings),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Zone)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotZone)
	}

	var (
		account cloudflare.Account
		err     error
	)

	// Configure account if user specified one
	if cr.Spec.ForProvider.AccountID != nil {
		account = cloudflare.Account{
			ID: *cr.Spec.ForProvider.AccountID,
		}
	}

	// This has a default set by CRD, so should not happen,
	// but we sanity check anyway to avoid a nil pointer
	// dereference calling CreateZone below.
	if cr.Spec.ForProvider.Type == nil {
		return managed.ExternalCreation{}, errors.New(errZoneCreation)
	}

	cr.SetConditions(rtv1.Creating())

	z, err := e.client.CreateZone(
		ctx,
		cr.Spec.ForProvider.Name,
		cr.Spec.ForProvider.JumpStart,
		account,
		*cr.Spec.ForProvider.Type,
	)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errZoneCreation)
	}

	cr.Status.AtProvider = zones.GenerateObservation(z)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Zone)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotZone)
	}

	return managed.ExternalUpdate{}, errors.Wrap(
		zones.UpdateZone(
			ctx,
			e.client,
			meta.GetExternalName(cr),
			&cr.Spec.ForProvider,
		),
		errZoneUpdate)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Zone)
	if !ok {
		return errors.New(errNotZone)
	}
	cr.SetConditions(rtv1.Deleting())
	_, err := e.client.DeleteZone(ctx, meta.GetExternalName(cr))
	return errors.Wrap(err, errZoneDeletion)
}
