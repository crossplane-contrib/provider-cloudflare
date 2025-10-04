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

package customhostname

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
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane-contrib/provider-cloudflare/apis/sslsaas/v1alpha1"
	clients "github.com/crossplane-contrib/provider-cloudflare/internal/clients"
	customhostnames "github.com/crossplane-contrib/provider-cloudflare/internal/clients/sslsaas/customhostnames"
	metrics "github.com/crossplane-contrib/provider-cloudflare/internal/metrics"
)

const (
	errNotCustomHostname = "managed resource is not a Custom Hostname custom resource"

	errClientConfig = "error getting client config"

	errCustomHostnameLookup   = "cannot lookup custom hostname"
	errCustomHostnameCreation = "cannot create custom hostname"
	errCustomHostnameUpdate   = "cannot update record"
	errCustomHostnameDeletion = "cannot delete record"
	errCustomHostnameNoZone   = "cannot create custom hostname no zone found"
)

const (
	customHostnameStatusActive = "active"

	maxConcurrency = 5
)

// Setup adds a controller that reconciles CustomHostname managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.CustomHostnameGroupKind)

	o := controller.Options{
		RateLimiter:             ratelimiter.NewDefaultManagedRateLimiter(rl),
		MaxConcurrentReconciles: maxConcurrency,
	}

	hc := metrics.NewInstrumentedHTTPClient(name)
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.CustomHostnameGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube: mgr.GetClient(),
			newCloudflareClientFn: func(cfg clients.Config) (customhostnames.Client, error) {
				return customhostnames.NewClient(cfg, hc)
			},
		}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithPollInterval(5*time.Minute),
		// Do not initialize external-name field.
		managed.WithInitializers(),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.CustomHostname{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                  client.Client
	newCloudflareClientFn func(cfg clients.Config) (customhostnames.Client, error)
}

// Connect produces a valid configuration for a Cloudflare API
// instance, and returns it as an external client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.CustomHostname)
	if !ok {
		return nil, errors.New(errNotCustomHostname)
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
	client customhostnames.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.CustomHostname)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCustomHostname)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalObservation{}, errors.New(errCustomHostnameNoZone)
	}

	// Custom Hostname does not exist if we dont have an ID stored in external-name
	chid := meta.GetExternalName(cr)
	if chid == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	ch, err := e.client.CustomHostname(ctx, *cr.Spec.ForProvider.Zone, chid)

	if err != nil {
		if customhostnames.IsCustomHostnameNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errCustomHostnameLookup)
	}

	cr.Status.AtProvider = customhostnames.GenerateObservation(ch)

	// Mark as ready when the Hostname is ready
	// Note that this does not mean that the SSL Certificate is ready
	// That status is available here - cr.Status.AtProvider.SSL.Status

	// We've made the decision to mark the resource as ready when
	// Cloudflare can accept traffic for it on any port (in this case
	// 80/http). 443/https traffic would receive a certificate error
	// until cr.Status.AtProvider.SSL.Status returns ready as well.
	// If this is necessary, both statuses can be checked by using a
	// readinessCheck in a Composition.

	if cr.Status.AtProvider.Status == customHostnameStatusActive {
		cr.Status.SetConditions(rtv1.Available())
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: customhostnames.UpToDate(&cr.Spec.ForProvider, ch),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.CustomHostname)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCustomHostname)
	}

	// Zone is required to create a custom hostname on. SSL Method and Type
	// Are required by the API call, but we default them. This is simply
	// protection from panic if an unvalidated resource is created.
	if cr.Spec.ForProvider.Zone == nil || cr.Spec.ForProvider.SSL.Method == nil ||
		cr.Spec.ForProvider.SSL.Type == nil {
		return managed.ExternalCreation{}, errors.New(errCustomHostnameCreation)
	}

	rch, err := e.client.CreateCustomHostname(
		ctx,
		*cr.Spec.ForProvider.Zone,
		customhostnames.ParametersToCustomHostname(cr.Spec.ForProvider),
	)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCustomHostnameCreation)
	}

	cr.Status.AtProvider = customhostnames.GenerateObservation(rch.Result)
	meta.SetExternalName(cr, rch.Result.ID)

	return managed.ExternalCreation{ExternalNameAssigned: true}, nil

}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.CustomHostname)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCustomHostname)
	}

	if cr.Spec.ForProvider.Zone == nil || cr.Spec.ForProvider.SSL.Method == nil ||
		cr.Spec.ForProvider.SSL.Type == nil {
		return managed.ExternalUpdate{}, errors.New(errCustomHostnameUpdate)
	}

	chid := meta.GetExternalName(cr)

	// Update should never be called on a nonexistent resource
	if chid == "" {
		return managed.ExternalUpdate{}, errors.New(errCustomHostnameUpdate)
	}

	_, err := e.client.UpdateCustomHostname(
		ctx,
		*cr.Spec.ForProvider.Zone,
		chid,
		customhostnames.ParametersToCustomHostname(cr.Spec.ForProvider),
	)
	return managed.ExternalUpdate{},
		errors.Wrap(
			err,
			errCustomHostnameUpdate,
		)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.CustomHostname)
	if !ok {
		return errors.New(errNotCustomHostname)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return errors.New(errCustomHostnameDeletion)
	}

	chid := meta.GetExternalName(cr)

	// Delete should never be called on a nonexistent resource
	if chid == "" {
		return errors.New(errCustomHostnameDeletion)
	}

	return errors.Wrap(
		e.client.DeleteCustomHostname(ctx, *cr.Spec.ForProvider.Zone, chid),
		errCustomHostnameDeletion)
}
