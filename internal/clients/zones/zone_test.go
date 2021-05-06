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
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"github.com/google/go-cmp/cmp"

	"github.com/pkg/errors"

	ptr "k8s.io/utils/pointer"

	"github.com/crossplane/crossplane-runtime/pkg/test"

	"github.com/benagricola/provider-cloudflare/apis/zone/v1alpha1"
	"github.com/benagricola/provider-cloudflare/internal/clients/zones/fake"
)

func TestLateInitialize(t *testing.T) {
	type args struct {
		zp  *v1alpha1.ZoneParameters
		z   cloudflare.Zone
		czs *v1alpha1.ZoneSettings
	}

	type want struct {
		o  bool
		zp *v1alpha1.ZoneParameters
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
		"Success": {
			reason: "LateInit should update fields from a Zone",
			args: args{
				zp: &v1alpha1.ZoneParameters{
					AccountID:         ptr.StringPtr("beef"),
					PlanID:            ptr.StringPtr("dead"),
					VanityNameServers: []string{},
					Settings:          v1alpha1.ZoneSettings{},
				},
				z: cloudflare.Zone{
					Account: cloudflare.Account{
						ID: "beef",
					},
					Plan: cloudflare.ZonePlan{
						ZonePlanCommon: cloudflare.ZonePlanCommon{
							ID: "dead",
						},
					},
					Paused:   false,
					VanityNS: []string{"ns1.lele.com", "ns2.woowoo.org"},
				},
				czs: &v1alpha1.ZoneSettings{},
			},
			want: want{
				o: true,
				zp: &v1alpha1.ZoneParameters{
					Paused:            ptr.BoolPtr(false),
					AccountID:         ptr.StringPtr("beef"),
					PlanID:            ptr.StringPtr("dead"),
					VanityNameServers: []string{"ns1.lele.com", "ns2.woowoo.org"},
					Settings:          v1alpha1.ZoneSettings{},
				},
			},
		},
		"SuccessSettings": {
			reason: "LateInit should update settings from a Zone",
			args: args{
				zp: &v1alpha1.ZoneParameters{
					AccountID:         ptr.StringPtr("beef"),
					Paused:            ptr.BoolPtr(false),
					PlanID:            ptr.StringPtr("dead"),
					VanityNameServers: []string{"ns1.lele.com", "ns2.woowoo.org"},
					Settings: v1alpha1.ZoneSettings{
						// This setting will be lateInited
						AdvancedDDOS: nil,
						// This setting will not be overwritten
						BrowserCacheTTL: ptr.Int64Ptr(900),
					},
				},
				z: cloudflare.Zone{
					Account: cloudflare.Account{
						ID: "beef",
					},
					Plan: cloudflare.ZonePlan{
						ZonePlanCommon: cloudflare.ZonePlanCommon{
							ID: "dead",
						},
					},
					// This field should not be late-inited, as the value
					// is already set false in zp
					Paused:   false,
					VanityNS: []string{"ns1.lele.com", "ns2.woowoo.org"},
				},
				// 'Current' Settings are those settings that were observed
				// from the API.
				// Only AdvancedDDOS should be late-inited here.
				czs: &v1alpha1.ZoneSettings{
					AdvancedDDOS:    ptr.StringPtr("yes"),
					BrowserCacheTTL: ptr.Int64Ptr(3600),
				},
			},
			want: want{
				o: true,
				zp: &v1alpha1.ZoneParameters{
					Paused:            ptr.BoolPtr(false),
					AccountID:         ptr.StringPtr("beef"),
					PlanID:            ptr.StringPtr("dead"),
					VanityNameServers: []string{"ns1.lele.com", "ns2.woowoo.org"},
					Settings: v1alpha1.ZoneSettings{
						AdvancedDDOS:    ptr.StringPtr("yes"),
						BrowserCacheTTL: ptr.Int64Ptr(900),
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := LateInitialize(tc.args.zp, tc.args.z, tc.args.czs)
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nLateInit(...): -want, +got:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.zp, tc.args.zp); diff != "" {
				t.Errorf("\n%s\nLateInit(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}
func TestUpToDate(t *testing.T) {
	type args struct {
		zp  *v1alpha1.ZoneParameters
		z   cloudflare.Zone
		ozs *v1alpha1.ZoneSettings
	}

	type want struct {
		o bool
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"SpecNil": {
			reason: "UpToDate should return true when not passed a spec",
			args:   args{},
			want: want{
				o: true,
			},
		},
		"EmptyParams": {
			reason: "UpToDate should return true and not panic with nil values",
			args: args{
				zp: &v1alpha1.ZoneParameters{},
				z: cloudflare.Zone{
					Paused: true,
					Plan: cloudflare.ZonePlan{
						ZonePlanCommon: cloudflare.ZonePlanCommon{
							ID: "beef",
						},
					},
					PlanPending: cloudflare.ZonePlan{
						ZonePlanCommon: cloudflare.ZonePlanCommon{
							ID: "cake",
						},
					},
					VanityNS: []string{},
				},
				ozs: &v1alpha1.ZoneSettings{},
			},
			want: want{
				o: true,
			},
		},
		"Paused": {
			reason: "UpToDate should return false if Paused is not up to date",
			args: args{
				zp: &v1alpha1.ZoneParameters{
					Paused: ptr.BoolPtr(false),
				},
				z: cloudflare.Zone{
					Paused: true,
				},
				ozs: &v1alpha1.ZoneSettings{},
			},
			want: want{
				o: false,
			},
		},
		"PlanFalse": {
			reason: "UpToDate should return false if PlanID is not one of Plan or PlanPending IDs",
			args: args{
				zp: &v1alpha1.ZoneParameters{
					PlanID: ptr.StringPtr("moo"),
				},
				z: cloudflare.Zone{
					Plan: cloudflare.ZonePlan{
						ZonePlanCommon: cloudflare.ZonePlanCommon{
							ID: "beef",
						},
					},
					PlanPending: cloudflare.ZonePlan{
						ZonePlanCommon: cloudflare.ZonePlanCommon{
							ID: "cake",
						},
					},
				},
				ozs: &v1alpha1.ZoneSettings{},
			},
			want: want{
				o: false,
			},
		},
		"PlanTrue": {
			reason: "UpToDate should return true if PlanID is current Plan ID",
			args: args{
				zp: &v1alpha1.ZoneParameters{
					PlanID: ptr.StringPtr("beef"),
				},
				z: cloudflare.Zone{
					Plan: cloudflare.ZonePlan{
						ZonePlanCommon: cloudflare.ZonePlanCommon{
							ID: "beef",
						},
					},
				},
				ozs: &v1alpha1.ZoneSettings{},
			},
			want: want{
				o: true,
			},
		},
		"PlanPendingTrue": {
			reason: "UpToDate should return true if PlanID is pending Plan ID",
			args: args{
				zp: &v1alpha1.ZoneParameters{
					PlanID:   ptr.StringPtr("cake"),
					Settings: v1alpha1.ZoneSettings{},
				},
				z: cloudflare.Zone{
					PlanPending: cloudflare.ZonePlan{
						ZonePlanCommon: cloudflare.ZonePlanCommon{
							ID: "cake",
						},
					},
				},
				ozs: &v1alpha1.ZoneSettings{},
			},
			want: want{
				o: true,
			},
		},
		"SettingsFalse": {
			reason: "UpToDate should return false if settings are different",
			args: args{
				zp: &v1alpha1.ZoneParameters{
					PlanID: ptr.StringPtr("cake"),
					Settings: v1alpha1.ZoneSettings{
						ZeroRTT: ptr.StringPtr("no"),
					},
				},
				z: cloudflare.Zone{
					PlanPending: cloudflare.ZonePlan{
						ZonePlanCommon: cloudflare.ZonePlanCommon{
							ID: "cake",
						},
					},
				},
				ozs: &v1alpha1.ZoneSettings{
					ZeroRTT: ptr.StringPtr("yes"),
				},
			},
			want: want{
				o: false,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := UpToDate(tc.args.zp, tc.args.z, tc.args.ozs)
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nUpToDate(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestUpdateZone(t *testing.T) {
	errBoom := errors.New("boom")
	type fields struct {
		client Client
	}

	type args struct {
		ctx context.Context
		id  string
		zp  v1alpha1.ZoneParameters
	}

	type want struct {
		err error
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"UpdateZoneNotFound": {
			reason: "UpdateZone should return errUpdateZone if the zone is not found",
			fields: fields{
				client: fake.MockClient{
					// When the ZoneDetails method is called on this fake client,
					// return an error that we can compare.
					MockZoneDetails: func(ctx context.Context, zoneID string) (cloudflare.Zone, error) {
						return cloudflare.Zone{}, errBoom
					},
				},
			},
			args: args{
				id: "abcd",
			},
			want: want{
				err: errors.Wrap(errBoom, errUpdateZone),
			},
		},
		"UpdateZoneOptions": {
			reason: "UpdateZone should return no error when updating zone options",
			fields: fields{
				client: fake.MockClient{
					MockZoneDetails: func(ctx context.Context, zoneID string) (cloudflare.Zone, error) {
						return cloudflare.Zone{
							ID:       zoneID,
							Name:     "testzone.com",
							Paused:   true,
							VanityNS: []string{"ns1.lele.com"},
						}, nil
					},
					// When EditZone is called, check it receives the expected arguments.
					// If it doesn't we return an error which will cause the test to fail.
					MockEditZone: func(ctx context.Context, zoneID string, zoneOpts cloudflare.ZoneOptions) (cloudflare.Zone, error) {
						var err error
						if zoneID != "abcd" {
							err = errors.New("zoneID value incorrect")
						}
						if *zoneOpts.Paused != false {
							err = errors.New("zoneOpts.Paused value incorrect")
						}

						if !cmp.Equal(zoneOpts.VanityNS,
							[]string{"ns1.lele.com", "ns2.woowoo.org"}) {
							err = errors.New("zoneOpts.VanityNS does not match")
						}
						// Returned zone is discarded by UpdateZone
						return cloudflare.Zone{}, err
					},

					MockZoneSettings: func(ctx context.Context, zoneID string) (*cloudflare.ZoneSettingResponse, error) {
						return &cloudflare.ZoneSettingResponse{}, nil
					},
				},
			},
			args: args{
				id: "abcd",
				zp: v1alpha1.ZoneParameters{
					Paused:            ptr.BoolPtr(false),
					VanityNameServers: []string{"ns1.lele.com", "ns2.woowoo.org"},
				},
			},
			want: want{
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := UpdateZone(tc.args.ctx, tc.fields.client, tc.args.id, tc.args.zp)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nUpdateZone(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestLoadSettingsForZone(t *testing.T) {
	type fields struct {
		client Client
	}

	type args struct {
		ctx context.Context
		id  string
		zs  v1alpha1.ZoneSettings
	}

	type want struct {
		err error
		o   v1alpha1.ZoneSettings
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"LoadUnknownSetting": {
			// Note: This is an academic test - all of the keys we use are static strings
			// So we _cannot_ load a setting into a struct without knowing about it.
			// We add this test to avoid regressions if the method used to load settings
			// is changed, as it has caused problems in the past.
			reason: "LoadSettingsForZone should not error when reading unknown settings",
			fields: fields{
				client: fake.MockClient{
					// When the ZoneDetails method is called on this fake client,
					// return an error that we can compare.
					MockZoneDetails: func(ctx context.Context, zoneID string) (cloudflare.Zone, error) {
						return cloudflare.Zone{}, nil
					},
					MockZoneSettings: func(ctx context.Context, zoneID string) (*cloudflare.ZoneSettingResponse, error) {
						return &cloudflare.ZoneSettingResponse{
							Result: []cloudflare.ZoneSetting{
								{ID: "unknownKey", Value: "foo"},
							},
						}, nil
					},
				},
			},
			args: args{
				id: "abcd",
				zs: v1alpha1.ZoneSettings{
					AdvancedDDOS: ptr.StringPtr("yes"),
				},
			},
			want: want{
				err: nil,
				o: v1alpha1.ZoneSettings{
					AdvancedDDOS: ptr.StringPtr("yes"),
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := tc.args.zs.DeepCopy()

			err := LoadSettingsForZone(tc.args.ctx, tc.fields.client, tc.args.id, &tc.args.zs)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nLoadSettingsForZone(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, *got); diff != "" {
				t.Errorf("\n%s\nLoadSettingsForZone(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}
