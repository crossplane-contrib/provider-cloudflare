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

package customhostname

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	ptr "k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	rtfake "github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	"github.com/crossplane-contrib/provider-cloudflare/apis/sslsaas/v1alpha1"
	pcv1alpha1 "github.com/crossplane-contrib/provider-cloudflare/apis/v1alpha1"
	clients "github.com/crossplane-contrib/provider-cloudflare/internal/clients"
	customhostnames "github.com/crossplane-contrib/provider-cloudflare/internal/clients/sslsaas/customhostnames"
	"github.com/crossplane-contrib/provider-cloudflare/internal/clients/sslsaas/customhostnames/fake"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

type customHostnameModifier func(*v1alpha1.CustomHostname)

func withExternalName(customHostnameModifier string) customHostnameModifier {
	return func(r *v1alpha1.CustomHostname) { meta.SetExternalName(r, customHostnameModifier) }
}

func withZone(zoneID string) customHostnameModifier {
	return func(r *v1alpha1.CustomHostname) { r.Spec.ForProvider.Zone = &zoneID }
}

func withHostname(hostname string) customHostnameModifier {
	return func(r *v1alpha1.CustomHostname) { r.Spec.ForProvider.Hostname = hostname }
}

func withSSLSettings(settings *v1alpha1.CustomHostnameSSL) customHostnameModifier {
	return func(r *v1alpha1.CustomHostname) { r.Spec.ForProvider.SSL = *settings }
}

func customHostname(m ...customHostnameModifier) *v1alpha1.CustomHostname {
	cr := &v1alpha1.CustomHostname{}
	for _, f := range m {
		f(cr)
	}
	return cr
}

const (
	externalName = "external-name"
	zone         = "zone.com"
	hostname     = "host.zone.com"
)

var sslSettings = &v1alpha1.CustomHostnameSSL{
	Method:            ptr.StringPtr("txt"),
	Type:              ptr.StringPtr("dv"),
	Wildcard:          ptr.BoolPtr(true),
	CustomCertificate: ptr.StringPtr("invalid cert"),
	CustomKey:         ptr.StringPtr("invalid key"),
	Settings: v1alpha1.CustomHostnameSSLSettings{
		HTTP2:         ptr.StringPtr("on"),
		TLS13:         ptr.StringPtr("on"),
		MinTLSVersion: ptr.StringPtr("1.2"),
		Ciphers: []string{
			"ECDHE-ECDSA-AES128-GCM-SHA256",
			"AEAD-CHACHA20-POLY1305-SHA256",
		},
	},
}

