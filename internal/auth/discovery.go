// Package auth provides the auth discovery layer that cross-references
// Caddy, Cloudflare Access, and Authentik to classify each hostname's
// authentication configuration into the WAN/LAN/API model.
//
// The discovery is read-only — it queries all three services and builds
// a HostAuth per hostname. No mutations are performed.
package auth

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

// DiscoveryOptions controls which auth sources are queried.
type DiscoveryOptions struct {
	// If a client is nil, that source is skipped and the corresponding
	// HostAuth fields are left empty.
}

// StreamEvent is a single event emitted by DiscoverStream.
type StreamEvent struct {
	Type string `json:"type"` // "base", "enrich", "done", "error"

	// For "base" and "enrich" events: the hosts being sent.
	Hosts []*models.HostAuth `json:"hosts,omitempty"`

	// For "enrich" events: which source completed ("cloudflare" or "authentik").
	Source string `json:"source,omitempty"`

	// For "done" events: which sources were queried.
	Sources *AuthSources `json:"sources,omitempty"`

	// For "error" events.
	Error string `json:"error,omitempty"`
}

// AuthSources indicates which auth discovery sources were queried.
type AuthSources struct {
	CloudflareAccess bool `json:"cloudflare_access"`
	Authentik        bool `json:"authentik"`
}

// DiscoverStream runs auth discovery and emits events as each phase completes.
// This allows the frontend to render hosts incrementally:
//  1. "base" — all hosts with Caddy-derived auth (instant, no API calls)
//  2. "enrich" — updated hosts after CF Access data arrives (source="cloudflare")
//  3. "enrich" — updated hosts after Authentik data arrives (source="authentik")
//  4. "done" — discovery complete, includes source availability
//
// If an API call fails, an "error" event is sent but discovery continues
// with the remaining sources.
func DiscoverStream(
	ctx context.Context,
	entries []*models.Entry,
	cfClient *api.CloudflareClient,
	akClient *api.AuthentikClient,
	emit func(StreamEvent),
) {
	if ctx == nil {
		ctx = context.Background()
	}

	akHostname := authentikHostnameFromClient(akClient)

	// Phase 1: Build base auth from entries (instant — no API calls).
	authMap := make(map[string]*models.HostAuth, len(entries))
	for _, e := range entries {
		if e == nil {
			continue
		}
		authMap[e.Hostname] = buildBaseAuth(e)
	}

	// Classify base auth and emit immediately.
	// For WAN-exposed hosts, we can't determine WAN auth until CF Access
	// enrichment completes (a wildcard CF Access app may cover them).
	// So we set status to "unknown" and leave WAN auth as "none" to indicate
	// "not yet classified" — the enrich event will re-classify with full data.
	baseHosts := make([]*models.HostAuth, 0, len(authMap))
	for _, ha := range authMap {
		if ha.WANExposed && cfClient != nil {
			// Don't classify WAN auth yet — CF Access data is pending.
			// Set LAN auth (known from Caddy) and mark status as unknown.
			if ha.HasForwardAuth {
				ha.LANAuth = models.LANAuthForwardAuth
			}
			ha.Status = models.AuthStatusUnknown
			ha.Notes = []string{"Awaiting CF Access enrichment…"}
		} else {
			classifyAuth(ha, akHostname)
		}
		baseHosts = append(baseHosts, ha)
	}
	emit(StreamEvent{Type: "base", Hosts: baseHosts})

	sources := AuthSources{
		CloudflareAccess: cfClient != nil,
		Authentik:        akClient != nil,
	}

	// Phase 2 + 3: Query CF Access and Authentik in parallel.
	// Each source emits an "enrich" event as soon as it completes.
	var wg sync.WaitGroup

	if cfClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfApps, cfPolicies, err := discoverCloudflareAccess(ctx, cfClient)
			if err != nil {
				logging.Warn("CF Access discovery failed", "error", err)
				emit(StreamEvent{Type: "error", Source: "cloudflare", Error: err.Error()})
				// Still re-classify all hosts that were pending (unknown status).
				updated := collectUpdatedHosts(authMap, func(ha *models.HostAuth) bool {
					return ha.Status == models.AuthStatusUnknown
				})
				for _, ha := range updated {
					classifyAuth(ha, akHostname)
				}
				if len(updated) > 0 {
					emit(StreamEvent{Type: "enrich", Source: "cloudflare", Hosts: updated})
				}
				return
			}
			// Enrich and emit ALL hosts — even if a host doesn't match a CF
			// Access app, it needs re-classification from "unknown" to its
			// final state (e.g., app_native with no CF Access).
			enrichWithCFAccess(authMap, cfApps, cfPolicies)
			updated := collectUpdatedHosts(authMap, func(ha *models.HostAuth) bool {
				// Send all WAN-exposed hosts (they were marked unknown in base).
				return ha.WANExposed
			})
			for _, ha := range updated {
				classifyAuth(ha, akHostname)
			}
			if len(updated) > 0 {
				emit(StreamEvent{Type: "enrich", Source: "cloudflare", Hosts: updated})
			}
		}()
	}

	if akClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			akProviders, akOutposts, err := discoverAuthentik(ctx, akClient)
			if err != nil {
				logging.Warn("Authentik discovery failed", "error", err)
				emit(StreamEvent{Type: "error", Source: "authentik", Error: err.Error()})
				return
			}
			// Enrich and emit updated hosts.
			enrichWithAuthentik(authMap, akProviders, akOutposts)
			updated := collectUpdatedHosts(authMap, func(ha *models.HostAuth) bool {
				return ha.AuthentikProviderPK > 0
			})
			for _, ha := range updated {
				classifyAuth(ha, akHostname)
			}
			if len(updated) > 0 {
				emit(StreamEvent{Type: "enrich", Source: "authentik", Hosts: updated})
			}
		}()
	}

	wg.Wait()

	// Phase 4: Re-classify all hosts (in case enrichment changed status)
	// and emit done.
	for _, ha := range authMap {
		classifyAuth(ha, akHostname)
	}
	emit(StreamEvent{Type: "done", Sources: &sources})
}

