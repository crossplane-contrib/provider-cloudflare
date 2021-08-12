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

package controller

import (
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/benagricola/provider-cloudflare/internal/controller/config"
	record "github.com/benagricola/provider-cloudflare/internal/controller/dns"
	filter "github.com/benagricola/provider-cloudflare/internal/controller/firewall/filter"
	rule "github.com/benagricola/provider-cloudflare/internal/controller/firewall/rule"
	application "github.com/benagricola/provider-cloudflare/internal/controller/spectrum"
	customhostname "github.com/benagricola/provider-cloudflare/internal/controller/sslsaas/customhostname"
	fallbackorigin "github.com/benagricola/provider-cloudflare/internal/controller/sslsaas/fallbackorigin"
	route "github.com/benagricola/provider-cloudflare/internal/controller/workers/route"
	zone "github.com/benagricola/provider-cloudflare/internal/controller/zone"
)

// Setup creates all Cloudflare controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, l logging.Logger, wl workqueue.RateLimiter) error {
	for _, setup := range []func(ctrl.Manager, logging.Logger, workqueue.RateLimiter) error{
		application.Setup,
		config.Setup,
		rule.Setup,
		filter.Setup,
		customhostname.Setup,
		zone.Setup,
		record.Setup,
		route.Setup,
		fallbackorigin.Setup,
	} {
		if err := setup(mgr, l, wl); err != nil {
			return err
		}
	}
	return nil
}
