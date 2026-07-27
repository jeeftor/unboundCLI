package sync

import (
	"fmt"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
)

// IsLegacyDescription checks if the description matches one of the known
// legacy description strings. This is the single shared implementation used
// by all sync flows and the startup migration logic.
func IsLegacyDescription(desc string, legacyDescriptions []string) bool {
	for _, legacyDesc := range legacyDescriptions {
		if desc == legacyDesc {
			return true
		}
	}
	return false
}

// IsSyncOwnedOverride returns true if the override's description matches the
// current entry description or any of the legacy descriptions. This determines
// whether the sync engine owns (can update/delete) the override.
func IsSyncOwnedOverride(override api.DNSOverride, entryDescription string, legacyDescriptions []string) bool {
	return override.Description == entryDescription ||
		IsLegacyDescription(override.Description, legacyDescriptions)
}

// OrganizeOverridesByOwnership splits existing Unbound overrides into two maps:
//   - syncCreated: overrides owned by the sync engine (description matches)
//   - other: overrides created manually or by other tools
//
// The key for each map is "host.domain" (the FQDN).
func OrganizeOverridesByOwnership(
	existingOverrides []api.DNSOverride,
	entryDescription string,
	legacyDescriptions []string,
) (syncCreated, other map[string]api.DNSOverride) {
	syncCreated = make(map[string]api.DNSOverride)
	other = make(map[string]api.DNSOverride)

	for _, override := range existingOverrides {
		key := fmt.Sprintf("%s.%s", override.Host, override.Domain)
		if IsSyncOwnedOverride(override, entryDescription, legacyDescriptions) {
			syncCreated[key] = override
		} else {
			other[key] = override
		}
	}
	return syncCreated, other
}

// SplitHostname splits a FQDN into host and domain parts using the first dot
// as the separator. For example "app.example.com" returns ("app", "example.com").
// If the hostname contains no dot, domain will be empty.
func SplitHostname(hostname string) (host, domain string) {
	parts := strings.SplitN(hostname, ".", 2)
	host = parts[0]
	if len(parts) > 1 {
		domain = parts[1]
	}
	return host, domain
}

// IsFQDN returns true if the hostname contains at least one dot.
func IsFQDN(hostname string) bool {
	return strings.Contains(hostname, ".")
}

// BaseSyncOptions contains the common fields shared by all sync option structs
// in the exec/sync package. Embed this in option structs to avoid duplicating
// these fields across CaddySyncOptions, CaddyAdguardSyncOptions, etc.
type BaseSyncOptions struct {
	DryRun             bool
	CaddyServerIP      string
	CaddyServerPort    int
	EntryDescription   string
	LegacyDescriptions []string
	Verbose            bool
}
