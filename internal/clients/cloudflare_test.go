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

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"

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
				// TODO: This test should FAIL if the json is valid...
				data: []byte("{\"invalid\":\"foo\""),
			},
			want: want{
				err: errJSON,
			},
		},
		"ErrMissingSecretField": {
			reason: "An error should be returned if the secret is missing one of the required fields",
			fields: fields{
				client: &test.MockClient{
					MockGet: test.NewMockGetFn(nil),
				},
			},
			args: args{
				data: []byte("{\"APIKey\":\"foo\"}"),
			},
			want: want{
				err: errors.New(errNoAuth),
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
				data: []byte("{\"APIKey\":\"foo\",\"Email\":\"foo@bar.com\"}"),
			},
			want: want{
				o: &Config{
					APIKey: "foo",
					Email:  "foo@bar.com",
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
