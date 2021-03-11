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

package zones

import (
	"context"
	"strings"

	"github.com/google/go-cmp/cmp"

	"github.com/pkg/errors"

	"github.com/cloudflare/cloudflare-go"

	"github.com/benagricola/provider-cloudflare/apis/zone/v1alpha1"
)

const (
	errSetPaused   = "error setting (un)pause"
	errSetPlan     = "error setting plan"
	errSetVanityNS = "error setting vanity nameservers"

	// Hardcoded string in cloudflare-go library.
	// It is used to detect a 'not found' zone
	// lookup vs. a failed lookup.
	// REF: https://github.com/cloudflare/cloudflare-go/blob/1dd2d1fe7d044b42d0b64c2f79b9e730c701ab75/cloudflare.go#L162
	// DO NOT CHANGE THIS
	errZoneNotFound = "Zone could not be found"

	// String returned by Cloudflare API if making a Zone
	// request for a Zone ID that doesn't exist.
	// It is used to detect a 'not found' zone
	// lookup vs. a failed lookup.
	// DO NOT CHANGE THIS
	errZoneInvalidID = "Invalid zone identifier"
)

// IsZoneNotFound returns true if the passed error indicates
// a Zone was not found.
func IsZoneNotFound(err error) bool {
	errStr := err.Error()
	return errStr == errZoneNotFound || strings.Contains(errStr, errZoneInvalidID)
}

// LookupZoneByIDOrName looks up a Zone by ID, if supplied,
// looking up by Name if not.
func LookupZoneByIDOrName(ctx context.Context, api cloudflare.API, zoneID string, name string) (cloudflare.Zone, error) {
	var err error
	if zoneID == "" {
		zoneID, err = api.ZoneIDByName(name)
		if err != nil {
			return cloudflare.Zone{}, err
		}
	}

	return api.ZoneDetails(ctx, zoneID)
}

// GenerateObservation creates an observation of a cloudflare Zone
func GenerateObservation(in cloudflare.Zone) v1alpha1.ZoneObservation {
	return v1alpha1.ZoneObservation{
		AccountID:         in.Account.ID,
		AccountName:       in.Account.Name,
		ID:                in.ID,
		DevMode:           in.DevMode,
		OriginalNS:        in.OriginalNS,
		OriginalRegistrar: in.OriginalRegistrar,
		OriginalDNSHost:   in.OriginalDNSHost,
		NameServers:       in.NameServers,
		Paused:            in.Paused,
		Permissions:       in.Permissions,
		PlanID:            in.Plan.ID,
		Plan:              in.Plan.Name,
		PlanPendingID:     in.PlanPending.ID,
		PlanPending:       in.PlanPending.Name,
		Status:            in.Status,
		Betas:             in.Betas,
		DeactReason:       in.DeactReason,
		VerificationKey:   in.VerificationKey,
		VanityNameServers: in.VanityNS,
	}
}

// LateInitialize initializes ZoneParameters based on the remote resource
func LateInitialize(spec *v1alpha1.ZoneParameters, o v1alpha1.ZoneObservation) bool {
	if spec == nil {
		return false
	}

	li := false
	if spec.AccountID == nil {
		spec.AccountID = &o.AccountID
		li = true
	}
	if spec.Paused == nil {
		spec.Paused = &o.Paused
		li = true
	}
	if spec.PlanID == nil {
		spec.PlanID = &o.PlanID
		li = true
	}
	if spec.VanityNameServers == nil {
		spec.VanityNameServers = o.VanityNameServers
		li = true
	}

	return li
}

// UpToDate checks if the remote resource is up to date with the
// requested resource parameters.
func UpToDate(spec *v1alpha1.ZoneParameters, o v1alpha1.ZoneObservation) bool {
	if spec == nil {
		return false
	}

	// Check if mutable fields are up to date with resource
	if *spec.Paused != o.Paused {
		return false
	}
	// We only detect the resource as not up to date if the requested
	// plan is not the current plan or the pending plan.
	// Since it can take a month for the plan to change from pending
	// to active.
	if *spec.PlanID != o.PlanID && *spec.PlanID != o.PlanPendingID {
		return false
	}
	if !cmp.Equal(spec.VanityNameServers, o.VanityNameServers) {
		return false
	}
	return true
}

// UpdateZone updates mutable values on a Zone
func UpdateZone(ctx context.Context, api *cloudflare.API, spec *v1alpha1.ZoneParameters, o *v1alpha1.ZoneObservation) error {
	var zone cloudflare.Zone
	var err error

	if spec.Paused != nil && *spec.Paused != o.Paused {
		zone, err = api.ZoneSetPaused(ctx, o.ID, *spec.Paused)
		if err != nil {
			return errors.Wrap(err, errSetPaused)
		}
		o.Paused = zone.Paused
	}

	// ZoneSetPlan does not return a copy of the updated zone
	// So we can't update the Plan until the next reconcile.
	if spec.PlanID != nil && *spec.PlanID != o.PlanID &&
		spec.PlanID != &o.PlanPendingID {
		err = api.ZoneSetPlan(ctx, o.ID, *spec.PlanID)
		if err != nil {
			return errors.Wrap(err, errSetPlan)
		}
	}

	if !cmp.Equal(spec.VanityNameServers, o.VanityNameServers) {
		zone, err = api.ZoneSetVanityNS(ctx, o.ID, spec.VanityNameServers)
		if err != nil {
			return errors.Wrap(err, errSetVanityNS)
		}
		o.VanityNameServers = zone.VanityNS
	}
	return nil
}
