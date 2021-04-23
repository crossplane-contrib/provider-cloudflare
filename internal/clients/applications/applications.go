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

package applications

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/benagricola/provider-cloudflare/apis/spectrum/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	"github.com/cloudflare/cloudflare-go"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Cloudflare returns this code when a application isnt found.
	errApplicationNotFound = "10006"

	// Returned when an invalid IP is supplied within spec
	errApplicationInvalidIP = "invalid IP within Edge IPs"
)

// Client is a Cloudflare API client that implements methods for working
// with Spectrum Applications.
type Client interface {
	CreateSpectrumApplication(ctx context.Context, zoneID string, appDetails cloudflare.SpectrumApplication) (cloudflare.SpectrumApplication, error)
	SpectrumApplication(ctx context.Context, zoneID string, applicationID string) (cloudflare.SpectrumApplication, error)
	UpdateSpectrumApplication(ctx context.Context, zoneID, appID string, appDetails cloudflare.SpectrumApplication) (cloudflare.SpectrumApplication, error)
	DeleteSpectrumApplication(ctx context.Context, zoneID string, applicationID string) error
}

// NewClient returns a new Cloudflare API client for working with Spectrum Applications.
func NewClient(cfg clients.Config) (Client, error) {
	return clients.NewClient(cfg)
}

// IsApplicationNotFound returns true if the passed error indicates
// a spectrum application was not found.
func IsApplicationNotFound(err error) bool {
	return strings.Contains(err.Error(), errApplicationNotFound)
}

// ConvertIPs converts slice of IPs in string form
// into net.IP for ease of use in YAML
// returns nil, error if any of the IPs are invalid
func ConvertIPs(ips []string) ([]net.IP, error) {
	rips := []net.IP{}
	for _, ip := range ips {
		cip := net.ParseIP(ip)
		if cip == nil {
			return nil, errors.New(errApplicationInvalidIP)
		}
		rips = append(rips, cip)
	}
	return rips, nil
}

// edgeIPsDontMatch returns true if the spec and observed IPs do not match
// returns false if the spec IPs do match
func edgeIPsDontMatch(spec []string, o []net.IP) bool {
	a := make(map[string]struct{})
	for _, ip := range spec {
		a[ip] = struct{}{}
	}

	b := make(map[string]struct{})
	for _, ip := range o {
		b[ip.String()] = struct{}{}
	}

	return !cmp.Equal(a, b)
}

// GenerateObservation creates an observation of a cloudflare Spectrum Application.
func GenerateObservation(in cloudflare.SpectrumApplication) v1alpha1.ApplicationObservation {
	o := v1alpha1.ApplicationObservation{}
	if in.CreatedOn != nil {
		o.CreatedOn = &metav1.Time{Time: *in.CreatedOn}
	}

	if in.ModifiedOn != nil {
		o.ModifiedOn = &metav1.Time{Time: *in.ModifiedOn}
	}

	return o
}

