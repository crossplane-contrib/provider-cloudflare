package filter

import (
	"context"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	"github.com/google/go-cmp/cmp"

	"github.com/crossplane-contrib/provider-cloudflare/apis/firewall/v1alpha1"
	"github.com/crossplane-contrib/provider-cloudflare/internal/clients/firewall/filter/fake"

	"github.com/pkg/errors"

	ptr "k8s.io/utils/pointer"
)

func TestLateInitialize(t *testing.T) {
	expression1 := `(http.request.uri.path ~ ".*wp-login.php" 
or http.request.uri.path ~ ".*xmlrpc.php") and ip.addr ne 172.16.22.155`

	expression2 := `(http.request.uri.path ~ ".*wp-login.php" 
or http.request.uri.path ~ ".*xmlrpc.php") and ip.addr ne 172.16.24.200`

	type args struct {
		fp *v1alpha1.FilterParameters
		f  cloudflare.Filter
	}

	type want struct {
		o  bool
		fp *v1alpha1.FilterParameters
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
			reason: "LateInit should not update already-set spec fields from a Filter",
			args: args{
				fp: &v1alpha1.FilterParameters{
					Expression:  expression1,
					Description: ptr.StringPtr("Test Description"),
					Paused:      ptr.Bool(false),
					Zone:        ptr.StringPtr("Test Zone"),
				},
				f: cloudflare.Filter{
					ID:          "372e67954025e0ba6aaa6d586b9e0b61",
					Expression:  expression2,
					Paused:      false,
					Description: "Test description - changed",
					Ref:         "SQ-101",
				},
			},
			want: want{
				o: false,
				fp: &v1alpha1.FilterParameters{
					Expression:  expression1,
					Description: ptr.StringPtr("Test Description"),
					Paused:      ptr.Bool(false),
					Zone:        ptr.StringPtr("Test Zone"),
				},
			},
		},
		"LateInitUpdate": {
			reason: "LateInit should update unset spec fields from a Filter",
			args: args{
				fp: &v1alpha1.FilterParameters{
					Expression: expression1,
					Zone:       ptr.StringPtr("Test Zone"),
				},
				f: cloudflare.Filter{
					ID:          "372e67954025e0ba6aaa6d586b9e0b61",
					Expression:  expression1,
					Paused:      false,
					Description: "Test Description",
					Ref:         "SQ-101",
				},
			},
			want: want{
				o: true,
				fp: &v1alpha1.FilterParameters{
					Expression:  expression1,
					Description: ptr.StringPtr("Test Description"),
					Paused:      ptr.Bool(false),
					Zone:        ptr.StringPtr("Test Zone"),
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := LateInitialize(tc.args.fp, tc.args.f)
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nLateInit(...): -want, +got:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.fp, tc.args.fp); diff != "" {
				t.Errorf("\n%s\nLateInit(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestUpToDate(t *testing.T) {
	expression1 := `(http.request.uri.path ~ ".*wp-login.php" 
or http.request.uri.path ~ ".*xmlrpc.php") and ip.addr ne 172.16.22.155`

	expression2 := `(http.request.uri.path ~ ".*wp-login.php" 
or http.request.uri.path ~ ".*xmlrpc.php") and ip.addr ne 172.16.24.200`

	type args struct {
		fp *v1alpha1.FilterParameters
		f  cloudflare.Filter
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
				fp: &v1alpha1.FilterParameters{},
				f:  cloudflare.Filter{},
			},
			want: want{
				o: true,
			},
		},
		"UpToDateDifferent": {
			reason: "UpToDate should return false if the spec does not match the filter",
			args: args{
				fp: &v1alpha1.FilterParameters{
					Expression:  expression1,
					Description: ptr.String("Test Description"),
					Paused:      ptr.Bool(false),
					Zone:        ptr.String("Test Zone"),
				},
				f: cloudflare.Filter{
					ID:          "372e67954025e0ba6aaa6d586b9e0b61",
					Expression:  expression2,
					Paused:      false,
					Description: "Test Description - changed",
					Ref:         "SQ-101",
				},
			},
			want: want{
				o: false,
			},
		},
		"UpToDateIdentical": {
			reason: "UpToDate should return true if the spec matches the filter",
			args: args{
				fp: &v1alpha1.FilterParameters{
					Expression:  expression1,
					Description: ptr.String("Test Description"),
					Paused:      ptr.Bool(false),
					Zone:        ptr.String("Test Zone"),
				},
				f: cloudflare.Filter{
					ID:          "372e67954025e0ba6aaa6d586b9e0b61",
					Expression:  expression1,
					Paused:      false,
					Description: "Test Description",
					Ref:         "SQ-101",
				},
			},
			want: want{
				o: true,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := UpToDate(tc.args.fp, tc.args.f)
			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("\n%s\nUpToDate(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestCreateFilter(t *testing.T) {
	errBoom := errors.New("boom")

	expression := `(http.request.uri.path ~ ".*wp-login.php" 
or http.request.uri.path ~ ".*xmlrpc.php") and ip.addr ne 172.16.22.155`

	type fields struct {
		client Client
	}

	type args struct {
		ctx context.Context
		fp  *v1alpha1.FilterParameters
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
		"CreateFilterSpecNil": {
			reason: "CreateFilter should return errSpecNil if not passed a spec",
			fields: fields{
				client: fake.MockClient{},
			},
			args: args{},
			want: want{
				err: errors.New(errSpecNil),
			},
		},
		"CreateFilter": {
			reason: "CreateFilter should return no error when creating a filter successfully",
			fields: fields{
				client: fake.MockClient{
					MockCreateFilters: func(ctx context.Context, zoneID string, firewallFilters []cloudflare.Filter) ([]cloudflare.Filter, error) {
						return []cloudflare.Filter{
							{
								ID:          "372e67954025e0ba6aaa6d586b9e0b61",
								Expression:  expression,
								Paused:      false,
								Description: "Test Description",
								Ref:         "SQ-101",
							},
						}, nil
					},
				},
			},
			args: args{
				fp: &v1alpha1.FilterParameters{
					Expression:  expression,
					Description: ptr.StringPtr("Test Description"),
					Paused:      ptr.Bool(false),
					Zone:        ptr.StringPtr("Test Zone"),
				},
			},
			want: want{
				err: nil,
			},
		},
		"CreateFilterFailed": {
			reason: "CreateFilter should return error when creating a filter fails",
			fields: fields{
				client: fake.MockClient{
					MockCreateFilters: func(ctx context.Context, zoneID string, firewallFilters []cloudflare.Filter) ([]cloudflare.Filter, error) {
						return []cloudflare.Filter{}, errBoom
					},
				},
			},
			args: args{
				fp: &v1alpha1.FilterParameters{
					Zone: ptr.String("Test Zone"),
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errCreateFilter),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := CreateFilter(tc.args.ctx, tc.fields.client, tc.args.fp)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nCreateFilter(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestUpdateFilter(t *testing.T) {
	expression := `(http.request.uri.path ~ ".*wp-login.php" 
or http.request.uri.path ~ ".*xmlrpc.php") and ip.addr ne 172.16.22.155`

	errBoom := errors.New("boom")
	type fields struct {
		client Client
	}

	type args struct {
		ctx context.Context
		id  string
		fp  *v1alpha1.FilterParameters
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
		"UpdateFilterNotFound": {
			reason: "UpdateFilter should return errFilterNotFound if the filter is not found",
			fields: fields{
				client: fake.MockClient{
					MockFilter: func(ctx context.Context, zoneID string, filterID string) (cloudflare.Filter, error) {
						return cloudflare.Filter{}, errBoom
					},
				},
			},
			args: args{
				id: "372e67954025e0ba6aaa6d586b9e0b61",
				fp: &v1alpha1.FilterParameters{
					Zone: ptr.String("Test Zone"),
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errFilterNotFound),
			},
		},
		"UpdateFilter": {
			reason: "UpdateFilter should return no error when updating a filter successfully",
			fields: fields{
				client: fake.MockClient{
					MockUpdateFilter: func(ctx context.Context, zoneID string, firewallFilter cloudflare.Filter) (cloudflare.Filter, error) {
						return cloudflare.Filter{}, nil
					},
					MockFilter: func(ctx context.Context, zoneID string, filterID string) (cloudflare.Filter, error) {
						return cloudflare.Filter{
							ID:          "372e67954025e0ba6aaa6d586b9e0b61",
							Expression:  expression,
							Paused:      false,
							Description: "Test Description",
							Ref:         "SQ-101",
						}, nil
					},
				},
			},
			args: args{
				fp: &v1alpha1.FilterParameters{
					Zone: ptr.String("Test Zone"),
				},
			},
			want: want{
				err: nil,
			},
		},
		"UpdateFilterFailed": {
			reason: "UpdateFilter should return an error if the update failed",
			fields: fields{
				client: fake.MockClient{
					MockUpdateFilter: func(ctx context.Context, zoneID string, firewallFilter cloudflare.Filter) (cloudflare.Filter, error) {
						return cloudflare.Filter{}, errBoom
					},
					MockFilter: func(ctx context.Context, zoneID string, filterID string) (cloudflare.Filter, error) {
						return cloudflare.Filter{
							ID:          "372e67954025e0ba6aaa6d586b9e0b61",
							Expression:  expression,
							Paused:      false,
							Description: "Test Description",
							Ref:         "SQ-101",
						}, nil
					},
				},
			},
			args: args{
				ctx: nil,
				id:  "372e67954025e0ba6aaa6d586b9e0b61",
				fp: &v1alpha1.FilterParameters{
					Expression:  "",
					Description: ptr.String("Test Description"),
					Paused:      ptr.Bool(false),
					Zone:        ptr.String("Test Zone"),
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errUpdateFilter),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := UpdateFilter(tc.args.ctx, tc.fields.client, tc.args.id, tc.args.fp)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nUpdateFilter(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
		})
	}
}
