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

package record

import (
	"context"

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

	"github.com/benagricola/provider-cloudflare/apis/dns/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	records "github.com/benagricola/provider-cloudflare/internal/clients/records"
)

const (
	errNotDNSRecord = "managed resource is not a DNSRecord custom resource"

	errClientConfig = "error getting client config"

	errDNSRecordLookup   = "cannot lookup record"
	errDNSRecordCreation = "cannot create record"
	errDNSRecordUpdate   = "cannot update record"
	errDNSRecordDeletion = "cannot delete record"
	errDNSRecordNoZone   = "cannot create record no zone found"

	maxConcurrency = 5

	// recordStatusActive = "active"
)

// Setup adds a controller that reconciles DNSRecord managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.DNSRecordGroupKind)

	o := controller.Options{
		RateLimiter:             ratelimiter.NewDefaultManagedRateLimiter(rl),
		MaxConcurrentReconciles: maxConcurrency,
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.DNSRecordGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:                  mgr.GetClient(),
			newCloudflareClientFn: records.NewClient}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.DNSRecord{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                  client.Client
	newCloudflareClientFn func(cfg clients.Config) records.Client
}

// Connect produces a valid configuration for a Cloudflare API
// instance, and returns it as an external client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.DNSRecord)
	if !ok {
		return nil, errors.New(errNotDNSRecord)
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
	client records.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.DNSRecord)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDNSRecord)
	}

	en := meta.GetExternalName(cr)

	// If the name & ExternalName are the same we know we haven't set the ExternalName
	// So attempt to create
	if en == cr.ObjectMeta.Name {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalObservation{}, errors.New(errDNSRecordNoZone)
	}

	record, err := e.client.DNSRecord(ctx, *cr.Spec.ForProvider.Zone, en)

	if err != nil {
		// Been deleted or doesnt exist
		if records.IsRecordNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errDNSRecordLookup)
	}

	cr.Status.AtProvider = records.GenerateObservation(record)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        records.UpToDate(&cr.Spec.ForProvider, record),
		ResourceLateInitialized: records.LateInitialize(&cr.Spec.ForProvider, record),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.DNSRecord)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDNSRecord)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalCreation{}, errors.New(errDNSRecordCreation)
	}

	if cr.Spec.ForProvider.TTL == nil {
		return managed.ExternalCreation{}, errors.New(errDNSRecordCreation)
	}

	// TODO: Add validation here for priority (only required for specific record types)

	cr.SetConditions(rtv1.Creating())

	res, err := e.client.CreateDNSRecord(
		ctx,
		*cr.Spec.ForProvider.Zone,
		cloudflare.DNSRecord{
			Type:     *cr.Spec.ForProvider.Type,
			Name:     cr.Spec.ForProvider.Name,
			TTL:      *cr.Spec.ForProvider.TTL,
			Content:  cr.Spec.ForProvider.Content,
			Proxied:  cr.Spec.ForProvider.Proxied,
			Priority: cr.Spec.ForProvider.Priority,
		},
	)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errDNSRecordCreation)
	}

	cr.Status.AtProvider = records.GenerateObservation(res.Result)

	// Update the external name with the ID of the new Zone
	meta.SetExternalName(cr, res.Result.ID)

	return managed.ExternalCreation{ExternalNameAssigned: true}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.DNSRecord)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotDNSRecord)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalUpdate{}, errors.New(errDNSRecordDeletion)
	}

	return managed.ExternalUpdate{},
		errors.Wrap(
			records.UpdateRecord(ctx, e.client, meta.GetExternalName(cr), &cr.Spec.ForProvider),
			errDNSRecordUpdate,
		)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.DNSRecord)
	if !ok {
		return errors.New(errNotDNSRecord)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return errors.New(errDNSRecordDeletion)
	}

	err := e.client.DeleteDNSRecord(ctx, *cr.Spec.ForProvider.Zone, meta.GetExternalName(cr))

	return errors.Wrap(err, errDNSRecordDeletion)
}
