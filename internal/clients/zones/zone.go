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
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/pkg/errors"

	"github.com/cloudflare/cloudflare-go"

	"github.com/crossplane-contrib/provider-cloudflare/apis/zone/v1alpha1"
	clients "github.com/crossplane-contrib/provider-cloudflare/internal/clients"
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

	cfsZeroRTT                                  = "0rtt"
	cfsAdvancedDDOS                             = "advanced_ddos"
	cfsAlwaysOnline                             = "always_online"
	cfsAlwaysUseHTTPS                           = "always_use_https"
	cfsAutomaticHTTPSRewrites                   = "automatic_https_rewrites"
	cfsBrotli                                   = "brotli"
	cfsBrowserCacheTTL                          = "browser_cache_ttl"
	cfsBrowserCheck                             = "browser_check"
	cfsCacheLevel                               = "cache_level"
	cfsChallengeTTL                             = "challenge_ttl"
	cfsCiphers                                  = "ciphers"
	cfsCnameFlattening                          = "cname_flattening"
	cfsDevelopmentMode                          = "development_mode"
	cfsEdgeCacheTTL                             = "edge_cache_ttl"
	cfsEmailObfuscation                         = "email_obfuscation"
	cfsHotlinkProtection                        = "hotlink_protection"
	cfsHTTP2                                    = "http2"
	cfsHTTP3                                    = "http3"
	cfsIPGeolocation                            = "ip_geolocation"
	cfsIPv6                                     = "ipv6"
	cfsLogToCloudflare                          = "log_to_cloudflare"
	cfsMaxUpload                                = "max_upload"
	cfsMinify                                   = "minify"
	cfsMinifyHTML                               = "html"
	cfsMinifyJS                                 = "js"
	cfsMinifyCSS                                = "css"
	cfsMinTLSVersion                            = "min_tls_version"
	cfsMirage                                   = "mirage"
	cfsMobileRedirect                           = "mobile_redirect"
	cfsMobileRedirectStatus                     = "status"
	cfsMobileRedirectSubdomain                  = "mobile_subdomain"
	cfsMobileRedirectStripURI                   = "strip_uri"
	cfsOpportunisticEncryption                  = "opportunistic_encryption"
	cfsOpportunisticOnion                       = "opportunistic_onion"
	cfsOrangeToOrange                           = "orange_to_orange"
	cfsOriginErrorPagePassThru                  = "origin_error_page_pass_thru"
	cfsPolish                                   = "polish"
	cfsPrefetchPreload                          = "prefetch_preload"
	cfsPrivacyPass                              = "privacy_pass"
	cfsPseudoIPv4                               = "pseudo_ipv4"
	cfsResponseBuffering                        = "response_buffering"
	cfsRocketLoader                             = "rocket_loader"
	cfsSecurityHeader                           = "security_header"
	cfsStrictTransportSecurity                  = "strict_transport_security"
	cfsStrictTransportSecurityEnabled           = "enabled"
	cfsStrictTransportSecurityIncludeSubdomains = "include_subdomains"
	cfsStrictTransportSecurityMaxAge            = "max_age"
	cfsStrictTransportSecurityNoSniff           = "nosniff"
	cfsSecurityLevel                            = "security_level"
	cfsServerSideExclude                        = "server_side_exclude"
	cfsSortQueryStringForCache                  = "sort_query_string_for_cache"
	cfsSSL                                      = "ssl"
	cfsTLS13                                    = "tls_1_3"
	cfsTLSClientAuth                            = "tls_client_auth"
	cfsTrueClientIPHeader                       = "true_client_ip_header"
	cfsVisitorIP                                = "visitor_ip"
	cfsWAF                                      = "waf"
	cfsWebP                                     = "webp"
	cfsWebSockets                               = "websockets"
)

// toMinifySettings converts an interface from the Cloudflare API
// into a MinifySettings type.
func toMinifySettings(in interface{}) *v1alpha1.MinifySettings {
	if m, ok := in.(map[string]interface{}); ok {
		minifySettings := &v1alpha1.MinifySettings{}
		for key, value := range m {
			sval := clients.ToString(value)
			switch key {
			case cfsMinifyCSS:
				minifySettings.CSS = sval
			case cfsMinifyJS:
				minifySettings.JS = sval
			case cfsMinifyHTML:
				minifySettings.HTML = sval
			}
		}

		return minifySettings
	}

	return nil
}

