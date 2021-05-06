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
	"testing"

	"github.com/cloudflare/cloudflare-go"

	"github.com/benagricola/provider-cloudflare/apis/dns/v1alpha1"
	"github.com/google/go-cmp/cmp"

	ptr "k8s.io/utils/pointer"
)

func uint16Ptr(v uint16) *uint16 {
	return &v
}

func TestLateInitialize(t *testing.T) {
	type args struct {
		rp *v1alpha1.RecordParameters
		r  cloudflare.DNSRecord
	}

	type want struct {
		o  bool
		rp *v1alpha1.RecordParameters
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"LateInitSpecNil": {
			reason: "LateInit should return false when not passed a spec",
			args:   args{},
			want: want{
				o: false,
			},
		},
		"LateInitDontUpdate": {
			reason: "LateInit should not update already-set spec fields from a Record",
			args: args{
				rp: &v1alpha1.RecordParameters{
					Proxied:  ptr.BoolPtr(false),
					Priority: ptr.Int32Ptr(4),
				},
				r: cloudflare.DNSRecord{
					Proxied:  ptr.BoolPtr(true),
					Priority: uint16Ptr(1),
				},
			},
			want: want{
				o: false,
				rp: &v1alpha1.RecordParameters{
					Proxied:  ptr.BoolPtr(false),
					Priority: ptr.Int32Ptr(4),
				},
			},
		},
		"LateInitUpdate": {
			reason: "LateInit should update unset spec fields from a Record",
			args: args{
				rp: &v1alpha1.RecordParameters{},
				r: cloudflare.DNSRecord{
					Proxied:  ptr.BoolPtr(true),
					Priority: uint16Ptr(1),
				},
			},
			want: want{
				o: true,
				rp: &v1alpha1.RecordParameters{
					Proxied:  ptr.BoolPtr(true),
					Priority: ptr.Int32Ptr(1),
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := LateInitialize(tc.args.rp, tc.args.r)
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nLateInit(...): -want, +got:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.rp, tc.args.rp); diff != "" {
				t.Errorf("\n%s\nLateInit(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestUpToDate(t *testing.T) {
	type args struct {
		rp *v1alpha1.RecordParameters
		r  cloudflare.DNSRecord
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
				rp: &v1alpha1.RecordParameters{},
				r:  cloudflare.DNSRecord{},
			},
			want: want{
				o: true,
			},
		},
		"UpToDateDifferent": {
			reason: "UpToDate should return false if the spec does not match the record",
			args: args{
				rp: &v1alpha1.RecordParameters{
					Type:    ptr.StringPtr("A"),
					Name:    "foo",
					Content: "127.0.0.1",
					TTL:     ptr.Int64Ptr(600),
					Proxied: ptr.BoolPtr(false),
				},
				r: cloudflare.DNSRecord{
					Type:    "A",
					Name:    "foo",
					Content: "127.0.0.2",
					TTL:     600,
					Proxied: ptr.BoolPtr(false),
				},
			},
			want: want{
				o: false,
			},
		},
		"UpToDateIdentical": {
			reason: "UpToDate should return true if the spec matches the record",
			args: args{
				rp: &v1alpha1.RecordParameters{
					Type:    ptr.StringPtr("A"),
					Name:    "foo",
					Content: "127.0.0.1",
					TTL:     ptr.Int64Ptr(600),
					Proxied: ptr.BoolPtr(false),
				},
				r: cloudflare.DNSRecord{
					Type:    "A",
					Name:    "foo",
					Content: "127.0.0.1",
					TTL:     600,
					Proxied: ptr.BoolPtr(false),
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
