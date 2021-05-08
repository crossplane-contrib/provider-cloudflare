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

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	reqInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_client_requests_in_flight",
			Help: "HTTP Requests in-flight at a point in time.",
		},
		[]string{"controller"},
	)
	reqTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_client_requests_total",
			Help: "Total HTTP Requests made, by code and method.",
		},
		[]string{"controller", "code", "method"},
	)
	reqLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_client_request_latency_seconds",
			Help:    "HTTP Request Latency histogram.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"controller", "code", "method"},
	)
	reqEventsLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_client_request_latency_events_seconds",
			Help:    "HTTP Request Latency histogram broken down by request event.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"controller", "event"},
	)
)

// Init registers metric types that can be instrumented on
// the http.Client passed to our clients.
func init() {
	metrics.Registry.MustRegister(
		reqInFlight,
		reqTotal,
		reqLatency,
		reqEventsLatency,
	)
}

// NewInstrumentedHTTPClient returns a *http.Client that has
// been instrumented to track request latencies, types and statuses.
func NewInstrumentedHTTPClient(n string) *http.Client {
	c := http.Client{}
	InstrumentHTTPClient(&c, n)
	return &c
}

// InstrumentHTTPClient instruments an existing *http.Client.
func InstrumentHTTPClient(hc *http.Client, n string) {
	l := prometheus.Labels{"controller": n}

	rt := reqTotal.MustCurryWith(l)
	rif := reqInFlight.With(l)
	rl := reqLatency.MustCurryWith(l)
	rbl := reqEventsLatency.MustCurryWith(l)

	trace := &promhttp.InstrumentTrace{
		DNSStart: func(t float64) {
			rbl.WithLabelValues("dns_start").Observe(t)
		},
		DNSDone: func(t float64) {
			rbl.WithLabelValues("dns_end").Observe(t)
		},
		ConnectStart: func(t float64) {
			rbl.WithLabelValues("connect_start").Observe(t)
		},
		ConnectDone: func(t float64) {
			rbl.WithLabelValues("connect_end").Observe(t)
		},
		TLSHandshakeStart: func(t float64) {
			rbl.WithLabelValues("tls_start").Observe(t)
		},
		TLSHandshakeDone: func(t float64) {
			rbl.WithLabelValues("tls_end").Observe(t)
		},
		GotFirstResponseByte: func(t float64) {
			rbl.WithLabelValues("ttfb").Observe(t)
		},
	}

	hc.Transport = promhttp.InstrumentRoundTripperInFlight(rif,
		promhttp.InstrumentRoundTripperCounter(rt,
			promhttp.InstrumentRoundTripperTrace(trace,
				promhttp.InstrumentRoundTripperDuration(rl, http.DefaultTransport),
			),
		),
	)
}
