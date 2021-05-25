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

package customhostnames

import (
	"testing"

	"github.com/cloudflare/cloudflare-go"

	"github.com/google/go-cmp/cmp"

	"github.com/benagricola/provider-cloudflare/apis/sslsaas/v1alpha1"

	ptr "k8s.io/utils/pointer"
)

const (
	hostname             = "myhostname.com"
	customOrigin         = "origin.zone.com"
	sslMethod            = "http"
	sslType              = "dv"
	sslWildcard          = true
	sslCustomCertificate = "invalid cert"
	sslCustomKey         = "invalid key"
)

func TestUpToDate(t *testing.T) {
	type args struct {
		chp *v1alpha1.CustomHostnameParameters
		ch  cloudflare.CustomHostname
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
				chp: &v1alpha1.CustomHostnameParameters{},
				ch:  cloudflare.CustomHostname{},
			},
			want: want{
				o: true,
			},
		},
		"UpToDateDifferent": {
			reason: "UpToDate should return false if the spec does not match the resource",
			args: args{
				chp: &v1alpha1.CustomHostnameParameters{
					Hostname:           hostname,
					CustomOriginServer: ptr.StringPtr(customOrigin),
					SSL: v1alpha1.CustomHostnameSSL{
						Method:            ptr.StringPtr(sslMethod),
						Type:              ptr.StringPtr(sslType),
						Wildcard:          ptr.BoolPtr(sslWildcard),
						CustomCertificate: ptr.StringPtr(sslCustomCertificate),
						CustomKey:         ptr.StringPtr(sslCustomKey),
					},
				},
				ch: cloudflare.CustomHostname{
					Hostname:           hostname,
					CustomOriginServer: "fancy.host.com",
					SSL: cloudflare.CustomHostnameSSL{
						Method:            "url",
						Type:              "email",
						Wildcard:          ptr.BoolPtr(true),
						CustomCertificate: "some cert",
						CustomKey:         "a key",
					},
				},
			},
			want: want{
				o: false,
			},
		},
		"UpToDateIdentical": {
			reason: "UpToDate should return true if the spec matches the resource",
			args: args{
				chp: &v1alpha1.CustomHostnameParameters{
					// Zone should be ignored as it cannot change
					Zone:               ptr.StringPtr("1234"),
					Hostname:           hostname,
					CustomOriginServer: ptr.StringPtr(customOrigin),
					SSL: v1alpha1.CustomHostnameSSL{
						Method:            ptr.StringPtr(sslMethod),
						Type:              ptr.StringPtr(sslType),
						Settings:          v1alpha1.CustomHostnameSSLSettings{},
						Wildcard:          ptr.BoolPtr(sslWildcard),
						CustomCertificate: ptr.StringPtr(sslCustomCertificate),
						CustomKey:         ptr.StringPtr(sslCustomKey),
					},
				},
				ch: cloudflare.CustomHostname{
					Hostname:           hostname,
					CustomOriginServer: customOrigin,
					SSL: cloudflare.CustomHostnameSSL{
						Method: sslMethod,
						Type:   sslType,
						Settings: cloudflare.CustomHostnameSSLSettings{
							// This should be converted to a nil pointer, which
							// should match the undefined value above.
							MinTLSVersion: "",
						},
						Wildcard:          ptr.BoolPtr(sslWildcard),
						CustomCertificate: sslCustomCertificate,
						CustomKey:         sslCustomKey,
					},
				},
			},
			want: want{
				o: true,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := UpToDate(tc.args.chp, tc.args.ch)
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nUpToDate(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}
