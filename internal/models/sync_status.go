package models

// SyncStatus represents the overall synchronization state of an entry
type SyncStatus int

const (
	// FullyInSync - All services are configured and match Caddy
	FullyInSync SyncStatus = iota

	// PartiallyInSync - Some services are configured, others missing
	PartiallyInSync

	// OutOfSync - Services are configured but with wrong IPs
	OutOfSync

	// CaddyOnly - Entry exists only in Caddy, not in DNS services
	CaddyOnly

	// Stale - Entry exists in DNS services but not in Caddy (should be removed)
	Stale

	// DHCPMismatch - DHCP lease IP doesn't match Caddy upstream
	DHCPMismatch
)

// String returns the string representation of SyncStatus
func (s SyncStatus) String() string {
	switch s {
	case FullyInSync:
		return "FullyInSync"
	case PartiallyInSync:
		return "PartiallyInSync"
	case OutOfSync:
		return "OutOfSync"
	case CaddyOnly:
		return "CaddyOnly"
	case Stale:
		return "Stale"
	case DHCPMismatch:
		return "DHCPMismatch"
	default:
		return "Unknown"
	}
}

// Icon returns the icon for this sync status
func (s SyncStatus) Icon() string {
	switch s {
	case FullyInSync:
		return "✓"
	case PartiallyInSync:
		return "⚠"
	case OutOfSync:
		return "✗"
	case CaddyOnly:
		return "📋"
	case Stale:
		return "🗑"
	case DHCPMismatch:
		return "⚠"
	default:
		return "?"
	}
}

// Label returns a human-readable label
func (s SyncStatus) Label() string {
	switch s {
	case FullyInSync:
		return "Synced"
	case PartiallyInSync:
		return "Partial"
	case OutOfSync:
		return "Out of Sync"
	case CaddyOnly:
		return "Caddy Only"
	case Stale:
		return "Stale"
	case DHCPMismatch:
		return "DHCP Mismatch"
	default:
		return "Unknown"
	}
}

// ComputeSyncStatus calculates the overall sync status for an entry
func ComputeSyncStatus(entry *Entry, caddyServerIP string) SyncStatus {
	inCaddy := entry.IsConfiguredInCaddy()
	inUnbound := entry.UnboundStatus.Configured
	inAdguard := entry.AdguardStatus.Configured

	// Stale: exists in DNS but not in Caddy
	if !inCaddy && (inUnbound || inAdguard) {
		return Stale
	}

	// Caddy Only: exists in Caddy but not configured in DNS services
	if inCaddy && !inUnbound && !inAdguard {
		return CaddyOnly
	}

	// Out of Sync: configured but IPs don't match
	if inCaddy {
		unboundWrong := inUnbound && !entry.UnboundStatus.InSync
		adguardWrong := inAdguard && !entry.AdguardStatus.InSync

		if unboundWrong || adguardWrong {
			return OutOfSync
		}
	}

	// Partially In Sync: one service configured, other missing
	if inCaddy && (inUnbound != inAdguard) {
		return PartiallyInSync
	}

	// Fully In Sync: both services configured with correct IPs
	if inCaddy && inUnbound && inAdguard &&
		entry.UnboundStatus.InSync && entry.AdguardStatus.InSync {
		return FullyInSync
	}

	// Default to Caddy Only if nothing else matches
	return CaddyOnly
}

// FilterMode represents different filtering modes for the table view
type FilterMode int

const (
	FilterNone FilterMode = iota
	FilterOutOfSync
	FilterMismatches
	FilterCaddyOnly
	FilterStale
	FilterUnboundIssues
	FilterAdguardIssues
	FilterDHCPMismatches
)

// String returns the string representation of FilterMode
func (f FilterMode) String() string {
	switch f {
	case FilterNone:
		return "All Services"
	case FilterOutOfSync:
		return "Out of Sync"
	case FilterMismatches:
		return "Mismatches"
	case FilterCaddyOnly:
		return "Caddy Only"
	case FilterStale:
		return "Stale Entries"
	case FilterUnboundIssues:
		return "Unbound Issues"
	case FilterAdguardIssues:
		return "AdGuard Issues"
	case FilterDHCPMismatches:
		return "DHCP Mismatches"
	default:
		return "Unknown"
	}
}

// ApplyFilter checks if an entry passes the given filter
func ApplyFilter(entry *Entry, filter FilterMode) bool {
	switch filter {
	case FilterNone:
		return true
	case FilterOutOfSync:
		return entry.OverallStatus == OutOfSync
	case FilterMismatches:
		return entry.OverallStatus == OutOfSync || entry.OverallStatus == PartiallyInSync
	case FilterCaddyOnly:
		return entry.OverallStatus == CaddyOnly
	case FilterStale:
		return entry.OverallStatus == Stale
	case FilterUnboundIssues:
		return entry.IsConfiguredInCaddy() && (!entry.UnboundStatus.Configured || !entry.UnboundStatus.InSync)
	case FilterAdguardIssues:
		return entry.IsConfiguredInCaddy() && (!entry.AdguardStatus.Configured || !entry.AdguardStatus.InSync)
	case FilterDHCPMismatches:
		return !entry.DHCPStatus.InSync && entry.DHCPStatus.Configured
	default:
		return true
	}
}
