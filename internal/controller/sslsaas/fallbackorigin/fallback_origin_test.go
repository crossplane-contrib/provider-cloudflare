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

package fallbackorigin

import (
	"context"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	rtfake "github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	"github.com/benagricola/provider-cloudflare/apis/sslsaas/v1alpha1"
	pcv1alpha1 "github.com/benagricola/provider-cloudflare/apis/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	fallbackorigins "github.com/benagricola/provider-cloudflare/internal/clients/sslsaas/fallbackorigins"
	"github.com/benagricola/provider-cloudflare/internal/clients/sslsaas/fallbackorigins/fake"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

type fallbackOriginModifier func(*v1alpha1.FallbackOrigin)

func withZone(zoneID string) fallbackOriginModifier {
	return func(r *v1alpha1.FallbackOrigin) { r.Spec.ForProvider.Zone = &zoneID }
}

func withOrigin(origin string) fallbackOriginModifier {
	return func(r *v1alpha1.FallbackOrigin) { r.Spec.ForProvider.Origin = &origin }
}

func fallbackOrigin(m ...fallbackOriginModifier) *v1alpha1.FallbackOrigin {
	cr := &v1alpha1.FallbackOrigin{}
	for _, f := range m {
		f(cr)
	}
	return cr
}

const (
	zone   = "zone.com"
	origin = "fallback.zone.com"
)

func TestConnect(t *testing.T) {
	mc := &test.MockClient{
		MockGet: test.NewMockGetFn(nil),
	}

	_, errGetProviderConfig := clients.GetConfig(context.Background(), mc, &rtfake.Managed{})

	type fields struct {
		kube      client.Client
		newClient func(cfg clients.Config) (fallbackorigins.Client, error)
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   error
	}{
		"ErrNotFallbackOrigin": {
			reason: "An error should be returned if the managed resource is not a *Fallback Origin",
			args: args{
				mg: nil,
			},
			want: errors.New(errNotFallbackOrigin),
		},
		"ErrGetConfig": {
			reason: "Any errors from GetConfig should be wrapped",
			fields: fields{
				kube: mc,
			},
			args: args{
				mg: &v1alpha1.FallbackOrigin{
					Spec: v1alpha1.FallbackOriginSpec{
						ResourceSpec: xpv1.ResourceSpec{},
					},
				},
			},
			want: errors.Wrap(errGetProviderConfig, errClientConfig),
		},
		"ConnectReturnOK": {
			reason: "Connect should return no error when passed the correct values",
			fields: fields{
				kube: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
						switch o := obj.(type) {
						case *pcv1alpha1.ProviderConfig:
							o.Spec.Credentials.Source = "Secret"
							o.Spec.Credentials.SecretRef = &xpv1.SecretKeySelector{
								Key: "creds",
							}
						case *corev1.Secret:
							o.Data = map[string][]byte{
								"creds": []byte("{\"APIKey\":\"foo\",\"Email\":\"foo@bar.com\"}"),
							}
						}
						return nil
					}),
				},
				newClient: fallbackorigins.NewClient,
			},
			args: args{
				mg: &v1alpha1.FallbackOrigin{
					Spec: v1alpha1.FallbackOriginSpec{
						ResourceSpec: xpv1.ResourceSpec{
							ProviderConfigReference: &xpv1.Reference{
								Name: "blah",
							},
						},
					},
				},
			},
			want: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &connector{kube: tc.fields.kube, newCloudflareClientFn: tc.fields.newClient}
			_, err := e.Connect(tc.args.ctx, tc.args.mg)
			if diff := cmp.Diff(tc.want, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Connect(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestObserve(t *testing.T) {
	errBoom := errors.New("boom")

	type fields struct {
		client fallbackorigins.Client
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"ErrNotFallbackOrigin": {
			reason: "An error should be returned if the managed resource is not a *FallbackOrigin",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotFallbackOrigin),
			},
		},
		"ErrNoFallbackOrigin": {
			reason: "We should return ResourceExists: false when the resource does not exist",
			fields: fields{
				client: fake.MockClient{
					MockCustomHostnameFallbackOrigin: func(ctx context.Context, zoneID string) (cloudflare.CustomHostnameFallbackOrigin, error) {
						return cloudflare.CustomHostnameFallbackOrigin{}, &fallbackorigins.ErrNotFound{}
					},
				},
			},
			args: args{
				mg: fallbackOrigin(
					withZone(zone),
					withOrigin(origin),
				),
			},
			want: want{
				o: managed.ExternalObservation{ResourceExists: false},
			},
		},
		"ErrFallbackOriginLookup": {
			reason: "We should return an empty observation and an error if the API returned an error",
			fields: fields{
				client: fake.MockClient{
					MockCustomHostnameFallbackOrigin: func(ctx context.Context, zoneID string) (cloudflare.CustomHostnameFallbackOrigin, error) {
						return cloudflare.CustomHostnameFallbackOrigin{}, errBoom
					},
				},
			},
			args: args{
				mg: fallbackOrigin(
					withZone(zone),
					withOrigin(origin),
				),
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errBoom, errFallbackOriginLookup),
			},
		},
		"ErrFallbackOriginNoZone": {
			reason: "We should return an error if the FallbackOrigin does not have a zone",
			fields: fields{
				client: fake.MockClient{
					MockCustomHostnameFallbackOrigin: func(ctx context.Context, zoneID string) (cloudflare.CustomHostnameFallbackOrigin, error) {
						return cloudflare.CustomHostnameFallbackOrigin{}, errBoom
					},
				},
			},
			args: args{
				mg: &v1alpha1.FallbackOrigin{},
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.New(errFallbackOriginNoZone),
			},
		},
		"Success": {
			reason: "We should return ResourceExists: true and no error when a FallbackOrigin is found",
			fields: fields{
				client: fake.MockClient{
					MockCustomHostnameFallbackOrigin: func(ctx context.Context, zoneID string) (cloudflare.CustomHostnameFallbackOrigin, error) {
						return cloudflare.CustomHostnameFallbackOrigin{}, nil
					},
				},
			},
			args: args{
				mg: fallbackOrigin(
					withZone(zone),
					withOrigin(origin),
				),
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{client: tc.fields.client}
			got, err := e.Observe(tc.args.ctx, tc.args.mg)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Observe(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	errBoom := errors.New("boom")

	type fields struct {
		client fallbackorigins.Client
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalCreation
		err error
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"ErrNotFallbackOrigin": {
			reason: "An error should be returned if the managed resource is not a *FallbackOrigin",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotFallbackOrigin),
			},
		},
		"ErrFallbackOriginCreate": {
			reason: "We should return any errors during the create process",
			fields: fields{
				client: fake.MockClient{
					MockUpdateCustomHostnameFallbackOrigin: func(ctx context.Context, zoneID string, chfo cloudflare.CustomHostnameFallbackOrigin) (*cloudflare.CustomHostnameFallbackOriginResponse, error) {
						return nil, errBoom
					},
				},
			},
			args: args{
				mg: fallbackOrigin(
					withZone(zone),
					withOrigin(origin),
				),
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errBoom, errFallbackOriginCreation),
			},
		},
		"Success": {
			reason: "We should return no error when a FallbackOrigin is created",
			fields: fields{
				client: fake.MockClient{
					MockUpdateCustomHostnameFallbackOrigin: func(ctx context.Context, zoneID string, chfo cloudflare.CustomHostnameFallbackOrigin) (*cloudflare.CustomHostnameFallbackOriginResponse, error) {
						return &cloudflare.CustomHostnameFallbackOriginResponse{
							Result: chfo,
						}, nil
					},
				},
			},
			args: args{
				mg: fallbackOrigin(
					withZone(zone),
					withOrigin(origin),
				),
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{client: tc.fields.client}
			got, err := e.Create(tc.args.ctx, tc.args.mg)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Create(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Create(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	errBoom := errors.New("boom")

	type fields struct {
		client fallbackorigins.Client
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
	}

	type want struct {
		o   managed.ExternalUpdate
		err error
	}

	cases := map[string]struct {
		reason string
		fields fields
		args   args
		want   want
	}{
		"ErrNotFallbackOrigin": {
			reason: "An error should be returned if the managed resource is not a *FallbackOrigin",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotFallbackOrigin),
			},
		},
		"ErrFallbackOriginUpdate": {
			reason: "We should return any errors during the update process",
			fields: fields{
				client: fake.MockClient{
					MockUpdateCustomHostnameFallbackOrigin: func(ctx context.Context, zoneID string, chfo cloudflare.CustomHostnameFallbackOrigin) (*cloudflare.CustomHostnameFallbackOriginResponse, error) {
						return &cloudflare.CustomHostnameFallbackOriginResponse{}, errBoom
					},
				},
			},
			args: args{
				mg: fallbackOrigin(
					withZone(zone),
					withOrigin(origin),
				),
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errBoom, errFallbackOriginUpdate),
			},
		},
		"Success": {
			reason: "We should return no error when a FallbackOrigin is updated",
			fields: fields{
				client: fake.MockClient{
					MockCustomHostnameFallbackOrigin: func(ctx context.Context, zoneID string) (cloudflare.CustomHostnameFallbackOrigin, error) {
						return cloudflare.CustomHostnameFallbackOrigin{
							Origin: origin,
						}, nil
					},
					MockUpdateCustomHostnameFallbackOrigin: func(ctx context.Context, zoneID string, chfo cloudflare.CustomHostnameFallbackOrigin) (*cloudflare.CustomHostnameFallbackOriginResponse, error) {
						return &cloudflare.CustomHostnameFallbackOriginResponse{}, nil
					},
				},
			},
			args: args{
				mg: fallbackOrigin(
					withZone(zone),
					withOrigin(origin),
				),
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{client: tc.fields.client}
			got, err := e.Update(tc.args.ctx, tc.args.mg)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Update(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\ne.Update(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	errBoom := errors.New("boom")

	type fields struct {
		client fallbackorigins.Client
	}

	type args struct {
		ctx context.Context
		mg  resource.Managed
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
		"ErrNotFallbackOrigin": {
			reason: "An error should be returned if the managed resource is not a *FallbackOrigin",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotFallbackOrigin),
			},
		},
		"ErrNoFallbackOrigin": {
			reason: "We should return an error when no external name is set",
			fields: fields{
				client: fake.MockClient{
					MockDeleteCustomHostnameFallbackOrigin: func(ctx context.Context, zoneID string) error {
						return nil
					},
				},
			},
			args: args{
				mg: fallbackOrigin(
					withZone(zone),
					withOrigin(origin),
				),
			},
			want: want{
				err: errors.New(errFallbackOriginDeletion),
			},
		},
		"ErrFallbackOriginDelete": {
			reason: "We should return any errors during the delete process",
			fields: fields{
				client: fake.MockClient{
					MockDeleteCustomHostnameFallbackOrigin: func(ctx context.Context, zoneID string) error {
						return errBoom
					},
				},
			},
			args: args{
				mg: fallbackOrigin(
					withZone(zone),
					withOrigin(origin),
				),
			},
			want: want{
				err: errors.Wrap(errBoom, errFallbackOriginDeletion),
			},
		},
		"Success": {
			reason: "We should return no error when a FallbackOrigin is deleted",
			fields: fields{
				client: fake.MockClient{
					MockDeleteCustomHostnameFallbackOrigin: func(ctx context.Context, zoneID string) error {
						return nil
					},
				},
			},
			args: args{
				mg: fallbackOrigin(
					withZone(zone),
					withOrigin(origin),
				),
			},
			want: want{
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{client: tc.fields.client}
			err := e.Delete(tc.args.ctx, tc.args.mg)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ne.Delete(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
		})
	}
}