func TestConnect(t *testing.T) {
	mc := &test.MockClient{
		MockGet: test.NewMockGetFn(nil),
	}

	_, errGetProviderConfig := clients.GetConfig(context.Background(), mc, &rtfake.Managed{})

	type fields struct {
		kube      client.Client
		newClient func(cfg clients.Config, hc *http.Client) (customhostnames.Client, error)
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
		"ErrNotCustomHostname": {
			reason: "An error should be returned if the managed resource is not a *CustomHostname",
			args: args{
				mg: nil,
			},
			want: errors.New(errNotCustomHostname),
		},
		"ErrGetConfig": {
			reason: "Any errors from GetConfig should be wrapped",
			fields: fields{
				kube: mc,
			},
			args: args{
				mg: &v1alpha1.CustomHostname{
					Spec: v1alpha1.CustomHostnameSpec{
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
				newClient: customhostnames.NewClient,
			},
			args: args{
				mg: &v1alpha1.CustomHostname{
					Spec: v1alpha1.CustomHostnameSpec{
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
			nc := func(cfg clients.Config) (customhostnames.Client, error) {
				return tc.fields.newClient(cfg, nil)
			}
			e := &connector{kube: tc.fields.kube, newCloudflareClientFn: nc}
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
		client customhostnames.Client
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
		"errNotCustomHostname": {
			reason: "An error should be returned if the managed resource is not a *CustomHostname",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotCustomHostname),
			},
		},
		"ErrNoCustomHostname": {
			reason: "We should return ResourceExists: false when no external name is set",
			fields: fields{
				client: fake.MockClient{},
			},
			args: args{
				mg: customHostname(withZone(zone)),
			},
			want: want{
				o: managed.ExternalObservation{ResourceExists: false},
			},
		},
		"ErrCustomHostnameLookup": {
			reason: "We should return an empty observation and an error if the API returned an error",
			fields: fields{
				client: fake.MockClient{
					MockCustomHostname: func(ctx context.Context, zoneID string, customHostnameID string) (cloudflare.CustomHostname, error) {
						return cloudflare.CustomHostname{}, errBoom
					},
				},
			},
			args: args{
				mg: customHostname(
					withZone(zone),
					withExternalName(externalName),
				),
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errBoom, errCustomHostnameLookup),
			},
		},
		"ErrCustomHostnameNoZone": {
			reason: "We should return an error if the CustomHostname does not have a zone",
			fields: fields{
				client: fake.MockClient{
					MockCustomHostname: func(ctx context.Context, zoneID string, customHostnameID string) (cloudflare.CustomHostname, error) {
						return cloudflare.CustomHostname{}, errBoom
					},
				},
			},
			args: args{
				mg: &v1alpha1.CustomHostname{},
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.New(errCustomHostnameNoZone),
			},
		},
		"Success": {
			reason: "We should return ResourceExists: true and no error when a CustomHostname is found",
			fields: fields{
				client: fake.MockClient{
					MockCustomHostname: func(ctx context.Context, zoneID, customHostnameID string) (cloudflare.CustomHostname, error) {
						return cloudflare.CustomHostname{}, nil
					},
				},
			},
			args: args{
				mg: customHostname(
					withZone(zone),
					withExternalName(externalName),
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
		client customhostnames.Client
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
		"ErrNotCustomHostname": {
			reason: "An error should be returned if the managed resource is not a *CustomHostname",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotCustomHostname),
			},
		},
		"ErrCustomHostnameCreate": {
			reason: "We should return any errors during the create process",
			fields: fields{
				client: fake.MockClient{
					MockCreateCustomHostname: func(ctx context.Context, zoneID string, rr cloudflare.CustomHostname) (*cloudflare.CustomHostnameResponse, error) {
						return nil, errBoom
					},
				},
			},
			args: args{
				mg: customHostname(
					withExternalName(externalName),
					withZone(zone),
					withHostname(hostname),
					withSSLSettings(sslSettings),
				),
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errBoom, errCustomHostnameCreation),
			},
		},
		"Success": {
			reason: "We should return ExternalNameAssigned: true and no error when a CustomHostname is created",
			fields: fields{
				client: fake.MockClient{
					MockCreateCustomHostname: func(ctx context.Context, zoneID string, rr cloudflare.CustomHostname) (*cloudflare.CustomHostnameResponse, error) {
						return &cloudflare.CustomHostnameResponse{
							Result: rr,
						}, nil
					},
				},
			},
			args: args{
				mg: customHostname(
					withZone(zone),
					withHostname(hostname),
					withSSLSettings(sslSettings),
				),
			},
			want: want{
				o: managed.ExternalCreation{
					ExternalNameAssigned: true,
				},
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
		client customhostnames.Client
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
		"ErrNotCustomHostname": {
			reason: "An error should be returned if the managed resource is not a *CustomHostname",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotCustomHostname),
			},
		}, "ErrNoCustomHostname": {
			reason: "We should return an error when no external name is set",
			fields: fields{
				client: fake.MockClient{
					MockUpdateCustomHostname: func(ctx context.Context, zoneID, CustomHostnameID string, rr cloudflare.CustomHostname) (*cloudflare.CustomHostnameResponse, error) {
						return &cloudflare.CustomHostnameResponse{}, nil
					},
				},
			},
			args: args{
				mg: customHostname(
					withZone(zone),
				),
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.New(errCustomHostnameUpdate),
			},
		},
		"ErrCustomHostnameUpdate": {
			reason: "We should return any errors during the update process",
			fields: fields{
				client: fake.MockClient{
					MockUpdateCustomHostname: func(ctx context.Context, zoneID, CustomHostnameID string, rr cloudflare.CustomHostname) (*cloudflare.CustomHostnameResponse, error) {
						return &cloudflare.CustomHostnameResponse{}, errBoom
					},
				},
			},
			args: args{
				mg: customHostname(
					withExternalName(externalName),
					withZone(zone),
					withHostname(hostname),
					withSSLSettings(sslSettings),
				),
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errBoom, errCustomHostnameUpdate),
			},
		},
		"Success": {
			reason: "We should return no error when a CustomHostname is updated",
			fields: fields{
				client: fake.MockClient{
					MockCustomHostname: func(ctx context.Context, zoneID string, CustomHostnameID string) (cloudflare.CustomHostname, error) {
						return cloudflare.CustomHostname{
							ID: zoneID,
						}, nil
					},
					MockUpdateCustomHostname: func(ctx context.Context, zoneID, CustomHostnameID string, rr cloudflare.CustomHostname) (*cloudflare.CustomHostnameResponse, error) {
						return &cloudflare.CustomHostnameResponse{}, nil
					},
				},
			},
			args: args{
				mg: customHostname(
					withExternalName(externalName),
					withZone(zone),
					withHostname(hostname),
					withSSLSettings(sslSettings),
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
		client customhostnames.Client
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
		"ErrNotCustomHostname": {
			reason: "An error should be returned if the managed resource is not a *CustomHostname",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotCustomHostname),
			},
		},
		"ErrNoCustomHostname": {
			reason: "We should return an error when no external name is set",
			fields: fields{
				client: fake.MockClient{
					MockDeleteCustomHostname: func(ctx context.Context, zoneID, CustomHostnameID string) error {
						return nil
					},
				},
			},
			args: args{
				mg: customHostname(
					withZone(zone),
				),
			},
			want: want{
				err: errors.New(errCustomHostnameDeletion),
			},
		},
		"ErrCustomHostnameDelete": {
			reason: "We should return any errors during the delete process",
			fields: fields{
				client: fake.MockClient{
					MockDeleteCustomHostname: func(ctx context.Context, zoneID, CustomHostnameID string) error {
						return errBoom
					},
				},
			},
			args: args{
				mg: customHostname(
					withExternalName(externalName),
					withZone(zone),
				),
			},
			want: want{
				err: errors.Wrap(errBoom, errCustomHostnameDeletion),
			},
		},
		"Success": {
			reason: "We should return no error when a CustomHostname is deleted",
			fields: fields{
				client: fake.MockClient{
					MockDeleteCustomHostname: func(ctx context.Context, zoneID, CustomHostnameID string) error {
						return nil
					},
				},
			},
			args: args{
				mg: customHostname(
					withExternalName(externalName),
					withZone(zone),
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
