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

package route

import (
	"testing"

	"github.com/cloudflare/cloudflare-go"

	"github.com/google/go-cmp/cmp"

	"github.com/benagricola/provider-cloudflare/apis/workers/v1alpha1"

	ptr "k8s.io/utils/pointer"
)

func TestUpToDate(t *testing.T) {
	type args struct {
		rp *v1alpha1.RouteParameters
		r  cloudflare.WorkerRoute
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
				rp: &v1alpha1.RouteParameters{},
				r:  cloudflare.WorkerRoute{},
			},
			want: want{
				o: true,
			},
		},
		"UpToDateDifferent": {
			reason: "UpToDate should return false if the spec does not match the route",
			args: args{
				rp: &v1alpha1.RouteParameters{
					Script:  ptr.StringPtr("test-worker"),
					Pattern: "example.com/*",
				},
				r: cloudflare.WorkerRoute{
					Script:  "",
					Pattern: "example.com/*",
				},
			},
			want: want{
				o: false,
			},
		},
		"UpToDateScriptDifferent": {
			reason: "UpToDate should return false if the spec does not match the route script (nil)",
			args: args{
				rp: &v1alpha1.RouteParameters{
					Script:  nil,
					Pattern: "example.com/*",
				},
				r: cloudflare.WorkerRoute{
					Script:  "test-worker",
					Pattern: "example.com/*",
				},
			},
			want: want{
				o: false,
			},
		},
		"UpToDateScriptDifferentRemote": {
			reason: "UpToDate should return false if the spec does not match the route script (nil)",
			args: args{
				rp: &v1alpha1.RouteParameters{
					Script:  ptr.StringPtr("test-script"),
					Pattern: "example.com/*",
				},
				r: cloudflare.WorkerRoute{
					Script:  "",
					Pattern: "example.com/*",
				},
			},
			want: want{
				o: false,
			},
		},
		"UpToDatePatternDifferent": {
			reason: "UpToDate should return false if the spec pattern does not match the route pattern",
			args: args{
				rp: &v1alpha1.RouteParameters{
					Script:  ptr.StringPtr("test-script"),
					Pattern: "example2.com/*",
				},
				r: cloudflare.WorkerRoute{
					Script:  "test-script",
					Pattern: "example.com/*",
				},
			},
			want: want{
				o: false,
			},
		},
		"UpToDateScriptTrue": {
			reason: "UpToDate should return true if the spec does match the route script (nil)",
			args: args{
				rp: &v1alpha1.RouteParameters{
					Script:  nil,
					Pattern: "example.com/*",
				},
				r: cloudflare.WorkerRoute{
					Script:  "",
					Pattern: "example.com/*",
				},
			},
			want: want{
				o: true,
			},
		},
		"UpToDateIdentical": {
			reason: "UpToDate should return true if the spec matches the route",
			args: args{
				rp: &v1alpha1.RouteParameters{
					Script:  ptr.StringPtr("test-worker"),
					Pattern: "example.com/*",
				},
				r: cloudflare.WorkerRoute{
					Script:  "test-worker",
					Pattern: "example.com/*",
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
