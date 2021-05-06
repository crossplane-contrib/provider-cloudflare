package rule

import (
	"context"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"

	"github.com/benagricola/provider-cloudflare/apis/firewall/v1alpha1"
	"github.com/benagricola/provider-cloudflare/internal/clients/firewall/rule/fake"
	"github.com/google/go-cmp/cmp"

	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/pkg/test"

	ptr "k8s.io/utils/pointer"
)

func TestLateInitialize(t *testing.T) {
	type args struct {
		rp *v1alpha1.RuleParameters
		r  cloudflare.FirewallRule
	}

	type want struct {
		o  bool
		rp *v1alpha1.RuleParameters
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
			reason: "LateInit should not update already-set spec fields from a Rule",
			args: args{
				rp: &v1alpha1.RuleParameters{
					Action:         "allow",
					Description:    ptr.String("Test Description - Original"),
					Priority:       ptr.Int32(4),
					Paused:         ptr.BoolPtr(false),
					BypassProducts: []v1alpha1.RuleBypassProduct{"waf"},
				},
				r: cloudflare.FirewallRule{
					Action:      "allow",
					Description: "Test Description - Changed",
					Priority:    ptr.Int32(1),
					Paused:      true,
					Products:    []string{"rateLimit"},
				},
			},
			want: want{
				o: false,
				rp: &v1alpha1.RuleParameters{
					Action:         "allow",
					Description:    ptr.String("Test Description - Original"),
					Priority:       ptr.Int32(4),
					Paused:         ptr.BoolPtr(false),
					BypassProducts: []v1alpha1.RuleBypassProduct{"waf"},
				},
			},
		},
		"LateInitUpdate": {
			reason: "LateInit should update unset spec fields from a Rule",
			args: args{
				rp: &v1alpha1.RuleParameters{
					Action: "allow",
					Filter: ptr.String("372e67954025e0ba6aaa6d586b9e0b61"),
				},
				r: cloudflare.FirewallRule{
					ID:          "f2d427378e7542acb295380d352e2ebd",
					Paused:      false,
					Description: "Test Description",
					Action:      "allow",
					Priority:    1.0,
					Filter: cloudflare.Filter{
						ID:          "372e67954025e0ba6aaa6d586b9e0b61",
						Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.155",
						Paused:      false,
						Description: "Test Description",
						Ref:         "SQ-100",
					},
					Products:   []string{"waf"},
					CreatedOn:  time.Time{},
					ModifiedOn: time.Time{},
				},
			},
			want: want{
				o: true,
				rp: &v1alpha1.RuleParameters{
					Action:         "allow",
					BypassProducts: []v1alpha1.RuleBypassProduct{"waf"},
					Description:    ptr.StringPtr("Test Description"),
					Filter:         ptr.String("372e67954025e0ba6aaa6d586b9e0b61"),
					Paused:         ptr.BoolPtr(false),
					Priority:       ptr.Int32(1),
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
		rp *v1alpha1.RuleParameters
		r  cloudflare.FirewallRule
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
				rp: &v1alpha1.RuleParameters{},
				r:  cloudflare.FirewallRule{},
			},
			want: want{
				o: true,
			},
		},
		"UpToDateDifferent": {
			reason: "UpToDate should return false if the spec does not match the record",
			args: args{
				rp: &v1alpha1.RuleParameters{
					Action:         "allow",
					BypassProducts: []v1alpha1.RuleBypassProduct{"waf"},
					Description:    ptr.StringPtr("Test Description - Original"),
					Filter:         ptr.StringPtr("372e67954025e0ba6aaa6d586b9e0b61"),
					Paused:         ptr.BoolPtr(false),
					Priority:       ptr.Int32(1),
					Zone:           ptr.StringPtr("Test Zone"),
				},
				r: cloudflare.FirewallRule{
					Action:      "allow",
					Description: "Test Description",
					Filter: cloudflare.Filter{
						ID:          "372e67954025e0ba6aaa6d586b9e0b61",
						Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.155",
						Paused:      false,
						Description: "Test description - Changed",
						Ref:         "SQ-100",
					},
					Paused:   true,
					Priority: 2.0,
					Products: []string{"rateLimit"},
				},
			},
			want: want{
				o: false,
			},
		},
		"UpToDateIdentical": {
			reason: "UpToDate should return true if the spec matches the record",
			args: args{
				rp: &v1alpha1.RuleParameters{
					Action:         "allow",
					BypassProducts: []v1alpha1.RuleBypassProduct{"waf"},
					Description:    ptr.StringPtr("Test Description"),
					Filter:         ptr.StringPtr("372e67954025e0ba6aaa6d586b9e0b61"),
					Paused:         ptr.BoolPtr(false),
					Priority:       ptr.Int32(1),
					Zone:           ptr.StringPtr("Test Zone"),
				},
				r: cloudflare.FirewallRule{
					Action:      "allow",
					Description: "Test Description",
					Filter: cloudflare.Filter{
						ID:          "372e67954025e0ba6aaa6d586b9e0b61",
						Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.155",
						Paused:      false,
						Description: "Test description",
						Ref:         "SQ-100",
					},
					Priority: 1.0,
					Products: []string{"waf"},
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

func TestCreateRule(t *testing.T) {
	errBoom := errors.New("boom")
	type fields struct {
		client Client
	}

	type args struct {
		ctx context.Context
		rp  *v1alpha1.RuleParameters
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
		"CreateRuleSpecNil": {
			reason: "CreateRule should return errSpecNil if not passed a spec",
			fields: fields{
				client: fake.MockClient{},
			},
			args: args{},
			want: want{
				err: errors.New(errSpecNil),
			},
		},
		"CreateRule": {
			reason: "CreateRule should return no error when creating a rule successfully",
			fields: fields{
				client: fake.MockClient{
					MockCreateFirewallRules: func(ctx context.Context, zoneID string, rr []cloudflare.FirewallRule) ([]cloudflare.FirewallRule, error) {
						return []cloudflare.FirewallRule{
							{
								Action:      "allow",
								Description: "Test Description",
								Filter: cloudflare.Filter{
									ID:          "372e67954025e0ba6aaa6d586b9e0b61",
									Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100",
									Paused:      false,
									Description: "Test description",
									Ref:         "SQ-100",
								},
								Priority: 1.0,
								Products: []string{"waf"},
							},
						}, nil
					},
				},
			},
			args: args{
				rp: &v1alpha1.RuleParameters{
					Action:         "allow",
					BypassProducts: []v1alpha1.RuleBypassProduct{"waf"},
					Description:    ptr.StringPtr("Test Description"),
					Filter:         ptr.StringPtr("372e67954025e0ba6aaa6d586b9e0b61"),
					Paused:         ptr.BoolPtr(false),
					Priority:       ptr.Int32(1),
					Zone:           ptr.StringPtr("Test Zone"),
				},
			},
			want: want{
				err: nil,
			},
		},
		"CreateRuleFailed": {
			reason: "CreateRule should return error when creating a rule fails",
			fields: fields{
				client: fake.MockClient{
					MockCreateFirewallRules: func(ctx context.Context, zoneID string, rr []cloudflare.FirewallRule) ([]cloudflare.FirewallRule, error) {
						return []cloudflare.FirewallRule{}, errBoom
					},
				},
			},
			args: args{
				rp: &v1alpha1.RuleParameters{
					Action:         "allow",
					BypassProducts: []v1alpha1.RuleBypassProduct{"waf"},
					Description:    ptr.StringPtr("Test Description"),
					Filter:         ptr.StringPtr("372e67954025e0ba6aaa6d586b9e0b61"),
					Paused:         ptr.BoolPtr(false),
					Priority:       ptr.Int32(1),
					Zone:           ptr.StringPtr("Test Zone"),
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errCreateRule),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := CreateRule(tc.args.ctx, tc.fields.client, tc.args.rp)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nCreateRule(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestUpdateRule(t *testing.T) {
	errBoom := errors.New("boom")
	type fields struct {
		client Client
	}

	type args struct {
		ctx context.Context
		id  string
		rp  *v1alpha1.RuleParameters
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
		"UpdateRuleNotFound": {
			reason: "UpdateRule should return errUpdateRule if the rule is not found",
			fields: fields{
				client: fake.MockClient{
					MockUpdateFirewallRule: func(ctx context.Context, zoneID string, rr cloudflare.FirewallRule) (cloudflare.FirewallRule, error) {
						return cloudflare.FirewallRule{}, errBoom
					},
					MockFirewallRule: func(ctx context.Context, zoneID, ruleID string) (cloudflare.FirewallRule, error) {
						return cloudflare.FirewallRule{
							Action:      "allow",
							Description: "Test Description",
							Filter: cloudflare.Filter{
								ID:          "372e67954025e0ba6aaa6d586b9e0b61",
								Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100",
								Paused:      false,
								Description: "Test description",
								Ref:         "SQ-100",
							},
							Priority: 1.0,
							Products: []string{"waf"},
						}, nil
					},
				},
			},
			args: args{
				rp: &v1alpha1.RuleParameters{
					Action:         "allow",
					BypassProducts: []v1alpha1.RuleBypassProduct{"waf"},
					Description:    ptr.StringPtr("Test Description"),
					Filter:         ptr.StringPtr("372e67954025e0ba6aaa6d586b9e0b61"),
					Paused:         ptr.BoolPtr(false),
					Priority:       ptr.Int32(1),
					Zone:           ptr.StringPtr("Test Zone"),
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errUpdateRule),
			},
		},
		"UpdateRule": {
			reason: "UpdateRule should return no error when updating a rule successfully",
			fields: fields{
				client: fake.MockClient{
					MockUpdateFirewallRule: func(ctx context.Context, zoneID string, rr cloudflare.FirewallRule) (cloudflare.FirewallRule, error) {
						return cloudflare.FirewallRule{
							Action:      "allow",
							Description: "New Description",
							Filter: cloudflare.Filter{
								ID:          "372e67954025e0ba6aaa6d586b9e0b61",
								Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.100",
								Paused:      false,
								Description: "Test description",
								Ref:         "SQ-100",
							},
							Priority: 1.0,
							Products: []string{"waf"},
						}, nil
					},
					MockFirewallRule: func(ctx context.Context, zoneID, ruleID string) (cloudflare.FirewallRule, error) {
						return cloudflare.FirewallRule{
							Action:      "allow",
							Description: "Old Description",
							Filter: cloudflare.Filter{
								ID:          "372e67954025e0ba6aaa6d586b9e0b61",
								Expression:  "http.request.uri.path ~ \".*wp-login.php\" or http.request.uri.path ~ \".*xmlrpc.php\") and ip.addr ne 172.16.22.155",
								Paused:      false,
								Description: "Test description",
								Ref:         "SQ-100",
							},
							Priority: 1.0,
							Products: []string{"waf"},
						}, nil
					},
				},
			},
			args: args{
				rp: &v1alpha1.RuleParameters{
					Action:         "allow",
					BypassProducts: []v1alpha1.RuleBypassProduct{"waf"},
					Description:    ptr.StringPtr("Test Description"),
					Filter:         ptr.StringPtr("372e67954025e0ba6aaa6d586b9e0b61"),
					Paused:         ptr.BoolPtr(false),
					Priority:       ptr.Int32(1),
					Zone:           ptr.StringPtr("Test Zone"),
				},
			},
			want: want{
				err: nil,
			},
		},
		"UpdateRuleFailed": {
			reason: "UpdateRule should return an error if the update failed",
			fields: fields{
				client: fake.MockClient{
					MockUpdateFirewallRule: func(ctx context.Context, zoneID string, rr cloudflare.FirewallRule) (cloudflare.FirewallRule, error) {
						return cloudflare.FirewallRule{}, errBoom
					},
					MockFirewallRule: func(ctx context.Context, zoneID, ruleID string) (cloudflare.FirewallRule, error) {
						return cloudflare.FirewallRule{}, nil
					},
				},
			},
			args: args{
				rp: &v1alpha1.RuleParameters{
					Action:         "allow",
					BypassProducts: []v1alpha1.RuleBypassProduct{"waf"},
					Description:    ptr.StringPtr("Test Description"),
					Filter:         ptr.StringPtr("372e67954025e0ba6aaa6d586b9e0b61"),
					Paused:         ptr.BoolPtr(false),
					Priority:       ptr.Int32(1),
					Zone:           ptr.StringPtr("Test Zone"),
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errUpdateRule),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := UpdateRule(tc.args.ctx, tc.fields.client, tc.args.id, tc.args.rp)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nUpdateRule(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
		})
	}
}
