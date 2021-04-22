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

package application

import (
	"context"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"github.com/benagricola/provider-cloudflare/apis/spectrum/v1alpha1"
	pcv1alpha1 "github.com/benagricola/provider-cloudflare/apis/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
	applications "github.com/benagricola/provider-cloudflare/internal/clients/applications"
	"github.com/benagricola/provider-cloudflare/internal/clients/applications/fake"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	rtfake "github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/test"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

type ApplicationModifier func(*v1alpha1.Application)

func withEdgeIPs(edgeIPs v1alpha1.SpectrumApplicationEdgeIPs) ApplicationModifier {
	return func(r *v1alpha1.Application) { r.Spec.ForProvider.EdgeIPs = &edgeIPs }
}

func withTrafficType(trafficType string) ApplicationModifier {
	return func(r *v1alpha1.Application) { r.Spec.ForProvider.TrafficType = &trafficType }
}

func withTLS(tls string) ApplicationModifier {
	return func(r *v1alpha1.Application) { r.Spec.ForProvider.TLS = &tls }
}

func withExternalName(applicationID string) ApplicationModifier {
	return func(r *v1alpha1.Application) { meta.SetExternalName(r, applicationID) }
}

func withZone(zoneID string) ApplicationModifier {
	return func(r *v1alpha1.Application) { r.Spec.ForProvider.Zone = &zoneID }
}

func Application(m ...ApplicationModifier) *v1alpha1.Application {
	cr := &v1alpha1.Application{}
	for _, f := range m {
		f(cr)
	}
	return cr
}

func TestConnect(t *testing.T) {
	mc := &test.MockClient{
		MockGet: test.NewMockGetFn(nil),
	}

	_, errGetProviderConfig := clients.GetConfig(context.Background(), mc, &rtfake.Managed{})

	type fields struct {
		kube      client.Client
		newClient func(cfg clients.Config) (applications.Client, error)
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
		"ErrNotApplication": {
			reason: "An error should be returned if the managed resource is not a *Application",
			args: args{
				mg: nil,
			},
			want: errors.New(errNotApplication),
		},
		"ErrGetConfig": {
			reason: "Any errors from GetConfig should be wrapped",
			fields: fields{
				kube: mc,
			},
			args: args{
				mg: &v1alpha1.Application{
					Spec: v1alpha1.ApplicationSpec{
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
				newClient: applications.NewClient,
			},
			args: args{
				mg: &v1alpha1.Application{
					Spec: v1alpha1.ApplicationSpec{
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
		client applications.Client
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
		"ErrNotApplication": {
			reason: "An error should be returned if the managed resource is not a *Application",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotApplication),
			},
		},
		"ErrNoApplication": {
			reason: "We should return ResourceExists: false when no external name is set",
			fields: fields{
				client: fake.MockClient{},
			},
			args: args{
				mg: &v1alpha1.Application{},
			},
			want: want{
				o: managed.ExternalObservation{ResourceExists: false},
			},
		},
		"ErrApplicationLookup": {
			reason: "We should return an empty observation and an error if the API returned an error",
			fields: fields{
				client: fake.MockClient{
					MockSpectrumApplication: func(ctx context.Context, zoneID string, ApplicationID string) (cloudflare.SpectrumApplication, error) {
						return cloudflare.SpectrumApplication{}, errBoom
					},
				},
			},
			args: args{
				mg: Application(
					withExternalName("1234beef"),
					withZone("foo.com"),
				),
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errBoom, errApplicationLookup),
			},
		},
		"ErrApplicationNoZone": {
			reason: "We should return an error if the Application does not have a zone",
			fields: fields{
				client: fake.MockClient{
					MockSpectrumApplication: func(ctx context.Context, zoneID string, ApplicationID string) (cloudflare.SpectrumApplication, error) {
						return cloudflare.SpectrumApplication{}, errBoom
					},
				},
			},
			args: args{
				mg: Application(
					withExternalName("1234beef"),
				),
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.New(errApplicationNoZone),
			},
		},
		"ErrApplicationNotFound": {
			reason: "We should return an error if the Application is not found (deleted on CF side)",
			fields: fields{
				client: fake.MockClient{
					MockSpectrumApplication: func(ctx context.Context, zoneID string, ApplicationID string) (cloudflare.SpectrumApplication, error) {
						return cloudflare.SpectrumApplication{}, errors.New("10006")
					},
				},
			},
			args: args{
				mg: Application(
					withExternalName("1234beef"),
					withZone("foo.com"),
				),
			},
			want: want{
				o:   managed.ExternalObservation{ResourceExists: false},
				err: nil,
			},
		},
		"Success": {
			reason: "We should return ResourceExists: true and no error when a Application is found",
			fields: fields{
				client: fake.MockClient{
					MockSpectrumApplication: func(ctx context.Context, zoneID, ApplicationID string) (cloudflare.SpectrumApplication, error) {
						return cloudflare.SpectrumApplication{
							ID: ApplicationID,
						}, nil
					},
				},
			},
			args: args{
				mg: Application(withExternalName("1234beef"), withZone("foo.com")),
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
		client applications.Client
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
		"ErrNotApplication": {
			reason: "An error should be returned if the managed resource is not a *Application",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotApplication),
			},
		},
		"ErrApplicationCreate": {
			reason: "We should return any errors during the create process",
			fields: fields{
				client: fake.MockClient{
					MockCreateSpectrumApplication: func(ctx context.Context, zoneID string, appDetails cloudflare.SpectrumApplication) (cloudflare.SpectrumApplication, error) {
						return cloudflare.SpectrumApplication{}, errBoom
					},
				},
			},
			args: args{
				mg: Application(
					withExternalName("1234beef"),
					withZone("foo.com"),
					withTLS("full"),
					withTrafficType("https"),
					withEdgeIPs(v1alpha1.SpectrumApplicationEdgeIPs{
						IPs: []string{"192.0.2.2", "2001:db8::1"},
					}),
				),
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errBoom, errApplicationCreation),
			},
		},
		"ErrApplicationNoZone": {
			reason: "We should return an error if the Application does not have a zone",
			fields: fields{
				client: fake.MockClient{
					MockCreateSpectrumApplication: func(ctx context.Context, zoneID string, appDetails cloudflare.SpectrumApplication) (cloudflare.SpectrumApplication, error) {
						return cloudflare.SpectrumApplication{}, errBoom
					},
				},
			},
			args: args{
				mg: Application(
					withExternalName("1234beef"),
					withTLS("full"),
					withTrafficType("https"),
					withEdgeIPs(v1alpha1.SpectrumApplicationEdgeIPs{
						IPs: []string{"192.0.2.2", "2001:db8::1"},
					}),
				),
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errors.New(errApplicationNoZone), errApplicationCreation),
			},
		},
		"Success": {
			reason: "We should return ExternalNameAssigned: true and no error when a Application is created",
			fields: fields{
				client: fake.MockClient{
					MockCreateSpectrumApplication: func(ctx context.Context, zoneID string, appDetails cloudflare.SpectrumApplication) (cloudflare.SpectrumApplication, error) {
						return appDetails, nil
					},
				},
			},
			args: args{
				mg: Application(
					withExternalName("1234beef"),
					withZone("foo.com"),
					withTLS("full"),
					withTrafficType("https"),
					withEdgeIPs(v1alpha1.SpectrumApplicationEdgeIPs{
						IPs: []string{"192.0.2.2", "2001:db8::1"},
					}),
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
		client applications.Client
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
		"ErrNotApplication": {
			reason: "An error should be returned if the managed resource is not a *Application",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotApplication),
			},
		},
		"ErrNoApplication": {
			reason: "We should return an error when no external name is set",
			fields: fields{
				client: fake.MockClient{
					MockUpdateSpectrumApplication: func(ctx context.Context, zoneID, ApplicationID string, appDetails cloudflare.SpectrumApplication) (cloudflare.SpectrumApplication, error) {
						return appDetails, nil
					},
				},
			},
			args: args{
				mg: Application(
					withZone("foo.com"),
					withTLS("full"),
					withTrafficType("https"),
					withEdgeIPs(v1alpha1.SpectrumApplicationEdgeIPs{
						IPs: []string{"192.0.2.2", "2001:db8::1"},
					}),
				),
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.New(errApplicationUpdate),
			},
		},
		"ErrApplicationUpdate": {
			reason: "We should return any errors during the update process",
			fields: fields{
				client: fake.MockClient{
					MockUpdateSpectrumApplication: func(ctx context.Context, zoneID, ApplicationID string, appDetails cloudflare.SpectrumApplication) (cloudflare.SpectrumApplication, error) {
						return cloudflare.SpectrumApplication{}, errBoom
					},
				},
			},
			args: args{
				mg: Application(
					withExternalName("1234beef"),
					withZone("foo.com"),
					withTLS("full"),
					withTrafficType("https"),
					withEdgeIPs(v1alpha1.SpectrumApplicationEdgeIPs{
						IPs: []string{"192.0.2.2", "2001:db8::1"},
					}),
				),
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errBoom, errApplicationUpdate),
			},
		},
		"Success": {
			reason: "We should return no error when a zone is updated",
			fields: fields{
				client: fake.MockClient{
					MockSpectrumApplication: func(ctx context.Context, zoneID, ApplicationID string) (cloudflare.SpectrumApplication, error) {
						return cloudflare.SpectrumApplication{
							ID: ApplicationID,
						}, nil
					},
					MockUpdateSpectrumApplication: func(ctx context.Context, zoneID, ApplicationID string, appDetails cloudflare.SpectrumApplication) (cloudflare.SpectrumApplication, error) {
						return appDetails, nil
					},
				},
			},
			args: args{
				mg: Application(
					withExternalName("1234beef"),
					withZone("foo.com"),
					withTLS("full"),
					withTrafficType("https"),
					withEdgeIPs(v1alpha1.SpectrumApplicationEdgeIPs{
						IPs: []string{"192.0.2.2", "2001:db8::1"},
					}),
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
		client applications.Client
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
		"ErrNotApplication": {
			reason: "An error should be returned if the managed resource is not a *Application",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotApplication),
			},
		},
		"ErrNoApplication": {
			reason: "We should return an error when no external name is set",
			fields: fields{
				client: fake.MockClient{
					MockDeleteSpectrumApplication: func(ctx context.Context, zoneID, ApplicationID string) error {
						return nil
					},
				},
			},
			args: args{
				mg: Application(
					withZone("foo.com"),
					withTLS("full"),
					withTrafficType("https"),
					withEdgeIPs(v1alpha1.SpectrumApplicationEdgeIPs{
						IPs: []string{"192.0.2.2", "2001:db8::1"},
					}),
				),
			},
			want: want{
				err: errors.New(errApplicationDeletion),
			},
		},
		"ErrApplicationDelete": {
			reason: "We should return any errors during the delete process",
			fields: fields{
				client: fake.MockClient{
					MockDeleteSpectrumApplication: func(ctx context.Context, zoneID, ApplicationID string) error {
						return errBoom
					},
				},
			},
			args: args{
				mg: Application(
					withExternalName("1234beef"),
					withZone("foo.com"),
					withTLS("full"),
					withTrafficType("https"),
					withEdgeIPs(v1alpha1.SpectrumApplicationEdgeIPs{
						IPs: []string{"192.0.2.2", "2001:db8::1"},
					}),
				),
			},
			want: want{
				err: errors.Wrap(errBoom, errApplicationDeletion),
			},
		},
		"Success": {
			reason: "We should return no error when a Application is deleted",
			fields: fields{
				client: fake.MockClient{
					MockDeleteSpectrumApplication: func(ctx context.Context, zoneID, ApplicationID string) error {
						return nil
					},
				},
			},
			args: args{
				mg: Application(
					withExternalName("1234beef"),
					withZone("foo.com"),
					withTLS("full"),
					withTrafficType("https"),
					withEdgeIPs(v1alpha1.SpectrumApplicationEdgeIPs{
						IPs: []string{"192.0.2.2", "2001:db8::1"},
					}),
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