// toMobileRedirectSettings converts an interface from the Cloudflare API
// into a MinifySettings type.
func toMobileRedirectSettings(in interface{}) *v1alpha1.MobileRedirectSettings {
	if m, ok := in.(map[string]interface{}); ok {
		mobileRedirectSettings := &v1alpha1.MobileRedirectSettings{}
		for key, value := range m {
			switch key {
			case cfsMobileRedirectStatus:
				mobileRedirectSettings.Status = clients.ToString(value)
			case cfsMobileRedirectSubdomain:
				mobileRedirectSettings.Subdomain = clients.ToString(value)
			case cfsMobileRedirectStripURI:
				mobileRedirectSettings.StripURI = clients.ToBool(value)
			}
		}

		return mobileRedirectSettings
	}

	return nil
}

// toStrictTransportSecuritySettings
func toStrictTransportSecuritySettings(in interface{}) *v1alpha1.StrictTransportSecuritySettings {
	if m, ok := in.(map[string]interface{}); ok {
		stsSettings := &v1alpha1.StrictTransportSecuritySettings{}
		for key, value := range m {
			switch key {
			case cfsStrictTransportSecurityEnabled:
				stsSettings.Enabled = clients.ToBool(value)
			case cfsStrictTransportSecurityMaxAge:
				stsSettings.MaxAge = clients.ToNumber(value)
			case cfsStrictTransportSecurityIncludeSubdomains:
				stsSettings.IncludeSubdomains = clients.ToBool(value)
			case cfsStrictTransportSecurityNoSniff:
				stsSettings.NoSniff = clients.ToBool(value)
			default:
			}
		}

		return stsSettings
	}

	return nil
}

// toSecurityHeaderSettings converts an interface from the Cloudflare API
// into a SecurityHeaderSettings type.
func toSecurityHeaderSettings(in interface{}) *v1alpha1.SecurityHeaderSettings {
	if m, ok := in.(map[string]interface{}); ok {
		securityHeaderSettings := &v1alpha1.SecurityHeaderSettings{}
		for key, value := range m {
			switch key { //nolint:gocritic
			case cfsStrictTransportSecurity:
				securityHeaderSettings.StrictTransportSecurity = toStrictTransportSecuritySettings(value)
			}
		}

		return securityHeaderSettings
	}

	return nil
}

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

func lateInitializeMinifySettings(observed, desired *v1alpha1.MinifySettings) bool {
	li := false

	if desired.CSS == nil {
		desired.CSS = observed.CSS
		li = true
	}
	if desired.HTML == nil {
		desired.HTML = observed.HTML
		li = true
	}
	if desired.JS == nil {
		desired.JS = observed.JS
		li = true
	}

	return li
}

func lateInitializeMobileRedirectSettings(observed, desired *v1alpha1.MobileRedirectSettings) bool {
	li := false

	if desired.Status == nil {
		desired.Status = observed.Status
		li = true
	}
	if desired.Subdomain == nil {
		desired.Subdomain = observed.Subdomain
		li = true
	}
	if desired.StripURI == nil {
		desired.StripURI = observed.StripURI
		li = true
	}

	return li
}

func lateInitializeSecurityHeaderSettings(observed, desired *v1alpha1.SecurityHeaderSettings) bool {
	li := false

	if desired.StrictTransportSecurity == nil {
		desired.StrictTransportSecurity = observed.StrictTransportSecurity
		return true
	}

	osts := observed.StrictTransportSecurity
	dsts := desired.StrictTransportSecurity

	if dsts.Enabled == nil {
		dsts.Enabled = osts.Enabled
		li = true
	}
	if dsts.MaxAge == nil {
		dsts.MaxAge = osts.MaxAge
		li = true
	}
	if dsts.IncludeSubdomains == nil {
		dsts.IncludeSubdomains = osts.IncludeSubdomains
		li = true
	}
	if dsts.NoSniff == nil {
		dsts.NoSniff = osts.NoSniff
		li = true
	}

	return li
}

// LateInitializeSettings initializes Settings based on the remote resource
func LateInitializeSettings(observed, desired ZoneSettingsMap, initOn *v1alpha1.ZoneSettings) bool { //nolint:gocyclo
	// Gocyclo disabled - perhaps the "complex" setting `else` branch should be extracted out?
	li := false
	nestedLateInit := false

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
		} else {
			// Handle "complex" settings specially.
			// These might be set in our spec, but still have nested settings
			// that need late initialisation from the remote state.
			switch k {
			case cfsMinify:
				obsMinify := toMinifySettings(v)
				if obsMinify != nil {
					nestedLateInit = lateInitializeMinifySettings(obsMinify, initOn.Minify)
				}

			case cfsMobileRedirect:
				obsMobileRedirect := toMobileRedirectSettings(v)
				if obsMobileRedirect != nil {
					nestedLateInit = lateInitializeMobileRedirectSettings(obsMobileRedirect, initOn.MobileRedirect)
				}

			case cfsSecurityHeader:
				obsSecurityHeader := toSecurityHeaderSettings(v)
				if obsSecurityHeader != nil {
					nestedLateInit = lateInitializeSecurityHeaderSettings(obsSecurityHeader, initOn.SecurityHeader)
				}
			}
		}
	}
	// If we lateInited any top-level fields, update them on the
	// Zone settings.
	if li {
		settingsMapToZone(desired, initOn)
	}

	return li || nestedLateInit
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
	zs.Ciphers = clients.ToStringSlice(sm[cfsCiphers])
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
	zs.Minify = toMinifySettings(sm[cfsMinify])
	zs.MinTLSVersion = clients.ToString(sm[cfsMinTLSVersion])
	zs.Mirage = clients.ToString(sm[cfsMirage])
	zs.MobileRedirect = toMobileRedirectSettings(sm[cfsMobileRedirect])
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
	zs.SecurityHeader = toSecurityHeaderSettings(sm[cfsSecurityHeader])
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

