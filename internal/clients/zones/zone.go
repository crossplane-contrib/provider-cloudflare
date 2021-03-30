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

package zones

import (
	"context"
	"strings"

	"github.com/google/go-cmp/cmp"

	"github.com/pkg/errors"

	"github.com/cloudflare/cloudflare-go"

	"github.com/benagricola/provider-cloudflare/apis/zone/v1alpha1"
	clients "github.com/benagricola/provider-cloudflare/internal/clients"
)

const (
	errLoadSettings   = "error loading settings"
	errUpdateZone     = "error updating zone"
	errSetPlan        = "error setting plan"
	errUpdateSettings = "error updating settings"

	// Hardcoded string in cloudflare-go library.
	// It is used to detect a 'not found' zone
	// lookup vs. a failed lookup.
	// REF: https://github.com/cloudflare/cloudflare-go/blob/1dd2d1fe7d044b42d0b64c2f79b9e730c701ab75/cloudflare.go#L162
	// DO NOT CHANGE THIS
	errZoneNotFound = "Zone could not be found"

	// String returned by Cloudflare API if making a Zone
	// request for a Zone ID that doesn't exist.
	// It is used to detect a 'not found' zone
	// lookup vs. a failed lookup.
	// DO NOT CHANGE THIS
	errZoneInvalidID = "Invalid zone identifier"

	cfsZeroRTT                 = "0rtt"
	cfsAdvancedDDOS            = "advanced_ddos"
	cfsAlwaysOnline            = "always_online"
	cfsAlwaysUseHTTPS          = "always_use_https"
	cfsAutomaticHTTPSRewrites  = "automatic_https_rewrites"
	cfsBrotli                  = "brotli"
	cfsBrowserCacheTTL         = "browser_cache_ttl"
	cfsBrowserCheck            = "browser_check"
	cfsCacheLevel              = "cache_level"
	cfsChallengeTTL            = "challenge_ttl"
	cfsCnameFlattening         = "cname_flattening"
	cfsDevelopmentMode         = "development_mode"
	cfsEdgeCacheTTL            = "edge_cache_ttl"
	cfsEmailObfuscation        = "email_obfuscation"
	cfsHotlinkProtection       = "hotlink_protection"
	cfsHTTP2                   = "http2"
	cfsHTTP3                   = "http3"
	cfsIPGeolocation           = "ip_geolocation"
	cfsIPv6                    = "ipv6"
	cfsLogToCloudflare         = "log_to_cloudflare"
	cfsMaxUpload               = "max_upload"
	cfsMinTLSVersion           = "min_tls_version"
	cfsMirage                  = "mirage"
	cfsOpportunisticEncryption = "opportunistic_encryption"
	cfsOpportunisticOnion      = "opportunistic_onion"
	cfsOrangeToOrange          = "orange_to_orange"
	cfsOriginErrorPagePassThru = "origin_error_page_pass_thru"
	cfsPolish                  = "polish"
	cfsPrefetchPreload         = "prefetch_preload"
	cfsPrivacyPass             = "privacy_pass"
	cfsPseudoIPv4              = "pseudo_ipv4"
	cfsResponseBuffering       = "response_buffering"
	cfsRocketLoader            = "rocket_loader"
	cfsSecurityLevel           = "security_level"
	cfsServerSideExclude       = "server_side_exclude"
	cfsSortQueryStringForCache = "sort_query_string_for_cache"
	cfsSSL                     = "ssl"
	cfsTLS13                   = "tls_1_3"
	cfsTLSClientAuth           = "tls_client_auth"
	cfsTrueClientIPHeader      = "true_client_ip_header"
	cfsVisitorIP               = "visitor_ip"
	cfsWAF                     = "waf"
	cfsWebP                    = "webp"
	cfsWebSockets              = "websockets"
)

// ZoneSettingsMap contains pairs of keys and values
// that represent settings on a Zone.
type ZoneSettingsMap map[string]interface{}

