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

package records

import (
	"context"
	"strings"

	"github.com/benagricola/provider-cloudflare/apis/dns/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	"github.com/cloudflare/cloudflare-go"
)

const (
	// Cloudflare returns this code when a record isnt found
	errRecordNotFound = "81044"

	errRecordUpdate = "cannot update record"
)

// Client is a Cloudflare API client that implements methods for working
// with DNS Records.
type Client interface {
	CreateDNSRecord(ctx context.Context, zoneID string, rr cloudflare.DNSRecord) (*cloudflare.DNSRecordResponse, error)
	UpdateDNSRecord(ctx context.Context, zoneID, recordID string, rr cloudflare.DNSRecord) error
	DNSRecord(ctx context.Context, zoneID, recordID string) (cloudflare.DNSRecord, error)
	DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error
}

// NewClient returns a new Cloudflare API client for working with DNS Records.
func NewClient(cfg clients.Config) Client {
	return clients.NewClient(cfg)
}

// IsRecordNotFound returns true if the passed error indicates
// a Record was not found.
func IsRecordNotFound(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, errRecordNotFound)
}

// GenerateObservation creates an observation of a cloudflare Record
func GenerateObservation(in cloudflare.DNSRecord) v1alpha1.DNSRecordObservation {
	return v1alpha1.DNSRecordObservation{
		Proxiable:  in.Proxiable,
		Zone:       in.ZoneName,
		Locked:     in.Locked,
		CreatedOn:  in.CreatedOn.String(),
		ModifiedOn: in.ModifiedOn.String(),
	}
}

// LateInitialize initializes RecordParameters based on the remote resource
func LateInitialize(spec *v1alpha1.DNSRecordParameters, o cloudflare.DNSRecord) bool {
	if spec == nil {
		return false
	}

	li := false
	if spec.Proxied == nil {
		spec.Proxied = o.Proxied
		li = true
	}

	if spec.Priority == nil {
		spec.Priority = o.Priority
		li = true
	}

	return li
}

// UpToDate checks if the remote resource is up to date with the
// requested resource parameters.
func UpToDate(spec *v1alpha1.DNSRecordParameters, o cloudflare.DNSRecord) bool {
	if spec == nil {
		return false
	}

	// Check if mutable fields are up to date with resource

	// If the Spec Name doesn't have the zone name on the end of it
	// Add it on the end when checking the result from the API
	// As CF returns the name as the full DNS record (including zone name)
	fn := spec.Name
	if !strings.HasSuffix(fn, o.ZoneName) {
		fn = fn + "." + o.ZoneName
	}

	if fn != o.Name {
		return false
	}

	if spec.Content != o.Content {
		return false
	}

	if spec.TTL != nil && *spec.TTL != o.TTL {
		return false
	}

	if spec.Proxied != nil && spec.Proxied != o.Proxied {
		return false
	}

	if spec.Priority != nil && spec.Priority != o.Priority {
		return false
	}

	return true
}

// UpdateRecord updates mutable values on a Record
func UpdateRecord(ctx context.Context, client Client, recordID string, spec *v1alpha1.DNSRecordParameters) error {

	rr := cloudflare.DNSRecord{
		Type:     *spec.Type,
		Name:     spec.Name,
		TTL:      *spec.TTL,
		Content:  spec.Content,
		Proxied:  spec.Proxied,
		Priority: spec.Priority,
	}

	return client.UpdateDNSRecord(ctx, *spec.Zone, recordID, rr)

}
