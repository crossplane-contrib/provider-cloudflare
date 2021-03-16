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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// ZoneSettings represents settings on a Zone
type ZoneSettings struct {
	// AlwaysOnline enables or disables Always Online
	// +optional
	AlwaysOnline *bool `json:"alwaysOnline,omitempty"`

	// AdvancedDDOS enables or disables Advanced DDoS mitigation
	// +optional
	AdvancedDDOS *bool `json:"advancedDdos,omitempty"`

	// AlwaysUseHTTPS enables or disables Always use HTTPS
	// +optional
	AlwaysUseHTTPS *bool `json:"alwaysUseHttps,omitempty"`

	// AutomaticHTTPSRewrites enables or disables Automatic HTTPS Rewrites
	// +optional
	AutomaticHTTPSRewrites *bool `json:"automaticHttpsRewrites,omitempty"`

	// Brotli enables or disables Brotli
	// +optional
	Brotli *bool `json:"brotli,omitempty"`

	// BrowserCacheTTL configures the browser cache ttl.
	// 0 means respect existing headers
	// +kubebuilder:validation:Enum=0;30;60;300;1200;1800;3600;7200;10800;14400;18000;28800;43200;57600;72000;86400;172800;259200;345600;432000;691200;1382400;2073600;2678400;5356800;16070400;31536000
	// +optional
	BrowserCacheTTL *int `json:"browserCacheTtl,omitempty"`

	// BrowserCheck enables or disables Browser check
	// +optional
	BrowserCheck *bool `json:"browserCheck,omitempty"`

	// CacheLevel configures the cache level
	// +kubebuilder:validation:Enum=bypass;basic;simplified;aggressive;cache_everything
	// +optional
	CacheLevel *string `json:"cacheLevel,omitempty"`

	// ChallengeTTL configures the edge cache ttl
	// +kubebuilder:validation:Enum=300;900;1800;2700;3600;7200;10800;14400;28800;57600;86400;604800;2592000;31536000
	// +optional
	ChallengeTTL *int `json:"challengeTtl,omitempty"`

	// CnameFlattening configures CNAME flattening
	// +kubebuilder:validation:Enum=flatten_at_root;flatten_all;flatten_none
	// +optional
	CnameFlattening *string `json:"cnameFlattening,omitempty"`

	// DevelopmentMode enables or disables Development mode
	// +optional
	DevelopmentMode *bool `json:"developmentMode,omitempty"`

	// EdgeCacheTTL configures the edge cache ttl
	// +optional
	EdgeCacheTTL *int `json:"edgeCacheTtl,omitempty"`

	// EmailObfuscation enables or disables Email obfuscation
	// +optional
	EmailObfuscation *bool `json:"emailObfuscation,omitempty"`

	// HotlinkProtection enables or disables Hotlink protection
	// +optional
	HotlinkProtection *bool `json:"hotlinkProtection,omitempty"`

	// HTTP2 enables or disables HTTP2
	// +optional
	HTTP2 *bool `json:"http2,omitempty"`

	// HTTP3 enables or disables HTTP3
	// +optional
	HTTP3 *bool `json:"http3,omitempty"`

	// IPGeolocation enables or disables IP Geolocation
	// +optional
	IPGeolocation *bool `json:"ipGeolocation,omitempty"`

	// IPv6 enables or disables IPv6
	// +optional
	IPv6 *bool `json:"ipv6,omitempty"`

	// LogToCloudflare enables or disables Logging to cloudflare
	// +optional
	LogToCloudflare *bool `json:"logToCloudflare,omitempty"`

	// MaxUpload configures the maximum upload payload size
	// +optional
	MaxUpload *int `json:"maxUpload,omitempty"`

	// MinTLSVersion configures the minimum TLS version
	// +kubebuilder:validation:Enum="1.0";"1.1";"1.2";"1.3"
	// +optional
	MinTLSVersion *string `json:"minTLSVersion,omitempty"`

	// Mirage enables or disables Mirage
	// +optional
	Mirage *bool `json:"mirage,omitempty"`

	// OpportunisticEncryption enables or disables Opportunistic encryption
	// +optional
	OpportunisticEncryption *bool `json:"opportunisticEncryption,omitempty"`

	// OpportunisticOnion enables or disables Opportunistic onion
	// +optional
	OpportunisticOnion *bool `json:"opportunisticOnion,omitempty"`

	// OrangeToOrange enables or disables Orange to orange
	// +optional
	OrangeToOrange *bool `json:"orangeToOrange,omitempty"`

	// OriginErrorPagePassThru enables or disables Mirage
	// +optional
	OriginErrorPagePassThru *bool `json:"originErrorPagePassThru,omitempty"`

	// Polish configures the Polish setting
	// +kubebuilder:validation:Enum=off;lossless;lossy
	// +optional
	Polish *string `json:"polish,omitempty"`

	// PrefetchPreload enables or disables Prefetch preload
	// +optional
	PrefetchPreload *bool `json:"prefetchPreload,omitempty"`

	// PrivacyPass enables or disables Privacy pass
	// +optional
	PrivacyPass *bool `json:"privacyPass,omitempty"`

	// PseudoIPv4 configures the Pseudo IPv4 setting
	// +kubebuilder:validation:Enum=off;add_header;overwrite_header
	// +optional
	PseudoIPv4 *string `json:"pseudoIpv4,omitempty"`

	// ResponseBuffering enables or disables Response buffering
	// +optional
	ResponseBuffering *bool `json:"responseBuffering,omitempty"`

	// RocketLoader enables or disables Rocket loader
	// +optional
	RocketLoader *bool `json:"rocketLoader,omitempty"`

	// SecurityLevel configures the Security level
	// +kubebuilder:validation:Enum=off;essentially_off;low;medium;high;under_attack
	// +optional
	SecurityLevel *string `json:"securityLevel,omitempty"`

	// ServerSideExclude enables or disables Server side exclude
	// +optional
	ServerSideExclude *bool `json:"serverSideExclude,omitempty"`

	// SortQueryStringForCache enables or disables Sort query string for cache
	// +optional
	SortQueryStringForCache *bool `json:"sortQueryStringForCache,omitempty"`

	// SSL configures the SSL mode
	// +kubebuilder:validation:Enum=off;flexible;full;strict;origin_pull
	// +optional
	SSL *string `json:"ssl,omitempty"`

	// TLS13 configures TLS 1.3
	// +kubebuilder:validation:Enum=off;on;zrt
	// +optional
	TLS13 *string `json:"tls13,omitempty"`

	// TLSClientAuth enables or disables TLS client authentication
	// +optional
	TLSClientAuth *bool `json:"tlsClientAuth,omitempty"`

	// TrueClientIPHeader enables or disables True client IP Header
	// +optional
	TrueClientIPHeader *bool `json:"trueClientIPHeader,omitempty"`

	// VisitorIP enables or disables Visitor IP
	// +optional
	VisitorIP *bool `json:"visitorIP,omitempty"`

	// WAF enables or disables the Web application firewall
	// +optional
	WAF *bool `json:"waf,omitempty"`

	// WebP enables or disables WebP
	// +optional
	WebP *bool `json:"webP,omitempty"`

	// WebSockets enables or disables Web sockets
	// +optional
	WebSockets *bool `json:"webSockets,omitempty"`

	// ZeroRTT enables or disables Zero RTT
	// +optional
	ZeroRTT *bool `json:"zeroRtt,omitempty"`
}

