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

package fallbackorigin

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	rtv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/cloudflare/cloudflare-go"

	"github.com/benagricola/provider-cloudflare/apis/sslsaas/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	fallbackorigins "github.com/benagricola/provider-cloudflare/internal/clients/sslsaas/fallbackorigins"
)

const (
	errNotFallbackOrigin = "managed resource is not a Fallback Origin custom resource"

	errClientConfig = "error getting client config"

	errFallbackOriginLookup   = "cannot lookup fallback origin"
	errFallbackOriginCreation = "cannot create fallback origin"
	errFallbackOriginUpdate   = "cannot update fallback origin"
	errFallbackOriginDeletion = "cannot delete fallback origin"
	errFallbackOriginNoZone   = "cannot create fallback origin no zone found"

	// String returned if the Fallback Origin is active
	fallbackOriginStatusActive = "active"

	maxConcurrency = 5
)

// Setup adds a controller that reconciles FallbackOrigin managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.FallbackOriginGroupKind)

	o := controller.Options{
		RateLimiter:             ratelimiter.NewDefaultManagedRateLimiter(rl),
		MaxConcurrentReconciles: maxConcurrency,
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.FallbackOriginGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:                  mgr.GetClient(),
			newCloudflareClientFn: fallbackorigins.NewClient}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithPollInterval(5*time.Minute),
		// Do not initialize external-name field.
		managed.WithInitializers(),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.FallbackOrigin{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                  client.Client
	newCloudflareClientFn func(cfg clients.Config) (fallbackorigins.Client, error)
}

// Connect produces a valid configuration for a Cloudflare API
// instance, and returns it as an external client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.FallbackOrigin)
	if !ok {
		return nil, errors.New(errNotFallbackOrigin)
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
	client fallbackorigins.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.FallbackOrigin)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotFallbackOrigin)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalObservation{}, errors.New(errFallbackOriginNoZone)
	}

	fallbackorigin, err := e.client.CustomHostnameFallbackOrigin(ctx, *cr.Spec.ForProvider.Zone)

	if err != nil {
		if fallbackorigins.IsFallbackOriginNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errFallbackOriginLookup)
	}

	cr.Status.AtProvider = fallbackorigins.GenerateObservation(fallbackorigin)

	if cr.Status.AtProvider.Status == fallbackOriginStatusActive {
		cr.Status.SetConditions(rtv1.Available())
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: fallbackorigins.UpToDate(&cr.Spec.ForProvider, fallbackorigin),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {

	cr, ok := mg.(*v1alpha1.FallbackOrigin)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotFallbackOrigin)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalCreation{}, errors.New(errFallbackOriginCreation)
	}

	if cr.Spec.ForProvider.Origin == nil {
		return managed.ExternalCreation{}, errors.New(errFallbackOriginCreation)
	}

	cr.SetConditions(rtv1.Creating())

	_, err := e.client.UpdateCustomHostnameFallbackOrigin(
		ctx,
		*cr.Spec.ForProvider.Zone,
		cloudflare.CustomHostnameFallbackOrigin{
			Origin: *cr.Spec.ForProvider.Origin,
		},
	)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errFallbackOriginCreation)
	}

	return managed.ExternalCreation{}, nil

}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.FallbackOrigin)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotFallbackOrigin)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalUpdate{}, errors.New(errFallbackOriginUpdate)
	}

	er := fallbackorigins.UpdateFallbackOrigin(ctx, e.client, &cr.Spec.ForProvider)

	return managed.ExternalUpdate{},
		errors.Wrap(
			er,
			errFallbackOriginUpdate,
		)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.FallbackOrigin)
	if !ok {
		return errors.New(errNotFallbackOrigin)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return errors.New(errFallbackOriginDeletion)
	}

	return errors.Wrap(
		e.client.DeleteCustomHostnameFallbackOrigin(ctx, *cr.Spec.ForProvider.Zone),
		errFallbackOriginDeletion)
}
