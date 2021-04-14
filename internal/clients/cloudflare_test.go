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

package clients

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudflare/cloudflare-go"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	ptr "k8s.io/utils/pointer"

	"sigs.k8s.io/controller-runtime/pkg/client"

	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"

	v1alpha1 "github.com/benagricola/provider-cloudflare/apis/v1alpha1"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	rtfake "github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/test"
)

func TestGetConfig(t *testing.T) {
	errBoom := errors.New("boom")

	mc := &test.MockClient{
		MockGet: test.NewMockGetFn(errBoom),
	}

	_, errGetCredentialsSecret := resource.ExtractSecret(context.Background(), mc, xpv1.CommonCredentialSelectors{
		SecretRef: &xpv1.SecretKeySelector{},
	})

	type fields struct {
		client client.Client
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   *Config
		err error
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"ErrProviderConfigNotSet": {
			reason: "An error should be returned if a providerConfigRef is not set",
			fields: fields{
				client: &test.MockClient{
					MockGet: test.NewMockGetFn(errBoom),
				},
			},
			args: args{
				mg: &rtfake.Managed{},
			},
			want: want{
				err: errors.New(errPCRef),
			},
		},
		"ErrGetProviderConfig": {
			reason: "An error should be returned if we can't get our ProviderConfig",
			fields: fields{
				client: &test.MockClient{
					// Return an error when attempting to 'Get' a field
					// on the managed resource - this has the effect of making the
					// reference lookup fail, which should return the error we're
					// expecting.
					MockGet: test.NewMockGetFn(errBoom),
				},
			},
			args: args{
				mg: &rtfake.Managed{
					ProviderConfigReferencer: rtfake.ProviderConfigReferencer{
						Ref: &xpv1.Reference{},
					},
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errGetPC),
			},
		},
		"ErrMissingConnectionSecret": {
			reason: "An error should be returned if our ProviderConfig can't return a connection secret",
			fields: fields{
				client: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
						switch o := obj.(type) {
						case *v1alpha1.ProviderConfig:
							o.Spec.Credentials.Source = "Secret"
							o.Spec.Credentials.SecretRef = &xpv1.SecretKeySelector{}
						case *corev1.Secret:
							return errBoom
						}
						return nil
					}),
				},
			},
			args: args{
				mg: &rtfake.Managed{
					ProviderConfigReferencer: rtfake.ProviderConfigReferencer{
						Ref: &xpv1.Reference{},
					},
				},
			},
			want: want{
				err: errors.Wrap(errGetCredentialsSecret, errGetPC),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := GetConfig(tc.args.ctx, tc.fields.client, tc.args.mg)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nGetConfig(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nGetConfig(...): -want, +got:\n%s\n", tc.reason, diff)
			}

		})
	}
}

func TestUseProviderSecret(t *testing.T) {
	errBoom := errors.New("boom")

	d := map[string]interface{}{}
	errJSON := json.Unmarshal([]byte("{"), &d)

	type fields struct {
		client client.Client
	}

	type args struct {
		ctx  context.Context
		data []byte
	}

	type want struct {
		o   *Config
		err error
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"ErrInvalidJson": {
			reason: "An error should be returned if the secret contains invalid JSON",
			fields: fields{
				client: &test.MockClient{
					MockGet: test.NewMockGetFn(errBoom),
				},
			},
			args: args{
				// Missing trailing } is invalid
				data: []byte("{\"invalid\":\"foo\""),
			},
			want: want{
				err: errJSON,
			},
		},
		"ValidSecret": {
			reason: "A valid Config should be returned when passed a valid secret",
			fields: fields{
				client: &test.MockClient{
					MockGet: test.NewMockGetFn(nil),
				},
			},
			args: args{
				data: []byte("{\"apiKey\":\"foo\",\"email\":\"foo@bar.com\",\"token\":\"A7E0BA00E5E44574BFEC828D3F895973\"}"),
			},
			want: want{
				o: &Config{
					AuthByAPIKey: &AuthByAPIKey{
						Key:   ptr.StringPtr("foo"),
						Email: ptr.StringPtr("foo@bar.com"),
					},
					AuthByAPIToken: &AuthByAPIToken{
						Token: ptr.StringPtr("A7E0BA00E5E44574BFEC828D3F895973"),
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := UseProviderSecret(tc.args.ctx, tc.args.data)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nGetConfig(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nGetConfig(...): -want, +got:\n%s\n", tc.reason, diff)
			}

		})
	}
}

func TestNewClient(t *testing.T) {
	type args struct {
		config Config
	}

	type want struct {
		o   *cloudflare.API
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ErrInvalidAuth": {
			reason: "An error should be returned if the config does not contain any valid authentication details",
			args: args{
				config: Config{},
			},
			want: want{
				err: errors.New(errNoAuth),
			},
		},
		"ErrInvalidAPIKeyAuth": {
			reason: "An error should be returned if the config does not contain required api key auth fields",
			args: args{
				config: Config{
					AuthByAPIKey: &AuthByAPIKey{
						Email: ptr.StringPtr("foo@bar.com"),
					},
				},
			},
			want: want{
				err: errors.New(errNoAuth),
			},
		},
		"ValidAPIKeyAuth": {
			reason: "A cloudflare client should be returned when config contains valid API key details",
			args: args{
				config: Config{
					AuthByAPIKey: &AuthByAPIKey{
						Key:   ptr.StringPtr("abcd"),
						Email: ptr.StringPtr("foo@bar.com"),
					},
				},
			},
			want: want{
				err: nil,
				o: func(key, email string) *cloudflare.API {
					api, _ := cloudflare.New(key, email)
					return api
				}("abcd", "foo@bar.com"),
			},
		},
		"ValidAPITokenAuth": {
			reason: "A cloudflare client should be returned when config contains valid API token details",
			args: args{
				config: Config{
					AuthByAPIToken: &AuthByAPIToken{
						Token: ptr.StringPtr("beef"),
					},
				},
			},
			want: want{
				err: nil,
				o: func(token string) *cloudflare.API {
					api, _ := cloudflare.NewWithAPIToken(token)
					return api
				}("beef"),
			},
		},
		"ValidAPIBothAuth": {
			reason: "A cloudflare client should be returned configured with API key details if both Auth types are provided",
			args: args{
				config: Config{
					AuthByAPIKey: &AuthByAPIKey{
						Key:   ptr.StringPtr("abcd"),
						Email: ptr.StringPtr("foo@bar.com"),
					},
					AuthByAPIToken: &AuthByAPIToken{
						Token: ptr.StringPtr("beef"),
					},
				},
			},
			want: want{
				err: nil,
				o: func(key, email string) *cloudflare.API {
					api, _ := cloudflare.New(key, email)
					return api
				}("abcd", "foo@bar.com"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := NewClient(tc.args.config)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nNewClient(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.o, got, cmpopts.IgnoreUnexported(cloudflare.API{})); diff != "" {
				t.Errorf("\n%s\nNewClient(...): -want, +got:\n%s\n", tc.reason, diff)
			}

		})
	}
}
