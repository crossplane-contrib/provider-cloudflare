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

	"github.com/benagricola/provider-cloudflare/apis/dns/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	records "github.com/benagricola/provider-cloudflare/internal/clients/records"
)

const (
	errNotRecord = "managed resource is not a Record custom resource"

	errClientConfig = "error getting client config"

	errRecordLookup   = "cannot lookup record"
	errRecordCreation = "cannot create record"
	errRecordUpdate   = "cannot update record"
	errRecordDeletion = "cannot delete record"
	errRecordNoZone   = "no zone found"

	maxConcurrency = 5

	// recordStatusActive = "active"
)

// Setup adds a controller that reconciles Record managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.RecordGroupKind)

	o := controller.Options{
		RateLimiter:             ratelimiter.NewDefaultManagedRateLimiter(rl),
		MaxConcurrentReconciles: maxConcurrency,
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RecordGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:                  mgr.GetClient(),
			newCloudflareClientFn: records.NewClient}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithPollInterval(5*time.Minute),
		// Do not initialize external-name field.
		managed.WithInitializers(),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.Record{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                  client.Client
	newCloudflareClientFn func(cfg clients.Config) (records.Client, error)
}

// Connect produces a valid configuration for a Cloudflare API
// instance, and returns it as an external client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Record)
	if !ok {
		return nil, errors.New(errNotRecord)
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
	client records.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Record)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRecord)
	}

	// Record does not exist if we dont have an ID stored in external-name
	rid := meta.GetExternalName(cr)
	if rid == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalObservation{}, errors.New(errRecordNoZone)
	}

	record, err := e.client.DNSRecord(ctx, *cr.Spec.ForProvider.Zone, rid)

	if err != nil {
		return managed.ExternalObservation{},
			errors.Wrap(resource.Ignore(records.IsRecordNotFound, err), errRecordLookup)
	}

	cr.Status.AtProvider = records.GenerateObservation(record)

	cr.SetConditions(rtv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceLateInitialized: records.LateInitialize(&cr.Spec.ForProvider, record),
		ResourceUpToDate:        records.UpToDate(&cr.Spec.ForProvider, record),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Record)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRecord)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalCreation{},
			errors.Wrap(errors.New(errRecordNoZone), errRecordCreation)
	}

	if cr.Spec.ForProvider.TTL == nil {
		return managed.ExternalCreation{}, errors.New(errRecordCreation)
	}

	if cr.Spec.ForProvider.Type == nil {
		return managed.ExternalCreation{}, errors.New(errRecordCreation)
	}

	// TODO: Add validation here for priority (only required for specific record types)

	cr.SetConditions(rtv1.Creating())

	ttl := int(*cr.Spec.ForProvider.TTL)

	res, err := e.client.CreateDNSRecord(
		ctx,
		*cr.Spec.ForProvider.Zone,
		cloudflare.DNSRecord{
			Type:     *cr.Spec.ForProvider.Type,
			Name:     cr.Spec.ForProvider.Name,
			TTL:      ttl,
			Content:  cr.Spec.ForProvider.Content,
			Proxied:  cr.Spec.ForProvider.Proxied,
			Priority: cr.Spec.ForProvider.Priority,
		},
	)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errRecordCreation)
	}

	cr.Status.AtProvider = records.GenerateObservation(res.Result)

	// Update the external name with the ID of the new DNS Record
	meta.SetExternalName(cr, res.Result.ID)

	return managed.ExternalCreation{ExternalNameAssigned: true}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Record)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRecord)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalUpdate{}, errors.Wrap(errors.New(errRecordNoZone), errRecordUpdate)
	}

	return managed.ExternalUpdate{},
		errors.Wrap(
			records.UpdateRecord(ctx, e.client, meta.GetExternalName(cr), &cr.Spec.ForProvider),
			errRecordUpdate,
		)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Record)
	if !ok {
		return errors.New(errNotRecord)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return errors.Wrap(errors.New(errRecordNoZone), errRecordDeletion)
	}

	return errors.Wrap(
		e.client.DeleteDNSRecord(ctx, *cr.Spec.ForProvider.Zone, meta.GetExternalName(cr)),
		errRecordDeletion)
}