// IsZoneNotFound returns true if the passed error indicates
// a Zone was not found.
func IsZoneNotFound(err error) bool {
	errStr := err.Error()
	return errStr == errZoneNotFound || strings.Contains(errStr, errZoneInvalidID)
}

// Client is a Cloudflare API client that implements methods for working
// with Zones.
type Client interface {
	CreateZone(ctx context.Context, name string, jumpstart bool, account cloudflare.Account, zoneType string) (cloudflare.Zone, error)
	DeleteZone(ctx context.Context, zoneID string) (cloudflare.ZoneID, error)
	EditZone(ctx context.Context, zoneID string, zoneOpts cloudflare.ZoneOptions) (cloudflare.Zone, error)
	UpdateZoneSettings(ctx context.Context, zoneID string, cs []cloudflare.ZoneSetting) (*cloudflare.ZoneSettingResponse, error)
	ZoneDetails(ctx context.Context, zoneID string) (cloudflare.Zone, error)
	ZoneIDByName(zoneName string) (string, error)
	ZoneSetPlan(ctx context.Context, zoneID string, planType string) error
	ZoneSettings(ctx context.Context, zoneID string) (*cloudflare.ZoneSettingResponse, error)
}

// NewClient returns a new Cloudflare API client for working with Zones.
func NewClient(cfg clients.Config) Client {
	return clients.NewClient(cfg)
}

// GenerateObservation creates an observation of a cloudflare Zone
func GenerateObservation(in cloudflare.Zone) v1alpha1.ZoneObservation {
	return v1alpha1.ZoneObservation{
		AccountID:         in.Account.ID,
		Account:           in.Account.Name,
		DevModeTimer:      in.DevMode,
		OriginalNS:        in.OriginalNS,
		OriginalRegistrar: in.OriginalRegistrar,
		OriginalDNSHost:   in.OriginalDNSHost,
		NameServers:       in.NameServers,
		PlanID:            in.Plan.ID,
		Plan:              in.Plan.Name,
		PlanPendingID:     in.PlanPending.ID,
		PlanPending:       in.PlanPending.Name,
		Status:            in.Status,
		Betas:             in.Betas,
		DeactReason:       in.DeactReason,
		VerificationKey:   in.VerificationKey,
		VanityNameServers: in.VanityNS,
	}
}

// LateInitialize initializes ZoneParameters based on the remote resource
func LateInitialize(spec *v1alpha1.ZoneParameters, z cloudflare.Zone,
	current, desired ZoneSettingsMap) bool {

	if spec == nil {
		return false
	}

	li := false
	if spec.AccountID == nil {
		spec.AccountID = &z.Account.ID
		li = true
	}
	if spec.Paused == nil {
		spec.Paused = &z.Paused
		li = true
	}
	if spec.PlanID == nil {
		spec.PlanID = &z.Plan.ID
		li = true
	}
	if spec.VanityNameServers == nil {
		spec.VanityNameServers = z.VanityNS
		li = true
	}

	if LateInitializeSettings(current, desired, &spec.Settings) {
		li = true
	}

	return li
}

// LateInitializeSettings initializes Settings based on the remote resource
func LateInitializeSettings(current, desired ZoneSettingsMap, initOn *v1alpha1.ZoneSettings) bool {
	li := false

	// For each retrieved setting
	for k, v := range current {
		// Check to see if setting is already desired
		if _, ok := desired[k]; !ok {
			// If not, late-init it
			desired[k] = v
			li = true
		}
	}
	// If we lateInited any fields, update them on the
	// Zone settings.
	if li {
		SettingsMapToZone(desired, initOn)
	}
	return li
}

// LoadSettingsForZone loads Zone settings from the cloudflare API
// and returns a ZoneSettingsMap.
func LoadSettingsForZone(ctx context.Context,
	client Client, zoneID string) (ZoneSettingsMap, error) {

	// Get settings
	sr, err := client.ZoneSettings(ctx, zoneID)
	if err != nil {
		return nil, errors.Wrap(err, errLoadSettings)
	}

	// Parse the result into a map based on key
	sbk := ZoneSettingsMap{}

	for _, setting := range sr.Result {
		// Ignore settings we cant edit
		if !setting.Editable {
			continue
		}
		sbk[setting.ID] = setting.Value
	}

	return sbk, nil
}

