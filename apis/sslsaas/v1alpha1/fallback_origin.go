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

// FallbackOriginParameters represents the settings of a FallbackOrigin
type FallbackOriginParameters struct {
	// Origin for the Fallback Origin.
	Origin *string `json:"origin,omitempty"`

	// ZoneID this Fallback Origin is for.
	// +immutable
	// +optional
	Zone *string `json:"zone,omitempty"`

	// ZoneRef references the zone object this Fallback Origin is for.
	// +immutable
	// +optional
	ZoneRef *xpv1.Reference `json:"zoneRef,omitempty"`

	// ZoneSelector selects the zone object this Fallback Origin is for.
	// +immutable
	// +optional
	ZoneSelector *xpv1.Selector `json:"zoneSelector,omitempty"`
}

// FallbackOriginObservation are the observable fields of a Fallback Origin.
type FallbackOriginObservation struct {
	Status string   `json:"status,omitempty"`
	Errors []string `json:"errors,omitempty"`
}

// A FallbackOriginSpec defines the desired state of a Fallback Origin.
type FallbackOriginSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       FallbackOriginParameters `json:"forProvider"`
}

// A FallbackOriginStatus represents the observed state of a Fallback Origin.
type FallbackOriginStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          FallbackOriginObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A FallbackOrigin is a fallback origin required to use SSL for SaaS.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.bindingPhase"
type FallbackOrigin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FallbackOriginSpec   `json:"spec"`
	Status FallbackOriginStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FallbackOriginList contains a list of FallbackOrigin
type FallbackOriginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FallbackOrigin `json:"items"`
}

// ResolveReferences of this Fallback Origin
func (dr *FallbackOrigin) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, dr)

	// Resolve spec.forProvider.zone
	rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(dr.Spec.ForProvider.Zone),
		Reference:    dr.Spec.ForProvider.ZoneRef,
		Selector:     dr.Spec.ForProvider.ZoneSelector,
		To:           reference.To{Managed: &v1alpha1.Zone{}, List: &v1alpha1.ZoneList{}},
		Extract:      reference.ExternalName(),
	})
	if err != nil {
		return errors.Wrap(err, "spec.forProvider.zone")
	}
	dr.Spec.ForProvider.Zone = reference.ToPtrValue(rsp.ResolvedValue)
	dr.Spec.ForProvider.ZoneRef = rsp.ResolvedReference

	return nil
}
