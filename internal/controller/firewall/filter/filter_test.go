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

package filter

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane-contrib/provider-cloudflare/apis/firewall/v1alpha1"

	"github.com/crossplane/crossplane-runtime/pkg/meta"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"github.com/crossplane-contrib/provider-cloudflare/internal/clients/firewall/filter"
	"github.com/crossplane-contrib/provider-cloudflare/internal/clients/firewall/filter/fake"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	rtfake "github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	corev1 "k8s.io/api/core/v1"

	pcv1alpha1 "github.com/crossplane-contrib/provider-cloudflare/apis/v1alpha1"
	clients "github.com/crossplane-contrib/provider-cloudflare/internal/clients"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

type filterModifier func(*v1alpha1.Filter)

func withExpression(expression string) filterModifier {
	return func(r *v1alpha1.Filter) { r.Spec.ForProvider.Expression = expression }
}

func withDescription(description string) filterModifier {
	return func(r *v1alpha1.Filter) { r.Spec.ForProvider.Description = &description }
}

func withPaused(paused bool) filterModifier {
	return func(r *v1alpha1.Filter) { r.Spec.ForProvider.Paused = &paused }
}

func withZone(zone string) filterModifier {
	return func(r *v1alpha1.Filter) { r.Spec.ForProvider.Zone = &zone }
}

func withExternalName(filterID string) filterModifier {
	return func(r *v1alpha1.Filter) { meta.SetExternalName(r, filterID) }
}

