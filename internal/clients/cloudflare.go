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
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/cloudflare/cloudflare-go"
)

const (
	errNoAuth = "email or api key not provided"
)

// Config represents the API configuration required to create
// a new client.
type Config struct {
	APIKey string `json:"APIKey"`
	Email  string `json:"Email"`
}

// NewClient creates new Cloudflare Client with provided Credentials.
func NewClient(c Config) (*cloudflare.API, error) {
	if c.APIKey == "" || c.Email == "" {
		return nil, errors.New(errNoAuth)
	}
	return cloudflare.New(c.APIKey, c.Email)
}

// GetConfig returns a valid Cloudflare API configuration
func GetConfig(data []byte) (*Config, error) {
	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}
	return config, nil
}
