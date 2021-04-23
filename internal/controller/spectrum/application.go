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

package application

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

	"github.com/cloudflare/cloudflare-go"

	"github.com/benagricola/provider-cloudflare/apis/spectrum/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	applications "github.com/benagricola/provider-cloudflare/internal/clients/applications"
)

const (
	errNotApplication = "managed resource is not a Application custom resource"

	errClientConfig = "error getting client config"

	errApplicationLookup   = "cannot lookup application"
	errApplicationCreation = "cannot create application"
	errApplicationUpdate   = "cannot update application"
	errApplicationDeletion = "cannot delete application"
	errApplicationNoZone   = "no zone found"

	maxConcurrency = 5
)

// Setup adds a controller that reconciles Spectrum managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.ApplicationGroupKind)

	o := controller.Options{
		RateLimiter:             ratelimiter.NewDefaultManagedRateLimiter(rl),
		MaxConcurrentReconciles: maxConcurrency,
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ApplicationGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:                  mgr.GetClient(),
			newCloudflareClientFn: applications.NewClient}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithPollInterval(5*time.Minute),
		// Do not initialize external-name field.
		managed.WithInitializers(),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.Application{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                  client.Client
	newCloudflareClientFn func(cfg clients.Config) (applications.Client, error)
}

// Connect produces a valid configuration for a Cloudflare API
// instance, and returns it as an external client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Application)
	if !ok {
		return nil, errors.New(errNotApplication)
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
	client applications.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Application)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotApplication)
	}

	// Application does not exist if we dont have an ID stored in external-name
	aid := meta.GetExternalName(cr)
	if aid == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalObservation{}, errors.New(errApplicationNoZone)
	}

	application, err := e.client.SpectrumApplication(ctx, *cr.Spec.ForProvider.Zone, aid)

	if err != nil {
		if applications.IsApplicationNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errApplicationLookup)
	}

	cr.Status.AtProvider = applications.GenerateObservation(application)

	cr.SetConditions(rtv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: applications.UpToDate(&cr.Spec.ForProvider, application),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { //nolint:gocyclo
	cr, ok := mg.(*v1alpha1.Application)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotApplication)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalCreation{},
			errors.Wrap(errors.New(errApplicationNoZone), errApplicationCreation)
	}

	cr.SetConditions(rtv1.Creating())

	dns := cloudflare.SpectrumApplicationDNS{
		Type: cr.Spec.ForProvider.DNS.Type,
		Name: cr.Spec.ForProvider.DNS.Name,
	}

	oport := cloudflare.SpectrumApplicationOriginPort{}
	if cr.Spec.ForProvider.OriginPort != nil {
		if cr.Spec.ForProvider.OriginPort.Port != nil {
			oport.Port = uint16(*cr.Spec.ForProvider.OriginPort.Port)
		}

		if cr.Spec.ForProvider.OriginPort.Start != nil {
			oport.Start = uint16(*cr.Spec.ForProvider.OriginPort.Start)
		}

		if cr.Spec.ForProvider.OriginPort.End != nil {
			oport.End = uint16(*cr.Spec.ForProvider.OriginPort.End)
		}
	}

	odns := cloudflare.SpectrumApplicationOriginDNS{}
	if cr.Spec.ForProvider.OriginDNS != nil {
		odns.Name = cr.Spec.ForProvider.OriginDNS.Name
	}

	eips := cloudflare.SpectrumApplicationEdgeIPs{}
	if cr.Spec.ForProvider.EdgeIPs != nil {
		eips.Type = cloudflare.SpectrumApplicationEdgeType(cr.Spec.ForProvider.EdgeIPs.Type)

		if cr.Spec.ForProvider.EdgeIPs.Connectivity != nil {
			eips.Connectivity = (*cloudflare.SpectrumApplicationConnectivity)(cr.Spec.ForProvider.EdgeIPs.Connectivity)
		}

		if cr.Spec.ForProvider.EdgeIPs.IPs != nil {
			ips, iperr := applications.ConvertIPs(cr.Spec.ForProvider.EdgeIPs.IPs)
			if iperr != nil {
				return managed.ExternalCreation{}, errors.Wrap(iperr, errApplicationCreation)
			}
			eips.IPs = ips
		}
	}

	ap := cloudflare.SpectrumApplication{
		Protocol:     cr.Spec.ForProvider.Protocol,
		DNS:          dns,
		OriginDirect: cr.Spec.ForProvider.OriginDirect,
		OriginPort:   &oport,
		OriginDNS:    &odns,
		EdgeIPs:      &eips,
	}

	if cr.Spec.ForProvider.ProxyProtocol != nil {
		ap.ProxyProtocol = cloudflare.ProxyProtocol(*cr.Spec.ForProvider.ProxyProtocol)
	}

	if cr.Spec.ForProvider.IPv4 != nil {
		ap.IPv4 = *cr.Spec.ForProvider.IPv4
	}

	if cr.Spec.ForProvider.IPFirewall != nil {
		ap.IPFirewall = *cr.Spec.ForProvider.IPFirewall
	}

	if cr.Spec.ForProvider.TLS != nil {
		ap.TLS = *cr.Spec.ForProvider.TLS
	}

	if cr.Spec.ForProvider.TrafficType != nil {
		ap.TrafficType = *cr.Spec.ForProvider.TrafficType
	}

	if cr.Spec.ForProvider.ArgoSmartRouting != nil {
		ap.ArgoSmartRouting = *cr.Spec.ForProvider.ArgoSmartRouting
	}

	res, err := e.client.CreateSpectrumApplication(
		ctx,
		*cr.Spec.ForProvider.Zone,
		ap,
	)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errApplicationCreation)
	}

	cr.Status.AtProvider = applications.GenerateObservation(res)

	// Update the external name with the ID of the new Spectrum Application
	meta.SetExternalName(cr, res.ID)

	return managed.ExternalCreation{ExternalNameAssigned: true}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Application)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotApplication)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalUpdate{}, errors.Wrap(errors.New(errApplicationNoZone), errApplicationUpdate)
	}

	aid := meta.GetExternalName(cr)

	// Update should never be called on a nonexistent resource
	if aid == "" {
		return managed.ExternalUpdate{}, errors.New(errApplicationUpdate)
	}

	return managed.ExternalUpdate{},
		errors.Wrap(
			applications.UpdateSpectrumApplication(ctx, e.client, meta.GetExternalName(cr), &cr.Spec.ForProvider),
			errApplicationUpdate,
		)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Application)
	if !ok {
		return errors.New(errNotApplication)
	}

	aid := meta.GetExternalName(cr)

	// Update should never be called on a nonexistent resource
	if aid == "" {
		return errors.New(errApplicationDeletion)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return errors.Wrap(errors.New(errApplicationNoZone), errApplicationDeletion)
	}

	return errors.Wrap(
		e.client.DeleteSpectrumApplication(ctx, *cr.Spec.ForProvider.Zone, meta.GetExternalName(cr)),
		errApplicationDeletion)
}