// SettingsMapToZone uses static definitions to map each setting
// to its' value on a ZoneSettings instance.
func SettingsMapToZone(sm ZoneSettingsMap, zs *v1alpha1.ZoneSettings) {
	zs.ZeroRTT = clients.ToString(sm[cfsZeroRTT])
	zs.AdvancedDDOS = clients.ToString(sm[cfsAdvancedDDOS])
	zs.AlwaysOnline = clients.ToString(sm[cfsAlwaysOnline])
	zs.AlwaysUseHTTPS = clients.ToString(sm[cfsAlwaysUseHTTPS])
	zs.AutomaticHTTPSRewrites = clients.ToString(sm[cfsAutomaticHTTPSRewrites])
	zs.Brotli = clients.ToString(sm[cfsBrotli])
	zs.BrowserCacheTTL = clients.ToNumber(sm[cfsBrowserCacheTTL])
	zs.BrowserCheck = clients.ToString(sm[cfsBrowserCheck])
	zs.CacheLevel = clients.ToString(sm[cfsCacheLevel])
	zs.ChallengeTTL = clients.ToNumber(sm[cfsChallengeTTL])
	zs.CnameFlattening = clients.ToString(sm[cfsCnameFlattening])
	zs.DevelopmentMode = clients.ToString(sm[cfsDevelopmentMode])
	zs.EdgeCacheTTL = clients.ToNumber(sm[cfsEdgeCacheTTL])
	zs.EmailObfuscation = clients.ToString(sm[cfsEmailObfuscation])
	zs.HotlinkProtection = clients.ToString(sm[cfsHotlinkProtection])
	zs.HTTP2 = clients.ToString(sm[cfsHTTP2])
	zs.HTTP3 = clients.ToString(sm[cfsHTTP3])
	zs.IPGeolocation = clients.ToString(sm[cfsIPGeolocation])
	zs.IPv6 = clients.ToString(sm[cfsIPv6])
	zs.LogToCloudflare = clients.ToString(sm[cfsLogToCloudflare])
	zs.MaxUpload = clients.ToNumber(sm[cfsMaxUpload])
	zs.MinTLSVersion = clients.ToString(sm[cfsMinTLSVersion])
	zs.Mirage = clients.ToString(sm[cfsMirage])
	zs.OpportunisticEncryption = clients.ToString(sm[cfsOpportunisticEncryption])
	zs.OpportunisticOnion = clients.ToString(sm[cfsOpportunisticOnion])
	zs.OrangeToOrange = clients.ToString(sm[cfsOrangeToOrange])
	zs.OriginErrorPagePassThru = clients.ToString(sm[cfsOriginErrorPagePassThru])
	zs.Polish = clients.ToString(sm[cfsPolish])
	zs.PrefetchPreload = clients.ToString(sm[cfsPrefetchPreload])
	zs.PrivacyPass = clients.ToString(sm[cfsPrivacyPass])
	zs.PseudoIPv4 = clients.ToString(sm[cfsPseudoIPv4])
	zs.ResponseBuffering = clients.ToString(sm[cfsResponseBuffering])
	zs.RocketLoader = clients.ToString(sm[cfsRocketLoader])
	zs.SecurityLevel = clients.ToString(sm[cfsSecurityLevel])
	zs.ServerSideExclude = clients.ToString(sm[cfsServerSideExclude])
	zs.SortQueryStringForCache = clients.ToString(sm[cfsSortQueryStringForCache])
	zs.SSL = clients.ToString(sm[cfsSSL])
	zs.TLS13 = clients.ToString(sm[cfsTLS13])
	zs.TLSClientAuth = clients.ToString(sm[cfsTLSClientAuth])
	zs.TrueClientIPHeader = clients.ToString(sm[cfsTrueClientIPHeader])
	zs.VisitorIP = clients.ToString(sm[cfsVisitorIP])
	zs.WAF = clients.ToString(sm[cfsWAF])
	zs.WebP = clients.ToString(sm[cfsWebP])
	zs.WebSockets = clients.ToString(sm[cfsWebSockets])
}