func filterBuild(m ...filterModifier) *v1alpha1.Filter {
	cr := &v1alpha1.Filter{}
	for _, f := range m {
		f(cr)
	}
	return cr
}
func TestObserve(t *testing.T) {
	errBoom := errors.New("boom")

	type fields struct {
		client filter.Client
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
		"ErrNotFilter": {
			reason: "An error should be returned if the managed resource is not a *Filter",
			fields: fields{
				client: fake.MockClient{},
			},
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotFilter),
			},
		},
		"ErrNoFilter": {
			reason: "We should return ResourceExists: false when no external name is set",
			fields: fields{
				client: fake.MockClient{},
			},
			args: args{
				mg: &v1alpha1.Filter{},
			},
			want: want{
				o: managed.ExternalObservation{ResourceExists: false},
			},
		},
		"ErrFilterLookup": {
			reason: "We should return an empty observation and an error if the API returned an error",
			fields: fields{
				client: fake.MockClient{
					MockFilter: func(ctx context.Context, zoneID string, filterID string) (cloudflare.Filter, error) {
						return cloudflare.Filter{}, errBoom
					},
				},
			},
			args: args{
				mg: filterBuild(
					withExternalName("372e67954025e0ba6aaa6d586b9e0b61"),
					withZone("Test Zone"),
				),
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.Wrap(errBoom, errFilterLookup),
			},
		},
		"ErrFilterNoZone": {
			reason: "We should return an error if the filter does not have a zone",
			fields: fields{
				client: fake.MockClient{},
			},
			args: args{
				mg: filterBuild(
					withExternalName("372e67954025e0ba6aaa6d586b9e0b61"),
				),
			},
			want: want{
				o:   managed.ExternalObservation{},
				err: errors.New(errNoZone),
			},
		},
		"Success": {
			reason: "We should return ResourceExists: true and no error when a filter is found",
			fields: fields{
				client: fake.MockClient{
					MockCreateFilters: func(ctx context.Context, zoneID string, firewallFilters []cloudflare.Filter) ([]cloudflare.Filter, error) {
						return []cloudflare.Filter{{
							ID:          "372e67954025e0ba6aaa6d586b9e0b61",
							Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100",
							Paused:      false,
							Description: "Test Description",
							Ref:         "SQ-100",
						}}, nil
					},
					MockFilter: func(ctx context.Context, zoneID string, filterID string) (cloudflare.Filter, error) {
						return cloudflare.Filter{
							ID:          "372e67954025e0ba6aaa6d586b9e0b61",
							Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100",
							Paused:      false,
							Description: "Test Description",
							Ref:         "SQ-100",
						}, nil
					},
				},
			},
			args: args{
				mg: filterBuild(
					withExternalName("372e67954025e0ba6aaa6d586b9e0b61"),
					withExpression("http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100"),
					withDescription("Test Description"),
					withPaused(false),
					withZone("Test Zone"),
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
		client filter.Client
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
		"ErrNotFilter": {
			reason: "An error should be returned if the managed resource is not a Filter",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotFilter),
			},
		},
		"ErrFilterCreate": {
			reason: "We should return any errors during the create process",
			fields: fields{
				client: fake.MockClient{
					MockCreateFilters: func(ctx context.Context, zoneID string, firewallFilters []cloudflare.Filter) ([]cloudflare.Filter, error) {
						return []cloudflare.Filter{{}}, errBoom
					},
				},
			},
			args: args{
				mg: filterBuild(
					withExternalName("372e67954025e0ba6aaa6d586b9e0b61"),
					withExpression("http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100"),
					withDescription("Test Description"),
					withPaused(false),
					withZone("Test Zone"),
				),
			},
			want: want{
				o:   managed.ExternalCreation{},
				err: errors.Wrap(errors.Wrap(errBoom, "error creating filter"), errFilterCreation),
			},
		},
		"Success": {
			reason: "We should return ExternalNameAssigned: true and no error when a record is created",
			fields: fields{
				client: fake.MockClient{
					MockCreateFilters: func(ctx context.Context, zoneID string, firewallFilters []cloudflare.Filter) ([]cloudflare.Filter, error) {
						return []cloudflare.Filter{{
							ID:          "372e67954025e0ba6aaa6d586b9e0b61",
							Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100",
							Paused:      false,
							Description: "Test Description",
							Ref:         "SQ-100",
						}}, nil
					},
				},
			},
			args: args{
				mg: filterBuild(
					withExternalName("372e67954025e0ba6aaa6d586b9e0b61"),
					withExpression("http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100"),
					withDescription("Test Description"),
					withPaused(false),
					withZone("Test Zone"),
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

func TestConnect(t *testing.T) {
	mc := &test.MockClient{
		MockGet: test.NewMockGetFn(nil),
	}

	_, errGetProviderConfig := clients.GetConfig(context.Background(), mc, &rtfake.Managed{})

	type fields struct {
		kube      client.Client
		newClient func(cfg clients.Config, hc *http.Client) (filter.Client, error)
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
		"ErrNotFilter": {
			reason: "An error should be returned if the managed resource is not a Filter",
			args: args{
				mg: nil,
			},
			want: errors.New(errNotFilter),
		},
		"ErrGetConfig": {
			reason: "Any errors from GetConfig should be wrapped",
			fields: fields{
				kube: mc,
			},
			args: args{
				mg: &v1alpha1.Filter{
					Spec: v1alpha1.FilterSpec{
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
				newClient: filter.NewClient,
			},
			args: args{
				mg: &v1alpha1.Filter{
					Spec: v1alpha1.FilterSpec{
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
			nc := func(cfg clients.Config) (filter.Client, error) {
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

func TestUpdate(t *testing.T) {
	errBoom := errors.New("boom")

	type fields struct {
		client filter.Client
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
		"ErrNotFilter": {
			reason: "An error should be returned if the managed resource is not a Filter",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotFilter),
			},
		}, "ErrNoFilter": {
			reason: "We should return an error when no external name is set",
			fields: fields{
				client: fake.MockClient{},
			},
			args: args{
				mg: filterBuild(
					withExpression("http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100"),
					withDescription("Test Description"),
					withPaused(false),
					withZone("Test Zone"),
				),
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.New(errFilterUpdate),
			},
		}, "ErrNoZone": {
			reason: "We should return an error when no Zone is set",
			fields: fields{
				client: fake.MockClient{},
			},
			args: args{
				mg: filterBuild(
					withExternalName("372e67954025e0ba6aaa6d586b9e0b61"),
					withExpression("http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100"),
					withDescription("Test Description"),
					withPaused(false),
				),
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errors.New(errNoZone), errFilterUpdate),
			},
		}, "ErrFilterUpdate": {
			reason: "We should return any errors during the update process",
			fields: fields{
				client: fake.MockClient{
					MockUpdateFilter: func(ctx context.Context, zoneID string, firewallFilter cloudflare.Filter) (cloudflare.Filter, error) {
						return cloudflare.Filter{}, errBoom
					},
					MockFilter: func(ctx context.Context, zoneID string, filterID string) (cloudflare.Filter, error) {
						return cloudflare.Filter{
							ID:          "372e67954025e0ba6aaa6d586b9e0b61",
							Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100",
							Paused:      false,
							Description: "Test Description",
							Ref:         "SQ-100",
						}, nil
					},
				},
			},
			args: args{
				mg: filterBuild(
					withExternalName("372e67954025e0ba6aaa6d586b9e0b61"),
					withExpression("http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100"),
					withDescription("Test Description"),
					withPaused(false),
					withZone("Test Zone"),
				),
			},
			want: want{
				o:   managed.ExternalUpdate{},
				err: errors.Wrap(errors.Wrap(errBoom, "error updating filter"), errFilterUpdate),
			},
		},
		"Success": {
			reason: "We should return no error when a filter is updated successfully",
			fields: fields{
				client: fake.MockClient{
					MockUpdateFilter: func(ctx context.Context, zoneID string, firewallFilter cloudflare.Filter) (cloudflare.Filter, error) {
						return cloudflare.Filter{
							ID:          "372e67954025e0ba6aaa6d586b9e0b61",
							Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100",
							Paused:      false,
							Description: "Test Description",
							Ref:         "SQ-100",
						}, nil
					},
					MockFilter: func(ctx context.Context, zoneID string, filterID string) (cloudflare.Filter, error) {
						return cloudflare.Filter{
							ID:          "372e67954025e0ba6aaa6d586b9e0b61",
							Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100",
							Paused:      false,
							Description: "Test Description",
							Ref:         "SQ-100",
						}, nil
					},
				},
			},
			args: args{
				mg: filterBuild(
					withExternalName("372e67954025e0ba6aaa6d586b9e0b61"),
					withExpression("http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100"),
					withDescription("Test Description"),
					withPaused(false),
					withZone("Test Zone"),
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
		client filter.Client
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
		"ErrNotFilter": {
			reason: "An error should be returned if the managed resource is not a Filter",
			args: args{
				mg: nil,
			},
			want: want{
				err: errors.New(errNotFilter),
			},
		},
		"ErrNoFilter": {
			reason: "We should return an error when no external name is set",
			fields: fields{
				client: fake.MockClient{},
			},
			args: args{
				mg: filterBuild(
					withExpression("http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100"),
					withDescription("Test Description"),
					withPaused(false),
					withZone("Test Zone"),
				),
			},
			want: want{
				err: errors.New(errFilterDeletion),
			},
		},
		"ErrFilterDelete": {
			reason: "We should return any errors during the delete process",
			fields: fields{
				client: fake.MockClient{
					MockDeleteFilter: func(ctx context.Context, zoneID string, firewallFilterID string) error {
						return errBoom
					},
				},
			},
			args: args{
				mg: filterBuild(
					withExternalName("372e67954025e0ba6aaa6d586b9e0b61"),
					withExpression("http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100"),
					withDescription("Test Description"),
					withPaused(false),
					withZone("Test Zone"),
				),
			},
			want: want{
				err: errors.Wrap(errBoom, errFilterDeletion),
			},
		},
		"Success": {
			reason: "We should return no error when a filter is deleted",
			fields: fields{
				client: fake.MockClient{
					MockDeleteFilter: func(ctx context.Context, zoneID string, firewallFilterID string) error {
						return nil
					},
				},
			},
			args: args{
				mg: filterBuild(
					withExternalName("372e67954025e0ba6aaa6d586b9e0b61"),
					withExpression("http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100"),
					withDescription("Test Description"),
					withPaused(false),
					withZone("Test Zone"),
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