// collectUpdatedHosts returns hosts that match the predicate, sorted by hostname.
func collectUpdatedHosts(authMap map[string]*models.HostAuth, pred func(*models.HostAuth) bool) []*models.HostAuth {
	var result []*models.HostAuth
	for _, ha := range authMap {
		if pred(ha) {
			result = append(result, ha)
		}
	}
	// Sort by hostname for stable ordering.
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j-1].Hostname > result[j].Hostname; j-- {
			result[j-1], result[j] = result[j], result[j-1]
		}
	}
	return result
}

// Discover queries all auth sources and returns a map of hostname → HostAuth.
//
// Parameters:
//   - entries: the current Entry list (provides Caddy forward_auth state and CF tunnel state)
//   - cfClient: Cloudflare client (for Access apps, policies, service tokens)
//   - akClient: Authentik client (for proxy providers, applications, outposts)
//
// If a client is nil, that source is skipped gracefully.
func Discover(
	ctx context.Context,
	entries []*models.Entry,
	cfClient *api.CloudflareClient,
	akClient *api.AuthentikClient,
) (map[string]*models.HostAuth, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	akHostname := authentikHostnameFromClient(akClient)

	// Build the initial HostAuth from Entry data (Caddy + CF tunnel state).
	// This gives us the base classification before we enrich with CF Access
	// and Authentik API data.
	authMap := make(map[string]*models.HostAuth, len(entries))
	for _, e := range entries {
		if e == nil {
			continue
		}
		authMap[e.Hostname] = buildBaseAuth(e)
	}

	// Query CF Access and Authentik in parallel.
	var (
		cfApps     []api.AccessAppInfo
		cfPolicies map[string][]api.AccessPolicyInfo // appID → policies
		akProviders []api.ProxyProviderInfo
		akOutposts  []api.OutpostInfo

		cfErr error
		akErr error

		wg sync.WaitGroup
	)

	if cfClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfApps, cfPolicies, cfErr = discoverCloudflareAccess(ctx, cfClient)
			if cfErr != nil {
				logging.Warn("CF Access discovery failed", "error", cfErr)
			}
		}()
	}

	if akClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			akProviders, akOutposts, akErr = discoverAuthentik(ctx, akClient)
			if akErr != nil {
				logging.Warn("Authentik discovery failed", "error", akErr)
			}
		}()
	}

	wg.Wait()

	// Enrich authMap with CF Access data.
	if cfErr == nil {
		enrichWithCFAccess(authMap, cfApps, cfPolicies)
	}

	// Enrich authMap with Authentik data.
	if akErr == nil {
		enrichWithAuthentik(authMap, akProviders, akOutposts)
	}

	// Classify each host's auth modes and status now that all data is merged.
	for hostname, ha := range authMap {
		classifyAuth(ha, akHostname)
		logging.Debug("Auth discovery result",
			"hostname", hostname,
			"wan", ha.WANAuth,
			"lan", ha.LANAuth,
			"api", ha.APIAuth,
			"status", ha.Status,
		)
	}

	return authMap, nil
}

