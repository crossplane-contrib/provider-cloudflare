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
	"net/http"
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
func NewClient(cfg clients.Config, hc *http.Client) (Client, error) {
	return clients.NewClient(cfg, hc)
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
	ozs *v1alpha1.ZoneSettings) bool {

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
	if len(spec.VanityNameServers) == 0 && len(z.VanityNS) > 0 {
		spec.VanityNameServers = z.VanityNS
		li = true
	}

	// Create a settings map from our Desired and Observed
	// Settings, so we can work out which fields need initialising.
	desired := zoneToSettingsMap(&spec.Settings)
	observed := zoneToSettingsMap(ozs)

	if LateInitializeSettings(observed, desired, &spec.Settings) {
		li = true
	}

	return li
}

// LateInitializeSettings initializes Settings based on the remote resource
func LateInitializeSettings(observed, desired ZoneSettingsMap, initOn *v1alpha1.ZoneSettings) bool {
	li := false

	// For each setting we retrieved from the API
	for k, v := range observed {
		// If the remote value is nil, skip
		if v == nil {
			continue
		}
		// If our local value is nil (i.e. unset), then init it
		// and set our late init state to true.
		if _, ok := desired[k]; !ok {
			desired[k] = v
			li = true
		}
	}
	// If we lateInited any fields, update them on the
	// Zone settings.
	if li {
		settingsMapToZone(desired, initOn)
	}
	return li
}

