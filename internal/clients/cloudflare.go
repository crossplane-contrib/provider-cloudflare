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
	"net/http"

	"github.com/cloudflare/cloudflare-go"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane-contrib/provider-cloudflare/apis/v1alpha1"
)

const (
	errGetPC        = "cannot get ProviderConfig"
	errPCRef        = "providerConfigRef not set"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errNoAuth       = "auth details not valid"
)

// AuthByAPIKey represents the details required to authenticate
// with the cloudflare API using a users' global API Key and
// Email address.
type AuthByAPIKey struct {
	Key   *string `json:"apiKey,omitempty"`
	Email *string `json:"email,omitempty"`
}

// AuthByAPIToken represents the details required to authenticate
// with the cloudflare API using an API token.
type AuthByAPIToken struct {
	Token *string `json:"token,omitempty"`
}

// Config represents the API configuration required to create
// a new client.
type Config struct {
	*AuthByAPIKey   `json:",inline"`
	*AuthByAPIToken `json:",inline"`
}

// NewClient creates a new Cloudflare Client with provided Credentials.
func NewClient(c Config, hc *http.Client) (*cloudflare.API, error) {
	if hc == nil {
		hc = http.DefaultClient
	}
	ohc := cloudflare.HTTPClient(hc)

	if c.AuthByAPIKey != nil && c.AuthByAPIKey.Key != nil &&
		c.AuthByAPIKey.Email != nil {
		return cloudflare.New(*c.AuthByAPIKey.Key, *c.AuthByAPIKey.Email, ohc)
	}
	if c.AuthByAPIToken != nil && c.AuthByAPIToken.Token != nil {
		return cloudflare.NewWithAPIToken(*c.AuthByAPIToken.Token, ohc)
	}
	return nil, errors.New(errNoAuth)
}

// GetConfig returns a valid Cloudflare API configuration
func GetConfig(ctx context.Context, c client.Client, mg resource.Managed) (*Config, error) {
	switch {
	case mg.GetProviderConfigReference() != nil:
		return UseProviderConfig(ctx, c, mg)
	default:
		return nil, errors.New(errPCRef)
	}

}

// UseProviderConfig produces a config that can be used to authenticate with Cloudflare.
func UseProviderConfig(ctx context.Context, c client.Client, mg resource.Managed) (*Config, error) {
	pc := &v1alpha1.ProviderConfig{}
	if err := c.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	t := resource.NewProviderConfigUsageTracker(c, &v1alpha1.ProviderConfigUsage{})
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}
	return UseProviderSecret(ctx, data)
}

// UseProviderSecret extracts a JSON blob containing configuration
// keys.
func UseProviderSecret(ctx context.Context, data []byte) (*Config, error) {
	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}
	return config, nil
}

// ToNumber converts an interface from the Cloudflare API
// into an int64 pointer, if it contains an existing int,
// int64 or float64 value.
func ToNumber(in interface{}) *int64 {
	// I believe cloudflare-go just decodes values using encoding/json,
	// which defaults to returning a float64 for numbers. We could probably
	// just cast and check for a float64 and ignore the int, but we don't
	// lose anything by simply allowing ints to be passed as that is our
	// storage type in kubernetes anyway.
	switch cv := in.(type) {
	case int:
		o := int64(cv)
		return &o
	case int64:
		return &cv
	case float64:
		o := int64(cv)
		return &o
	default:
	}
	return nil
}

// ToString converts an interface from the Cloudflare API
// into a string pointer, if it contains an existing string.
func ToString(in interface{}) *string {
	return toString(in, false)
}

// ToOptionalString converts an interface from the Cloudflare API
// into a string pointer, if it contains an existing string or
// string pointer.
// If the existing string is empty (""), it returns nil.
func ToOptionalString(in interface{}) *string {
	return toString(in, true)
}

// The assumption here is that the input value has no special
// state for empty string and it means the same as "unset".
func toString(in interface{}, optional bool) *string {
	switch v := in.(type) {
	case string:
		if v != "" || !optional {
			return &v
		}
	// No need for optional discovery here. If input is a
	// pointer then we assume the optional case is when the
	// pointer is nil.
	case *string:
		return v
	}
	return nil
}

// ToBool converts an interface from the Cloudflare API
// into a bool pointer, if it contains an existing bool
// or bool pointer.
func ToBool(in interface{}) *bool {
	switch v := in.(type) {
	case bool:
		return &v
	case *bool:
		return v
	}
	return nil
}

// ToStringSlice converts an interface from the Cloudflare API
// into a string slice.
func ToStringSlice(in interface{}) []string {
	if v, ok := in.([]string); ok {
		return v
	}
	return nil
}
