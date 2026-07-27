package models

// WANAuthMode describes how WAN (internet-facing) traffic is authenticated.
type WANAuthMode string

const (
	// WANAuthNone — no WAN authentication configured.
	// This is a misconfiguration for exposed hosts — they're reachable from
	// the internet without any auth barrier.
	WANAuthNone WANAuthMode = "none"
	// WANAuthCFAccess — Cloudflare Access sits at the edge and requires
	// IdP login before traffic reaches the tunnel.
	WANAuthCFAccess WANAuthMode = "cf_access"
	// WANAuthForwardAuth — Caddy's forward_auth delegates to Authentik.
	// CF Access must have a bypass policy to avoid double-login.
	WANAuthForwardAuth WANAuthMode = "forward_auth"
	// WANAuthAppNative — the application handles its own authentication
	// (e.g., Jellyfin, Audiobookshelf have built-in login).
	WANAuthAppNative WANAuthMode = "app_native"
)

// LANAuthMode describes how LAN (internal) traffic is authenticated.
type LANAuthMode string

const (
	// LANAuthNone — no LAN authentication. The app is directly accessible
	// on the LAN. This is the default for apps with their own auth.
	LANAuthNone LANAuthMode = "none"
	// LANAuthForwardAuth — Caddy's forward_auth delegates to Authentik
	// even on LAN requests.
	LANAuthForwardAuth LANAuthMode = "forward_auth"
	// LANAuthAppNative — the application handles its own authentication.
	LANAuthAppNative LANAuthMode = "app_native"
)

// APIAuthMode describes how API/machine-to-machine access is authenticated.
type APIAuthMode string

const (
	// APIAuthNone — no API-specific auth. API calls use the same auth
	// as browser traffic (or none if WAN/LAN is none).
	APIAuthNone APIAuthMode = "none"
	// APIAuthCFServiceToken — Cloudflare Access service token
	// (CF-Access-Client-Id / CF-Access-Client-Secret headers).
	APIAuthCFServiceToken APIAuthMode = "cf_service_token"
	// APIAuthAuthentikBearer — Authentik bearer token
	// (Authorization: Bearer <token>).
	APIAuthAuthentikBearer APIAuthMode = "authentik_bearer"
	// APIAuthAppNativeKey — the application's own API key mechanism.
	APIAuthAppNativeKey APIAuthMode = "app_native_key"
)

// AuthStatus indicates the health/classification of a host's auth configuration.
type AuthStatus string

const (
	// AuthStatusOK — auth is properly configured for this host.
	AuthStatusOK AuthStatus = "ok"
	// AuthStatusWarning — auth works but has a non-ideal configuration
	// (e.g., split WAN/LAN modes, forward_auth without CF bypass).
	AuthStatusWarning AuthStatus = "warning"
	// AuthStatusError — auth is missing or broken
	// (e.g., WAN-exposed host with no auth, double-login risk).
	AuthStatusError AuthStatus = "error"
	// AuthStatusUnknown — auth state couldn't be determined
	// (e.g., Authentik/CF Access API unavailable).
	AuthStatusUnknown AuthStatus = "unknown"
)

// HostAuth captures the discovered authentication state for a hostname.
// This is populated by the auth discovery layer (internal/auth) which
// cross-references Caddy, Cloudflare Access, and Authentik.
type HostAuth struct {
	Hostname string `json:"hostname"`

	// Auth modes (classified by the discovery layer)
	WANAuth WANAuthMode `json:"wan_auth"`
	LANAuth LANAuthMode `json:"lan_auth"`
	APIAuth APIAuthMode `json:"api_auth"`

	// Overall status
	Status AuthStatus `json:"status"`

	// Discovered Cloudflare Access state
	CFAccessAppID       string   `json:"cf_access_app_id,omitempty"`
	CFAccessAppDomain   string   `json:"cf_access_app_domain,omitempty"`
	CFAccessPolicyIDs   []string `json:"cf_access_policy_ids,omitempty"`
	CFAccessDecisions   []string `json:"cf_access_decisions,omitempty"` // "allow", "bypass", "service_auth", "deny"
	CFAccessAppType     string   `json:"cf_access_app_type,omitempty"`  // "self_hosted", "wildcard", etc

	// Discovered Authentik state
	AuthentikProviderPK  int32  `json:"authentik_provider_pk,omitempty"`
	AuthentikAppSlug     string `json:"authentik_app_slug,omitempty"`
	AuthentikProviderMode string `json:"authentik_provider_mode,omitempty"` // forward_single, proxy, etc
	AuthentikOutpostUUID string `json:"authentik_outpost_uuid,omitempty"`

	// Caddy-side detection (already exists in CaddyRouteInfo, mirrored here for the auth view)
	HasForwardAuth bool `json:"has_forward_auth"`

	// Whether this host is WAN-exposed (has a CF tunnel ingress rule)
	WANExposed bool `json:"wan_exposed"`

	// Human-readable notes about the auth configuration
	Notes []string `json:"notes,omitempty"`
}

// HasCFAccess returns true if this host has a CF Access application.
func (h *HostAuth) HasCFAccess() bool {
	return h.CFAccessAppID != ""
}

// HasAuthentikProvider returns true if this host has an Authentik proxy provider.
func (h *HostAuth) HasAuthentikProvider() bool {
	return h.AuthentikProviderPK > 0
}

// IsDoubleLoginRisk returns true if both CF Access and forward_auth are
// active on WAN without a bypass policy — this causes the Pattern F
// double-login anti-pattern.
func (h *HostAuth) IsDoubleLoginRisk() bool {
	return h.WANAuth == WANAuthCFAccess && h.HasForwardAuth && h.WANExposed
}
