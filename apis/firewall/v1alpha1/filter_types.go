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

	"github.com/benagricola/provider-cloudflare/apis/zone/v1alpha1"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reference"

	"github.com/pkg/errors"
)

// FilterParameters are the configurable fields of a Filter.
type FilterParameters struct {
	// Expression is the filter expression used to match traffic.
	Expression string `json:"expression"`

	// Description is a human readable description of this rule.
	// +kubebuilder:validation:MaxLength=500
	// +optional
	Description *string `json:"description,omitempty"`

	// Paused indicates if this rule is paused or not.
	// +optional
	Paused *bool `json:"paused,omitempty"`

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

// FilterObservation is the observable fields of a Filter.
type FilterObservation struct{}

// A FilterSpec defines the desired state of a Filter.
type FilterSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       FilterParameters `json:"forProvider"`
}

// A FilterStatus represents the observed state of a Filter.
type FilterStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          FilterObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Filter is a matching expression that can be referenced by one or more
// firewall rules.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,cloudflare}
type Filter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FilterSpec   `json:"spec"`
	Status FilterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FilterList contains a list of Filter
type FilterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Filter `json:"items"`
}

// ResolveReferences of this Filter
func (f *Filter) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, f)

	// Resolve spec.forProvider.zone
	rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(f.Spec.ForProvider.Zone),
		Reference:    f.Spec.ForProvider.ZoneRef,
		Selector:     f.Spec.ForProvider.ZoneSelector,
		To:           reference.To{Managed: &v1alpha1.Zone{}, List: &v1alpha1.ZoneList{}},
		Extract:      reference.ExternalName(),
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.zone")
	}
	f.Spec.ForProvider.Zone = reference.ToPtrValue(rsp.ResolvedValue)
	f.Spec.ForProvider.ZoneRef = rsp.ResolvedReference
	return nil
}
