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

package v1alpha1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	zone "github.com/benagricola/provider-cloudflare/apis/zone/v1alpha1"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reference"

	"github.com/pkg/errors"
)

// RuleBypassProduct identifies a product that will be
// bypassed when the bypass action is used.
// +kubebuilder:validation:Enum=zoneLockdown;uaBlock;bic;hot;securityLevel;rateLimit;waf
type RuleBypassProduct string

// RuleParameters are the configurable fields of a Rule.
type RuleParameters struct {
	// Action is the action to apply to a matching request.
	// +kubebuilder:validation:Enum=block;challenge;js_challenge;allow;log;bypass
	Action string `json:"action"`

	// BypassProducts lists the products by identifier that should be
	// bypassed when the bypass action is used.
	// +optional
	BypassProducts []RuleBypassProduct `json:"bypassProducts,omitempty"`

	// Description is a human readable description of this rule.
	// +kubebuilder:validation:MaxLength=500
	// +optional
	Description *string `json:"description,omitempty"`

	// Filter refers to a Filter ID that this rule uses to match
	// traffic.
	// +optional
	Filter *string `json:"filter,omitempty"`

	// FilterRef references the filter object this rule uses to match traffic.
	// +optional
	FilterRef *xpv1.Reference `json:"filterRef,omitempty"`

	// FilterSelector selects the filter object this rule uses to match traffic.
	// +optional
	FilterSelector *xpv1.Selector `json:"filterSelector,omitempty"`

	// Paused indicates if this rule is paused or not.
	// +optional
	Paused *bool `json:"paused,omitempty"`

	// NOTE(bagricola): Cloudflare's API documentation says this has a range of
	// 0 - 2147483647 - but in reality, you get an error trying to set it to 0 and
	// it seems you can set it HIGHER than 2147483647.
	// I'm going off their API documentation here, except setting the minimum to
	// 1 to avoid the 400 error that causes.

	// Priority is the priority of this Firewall Rule, that controls
	// processing order. Rules without a priority set will be sequenced
	// after rules with a priority set.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=2147483647
	// +optional
	Priority *int32 `json:"priority,omitempty"`

	// ZoneID this Firewall Rule is for.
	// +immutable
	// +optional
	Zone *string `json:"zone,omitempty"`

	// ZoneRef references the zone object this Firewall Rule is for.
	// +immutable
	// +optional
	ZoneRef *xpv1.Reference `json:"zoneRef,omitempty"`

	// ZoneSelector selects the zone object this Firewall Rule is for.
	// +immutable
	// +optional
	ZoneSelector *xpv1.Selector `json:"zoneSelector,omitempty"`
}

// RuleObservation are the observable fields of a Rule.
type RuleObservation struct{}

// A RuleSpec defines the desired state of a Rule.
type RuleSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RuleParameters `json:"forProvider"`
}

// A RuleStatus represents the observed state of a Rule.
type RuleStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RuleObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Rule is a firewall filter applied in a particular order to a Zone.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,cloudflare}
type Rule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuleSpec   `json:"spec"`
	Status RuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RuleList contains a list of Rule
type RuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Rule `json:"items"`
}

// ResolveReferences of this DNS Record
func (fr *Rule) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, fr)

	// Resolve spec.forProvider.zone
	rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(fr.Spec.ForProvider.Zone),
		Reference:    fr.Spec.ForProvider.ZoneRef,
		Selector:     fr.Spec.ForProvider.ZoneSelector,
		To:           reference.To{Managed: &zone.Zone{}, List: &zone.ZoneList{}},
		Extract:      reference.ExternalName(),
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.zone")
	}
	fr.Spec.ForProvider.Zone = reference.ToPtrValue(rsp.ResolvedValue)
	fr.Spec.ForProvider.ZoneRef = rsp.ResolvedReference

	// Resolve spec.forProvider.filter
	// NOTE(bagricola): It is _possible_ for poor implementation during usage
	// of this resource to resolve a Filter that is not on the Zone we resolved
	// above. We rely on the Cloudflare API returning an error here, in that it
	// should reject our creation attempt if the Filter ID we pass is not
	// valid on the Zone in question.
	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(fr.Spec.ForProvider.Filter),
		Reference:    fr.Spec.ForProvider.FilterRef,
		Selector:     fr.Spec.ForProvider.FilterSelector,
		To:           reference.To{Managed: &Filter{}, List: &FilterList{}},
		Extract:      reference.ExternalName(),
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.filter")
	}
	fr.Spec.ForProvider.Filter = reference.ToPtrValue(rsp.ResolvedValue)
	fr.Spec.ForProvider.FilterRef = rsp.ResolvedReference
	return nil
}
