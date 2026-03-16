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

	"github.com/crossplane-contrib/provider-cloudflare/apis/zone/v1alpha1"
	"github.com/crossplane-contrib/provider-cloudflare/internal/clients/zones/fake"
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
						// These settings will be lateInited
						AdvancedDDOS:   nil,
						Minify:         nil,
						MobileRedirect: nil,
						SecurityHeader: nil,
						Ciphers:        nil,
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
				// AdvancedDDOS, Minify and SecurityHeader should be late-inited here.
				czs: &v1alpha1.ZoneSettings{
					AdvancedDDOS: ptr.StringPtr("yes"),
					Minify: &v1alpha1.MinifySettings{
						CSS:  ptr.StringPtr("on"),
						HTML: ptr.StringPtr("on"),
						JS:   ptr.StringPtr("on"),
					},
					MobileRedirect: &v1alpha1.MobileRedirectSettings{
						Status:    ptr.StringPtr("on"),
						Subdomain: ptr.StringPtr("m"),
						StripURI:  ptr.BoolPtr(false),
					},
					SecurityHeader: &v1alpha1.SecurityHeaderSettings{
						StrictTransportSecurity: &v1alpha1.StrictTransportSecuritySettings{
							Enabled:           ptr.BoolPtr(true),
							MaxAge:            ptr.Int64(86400),
							IncludeSubdomains: ptr.BoolPtr(true),
							NoSniff:           ptr.BoolPtr(true),
						},
					},
					Ciphers: []string{
						"ECDHE-RSA-AES128-GCM-SHA256",
						"AES128-SHA",
					},
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
						AdvancedDDOS: ptr.StringPtr("yes"),
						Minify: &v1alpha1.MinifySettings{
							CSS:  ptr.StringPtr("on"),
							HTML: ptr.StringPtr("on"),
							JS:   ptr.StringPtr("on"),
						},
						MobileRedirect: &v1alpha1.MobileRedirectSettings{
							Status:    ptr.StringPtr("on"),
							Subdomain: ptr.StringPtr("m"),
							StripURI:  ptr.BoolPtr(false),
						},
						SecurityHeader: &v1alpha1.SecurityHeaderSettings{
							StrictTransportSecurity: &v1alpha1.StrictTransportSecuritySettings{
								Enabled:           ptr.BoolPtr(true),
								MaxAge:            ptr.Int64(86400),
								IncludeSubdomains: ptr.BoolPtr(true),
								NoSniff:           ptr.BoolPtr(true),
							},
						},
						Ciphers: []string{
							"ECDHE-RSA-AES128-GCM-SHA256",
							"AES128-SHA",
						},
						BrowserCacheTTL: ptr.Int64Ptr(900),
					},
				},
			},
		},
		"SuccessSettingsPartial": {
			reason: "LateInit should update partially set settings from a Zone",
			args: args{
				zp: &v1alpha1.ZoneParameters{
					AccountID:         ptr.StringPtr("beef"),
					Paused:            ptr.BoolPtr(false),
					PlanID:            ptr.StringPtr("dead"),
					VanityNameServers: []string{"ns1.lele.com", "ns2.woowoo.org"},
					Settings: v1alpha1.ZoneSettings{
						// nil settings under the top-level setting will be lateInited
						Minify: &v1alpha1.MinifySettings{
							CSS:  nil,
							HTML: nil,
							JS:   ptr.StringPtr("off"),
						},
						MobileRedirect: &v1alpha1.MobileRedirectSettings{
							Status:    ptr.StringPtr("on"),
							Subdomain: nil,
							StripURI:  ptr.BoolPtr(true),
						},
						SecurityHeader: &v1alpha1.SecurityHeaderSettings{
							StrictTransportSecurity: &v1alpha1.StrictTransportSecuritySettings{
								Enabled:           ptr.BoolPtr(true),
								MaxAge:            ptr.Int64Ptr(86400),
								NoSniff:           nil,
								IncludeSubdomains: nil,
							},
						},
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
				// CSS and HTML under Minify should be late-inited here.
				czs: &v1alpha1.ZoneSettings{
					Minify: &v1alpha1.MinifySettings{
						CSS:  ptr.StringPtr("on"),
						HTML: ptr.StringPtr("off"),
						JS:   ptr.StringPtr("on"),
					},
					MobileRedirect: &v1alpha1.MobileRedirectSettings{
						Status:    ptr.StringPtr("on"),
						Subdomain: ptr.StringPtr("m"),
						StripURI:  ptr.BoolPtr(false),
					},
					SecurityHeader: &v1alpha1.SecurityHeaderSettings{
						StrictTransportSecurity: &v1alpha1.StrictTransportSecuritySettings{
							Enabled:           ptr.BoolPtr(false),
							MaxAge:            ptr.Int64Ptr(66700),
							NoSniff:           ptr.BoolPtr(true),
							IncludeSubdomains: ptr.BoolPtr(true),
						},
					},
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
						Minify: &v1alpha1.MinifySettings{
							CSS:  ptr.StringPtr("on"),
							HTML: ptr.StringPtr("off"),
							JS:   ptr.StringPtr("off"),
						},
						MobileRedirect: &v1alpha1.MobileRedirectSettings{
							Status:    ptr.StringPtr("on"),
							Subdomain: ptr.StringPtr("m"),
							StripURI:  ptr.BoolPtr(true),
						},
						SecurityHeader: &v1alpha1.SecurityHeaderSettings{
							StrictTransportSecurity: &v1alpha1.StrictTransportSecuritySettings{
								Enabled:           ptr.BoolPtr(true),
								MaxAge:            ptr.Int64Ptr(86400),
								NoSniff:           ptr.BoolPtr(true),
								IncludeSubdomains: ptr.BoolPtr(true),
							},
						},
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
		"VanityNSTrue": {
			reason: "UpToDate should return true if VanityNS field matches in any order",
			args: args{
				zp: &v1alpha1.ZoneParameters{
					PlanID:            ptr.StringPtr("cake"),
					Settings:          v1alpha1.ZoneSettings{},
					VanityNameServers: []string{"ns2.woowoo.org", "ns1.lele.com"},
				},
				z: cloudflare.Zone{
					PlanPending: cloudflare.ZonePlan{
						ZonePlanCommon: cloudflare.ZonePlanCommon{
							ID: "cake",
						},
					},
					VanityNS: []string{"ns1.lele.com", "ns2.woowoo.org"},
				},
				ozs: &v1alpha1.ZoneSettings{},
			},
			want: want{
				o: true,
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

	inputZoneID := "1234"
	nsKey := cfsMinify

	nsInputValue := v1alpha1.MinifySettings{
		CSS:  ptr.StringPtr("on"),
		HTML: ptr.StringPtr("off"),
		JS:   ptr.StringPtr("bar"),
	}

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
				id: inputZoneID,
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
						if zoneID != inputZoneID {
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
				id: inputZoneID,
				zp: v1alpha1.ZoneParameters{
					Paused:            ptr.BoolPtr(false),
					VanityNameServers: []string{"ns1.lele.com", "ns2.woowoo.org"},
				},
			},
			want: want{
				err: nil,
			},
		},
		"UpdateZoneSettings": {
			reason: "UpdateZone should return no error when updating zone settings",
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
						if zoneID != inputZoneID {
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
						return &cloudflare.ZoneSettingResponse{
							Result: []cloudflare.ZoneSetting{
								{
									ID:       nsKey,
									Editable: true,
									// Client should decode nested values from map string interface
									Value: map[string]interface{}{
										cfsMinifyCSS:  nsInputValue.CSS,
										cfsMinifyHTML: nsInputValue.HTML,
										cfsMinifyJS:   "foo", // This value should be overwritten
									},
								},
							},
						}, nil
					},
					MockUpdateZoneSettings: func(ctx context.Context, zoneID string, cs []cloudflare.ZoneSetting) (*cloudflare.ZoneSettingResponse, error) {
						if zoneID != inputZoneID {
							return nil, errors.New("zoneID value incorrect")
						}
						nsInputExpectedValue := map[string]interface{}{
							cfsMinifyCSS:  *nsInputValue.CSS,
							cfsMinifyHTML: *nsInputValue.HTML,
							cfsMinifyJS:   *nsInputValue.JS,
						}
						// Must match our requested setting ID and value.
						// If not, we return an error.
						for _, setting := range cs {
							if setting.ID == nsKey {
								if cmp.Equal(nsInputExpectedValue, setting.Value) {
									return nil, nil
								}
							}
						}
						return nil, errors.New("Nested complex setting not updated or invalid")
					},
				},
			},
			args: args{
				id: inputZoneID,
				zp: v1alpha1.ZoneParameters{
					Paused:            ptr.BoolPtr(false),
					VanityNameServers: []string{"ns1.lele.com", "ns2.woowoo.org"},
					Settings: v1alpha1.ZoneSettings{
						Minify: &nsInputValue,
					},
				},
			},
			want: want{
				err: nil,
			},
		},
		// TODO: Test SetPlan
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
	errBoom := errors.New("boom")
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
		"ErrorLookupSettings": {
			reason: "LoadSettingsForZone should return an error when the API call returns an error",
			fields: fields{
				client: fake.MockClient{
					MockZoneSettings: func(ctx context.Context, zoneID string) (*cloudflare.ZoneSettingResponse, error) {
						return nil, errBoom
					},
				},
			},
			args: args{
				id: "abcd",
				zs: v1alpha1.ZoneSettings{ZeroRTT: ptr.StringPtr("yes")},
			},
			want: want{
				err: errors.Wrap(errBoom, errLoadSettings),
				o:   v1alpha1.ZoneSettings{ZeroRTT: ptr.StringPtr("yes")},
			},
		},
		"LoadUnknownSetting": {
			// Note: This is an academic test - all of the keys we use are static strings
			// So we _cannot_ load a setting into a struct without knowing about it.
			// We add this test to avoid regressions if the method used to load settings
			// is changed, as it has caused problems in the past.
			reason: "LoadSettingsForZone should not error when reading unknown settings",
			fields: fields{
				client: fake.MockClient{
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

func TestSecurityHeaderSettingsToMap(t *testing.T) {
	type args struct {
		settings *v1alpha1.SecurityHeaderSettings
	}

	type want struct {
		o map[string]interface{}
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"Success": {
			reason: "securityHeaderSettingsToMap should return a valid map type",
			args: args{
				settings: &v1alpha1.SecurityHeaderSettings{
					StrictTransportSecurity: &v1alpha1.StrictTransportSecuritySettings{
						Enabled:           ptr.BoolPtr(true),
						MaxAge:            ptr.Int64Ptr(86400),
						IncludeSubdomains: ptr.BoolPtr(true),
						NoSniff:           ptr.BoolPtr(true),
					},
				},
			},
			want: want{
				o: map[string]interface{}{
					cfsStrictTransportSecurity: map[string]interface{}{
						cfsStrictTransportSecurityEnabled:           true,
						cfsStrictTransportSecurityIncludeSubdomains: true,
						cfsStrictTransportSecurityMaxAge:            int64(86400),
						cfsStrictTransportSecurityNoSniff:           true,
					},
				},
			},
		},
		"SuccessEmpty": {
			reason: "securityHeaderSettingsToMap should return an empty map when no settings are provided",
			args: args{
				settings: &v1alpha1.SecurityHeaderSettings{},
			},
			want: want{
				o: map[string]interface{}{},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := securityHeaderSettingsToMap(tc.args.settings)
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nsecurityHeaderSettingsToMap(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestMobileRedirectSettingsToMap(t *testing.T) {
	type args struct {
		settings *v1alpha1.MobileRedirectSettings
	}

	type want struct {
		o map[string]interface{}
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"Success": {
			reason: "mobileRedirectSettingsToMap should return a valid map type",
			args: args{
				settings: &v1alpha1.MobileRedirectSettings{
					Status:    ptr.StringPtr("on"),
					Subdomain: ptr.StringPtr("m"),
					StripURI:  ptr.BoolPtr(false),
				},
			},
			want: want{
				o: map[string]interface{}{
					cfsMobileRedirectStatus:    "on",
					cfsMobileRedirectSubdomain: "m",
					cfsMobileRedirectStripURI:  false,
				},
			},
		},
		"SuccessEmpty": {
			reason: "mobileRedirectSettingsToMap should return an empty map when no settings are provided",
			args: args{
				settings: &v1alpha1.MobileRedirectSettings{},
			},
			want: want{
				o: map[string]interface{}{},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := mobileRedirectSettingsToMap(tc.args.settings)
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nmobileRedirectSettingsToMap(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestMinifySettingsToMap(t *testing.T) {
	type args struct {
		settings *v1alpha1.MinifySettings
	}

	type want struct {
		o map[string]interface{}
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"Success": {
			reason: "minifySettingsToMap should return a valid map type",
			args: args{
				settings: &v1alpha1.MinifySettings{
					CSS:  ptr.StringPtr("on"),
					HTML: ptr.StringPtr("on"),
					JS:   ptr.StringPtr("on"),
				},
			},
			want: want{
				o: map[string]interface{}{
					cfsMinifyCSS:  "on",
					cfsMinifyHTML: "on",
					cfsMinifyJS:   "on",
				},
			},
		},
		"SuccessEmpty": {
			reason: "minifySettingsToMap should return an empty map when no settings are provided",
			args: args{
				settings: &v1alpha1.MinifySettings{},
			},
			want: want{
				o: map[string]interface{}{},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := minifySettingsToMap(tc.args.settings)
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nminifySettingsToMap(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}
