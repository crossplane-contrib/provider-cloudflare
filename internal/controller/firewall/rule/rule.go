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

package rule

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

	"github.com/benagricola/provider-cloudflare/apis/firewall/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	rule "github.com/benagricola/provider-cloudflare/internal/clients/firewall/rule"
	metrics "github.com/benagricola/provider-cloudflare/internal/metrics"
)

const (
	errNotRule = "managed resource is not a Rule custom resource"

	errClientConfig = "error getting client config"

	errRuleLookup   = "cannot lookup firewall rule"
	errRuleCreation = "cannot create firewall rule"
	errRuleUpdate   = "cannot update firewall rule"
	errRuleDeletion = "cannot delete firewall rule"
	errNoZone       = "no zone found"
	errNoFilter     = "no filter found"

	maxConcurrency = 5
)

// Setup adds a controller that reconciles Rule managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.RuleGroupKind)

	o := controller.Options{
		RateLimiter:             ratelimiter.NewDefaultManagedRateLimiter(rl),
		MaxConcurrentReconciles: maxConcurrency,
	}

	hc := metrics.NewInstrumentedHTTPClient(name)
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.RuleGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube: mgr.GetClient(),
			newCloudflareClientFn: func(cfg clients.Config) (rule.Client, error) {
				return rule.NewClient(cfg, hc)
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
		For(&v1alpha1.Rule{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                  client.Client
	newCloudflareClientFn func(cfg clients.Config) (rule.Client, error)
}

// Connect produces a valid configuration for a Cloudflare API
// instance, and returns it as an external client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Rule)
	if !ok {
		return nil, errors.New(errNotRule)
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
	client rule.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Rule)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRule)
	}

	// Rule does not exist if we dont have an ID stored in external-name
	rid := meta.GetExternalName(cr)
	if rid == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalObservation{}, errors.New(errNoZone)
	}

	r, err := e.client.FirewallRule(ctx, *cr.Spec.ForProvider.Zone, rid)

	if err != nil {
		return managed.ExternalObservation{},
			errors.Wrap(resource.Ignore(rule.IsRuleNotFound, err), errRuleLookup)
	}

	cr.Status.AtProvider = rule.GenerateObservation(r)

	cr.Status.SetConditions(rtv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceLateInitialized: rule.LateInitialize(&cr.Spec.ForProvider, r),
		ResourceUpToDate:        rule.UpToDate(&cr.Spec.ForProvider, r),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Rule)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRule)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalCreation{}, errors.New(errNoZone)
	}

	if cr.Spec.ForProvider.Filter == nil {
		return managed.ExternalCreation{}, errors.New(errNoFilter)
	}

	nr, err := rule.CreateRule(ctx, e.client, &cr.Spec.ForProvider)

	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errRuleCreation)
	}

	cr.Status.AtProvider = rule.GenerateObservation(*nr)

	// Update the external name with the ID of the new Rule
	meta.SetExternalName(cr, nr.ID)

	return managed.ExternalCreation{ExternalNameAssigned: true}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Rule)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRule)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return managed.ExternalUpdate{}, errors.Wrap(errors.New(errNoZone), errRuleUpdate)
	}

	rid := meta.GetExternalName(cr)

	// Update should never be called on a nonexistent resource
	if rid == "" {
		return managed.ExternalUpdate{}, errors.New(errRuleUpdate)
	}

	return managed.ExternalUpdate{},
		errors.Wrap(
			rule.UpdateRule(ctx, e.client, meta.GetExternalName(cr), &cr.Spec.ForProvider),
			errRuleUpdate,
		)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Rule)
	if !ok {
		return errors.New(errNotRule)
	}

	if cr.Spec.ForProvider.Zone == nil {
		return errors.Wrap(errors.New(errNoZone), errRuleDeletion)
	}

	rid := meta.GetExternalName(cr)

	// Delete should never be called on a nonexistent resource
	if rid == "" {
		return errors.New(errRuleDeletion)
	}

	return errors.Wrap(
		e.client.DeleteFirewallRule(ctx, *cr.Spec.ForProvider.Zone, meta.GetExternalName(cr)),
		errRuleDeletion)
}