// ZoneParameters are the configurable fields of a Zone.
type ZoneParameters struct {
	// Name is the name of the Zone, which should be a valid
	// domain.
	// +kubebuilder:validation:Format=hostname
	// +kubebuilder:validation:MaxLength=253
	// +immutable
	Name string `json:"name"`

	// AccountID is the account ID under which this Zone will be
	// created.
	// +immutable
	// +optional
	AccountID *string `json:"accountId,omitempty"`

	// TODO: Work out what to do with this one. In Cloudflare, it causes
	// Existing DNS Records to be imported, which means we have
	// records in Cloudflare that would not be managed by Crossplane.
	// Should we try to import those when creating a Zone with
	// JumpStart enabled?

	// JumpStart enables attempting to import existing DNS records
	// when a new Zone is created.
	// WARNING: JumpStart causes Cloudflare to automatically create
	// DNS records without the involvement of Crossplane. This means
	// you will have no DNSRecord instances representing records
	// created in this manner, and you will have to import them
	// manually if you want to manage them with Crossplane.
	// +kubebuilder:default=false
	// +immutable
	// +optional
	JumpStart bool `json:"jumpStart"`

	// Paused indicates if the zone is only using Cloudflare DNS services.
	// +optional
	Paused *bool `json:"paused,omitempty"`

	// PlanID indicates the plan that this Zone will be subscribed
	// to.
	// +optional
	PlanID *string `json:"planId,omitempty"`

	// Type indicates the type of this zone - partial (partner-hosted
	// or CNAME only) or full.
	// +kubebuilder:validation:Enum=full;partial
	// +kubebuilder:default=full
	// +immutable
	// +optional
	Type *string `json:"type,omitempty"`

	// Settings contains a Zone settings that can be applied
	// to this zone.
	// +optional
	Settings ZoneSettings `json:"settings,omitempty"`

	// VanityNameServers lists an array of domains to use for custom
	// nameservers.
	// +optional
	VanityNameServers []string `json:"vanityNameServers,omitempty"`
}