// buildBaseAuth creates the initial HostAuth from Entry data.
// This captures what we already know from Caddy route parsing and CF tunnel state.
func buildBaseAuth(e *models.Entry) *models.HostAuth {
	ha := &models.HostAuth{
		Hostname:               e.Hostname,
		HasForwardAuth:         e.CaddyRoute.HasForwardAuth,
		ConditionalForwardAuth: e.CaddyRoute.ConditionalForwardAuth,
		WANExposed:             e.CloudflareStatus.Configured,
		WANAuth:                models.WANAuthNone,
		LANAuth:                models.LANAuthNone,
		APIAuth:                models.APIAuthNone,
		Status:                 models.AuthStatusUnknown,
	}

	// If Caddy has forward_auth, the LAN auth is forward_auth.
	if e.CaddyRoute.HasForwardAuth {
		ha.LANAuth = models.LANAuthForwardAuth
	}

	return ha
}

// discoverCloudflareAccess queries CF Access apps and their policies.
func discoverCloudflareAccess(ctx context.Context, cfClient *api.CloudflareClient) (
	[]api.AccessAppInfo,
	map[string][]api.AccessPolicyInfo,
	error,
) {
	apps, err := cfClient.ListAccessApps()
	if err != nil {
		return nil, nil, fmt.Errorf("listing CF Access apps: %w", err)
	}

	policies := make(map[string][]api.AccessPolicyInfo)
	for _, app := range apps {
		pols, err := cfClient.ListAccessPolicies(app.ID)
		if err != nil {
			logging.Warn("Failed to list CF Access policies", "appID", app.ID, "error", err)
			continue
		}
		policies[app.ID] = pols
	}

	return apps, policies, nil
}

// discoverAuthentik queries Authentik proxy providers and outposts.
func discoverAuthentik(ctx context.Context, akClient *api.AuthentikClient) (
	[]api.ProxyProviderInfo,
	[]api.OutpostInfo,
	error,
) {
	providers, err := akClient.ListProxyProviders()
	if err != nil {
		return nil, nil, fmt.Errorf("listing Authentik proxy providers: %w", err)
	}

	outposts, err := akClient.ListOutposts()
	if err != nil {
		logging.Warn("Failed to list Authentik outposts", "error", err)
		outposts = nil
	}

	return providers, outposts, nil
}

// enrichWithCFAccess merges CF Access app and policy data into the auth map.
func enrichWithCFAccess(authMap map[string]*models.HostAuth, apps []api.AccessAppInfo, policies map[string][]api.AccessPolicyInfo) {
	// Build a hostname → app index. Apps can be per-hostname (domain="jellyfin.vookie.net")
	// or wildcard (domain="*.vookie.net").
	type appMatch struct {
		app      api.AccessAppInfo
		wildcard bool
	}
	exactMatches := make(map[string]appMatch)
	var wildcardMatches []appMatch

	for _, app := range apps {
		if isWildcardDomain(app.Domain) {
			wildcardMatches = append(wildcardMatches, appMatch{app: app, wildcard: true})
		} else {
			exactMatches[app.Domain] = appMatch{app: app}
		}
	}

	for hostname, ha := range authMap {
		var matched *appMatch

		// Try exact match first.
		if m, ok := exactMatches[hostname]; ok {
			matched = &m
		} else {
			// Try wildcard matches.
			for i, wm := range wildcardMatches {
				if wildcardMatchesDomain(wm.app.Domain, hostname) {
					matched = &wildcardMatches[i]
					break
				}
			}
		}

		if matched == nil {
			continue
		}

		ha.CFAccessAppID = matched.app.ID
		ha.CFAccessAppDomain = matched.app.Domain
		ha.CFAccessAppType = matched.app.Type

		// Get policies for this app.
		if pols, ok := policies[matched.app.ID]; ok {
			ha.CFAccessPolicyIDs = make([]string, 0, len(pols))
			ha.CFAccessDecisions = make([]string, 0, len(pols))
			for _, p := range pols {
				ha.CFAccessPolicyIDs = append(ha.CFAccessPolicyIDs, p.ID)
				ha.CFAccessDecisions = append(ha.CFAccessDecisions, string(p.Decision))
			}
		}
	}
}

