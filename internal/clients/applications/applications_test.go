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
		"UpToDateIdentical": {
			reason: "UpToDate should return true if the spec matches the record",
			args: args{
				rp: &v1alpha1.ApplicationParameters{
					Protocol:    ptr.StringPtr("tcp/80"),
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
