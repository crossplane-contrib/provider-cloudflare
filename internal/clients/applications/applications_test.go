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
	"net"
	"testing"

	"github.com/cloudflare/cloudflare-go"

	"github.com/benagricola/provider-cloudflare/apis/spectrum/v1alpha1"
	"github.com/google/go-cmp/cmp"

	ptr "k8s.io/utils/pointer"
)

func TestUpToDate(t *testing.T) {

	port := uint32(2022)
	start := uint32(2020)
	end := uint32(2024)
	connectivityAll := cloudflare.SpectrumConnectivityAll

	type args struct {
		rp *v1alpha1.ApplicationParameters
		r  cloudflare.SpectrumApplication
	}

	type want struct {
		o bool
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"UpToDateSpecNil": {
			reason: "UpToDate should return true when not passed a spec",
			args:   args{},
			want: want{
				o: true,
			},
		},
		"UpToDateEmptyParams": {
			reason: "UpToDate should return true and not panic with nil values",
			args: args{
				rp: &v1alpha1.ApplicationParameters{},
				r:  cloudflare.SpectrumApplication{},
			},
			want: want{
				o: true,
			},
		},
		"UpToDateEmptyEdgeIps": {
			reason: "UpToDate should return false and not panic if we supply NIL EdgeIPs but the observed settings still has them",
			args: args{
				rp: &v1alpha1.ApplicationParameters{},
				r: cloudflare.SpectrumApplication{
					OriginPort: &cloudflare.SpectrumApplicationOriginPort{
						Port: 8000,
					},
					EdgeIPs: &cloudflare.SpectrumApplicationEdgeIPs{
						Type: cloudflare.SpectrumEdgeTypeStatic,
					},
				},
			},
			want: want{
				o: false,
			},
		},
		"UpToDateEmptyOriginPort": {
			reason: "UpToDate should return false and not panic if we supply NIL OriginPort but the observed settings still has them",
			args: args{
				rp: &v1alpha1.ApplicationParameters{},
				r: cloudflare.SpectrumApplication{
					OriginPort: &cloudflare.SpectrumApplicationOriginPort{
						Port: 8000,
					},
				},
			},
			want: want{
				o: false,
			},
		},
		"UpToDateDifferentEdgeIPs": {
			reason: "UpToDate should return false and not panic when EdgeIPs IPs do not match",
			args: args{
				rp: &v1alpha1.ApplicationParameters{
					EdgeIPs: &v1alpha1.SpectrumApplicationEdgeIPs{
						IPs: []string{"192.0.2.2", "2001:db8::1"},
					},
				},
				r: cloudflare.SpectrumApplication{
					EdgeIPs: &cloudflare.SpectrumApplicationEdgeIPs{
						IPs: []net.IP{net.ParseIP("192.0.2.1"), net.ParseIP("2001:db8::1")},
					},
				},
			},
			want: want{
				o: false,
			},
		},
		"UpToDateDifferentOrderEdgeIPs": {
			reason: "UpToDate should return true and not panic when EdgeIPs IPs match but in different order",
			args: args{
				rp: &v1alpha1.ApplicationParameters{
					EdgeIPs: &v1alpha1.SpectrumApplicationEdgeIPs{
						IPs: []string{"2001:db8::1", "192.0.2.1"},
					},
				},
				r: cloudflare.SpectrumApplication{
					EdgeIPs: &cloudflare.SpectrumApplicationEdgeIPs{
						IPs: []net.IP{net.ParseIP("192.0.2.1"), net.ParseIP("2001:db8::1")},
					},
				},
			},
			want: want{
				o: true,
			},
		},
		"SuccessSpectrumDNS": {
			reason: "UpToDate should return true and not panic with a Application with Spectrum DNS",
			args: args{
				rp: &v1alpha1.ApplicationParameters{
					IPv4: ptr.BoolPtr(true),
					DNS: v1alpha1.SpectrumApplicationDNS{
						Type: "CNAME",
						Name: "spectrum.foo.com",
					},
					OriginDNS: &v1alpha1.SpectrumApplicationOriginDNS{
						Name: "spectrum.origin.foo.com",
					},
					OriginPort: &v1alpha1.SpectrumApplicationOriginPort{
						Port: &port,
					},
					IPFirewall:    ptr.BoolPtr(true),
					ProxyProtocol: ptr.StringPtr("off"),
					TLS:           ptr.StringPtr("full"),
				},
				r: cloudflare.SpectrumApplication{
					IPv4: true,
					DNS: cloudflare.SpectrumApplicationDNS{
						Type: "CNAME",
						Name: "spectrum.foo.com",
					},
					OriginDNS: &cloudflare.SpectrumApplicationOriginDNS{
						Name: "spectrum.origin.foo.com",
					},
					OriginPort: &cloudflare.SpectrumApplicationOriginPort{
						Port: 2022,
					},
					IPFirewall:    true,
					ProxyProtocol: "off",
					TLS:           "full",
				},
			},
			want: want{
				o: true,
			},
		},
		"SuccessSpectrumDNSPortRange": {
			reason: "UpToDate should return true and not panic with a Application with Spectrum DNS with port range",
			args: args{
				rp: &v1alpha1.ApplicationParameters{
					IPv4: ptr.BoolPtr(true),
					DNS: v1alpha1.SpectrumApplicationDNS{
						Type: "CNAME",
						Name: "spectrum.foo.com",
					},
					OriginDNS: &v1alpha1.SpectrumApplicationOriginDNS{
						Name: "spectrum.origin.foo.com",
					},
					OriginPort: &v1alpha1.SpectrumApplicationOriginPort{
						Start: &start,
						End:   &end,
					},
					IPFirewall:    ptr.BoolPtr(true),
					ProxyProtocol: ptr.StringPtr("off"),
					TLS:           ptr.StringPtr("full"),
				},
				r: cloudflare.SpectrumApplication{
					IPv4: true,
					DNS: cloudflare.SpectrumApplicationDNS{
						Type: "CNAME",
						Name: "spectrum.foo.com",
					},
					OriginDNS: &cloudflare.SpectrumApplicationOriginDNS{
						Name: "spectrum.origin.foo.com",
					},
					OriginPort: &cloudflare.SpectrumApplicationOriginPort{
						Start: 2020,
						End:   2024,
					},
					IPFirewall:    true,
					ProxyProtocol: "off",
					TLS:           "full",
				},
			},
			want: want{
				o: true,
			},
		},
		"SuccessSpectrumEdgeIPsAnycast": {
			reason: "UpToDate should return true and not panic with a Application with Spectrum Edge IPs Anycast",
			args: args{
				rp: &v1alpha1.ApplicationParameters{
					IPv4: ptr.BoolPtr(true),
					DNS: v1alpha1.SpectrumApplicationDNS{
						Type: "CNAME",
						Name: "spectrum.foo.com",
					},
					EdgeIPs: &v1alpha1.SpectrumApplicationEdgeIPs{
						Type: "static",
						IPs:  []string{"2001:db8::1", "192.0.2.1"},
					},
					IPFirewall:    ptr.BoolPtr(true),
					ProxyProtocol: ptr.StringPtr("off"),
					TLS:           ptr.StringPtr("full"),
					OriginDirect:  []string{"tcp://192.0.2.1:22"},
				},
				r: cloudflare.SpectrumApplication{
					IPv4: true,
					DNS: cloudflare.SpectrumApplicationDNS{
						Type: "CNAME",
						Name: "spectrum.foo.com",
					},
					IPFirewall:    true,
					ProxyProtocol: "off",
					TLS:           "full",
					OriginDirect:  []string{"tcp://192.0.2.1:22"},
					EdgeIPs: &cloudflare.SpectrumApplicationEdgeIPs{
						Type: "static",
						IPs:  []net.IP{net.ParseIP("192.0.2.1"), net.ParseIP("2001:db8::1")},
					},
				},
			},
			want: want{
				o: true,
			},
		},
		"SuccessSpectrumEdgeIPsDynamic": {
			reason: "UpToDate should return true and not panic with a Application with Spectrum Edge IPs Dynamic",
			args: args{
				rp: &v1alpha1.ApplicationParameters{
					IPv4: ptr.BoolPtr(true),
					DNS: v1alpha1.SpectrumApplicationDNS{
						Type: "CNAME",
						Name: "spectrum.foo.com",
					},
					EdgeIPs: &v1alpha1.SpectrumApplicationEdgeIPs{
						Type:         "static",
						Connectivity: ptr.StringPtr("all"),
					},
					IPFirewall:    ptr.BoolPtr(true),
					ProxyProtocol: ptr.StringPtr("off"),
					TLS:           ptr.StringPtr("full"),
					OriginDirect:  []string{"tcp://192.0.2.1:22"},
				},
				r: cloudflare.SpectrumApplication{
					IPv4: true,
					DNS: cloudflare.SpectrumApplicationDNS{
						Type: "CNAME",
						Name: "spectrum.foo.com",
					},
					IPFirewall:    true,
					ProxyProtocol: "off",
					TLS:           "full",
					OriginDirect:  []string{"tcp://192.0.2.1:22"},
					EdgeIPs: &cloudflare.SpectrumApplicationEdgeIPs{
						Type:         "static",
						Connectivity: &connectivityAll,
					},
				},
			},
			want: want{
				o: true,
			},
		},
		"UpToDateIdentical": {
			reason: "UpToDate should return true if the spec matches the record",
			args: args{
				rp: &v1alpha1.ApplicationParameters{
					Protocol:    "tcp/80",
					TrafficType: ptr.StringPtr("http"),
					IPFirewall:  ptr.BoolPtr(false),
				},
				r: cloudflare.SpectrumApplication{
					Protocol:    "tcp/80",
					TrafficType: "http",
					IPFirewall:  false,
				},
			},
			want: want{
				o: true,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := UpToDate(tc.args.rp, tc.args.r)
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nUpToDate(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}
