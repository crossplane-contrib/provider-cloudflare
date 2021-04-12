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

	"github.com/cloudflare/cloudflare-go"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/benagricola/provider-cloudflare/apis/v1alpha1"
)

const (
	errGetPC        = "cannot get ProviderConfig"
	errPCRef        = "providerConfigRef not set"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errNoAuth       = "email or api key not provided"
)

// Config represents the API configuration required to create
// a new client.
// TODO: camelCase the JSON keys
type Config struct {
	APIKey string `json:"APIKey"`
	Email  string `json:"Email"`
}

// NewClient creates new Cloudflare Client with provided Credentials.
func NewClient(c Config) *cloudflare.API {
	api, err := cloudflare.New(c.APIKey, c.Email)
	if err != nil {
		panic(err)
	}
	return api
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
	if config.APIKey == "" || config.Email == "" {
		return nil, errors.New(errNoAuth)
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
	if v, ok := in.(string); ok {
		return &v
	}
	return nil
}
