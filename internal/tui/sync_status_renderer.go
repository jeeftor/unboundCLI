package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// SyncStatusRenderer handles the visual rendering of the sync status dashboard
type SyncStatusRenderer struct {
	dashboard *SyncStatusDashboard
	width     int
	showIPs   bool
}

// NewSyncStatusRenderer creates a new renderer for the sync status dashboard
func NewSyncStatusRenderer(dashboard *SyncStatusDashboard) *SyncStatusRenderer {
	return &SyncStatusRenderer{
		dashboard: dashboard,
		width:     120, // Default terminal width
		showIPs:   false,
	}
}

// SetWidth sets the rendering width
func (r *SyncStatusRenderer) SetWidth(width int) {
	r.width = width
}

// SetShowIPs controls whether to show IP addresses in the table
func (r *SyncStatusRenderer) SetShowIPs(show bool) {
	r.showIPs = show
}

// RenderDashboard renders the complete sync status dashboard
func (r *SyncStatusRenderer) RenderDashboard() string {
	var sb strings.Builder

	// Header
	sb.WriteString(r.renderHeader())
	sb.WriteString("\n")

	// Summary
	sb.WriteString(r.renderSummary())
	sb.WriteString("\n")

	// Filters (if any are active)
	if r.hasActiveFilters() {
		sb.WriteString(r.renderActiveFilters())
		sb.WriteString("\n")
	}

	// Main table
	sb.WriteString(r.renderTable())
	sb.WriteString("\n")

	// Legend
	sb.WriteString(r.renderLegend())

	return sb.String()
}

// renderHeader renders the dashboard header
func (r *SyncStatusRenderer) renderHeader() string {
	var sb strings.Builder

	title := "🚀 CADDY-CENTRIC DNS SYNC STATUS DASHBOARD 🚀"
	padding := (r.width - utf8.RuneCountInString(title)) / 2
	if padding < 0 {
		padding = 0
	}

	sb.WriteString(strings.Repeat("=", r.width))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat(" ", padding))
	sb.WriteString(title)
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("=", r.width))

	return sb.String()
}

// renderSummary renders the summary statistics
func (r *SyncStatusRenderer) renderSummary() string {
	summary := r.dashboard.GetSummary()
	var sb strings.Builder

	sb.WriteString("\n📊 SUMMARY:\n")
	sb.WriteString(fmt.Sprintf("  Total Services: %d\n", summary.Total))
	sb.WriteString(fmt.Sprintf("  ✅ Fully In Sync: %d\n", summary.FullyInSync))

	if summary.PartiallyInSync > 0 {
		sb.WriteString(fmt.Sprintf("  ⚠️  Partially In Sync: %d\n", summary.PartiallyInSync))
	}

	if summary.OutOfSync > 0 {
		sb.WriteString(fmt.Sprintf("  ❌ Out of Sync: %d\n", summary.OutOfSync))
	}

	if summary.CaddyOnly > 0 {
		sb.WriteString(fmt.Sprintf("  📋 Caddy Only: %d\n", summary.CaddyOnly))
	}

	sb.WriteString("\n📍 SYSTEM COVERAGE:\n")
	sb.WriteString(fmt.Sprintf("  🎯 Caddy Config: %d services\n", summary.InCaddy))
	sb.WriteString(fmt.Sprintf("  🌐 UnboundDNS: %d entries\n", summary.InUnbound))
	sb.WriteString(fmt.Sprintf("  🛡️  AdguardHome: %d rewrites\n", summary.InAdguard))

	return sb.String()
}

// renderTable renders the main comparison table
func (r *SyncStatusRenderer) renderTable() string {
	statuses := r.dashboard.GetFilteredStatuses()
	if len(statuses) == 0 {
		return "\n📭 No services match the current filters.\n"
	}

	var sb strings.Builder

	// Table header
	sb.WriteString("\n")
	sb.WriteString(r.renderTableHeader())
	sb.WriteString(r.renderTableDivider())

	// Table rows
	for _, status := range statuses {
		sb.WriteString(r.renderTableRow(status))
	}

	sb.WriteString(r.renderTableDivider())

	return sb.String()
}

// renderTableHeader renders the table column headers
func (r *SyncStatusRenderer) renderTableHeader() string {
	if r.showIPs {
		return fmt.Sprintf("│ %-30s │ %-20s │ %-20s │ %-20s │ %-8s │\n",
			"HOSTNAME",
			"CADDY CONFIG",
			"UNBOUNDDNS",
			"ADGUARDHOME",
			"STATUS")
	} else {
		return fmt.Sprintf("│ %-35s │ %-12s │ %-12s │ %-12s │ %-8s │\n",
			"HOSTNAME",
			"CADDY",
			"UNBOUND",
			"ADGUARD",
			"OVERALL")
	}
}

// renderTableDivider renders a table divider line
func (r *SyncStatusRenderer) renderTableDivider() string {
	if r.showIPs {
		return "├────────────────────────────────┼──────────────────────┼──────────────────────┼──────────────────────┼──────────┤\n"
	} else {
		return "├─────────────────────────────────────┼──────────────┼──────────────┼──────────────┼──────────┤\n"
	}
}