func mapSetString(sm ZoneSettingsMap, key string, value *string) {
	// Ignore nil pointers
	if value == nil {
		return
	}
	sm[key] = *value
}

func mapSetNumber(sm ZoneSettingsMap, key string, value *int) {
	// Ignore nil pointers
	if value == nil {
		return
	}
	sm[key] = float64(*value)
}

// ZoneToSettingsMap uses static definitions to map each setting
// from its' value on a ZoneSettings instance.
func ZoneToSettingsMap(zs *v1alpha1.ZoneSettings) ZoneSettingsMap {
	sm := ZoneSettingsMap{}
	mapSetString(sm, cfsZeroRTT, zs.ZeroRTT)
	mapSetString(sm, cfsAdvancedDDOS, zs.AdvancedDDOS)
	mapSetString(sm, cfsAlwaysOnline, zs.AlwaysOnline)
	mapSetString(sm, cfsAlwaysUseHTTPS, zs.AlwaysUseHTTPS)
	mapSetString(sm, cfsAutomaticHTTPSRewrites, zs.AutomaticHTTPSRewrites)
	mapSetString(sm, cfsBrotli, zs.Brotli)
	mapSetNumber(sm, cfsBrowserCacheTTL, zs.BrowserCacheTTL)
	mapSetString(sm, cfsBrowserCheck, zs.BrowserCheck)
	mapSetString(sm, cfsCacheLevel, zs.CacheLevel)
	mapSetNumber(sm, cfsChallengeTTL, zs.ChallengeTTL)
	mapSetString(sm, cfsCnameFlattening, zs.CnameFlattening)
	mapSetString(sm, cfsDevelopmentMode, zs.DevelopmentMode)
	mapSetNumber(sm, cfsEdgeCacheTTL, zs.EdgeCacheTTL)
	mapSetString(sm, cfsEmailObfuscation, zs.EmailObfuscation)
	mapSetString(sm, cfsHotlinkProtection, zs.HotlinkProtection)
	mapSetString(sm, cfsHTTP2, zs.HTTP2)
	mapSetString(sm, cfsHTTP3, zs.HTTP3)
	mapSetString(sm, cfsIPGeolocation, zs.IPGeolocation)
	mapSetString(sm, cfsIPv6, zs.IPv6)
	mapSetString(sm, cfsLogToCloudflare, zs.LogToCloudflare)
	mapSetNumber(sm, cfsMaxUpload, zs.MaxUpload)
	mapSetString(sm, cfsMinTLSVersion, zs.MinTLSVersion)
	mapSetString(sm, cfsMirage, zs.Mirage)
	mapSetString(sm, cfsOpportunisticEncryption, zs.OpportunisticEncryption)
	mapSetString(sm, cfsOpportunisticOnion, zs.OpportunisticOnion)
	mapSetString(sm, cfsOrangeToOrange, zs.OrangeToOrange)
	mapSetString(sm, cfsOriginErrorPagePassThru, zs.OriginErrorPagePassThru)
	mapSetString(sm, cfsPolish, zs.Polish)
	mapSetString(sm, cfsPrefetchPreload, zs.PrefetchPreload)
	mapSetString(sm, cfsPrivacyPass, zs.PrivacyPass)
	mapSetString(sm, cfsPseudoIPv4, zs.PseudoIPv4)
	mapSetString(sm, cfsResponseBuffering, zs.ResponseBuffering)
	mapSetString(sm, cfsRocketLoader, zs.RocketLoader)
	mapSetString(sm, cfsSecurityLevel, zs.SecurityLevel)
	mapSetString(sm, cfsServerSideExclude, zs.ServerSideExclude)
	mapSetString(sm, cfsSortQueryStringForCache, zs.SortQueryStringForCache)
	mapSetString(sm, cfsSSL, zs.SSL)
	mapSetString(sm, cfsTLS13, zs.TLS13)
	mapSetString(sm, cfsTLSClientAuth, zs.TLSClientAuth)
	mapSetString(sm, cfsTrueClientIPHeader, zs.TrueClientIPHeader)
	mapSetString(sm, cfsVisitorIP, zs.VisitorIP)
	mapSetString(sm, cfsWAF, zs.WAF)
	mapSetString(sm, cfsWebP, zs.WebP)
	mapSetString(sm, cfsWebSockets, zs.WebSockets)
	return sm
}