// enrichWithAuthentik merges Authentik proxy provider data into the auth map.
func enrichWithAuthentik(authMap map[string]*models.HostAuth, providers []api.ProxyProviderInfo, outposts []api.OutpostInfo) {
	// Build a hostname → provider index. The provider's ExternalHost is
	// "https://hostname" so we strip the scheme.
	providerByHost := make(map[string]api.ProxyProviderInfo)
	for _, p := range providers {
		hostname := stripScheme(p.ExternalHost)
		if hostname != "" {
			providerByHost[hostname] = p
		}
	}

	// Build a provider PK → outpost UUID index.
	outpostByProvider := make(map[int32]string)
	for _, o := range outposts {
		for _, pk := range o.Providers {
			outpostByProvider[pk] = o.UUID
		}
	}

	for hostname, ha := range authMap {
		p, ok := providerByHost[hostname]
		if !ok {
			continue
		}

		ha.AuthentikProviderPK = p.PK
		ha.AuthentikAppSlug = p.AssignedAppSlug
		ha.AuthentikProviderMode = string(p.Mode)

		if outpostUUID, ok := outpostByProvider[p.PK]; ok {
			ha.AuthentikOutpostUUID = outpostUUID
		}
	}
}

// authentikHostnameFromClient extracts the hostname from the Authentik base URL.
// Returns empty string if akClient is nil or the URL is invalid.
func authentikHostnameFromClient(akClient *api.AuthentikClient) string {
	if akClient == nil {
		return ""
	}
	u, err := url.Parse(akClient.BaseURL())
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// classifyAuth determines the WAN/LAN/API auth modes and overall status
// based on the merged data from all sources.
//
// Classification logic:
//
// WAN auth (only relevant if WANExposed):
//   - CF Access app exists + no forward_auth → cf_access
//   - CF Access app exists + forward_auth + bypass policy → forward_auth
//   - CF Access app exists + forward_auth + no bypass → ERROR (double-login)
//   - CF Access app exists + conditional forward_auth (Caddy skips FA for CF tunnel) → cf_access (OK)
//   - No CF Access + forward_auth → forward_auth (Authentik handles it)
//   - No CF Access + no forward_auth → app_native (or none if we can't tell)
//
// LAN auth:
//   - forward_auth in Caddy → forward_auth
//   - No forward_auth → none (or app_native if the app has its own login)
//
// API auth:
//   - service_auth policy on CF Access app → cf_service_token
//   - Authentik provider with intercept_header_auth → authentik_bearer
//   - Otherwise → none
func classifyAuth(ha *models.HostAuth, authentikHostname string) {
	var notes []string

	// --- WAN auth ---
	if ha.WANExposed {
		hasCFAccess := ha.CFAccessAppID != ""
		hasBypassPolicy := hasPolicyDecision(ha, "bypass")
		hasAllowPolicy := hasPolicyDecision(ha, "allow")
		hasServiceAuthPolicy := hasPolicyDecision(ha, "service_auth")

		// Check if this host IS the Authentik identity provider itself.
		// The IdP must have a CF Access bypass policy (circular dependency —
		// can't put auth in front of the auth provider). This is expected
		// and safe — Authentik has its own native login.
		isAuthProvider := authentikHostname != "" && ha.Hostname == authentikHostname

		switch {
		case isAuthProvider && hasCFAccess && hasBypassPolicy && !ha.HasForwardAuth:
			// This is the Authentik IdP itself — bypass is required (circular dependency).
			ha.WANAuth = models.WANAuthAppNative
			notes = append(notes, "Authentik identity provider — CF Access bypass required (circular dependency), native auth handles login")
			ha.Status = models.AuthStatusOK

		case hasCFAccess && !ha.HasForwardAuth && hasBypassPolicy && !hasAllowPolicy && !hasServiceAuthPolicy:
			// CRITICAL: CF Access has only bypass policies and no forward_auth.
			// This means the host is WIDE OPEN to the internet with zero auth.
			ha.WANAuth = models.WANAuthNone
			notes = append(notes, "CRITICAL: CF Access bypass-only with no forward_auth — host is OPEN to the internet")
			ha.Status = models.AuthStatusError

		case hasCFAccess && ha.HasForwardAuth && !hasBypassPolicy && !ha.ConditionalForwardAuth:
			// Pattern F: CF Access + forward_auth without bypass = double login
			ha.WANAuth = models.WANAuthCFAccess
			notes = append(notes, "Double-login risk: CF Access + forward_auth without bypass policy")
			ha.Status = models.AuthStatusError

		case hasCFAccess && ha.HasForwardAuth && !hasBypassPolicy && ha.ConditionalForwardAuth:
			// Pattern E (DEPRECATED): CF Access + conditional forward_auth (Caddy skips FA for CF tunnel)
			// With CF Access auto_redirect_to_identity, the split-horizon pattern is no longer
			// needed. It adds complexity and risk (misconfigured matchers can leave hosts open).
			// Simplify to CF Access only (remove forward_auth + matchers from Caddyfile).
			ha.WANAuth = models.WANAuthCFAccess
			ha.Status = models.AuthStatusWarning
			notes = append(notes, "DEPRECATED: conditional forward_auth — simplify to CF Access only (auto_redirect_to_identity makes split-horizon unnecessary)")

		case hasCFAccess && ha.HasForwardAuth && hasBypassPolicy:
			// Pattern D: CF Access bypasses, forward_auth handles actual auth
			ha.WANAuth = models.WANAuthForwardAuth
			notes = append(notes, "CF Access bypass → forward_auth")

		case hasCFAccess && !ha.HasForwardAuth:
			// Pattern A/B: CF Access handles auth
			ha.WANAuth = models.WANAuthCFAccess

		case !hasCFAccess && ha.HasForwardAuth:
			// Pattern C: forward_auth without CF Access
			ha.WANAuth = models.WANAuthForwardAuth

		case !hasCFAccess && !ha.HasForwardAuth:
			// No auth layer at all — either app-native or completely open
			// We can't distinguish app-native from "no auth" without app-specific
			// knowledge, so we classify as app_native (optimistic) but flag it.
			ha.WANAuth = models.WANAuthAppNative
			notes = append(notes, "No CF Access or forward_auth — assuming app-native auth")
		}

		// --- API auth (only relevant for WAN-exposed hosts) ---
		switch {
		case hasServiceAuthPolicy:
			ha.APIAuth = models.APIAuthCFServiceToken
		case ha.AuthentikProviderPK > 0:
			// We don't currently check intercept_header_auth in the discovery
			// (it's not exposed in ProxyProviderInfo). Assume the provider
			// could support bearer auth if it exists.
			ha.APIAuth = models.APIAuthAuthentikBearer
		default:
			ha.APIAuth = models.APIAuthNone
		}
	} else {
		// Not WAN-exposed — WAN auth is N/A
		ha.WANAuth = models.WANAuthNone
	}

	// --- LAN auth ---
	if ha.HasForwardAuth {
		ha.LANAuth = models.LANAuthForwardAuth
	} else {
		ha.LANAuth = models.LANAuthNone
	}

	// --- Status ---
	if ha.Status != models.AuthStatusError {
		// Check for warnings
		if ha.WANExposed && ha.WANAuth == models.WANAuthNone {
			ha.Status = models.AuthStatusError
			notes = append(notes, "WAN-exposed host with no auth")
		} else if ha.WANExposed && ha.WANAuth == models.WANAuthAppNative && ha.CFAccessAppID == "" && !ha.HasForwardAuth {
			// WAN-exposed with no auth layer — could be app-native or could be open
			ha.Status = models.AuthStatusWarning
		} else if ha.LANAuth == models.LANAuthForwardAuth && ha.WANAuth == models.WANAuthCFAccess {
			// Split auth: CF Access on WAN, forward_auth on LAN
			ha.Status = models.AuthStatusWarning
			notes = append(notes, "Split auth: CF Access (WAN) + forward_auth (LAN)")
		} else {
			ha.Status = models.AuthStatusOK
		}
	}

	ha.Notes = notes
}

// hasPolicyDecision checks if any of the host's CF Access policies has the
// given decision (e.g., "bypass", "service_auth", "allow", "deny").
func hasPolicyDecision(ha *models.HostAuth, decision string) bool {
	for _, d := range ha.CFAccessDecisions {
		if d == decision {
			return true
		}
	}
	return false
}

// --- helpers ---

func isWildcardDomain(domain string) bool {
	return len(domain) > 2 && domain[:2] == "*."
}

func wildcardMatchesDomain(wildcard, hostname string) bool {
	if !isWildcardDomain(wildcard) {
		return false
	}
	suffix := wildcard[1:] // ".vookie.net"
	return len(hostname) > len(suffix) && hostname[len(hostname)-len(suffix):] == suffix
}

func stripScheme(url string) string {
	if len(url) > 8 && url[:8] == "https://" {
		return url[8:]
	}
	if len(url) > 7 && url[:7] == "http://" {
		return url[7:]
	}
	return url
}