// ZoneObservation are the observable fields of a Zone.
type ZoneObservation struct {
	// AccountID is the account ID that this zone exists under
	AccountID string `json:"accountId,omitempty"`

	// AccountName is the account name that this zone exists under
	Account string `json:"accountName,omitempty"`

	// DevModeTimer indicates the number of seconds left
	// in dev mode (if positive), otherwise the number
	// of seconds since dev mode expired.
	DevModeTimer int `json:"devModeTimer,omitempty"`

	// OriginalNS lists the original nameservers when
	// this Zone was created.
	OriginalNS []string `json:"originalNameServers,omitempty"`

	// OriginalRegistrar indicates the original registrar
	// when this Zone was created.
	OriginalRegistrar string `json:"originalRegistrar,omitempty"`

	// OriginalDNSHost indicates the original DNS host
	// when this Zone was created.
	OriginalDNSHost string `json:"originalDNSHost,omitempty"`

	// NameServers lists the Name servers that are assigned
	// to this Zone.
	NameServers []string `json:"nameServers,omitempty"`

	// PlanID indicates the billing plan ID assigned
	// to this Zone.
	PlanID string `json:"planId,omitempty"`

	// Plan indicates the name of the plan assigned
	// to this Zone.
	Plan string `json:"plan,omitempty"`

	// PlanPendingID indicates the ID of the pending plan
	// assigned to this Zone.
	PlanPendingID string `json:"planPendingId,omitempty"`

	// PlanPending indicates the name of the pending plan
	// assigned to this Zone.
	PlanPending string `json:"planPending,omitempty"`

	// Status indicates the status of this Zone.
	Status string `json:"status,omitempty"`

	// Betas indicates the betas available on this Zone.
	Betas []string `json:"betas,omitempty"`

	// DeactReason indicates the deactivation reason on
	// this Zone.
	DeactReason string `json:"deactivationReason,omitempty"`

	// VerificationKey indicates the Verification key set
	// on this Zone.
	VerificationKey string `json:"verificationKey,omitempty"`

	// VanityNameServers lists the currently assigned vanity
	// name server addresses.
	VanityNameServers []string `json:"vanityNameServers,omitempty"`
}

// A ZoneSpec defines the desired state of a Zone.
type ZoneSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ZoneParameters `json:"forProvider"`
}

// A ZoneStatus represents the observed state of a Zone.
type ZoneStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ZoneObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Zone is a set of common settings applied to one or more domains.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.bindingPhase"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="ACCOUNT",type="string",JSONPath=".status.atProvider.accountId"
// +kubebuilder:printcolumn:name="PLAN",type="string",JSONPath=".status.atProvider.plan"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,cloudflare}
type Zone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZoneSpec   `json:"spec"`
	Status ZoneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ZoneList contains a list of Zone
type ZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Zone `json:"items"`
}