// GetChangedSettings builds a map of only the settings whose
// values need to be updated.
func GetChangedSettings(current, desired ZoneSettingsMap) []cloudflare.ZoneSetting {
	out := []cloudflare.ZoneSetting{}
	for k, nv := range desired {
		cv := current[k]
		// If the current value and new value are not the same,
		// append a ZoneSetting entry to the output list, in
		// preparation for updating.
		if cv != nv {
			zs := cloudflare.ZoneSetting{
				ID:    k,
				Value: nv,
			}
			out = append(out, zs)
		}
	}
	return out
}

// UpToDate checks if the remote resource is up to date with the
// requested resource parameters.
func UpToDate(spec *v1alpha1.ZoneParameters, z cloudflare.Zone) bool {
	// If we don't have a spec, we _must_ be up to date.
	if spec == nil {
		return true
	}

	// Check if mutable fields are up to date with resource
	if *spec.Paused != z.Paused {
		return false
	}

	// We only detect the resource as not up to date if the requested
	// plan is not the current plan or the pending plan.
	// Since it can take a month for the plan to change from pending
	// to active.
	if spec.PlanID != nil && *spec.PlanID != z.Plan.ID && *spec.PlanID != z.PlanPending.ID {
		return false
	}

	if !cmp.Equal(spec.VanityNameServers, z.VanityNS) {
		return false
	}

	return true
}

// UpdateZone updates mutable values on a Zone
func UpdateZone(ctx context.Context, client Client, zoneID string, spec *v1alpha1.ZoneParameters) error { //nolint:gocyclo
	// Get current zone status
	z, err := client.ZoneDetails(ctx, zoneID)
	if err != nil {
		return errors.Wrap(err, errUpdateZone)
	}

	zo := cloudflare.ZoneOptions{}
	u := false

	if spec.Paused != nil && *spec.Paused != z.Paused {
		zo.Paused = spec.Paused
		u = true
	}

	if !cmp.Equal(spec.VanityNameServers, z.VanityNS) {
		zo.VanityNS = spec.VanityNameServers
		u = true
	}

	// Update zone options if necessary
	if u {
		_, err := client.EditZone(ctx, zoneID, zo)
		if err != nil {
			return errors.Wrap(err, errUpdateZone)
		}
	}

	// ZoneSetPlan appears to use a zone subscriptions endpoint
	// Rather than just EditZone, so we implement it separately.
	// We only update if the requested plan is not the current plan
	// OR the pending plan, as it may take a long time for the plan
	// change to take effect.
	if spec.PlanID != nil && *spec.PlanID != z.Plan.ID &&
		spec.PlanID != &z.PlanPending.ID {
		err := client.ZoneSetPlan(ctx, zoneID, *spec.PlanID)
		if err != nil {
			return errors.Wrap(err, errSetPlan)
		}
	}

	// We don't store observed settings so look them up before changing.
	curSettings, err := LoadSettingsForZone(ctx, client, zoneID)
	if err != nil {
		return errors.Wrap(err, errUpdateSettings)
	}

	// See if any settings were updated, otherwise return
	// update is complete.
	cs := GetChangedSettings(curSettings, ZoneToSettingsMap(&spec.Settings))
	if len(cs) < 1 {
		return nil
	}

	// One or more settings were changed, so update them and return.
	_, err = client.UpdateZoneSettings(ctx, zoneID, cs)
	return errors.Wrap(err, errUpdateSettings)
}