// minifySettingsToMap converts a MinifySettings struct to the shape expected by the
// Cloudflare API. This may not necessarily exactly match our local JSON format
func minifySettingsToMap(settings *v1alpha1.MinifySettings) map[string]interface{} {
	m := make(map[string]interface{})

	if settings.CSS != nil {
		m[cfsMinifyCSS] = *settings.CSS
	}
	if settings.HTML != nil {
		m[cfsMinifyHTML] = *settings.HTML
	}
	if settings.JS != nil {
		m[cfsMinifyJS] = *settings.JS
	}

	return m
}

// mobileRedirectSettingsToMap converts a MobileRedirectSettings struct to the shape expected by the
// Cloudflare API. This may not necessarily exactly match our local JSON format
func mobileRedirectSettingsToMap(settings *v1alpha1.MobileRedirectSettings) map[string]interface{} {
	m := make(map[string]interface{})

	if settings.Status != nil {
		m[cfsMobileRedirectStatus] = *settings.Status
	}
	if settings.StripURI != nil {
		m[cfsMobileRedirectStripURI] = *settings.StripURI
	}
	if settings.Subdomain != nil {
		m[cfsMobileRedirectSubdomain] = *settings.Subdomain
	}

	return m
}

// securityHeaderSettingsToMap converts a MobileRedirectSettings struct to the shape expected by the
// Cloudflare API. This may not necessarily exactly match our local JSON format
func securityHeaderSettingsToMap(settings *v1alpha1.SecurityHeaderSettings) map[string]interface{} {
	m := make(map[string]interface{})

	if settings.StrictTransportSecurity != nil {
		sts := settings.StrictTransportSecurity
		stsSettings := make(map[string]interface{})

		if sts.Enabled != nil {
			stsSettings[cfsStrictTransportSecurityEnabled] = *sts.Enabled
		}
		if sts.IncludeSubdomains != nil {
			stsSettings[cfsStrictTransportSecurityIncludeSubdomains] = *sts.IncludeSubdomains
		}
		if sts.MaxAge != nil {
			stsSettings[cfsStrictTransportSecurityMaxAge] = *sts.MaxAge
		}
		if sts.NoSniff != nil {
			stsSettings[cfsStrictTransportSecurityNoSniff] = *sts.NoSniff
		}

		m[cfsStrictTransportSecurity] = stsSettings
	}

	return m
}

func mapSet(sm ZoneSettingsMap, key string, value interface{}) { //nolint:gocyclo
	// Gocyclo ignored here in anticipation of later refactoring
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
	case []string:
		if vt != nil {
			sm[key] = vt
		}
	case *v1alpha1.MinifySettings:
		if vt != nil {
			sm[key] = minifySettingsToMap(vt)
		}
	case *v1alpha1.MobileRedirectSettings:
		if vt != nil {
			sm[key] = mobileRedirectSettingsToMap(vt)
		}
	case *v1alpha1.SecurityHeaderSettings:
		if vt != nil {
			sm[key] = securityHeaderSettingsToMap(vt)
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
	mapSet(sm, cfsCiphers, zs.Ciphers)
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
	mapSet(sm, cfsMinify, zs.Minify)
	mapSet(sm, cfsMinTLSVersion, zs.MinTLSVersion)
	mapSet(sm, cfsMirage, zs.Mirage)
	mapSet(sm, cfsMobileRedirect, zs.MobileRedirect)
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
	mapSet(sm, cfsSecurityHeader, zs.SecurityHeader)
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
		if !cmp.Equal(cv, nv) {
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

	sortSlicesOpt := cmpopts.SortSlices(func(x, y string) bool {
		return x < y
	})

	if !cmp.Equal(spec.VanityNameServers, z.VanityNS, cmpopts.EquateEmpty(), sortSlicesOpt) {
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
