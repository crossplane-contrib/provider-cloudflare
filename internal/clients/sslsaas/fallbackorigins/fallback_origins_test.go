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

package fallbackorigins

import (
	"testing"

	"github.com/cloudflare/cloudflare-go"

	"github.com/google/go-cmp/cmp"

	"github.com/crossplane-contrib/provider-cloudflare/apis/sslsaas/v1alpha1"

	ptr "k8s.io/utils/pointer"
)

const (
	origin = "fallback.origin.com"
)

func TestUpToDate(t *testing.T) {
	type args struct {
		fop *v1alpha1.FallbackOriginParameters
		fo  cloudflare.CustomHostnameFallbackOrigin
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
				fop: &v1alpha1.FallbackOriginParameters{},
				fo:  cloudflare.CustomHostnameFallbackOrigin{},
			},
			want: want{
				o: true,
			},
		},
		"UpToDateDifferent": {
			reason: "UpToDate should return false if the spec does not match the resource",
			args: args{
				fop: &v1alpha1.FallbackOriginParameters{
					Origin: ptr.StringPtr(origin),
				},
				fo: cloudflare.CustomHostnameFallbackOrigin{
					Origin: "crazy.origin.com",
				},
			},
			want: want{
				o: false,
			},
		},
		"UpToDateIdentical": {
			reason: "UpToDate should return true if the spec matches the resource",
			args: args{
				fop: &v1alpha1.FallbackOriginParameters{
					Origin: ptr.StringPtr(origin),
				},
				fo: cloudflare.CustomHostnameFallbackOrigin{
					Origin: origin,
				},
			},
			want: want{
				o: true,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := UpToDate(tc.args.fop, tc.args.fo)
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nUpToDate(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}
