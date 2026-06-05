package models

// Entry represents a unified DNS/service entry with data from all sources
type Entry struct {
	// Identification
	Hostname string // e.g., "jellyfin.vookie.net"

	// Caddy (Source of Truth)
	CaddyUpstream string         // e.g., "192.168.1.112:8096"
	CaddyIP       string         // Extracted IP: "192.168.1.112"
	CaddyPort     string         // Extracted port: "8096"
	CaddyRoute    CaddyRouteInfo // full handler chain from Caddy config

	// DNS Services
	UnboundStatus ServiceStatus
	AdguardStatus ServiceStatus

	// DHCP
	DHCPStatus DHCPStatus

	// DNS Resolution (what DNS actually resolves to)
	DNSResolved string // Current DNS resolution result

	// Cloudflare
	CloudflareStatus CloudflareStatus

	// Computed Status
	OverallStatus SyncStatus // Overall sync state
	DataSource    string     // Where this entry came from
}

// IsConfiguredInCaddy returns true if this entry exists in Caddy
func (e *Entry) IsConfiguredInCaddy() bool {
	return e.CaddyUpstream != ""
}

// IsConfiguredInCloudflare returns true if this hostname has an ingress rule in any CF tunnel.
func (e *Entry) IsConfiguredInCloudflare() bool {
	return e.CloudflareStatus.Configured
}

// NeedsHTTPHostHeader returns true if the entry is in CF but HTTPHostHeader is not set.
// Without HTTPHostHeader, CF→Caddy routing typically breaks.
func (e *Entry) NeedsHTTPHostHeader() bool {
	return e.CloudflareStatus.Configured && e.CloudflareStatus.HTTPHostHeader == ""
}

// HasDNSMismatch returns true if DNS resolves to something other than Caddy
func (e *Entry) HasDNSMismatch() bool {
	if e.DNSResolved == "" || e.DNSResolved == "NONE" || e.DNSResolved == "FAIL" {
		return false
	}
	// Caddy server IP would typically be the target (e.g., 192.168.1.15)
	// This would need to be passed in from configuration
	return false // TODO: Implement proper DNS mismatch detection
}

// NeedsSyncToUnbound returns true if Unbound needs to be updated
func (e *Entry) NeedsSyncToUnbound() bool {
	if !e.IsConfiguredInCaddy() {
		return false
	}
	return !e.UnboundStatus.InSync
}

// NeedsSyncToAdguard returns true if AdGuard needs to be updated
func (e *Entry) NeedsSyncToAdguard() bool {
	if !e.IsConfiguredInCaddy() {
		return false
	}
	return !e.AdguardStatus.InSync
}

// NeedsDHCPStaticEntry returns true if a static DHCP entry should be created
func (e *Entry) NeedsDHCPStaticEntry() bool {
	if !e.IsConfiguredInCaddy() {
		return false
	}
	// Suggest static entry if currently dynamic
	return e.DHCPStatus.Configured && e.DHCPStatus.Type == "dynamic"
}

// NeedsRemovalFromUnbound returns true if this entry should be removed from Unbound
// (configured in Unbound but NOT in Caddy anymore)
func (e *Entry) NeedsRemovalFromUnbound() bool {
	// Only remove if it's configured in Unbound but NOT in Caddy
	return e.UnboundStatus.Configured && !e.IsConfiguredInCaddy()
}

// NeedsRemovalFromAdguard returns true if this entry should be removed from AdGuard
// (configured in AdGuard but NOT in Caddy anymore)
func (e *Entry) NeedsRemovalFromAdguard() bool {
	// Only remove if it's configured in AdGuard but NOT in Caddy
	return e.AdguardStatus.Configured && !e.IsConfiguredInCaddy()
}