// renderTableRow renders a single table row
func (r *SyncStatusRenderer) renderTableRow(status SyncStatus) string {
	hostname := r.truncateString(status.Hostname, 30)

	if r.showIPs {
		caddyInfo := r.formatCaddyInfo(status)
		unboundInfo := r.formatUnboundInfo(status)
		adguardInfo := r.formatAdguardInfo(status)
		overallStatus := r.formatOverallStatus(status)

		return fmt.Sprintf("│ %-30s │ %-20s │ %-20s │ %-20s │ %-8s │\n",
			hostname,
			caddyInfo,
			unboundInfo,
			adguardInfo,
			overallStatus)
	} else {
		caddyIcon := r.getCaddyIcon(status)
		unboundIcon := GetStatusIcon(status.UnboundStatus, status.CaddyIP != "")
		adguardIcon := GetStatusIcon(status.AdguardStatus, status.CaddyIP != "")
		overallIcon := GetOverallStatusIcon(status.Overall)

		return fmt.Sprintf("│ %-35s │ %-12s │ %-12s │ %-12s │ %-8s │\n",
			hostname,
			caddyIcon,
			unboundIcon,
			adguardIcon,
			overallIcon)
	}
}

// formatCaddyInfo formats Caddy information for display
func (r *SyncStatusRenderer) formatCaddyInfo(status SyncStatus) string {
	if status.CaddyIP == "" {
		return "❌ Not configured"
	}
	return fmt.Sprintf("✅ %s", r.truncateString(status.CaddyIP, 12))
}

// formatUnboundInfo formats UnboundDNS information for display
func (r *SyncStatusRenderer) formatUnboundInfo(status SyncStatus) string {
	if !status.UnboundStatus.Present {
		if status.CaddyIP == "" {
			return "✅ Not present"
		}
		return "❌ Missing"
	}

	icon := "⚠️"
	if status.UnboundStatus.InSync {
		icon = "✅"
	}

	return fmt.Sprintf("%s %s", icon, r.truncateString(status.UnboundStatus.IP, 12))
}

// formatAdguardInfo formats AdguardHome information for display
func (r *SyncStatusRenderer) formatAdguardInfo(status SyncStatus) string {
	if !status.AdguardStatus.Present {
		if status.CaddyIP == "" {
			return "✅ Not present"
		}
		return "❌ Missing"
	}

	icon := "⚠️"
	if status.AdguardStatus.InSync {
		icon = "✅"
	}

	return fmt.Sprintf("%s %s", icon, r.truncateString(status.AdguardStatus.IP, 12))
}

// formatOverallStatus formats the overall status for display
func (r *SyncStatusRenderer) formatOverallStatus(status SyncStatus) string {
	switch status.Overall {
	case FullyInSync:
		return "✅ Synced"
	case PartiallyInSync:
		return "⚠️ Partial"
	case OutOfSync:
		return "❌ Out of Sync"
	case CaddyOnly:
		return "📋 Caddy Only"
	default:
		return "❓ Unknown"
	}
}

// getCaddyIcon returns the appropriate icon for Caddy status
func (r *SyncStatusRenderer) getCaddyIcon(status SyncStatus) string {
	if status.CaddyIP == "" {
		return "❌"
	}
	return "✅"
}

// renderLegend renders the status legend
func (r *SyncStatusRenderer) renderLegend() string {
	var sb strings.Builder

	sb.WriteString("\n📖 LEGEND:\n")
	sb.WriteString("  ✅ In sync / Present and correct\n")
	sb.WriteString("  ⚠️  Present but incorrect IP\n")
	sb.WriteString("  ❌ Missing or should not be present\n")
	sb.WriteString("  📋 Only in Caddy (needs sync)\n")

	sb.WriteString("\n🎯 OVERALL STATUS:\n")
	sb.WriteString("  ✅ Synced: All systems match Caddy\n")
	sb.WriteString("  ⚠️  Partial: Some systems match Caddy\n")
	sb.WriteString("  ❌ Out of Sync: No systems match Caddy\n")
	sb.WriteString("  📋 Caddy Only: In Caddy but not synced yet\n")

	return sb.String()
}

// renderActiveFilters renders information about active filters
func (r *SyncStatusRenderer) renderActiveFilters() string {
	var sb strings.Builder
	var filters []string

	if r.dashboard.filters.ShowOnlyOutOfSync {
		filters = append(filters, "Out of sync only")
	}
	if r.dashboard.filters.HostnameFilter != "" {
		filters = append(filters, fmt.Sprintf("Hostname: %s", r.dashboard.filters.HostnameFilter))
	}

	if len(filters) > 0 {
		sb.WriteString("\n🔍 ACTIVE FILTERS: ")
		sb.WriteString(strings.Join(filters, ", "))
		sb.WriteString("\n")
	}

	return sb.String()
}

// hasActiveFilters checks if any filters are currently active
func (r *SyncStatusRenderer) hasActiveFilters() bool {
	return r.dashboard.filters.ShowOnlyOutOfSync ||
		r.dashboard.filters.HostnameFilter != ""
}

// truncateString truncates a string to the specified length
func (r *SyncStatusRenderer) truncateString(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}

	runes := []rune(s)
	if maxLen > 3 {
		return string(runes[:maxLen-3]) + "..."
	}
	return string(runes[:maxLen])
}

// RenderCompactSummary renders a one-line summary for quick status checks
func (r *SyncStatusRenderer) RenderCompactSummary() string {
	summary := r.dashboard.GetSummary()

	return fmt.Sprintf("📊 %d services: %d✅ %d⚠️ %d❌ %d📋",
		summary.Total,
		summary.FullyInSync,
		summary.PartiallyInSync,
		summary.OutOfSync,
		summary.CaddyOnly)
}