// LoadSettingsForZone loads Zone settings from the cloudflare API
// and returns a ZoneSettingsMap.
func LoadSettingsForZone(ctx context.Context,
	client Client, zoneID string, zs *v1alpha1.ZoneSettings) error {

	// Get settings
	sr, err := client.ZoneSettings(ctx, zoneID)
	if err != nil {
		return errors.Wrap(err, errLoadSettings)
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
	settingsMapToZone(sbk, zs)
	return nil
}

// settingsMapToZone uses static definitions to map each setting
// to its' value on a ZoneSettings instance.
func settingsMapToZone(sm ZoneSettingsMap, zs *v1alpha1.ZoneSettings) {
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

func mapSet(sm ZoneSettingsMap, key string, value interface{}) {
	// Note for clarity: These case statements _cannot_ be combined
	// as they are extracting the individual value type pointers
	// from the interface that is passed in.
	switch vt := (value).(type) {
	case *string:
		if vt != nil {
			sm[key] = *vt
		}
	case *int64:
		if vt != nil {
			sm[key] = *vt
		}
	// Empty pointer values are ignored
	default:
		return
	}
}

// ZoneToSettingsMap uses static definitions to map each setting
// from its' value on a ZoneSettings instance.
func zoneToSettingsMap(zs *v1alpha1.ZoneSettings) ZoneSettingsMap {
	sm := ZoneSettingsMap{}
	mapSet(sm, cfsZeroRTT, zs.ZeroRTT)
	mapSet(sm, cfsAdvancedDDOS, zs.AdvancedDDOS)
	mapSet(sm, cfsAlwaysOnline, zs.AlwaysOnline)
	mapSet(sm, cfsAlwaysUseHTTPS, zs.AlwaysUseHTTPS)
	mapSet(sm, cfsAutomaticHTTPSRewrites, zs.AutomaticHTTPSRewrites)
	mapSet(sm, cfsBrotli, zs.Brotli)
	mapSet(sm, cfsBrowserCacheTTL, zs.BrowserCacheTTL)
	mapSet(sm, cfsBrowserCheck, zs.BrowserCheck)
	mapSet(sm, cfsCacheLevel, zs.CacheLevel)
	mapSet(sm, cfsChallengeTTL, zs.ChallengeTTL)
	mapSet(sm, cfsCnameFlattening, zs.CnameFlattening)
	mapSet(sm, cfsDevelopmentMode, zs.DevelopmentMode)
	mapSet(sm, cfsEdgeCacheTTL, zs.EdgeCacheTTL)
	mapSet(sm, cfsEmailObfuscation, zs.EmailObfuscation)
	mapSet(sm, cfsHotlinkProtection, zs.HotlinkProtection)
	mapSet(sm, cfsHTTP2, zs.HTTP2)
	mapSet(sm, cfsHTTP3, zs.HTTP3)
	mapSet(sm, cfsIPGeolocation, zs.IPGeolocation)
	mapSet(sm, cfsIPv6, zs.IPv6)
	mapSet(sm, cfsLogToCloudflare, zs.LogToCloudflare)
	mapSet(sm, cfsMaxUpload, zs.MaxUpload)
	mapSet(sm, cfsMinTLSVersion, zs.MinTLSVersion)
	mapSet(sm, cfsMirage, zs.Mirage)
	mapSet(sm, cfsOpportunisticEncryption, zs.OpportunisticEncryption)
	mapSet(sm, cfsOpportunisticOnion, zs.OpportunisticOnion)
	mapSet(sm, cfsOrangeToOrange, zs.OrangeToOrange)
	mapSet(sm, cfsOriginErrorPagePassThru, zs.OriginErrorPagePassThru)
	mapSet(sm, cfsPolish, zs.Polish)
	mapSet(sm, cfsPrefetchPreload, zs.PrefetchPreload)
	mapSet(sm, cfsPrivacyPass, zs.PrivacyPass)
	mapSet(sm, cfsPseudoIPv4, zs.PseudoIPv4)
	mapSet(sm, cfsResponseBuffering, zs.ResponseBuffering)
	mapSet(sm, cfsRocketLoader, zs.RocketLoader)
	mapSet(sm, cfsSecurityLevel, zs.SecurityLevel)
	mapSet(sm, cfsServerSideExclude, zs.ServerSideExclude)
	mapSet(sm, cfsSortQueryStringForCache, zs.SortQueryStringForCache)
	mapSet(sm, cfsSSL, zs.SSL)
	mapSet(sm, cfsTLS13, zs.TLS13)
	mapSet(sm, cfsTLSClientAuth, zs.TLSClientAuth)
	mapSet(sm, cfsTrueClientIPHeader, zs.TrueClientIPHeader)
	mapSet(sm, cfsVisitorIP, zs.VisitorIP)
	mapSet(sm, cfsWAF, zs.WAF)
	mapSet(sm, cfsWebP, zs.WebP)
	mapSet(sm, cfsWebSockets, zs.WebSockets)
	return sm
}

// GetChangedSettings builds a map of only the settings whose
// values need to be updated.
func GetChangedSettings(czs, dzs *v1alpha1.ZoneSettings) []cloudflare.ZoneSetting {
	out := []cloudflare.ZoneSetting{}

	current := zoneToSettingsMap(czs)
	desired := zoneToSettingsMap(dzs)

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
func UpToDate(spec *v1alpha1.ZoneParameters, z cloudflare.Zone, ozs *v1alpha1.ZoneSettings) bool { //nolint:gocyclo
	// NOTE: Gocyclo ignored here because this method has to check each field
	// properly. Avoid putting any more complex logic here, if possible.

	// If we don't have a spec, we _must_ be up to date.
	if spec == nil {
		return true
	}

	// Check if mutable fields are up to date with resource
	if spec.Paused != nil && *spec.Paused != z.Paused {
		return false
	}

	// We only detect the resource as not up to date if the requested
	// plan is not the current plan or the pending plan.
	// Since it can take a month for the plan to change from pending
	// to active.
	if spec.PlanID != nil && *spec.PlanID != z.Plan.ID && *spec.PlanID != z.PlanPending.ID {
		return false
	}

	// TODO: Does this handle nameservers in the wrong order?
	if (spec.VanityNameServers != nil && !cmp.Equal(spec.VanityNameServers, z.VanityNS)) ||
		(spec.VanityNameServers == nil && len(z.VanityNS) > 0) {
		return false
	}

	// Compare settings
	// NOTE: If any settings contain lists or complex structures
	// it may be necessary to modify this to sort those structures or
	// compare them in a different manner.
	// Have a look at https://pkg.go.dev/github.com/google/go-cmp@v0.5.4/cmp/cmpopts
	// to see if what you're looking for is supported by the cmp library
	// before implementing here.
	if !cmp.Equal(*ozs, spec.Settings) {
		return false
	}
	return true
}

// UpdateZone updates mutable values on a Zone
func UpdateZone(ctx context.Context, client Client, zoneID string, spec v1alpha1.ZoneParameters) error { //nolint:gocyclo
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
			return err
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
	curSettings := v1alpha1.ZoneSettings{}
	err = LoadSettingsForZone(ctx, client, zoneID, &curSettings)
	if err != nil {
		return errors.Wrap(err, errUpdateSettings)
	}

	// See if any settings were updated, otherwise return
	// update is complete.
	cs := GetChangedSettings(&curSettings, &spec.Settings)
	if len(cs) < 1 {
		return nil
	}

	// One or more settings were changed, so update them and return.
	_, err = client.UpdateZoneSettings(ctx, zoneID, cs)
	return errors.Wrap(err, errUpdateSettings)
}