// UpToDate checks if the remote Application is up to date with the
// requested resource parameters.
func UpToDate(spec *v1alpha1.ApplicationParameters, o cloudflare.SpectrumApplication) bool { //nolint:gocyclo
	// NOTE(bagricola): The complexity here is simply repeated
	// if statements checking for updated fields. You should think
	// before adding further complexity to this method, but adding
	// more field checks should not be an issue.
	if spec == nil {
		return true
	}

	if spec.DNS.Type != o.DNS.Type {
		return false
	}

	if spec.DNS.Name != o.DNS.Name {
		return false
	}

	if spec.OriginPort == nil && o.OriginPort != nil {
		return false
	}

	if spec.OriginPort != nil {
		if spec.OriginPort.Port != nil && uint16(*spec.OriginPort.Port) != o.OriginPort.Port {
			return false
		}

		if spec.OriginPort.Start != nil && uint16(*spec.OriginPort.Start) != o.OriginPort.Start {
			return false
		}

		if spec.OriginPort.End != nil && uint16(*spec.OriginPort.End) != o.OriginPort.End {
			return false
		}
	}

	if spec.OriginDNS == nil && o.OriginDNS != nil {
		return false
	}

	if spec.OriginDNS != nil && spec.OriginDNS.Name != o.OriginDNS.Name {
		return false
	}

	if spec.EdgeIPs == nil && o.EdgeIPs != nil {
		return false
	}

	if spec.EdgeIPs != nil {
		if o.EdgeIPs.Type != cloudflare.SpectrumApplicationEdgeType(spec.EdgeIPs.Type) {
			return false
		}

		if spec.EdgeIPs.Connectivity != nil && o.EdgeIPs.Connectivity.String() != *spec.EdgeIPs.Connectivity {
			return false
		}

		if spec.EdgeIPs.IPs != nil && edgeIPsDontMatch(spec.EdgeIPs.IPs, o.EdgeIPs.IPs) {
			return false
		}
	}

	if spec.ProxyProtocol != nil && o.ProxyProtocol != cloudflare.ProxyProtocol(*spec.ProxyProtocol) {
		return false
	}

	if !cmp.Equal(spec.OriginDirect, o.OriginDirect) {
		return false
	}

	if spec.Protocol != o.Protocol {
		return false
	}

	if spec.IPv4 != nil && *spec.IPv4 != o.IPv4 {
		return false
	}

	if spec.IPFirewall != nil && *spec.IPFirewall != o.IPFirewall {
		return false
	}

	if spec.TLS != nil && *spec.TLS != o.TLS {
		return false
	}

	if spec.TrafficType != nil && *spec.TrafficType != o.TrafficType {
		return false
	}

	if spec.ArgoSmartRouting != nil && *spec.ArgoSmartRouting != o.ArgoSmartRouting {
		return false
	}

	return true
}

// UpdateSpectrumApplication updates mutable values on a Spectrum Application.
func UpdateSpectrumApplication(ctx context.Context, client Client, applicationID string, spec *v1alpha1.ApplicationParameters) error { //nolint:gocyclo

	dns := cloudflare.SpectrumApplicationDNS{
		Type: spec.DNS.Type,
		Name: spec.DNS.Name,
	}

	oport := cloudflare.SpectrumApplicationOriginPort{}
	if spec.OriginPort != nil {
		if spec.OriginPort.Port != nil {
			oport.Port = uint16(*spec.OriginPort.Port)
		}

		if spec.OriginPort.Start != nil {
			oport.Start = uint16(*spec.OriginPort.Start)
		}

		if spec.OriginPort.End != nil {
			oport.End = uint16(*spec.OriginPort.End)
		}
	}

	odns := cloudflare.SpectrumApplicationOriginDNS{}
	if spec.OriginDNS != nil {
		odns.Name = spec.OriginDNS.Name
	}

	eips := cloudflare.SpectrumApplicationEdgeIPs{}
	if spec.EdgeIPs != nil {
		eips.Type = cloudflare.SpectrumApplicationEdgeType(spec.EdgeIPs.Type)

		if spec.EdgeIPs.Connectivity != nil {
			eips.Connectivity = (*cloudflare.SpectrumApplicationConnectivity)(spec.EdgeIPs.Connectivity)
		}

		if spec.EdgeIPs.IPs != nil {
			ips, iperr := ConvertIPs(spec.EdgeIPs.IPs)
			if iperr != nil {
				return iperr
			}
			eips.IPs = ips
		}
	}

	ap := cloudflare.SpectrumApplication{
		Protocol:     spec.Protocol,
		DNS:          dns,
		OriginDirect: spec.OriginDirect,
		OriginPort:   &oport,
		OriginDNS:    &odns,
		EdgeIPs:      &eips,
	}

	if spec.ProxyProtocol != nil {
		ap.ProxyProtocol = cloudflare.ProxyProtocol(*spec.ProxyProtocol)
	}

	if spec.IPv4 != nil {
		ap.IPv4 = *spec.IPv4
	}

	if spec.IPFirewall != nil {
		ap.IPFirewall = *spec.IPFirewall
	}

	if spec.TLS != nil {
		ap.TLS = *spec.TLS
	}

	if spec.TrafficType != nil {
		ap.TrafficType = *spec.TrafficType
	}

	if spec.ArgoSmartRouting != nil {
		ap.ArgoSmartRouting = *spec.ArgoSmartRouting
	}

	_, err := client.UpdateSpectrumApplication(ctx, *spec.Zone, applicationID, ap)

	return err

}
