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

	cfsBoolTrue  = "on"
	cfsBoolFalse = "off"

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

// LookupZoneByIDOrName looks up a Zone by ID, if supplied,
// looking up by Name if not.
func LookupZoneByIDOrName(ctx context.Context, client Client,
	zoneIDOrName string) (*cloudflare.Zone, error) {

	// Lookup Zone by ID, return if no error
	zone, err := client.ZoneDetails(ctx, zoneIDOrName)
	if err == nil {
		return &zone, nil
	}

	// Otherwise, try to get the zone ID from the name and
	// retrieve the zone.
	zoneID, err := client.ZoneIDByName(zoneIDOrName)
	if err != nil {
		return nil, err
	}
	zone, err = client.ZoneDetails(ctx, zoneID)
	return &zone, err
}

// GenerateObservation creates an observation of a cloudflare Zone
func GenerateObservation(in cloudflare.Zone) v1alpha1.ZoneObservation {
	return v1alpha1.ZoneObservation{
		AccountID:         in.Account.ID,
		Account:           in.Account.Name,
		DevMode:           in.DevMode,
		OriginalNS:        in.OriginalNS,
		OriginalRegistrar: in.OriginalRegistrar,
		OriginalDNSHost:   in.OriginalDNSHost,
		NameServers:       in.NameServers,
		Paused:            in.Paused,
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
func LateInitialize(spec *v1alpha1.ZoneParameters, o v1alpha1.ZoneObservation,
	current, desired ZoneSettingsMap) bool {

	if spec == nil {
		return false
	}

	li := false
	if spec.AccountID == nil {
		spec.AccountID = &o.AccountID
		li = true
	}
	if spec.Paused == nil {
		spec.Paused = &o.Paused
		li = true
	}
	if spec.PlanID == nil {
		spec.PlanID = &o.PlanID
		li = true
	}
	if spec.VanityNameServers == nil {
		spec.VanityNameServers = o.VanityNameServers
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
	zs.ZeroRTT = clients.ToBoolean(sm[cfsZeroRTT])
	zs.AdvancedDDOS = clients.ToBoolean(sm[cfsAdvancedDDOS])
	zs.AlwaysOnline = clients.ToBoolean(sm[cfsAlwaysOnline])
	zs.AlwaysUseHTTPS = clients.ToBoolean(sm[cfsAlwaysUseHTTPS])
	zs.AutomaticHTTPSRewrites = clients.ToBoolean(sm[cfsAutomaticHTTPSRewrites])
	zs.Brotli = clients.ToBoolean(sm[cfsBrotli])
	zs.BrowserCacheTTL = clients.ToNumber(sm[cfsBrowserCacheTTL])
	zs.BrowserCheck = clients.ToBoolean(sm[cfsBrowserCheck])
	zs.CacheLevel = clients.ToString(sm[cfsCacheLevel])
	zs.ChallengeTTL = clients.ToNumber(sm[cfsChallengeTTL])
	zs.CnameFlattening = clients.ToString(sm[cfsCnameFlattening])
	zs.DevelopmentMode = clients.ToBoolean(sm[cfsDevelopmentMode])
	zs.EdgeCacheTTL = clients.ToNumber(sm[cfsEdgeCacheTTL])
	zs.EmailObfuscation = clients.ToBoolean(sm[cfsEmailObfuscation])
	zs.HotlinkProtection = clients.ToBoolean(sm[cfsHotlinkProtection])
	zs.HTTP2 = clients.ToBoolean(sm[cfsHTTP2])
	zs.HTTP3 = clients.ToBoolean(sm[cfsHTTP3])
	zs.IPGeolocation = clients.ToBoolean(sm[cfsIPGeolocation])
	zs.IPv6 = clients.ToBoolean(sm[cfsIPv6])
	zs.LogToCloudflare = clients.ToBoolean(sm[cfsLogToCloudflare])
	zs.MaxUpload = clients.ToNumber(sm[cfsMaxUpload])
	zs.MinTLSVersion = clients.ToString(sm[cfsMinTLSVersion])
	zs.Mirage = clients.ToBoolean(sm[cfsMirage])
	zs.OpportunisticEncryption = clients.ToBoolean(sm[cfsOpportunisticEncryption])
	zs.OpportunisticOnion = clients.ToBoolean(sm[cfsOpportunisticOnion])
	zs.OrangeToOrange = clients.ToBoolean(sm[cfsOrangeToOrange])
	zs.OriginErrorPagePassThru = clients.ToBoolean(sm[cfsOriginErrorPagePassThru])
	zs.Polish = clients.ToString(sm[cfsPolish])
	zs.PrefetchPreload = clients.ToBoolean(sm[cfsPrefetchPreload])
	zs.PrivacyPass = clients.ToBoolean(sm[cfsPrivacyPass])
	zs.PseudoIPv4 = clients.ToString(sm[cfsPseudoIPv4])
	zs.ResponseBuffering = clients.ToBoolean(sm[cfsResponseBuffering])
	zs.RocketLoader = clients.ToBoolean(sm[cfsRocketLoader])
	zs.SecurityLevel = clients.ToString(sm[cfsSecurityLevel])
	zs.ServerSideExclude = clients.ToBoolean(sm[cfsServerSideExclude])
	zs.SortQueryStringForCache = clients.ToBoolean(sm[cfsSortQueryStringForCache])
	zs.SSL = clients.ToString(sm[cfsSSL])
	zs.TLS13 = clients.ToString(sm[cfsTLS13])
	zs.TLSClientAuth = clients.ToBoolean(sm[cfsTLSClientAuth])
	zs.TrueClientIPHeader = clients.ToBoolean(sm[cfsTrueClientIPHeader])
	zs.VisitorIP = clients.ToBoolean(sm[cfsVisitorIP])
	zs.WAF = clients.ToBoolean(sm[cfsWAF])
	zs.WebP = clients.ToBoolean(sm[cfsWebP])
	zs.WebSockets = clients.ToBoolean(sm[cfsWebSockets])
}

func mapSetBool(sm ZoneSettingsMap, key string, value *bool) {
	// Ignore nil pointers
	if value == nil {
		return
	}
	if *value {
		sm[key] = cfsBoolTrue
		return
	}
	sm[key] = cfsBoolFalse
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
	mapSetBool(sm, cfsZeroRTT, zs.ZeroRTT)
	mapSetBool(sm, cfsAdvancedDDOS, zs.AdvancedDDOS)
	mapSetBool(sm, cfsAlwaysOnline, zs.AlwaysOnline)
	mapSetBool(sm, cfsAlwaysUseHTTPS, zs.AlwaysUseHTTPS)
	mapSetBool(sm, cfsAutomaticHTTPSRewrites, zs.AutomaticHTTPSRewrites)
	mapSetBool(sm, cfsBrotli, zs.Brotli)
	mapSetNumber(sm, cfsBrowserCacheTTL, zs.BrowserCacheTTL)
	mapSetBool(sm, cfsBrowserCheck, zs.BrowserCheck)
	mapSetString(sm, cfsCacheLevel, zs.CacheLevel)
	mapSetNumber(sm, cfsChallengeTTL, zs.ChallengeTTL)
	mapSetString(sm, cfsCnameFlattening, zs.CnameFlattening)
	mapSetBool(sm, cfsDevelopmentMode, zs.DevelopmentMode)
	mapSetNumber(sm, cfsEdgeCacheTTL, zs.EdgeCacheTTL)
	mapSetBool(sm, cfsEmailObfuscation, zs.EmailObfuscation)
	mapSetBool(sm, cfsHotlinkProtection, zs.HotlinkProtection)
	mapSetBool(sm, cfsHTTP2, zs.HTTP2)
	mapSetBool(sm, cfsHTTP3, zs.HTTP3)
	mapSetBool(sm, cfsIPGeolocation, zs.IPGeolocation)
	mapSetBool(sm, cfsIPv6, zs.IPv6)
	mapSetBool(sm, cfsLogToCloudflare, zs.LogToCloudflare)
	mapSetNumber(sm, cfsMaxUpload, zs.MaxUpload)
	mapSetString(sm, cfsMinTLSVersion, zs.MinTLSVersion)
	mapSetBool(sm, cfsMirage, zs.Mirage)
	mapSetBool(sm, cfsOpportunisticEncryption, zs.OpportunisticEncryption)
	mapSetBool(sm, cfsOpportunisticOnion, zs.OpportunisticOnion)
	mapSetBool(sm, cfsOrangeToOrange, zs.OrangeToOrange)
	mapSetBool(sm, cfsOriginErrorPagePassThru, zs.OriginErrorPagePassThru)
	mapSetString(sm, cfsPolish, zs.Polish)
	mapSetBool(sm, cfsPrefetchPreload, zs.PrefetchPreload)
	mapSetBool(sm, cfsPrivacyPass, zs.PrivacyPass)
	mapSetString(sm, cfsPseudoIPv4, zs.PseudoIPv4)
	mapSetBool(sm, cfsResponseBuffering, zs.ResponseBuffering)
	mapSetBool(sm, cfsRocketLoader, zs.RocketLoader)
	mapSetString(sm, cfsSecurityLevel, zs.SecurityLevel)
	mapSetBool(sm, cfsServerSideExclude, zs.ServerSideExclude)
	mapSetBool(sm, cfsSortQueryStringForCache, zs.SortQueryStringForCache)
	mapSetString(sm, cfsSSL, zs.SSL)
	mapSetString(sm, cfsTLS13, zs.TLS13)
	mapSetBool(sm, cfsTLSClientAuth, zs.TLSClientAuth)
	mapSetBool(sm, cfsTrueClientIPHeader, zs.TrueClientIPHeader)
	mapSetBool(sm, cfsVisitorIP, zs.VisitorIP)
	mapSetBool(sm, cfsWAF, zs.WAF)
	mapSetBool(sm, cfsWebP, zs.WebP)
	mapSetBool(sm, cfsWebSockets, zs.WebSockets)
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
func UpToDate(spec *v1alpha1.ZoneParameters, o v1alpha1.ZoneObservation) bool {
	if spec == nil {
		return false
	}

	// Check if mutable fields are up to date with resource
	if *spec.Paused != o.Paused {
		return false
	}
	// We only detect the resource as not up to date if the requested
	// plan is not the current plan or the pending plan.
	// Since it can take a month for the plan to change from pending
	// to active.
	if *spec.PlanID != o.PlanID && *spec.PlanID != o.PlanPendingID {
		return false
	}
	if !cmp.Equal(spec.VanityNameServers, o.VanityNameServers) {
		return false
	}

	return true
}

// UpdateZone updates mutable values on a Zone
func UpdateZone(ctx context.Context, client Client, zoneID string, spec *v1alpha1.ZoneParameters, o *v1alpha1.ZoneObservation) error { //nolint:gocyclo
	zo := cloudflare.ZoneOptions{}
	u := false

	if spec.Paused != nil && *spec.Paused != o.Paused {
		zo.Paused = spec.Paused
		u = true
	}

	if !cmp.Equal(spec.VanityNameServers, o.VanityNameServers) {
		zo.VanityNS = spec.VanityNameServers
		u = true
	}

	// Update zone options if necessary
	if u {
		zone, err := client.EditZone(ctx, zoneID, zo)
		if err != nil {
			return errors.Wrap(err, errUpdateZone)
		}
		// Update observation with returned values
		o.Paused = zone.Paused
		o.VanityNameServers = zone.VanityNS
	}

	// ZoneSetPlan appears to use a zone subscriptions endpoint
	// Rather than just EditZone, so we implement it separately.
	// We only update if the requested plan is not the current plan
	// OR the pending plan, as it may take a long time for the plan
	// change to take effect.
	if spec.PlanID != nil && *spec.PlanID != o.PlanID &&
		spec.PlanID != &o.PlanPendingID {
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
