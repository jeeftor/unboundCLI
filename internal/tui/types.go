package tui

// This file contains stub types to maintain compatibility
// while the TUI is being rebuilt from scratch.
// These types should be moved to a proper package (e.g., internal/status) later.

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/api"
)

// OverallSyncStatus represents the overall sync state
type OverallSyncStatus int

const (
	FullyInSync OverallSyncStatus = iota
	PartiallyInSync
	OutOfSync
	CaddyOnly
)

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Present bool
	InSync  bool
	IP      string
}

// SyncStatus represents the full sync status for a hostname
type SyncStatus struct {
	Hostname      string
	DataSource    string
	CaddyIP       string
	DNSResolvedIP string
	UpstreamIP    string
	DHCPLeaseIP   string
	DHCPLeaseType string
	DHCPMismatch  bool
	UnboundStatus ServiceStatus
	AdguardStatus ServiceStatus
	Overall       OverallSyncStatus
}

// StatusFilters represents filters for status display
type StatusFilters struct {
	ShowOutOfSync     bool
	ShowMismatches    bool
	ShowCaddyOnly     bool
	ShowStale         bool
	ShowOnlyOutOfSync bool
	HostnameFilter    string
}

// SyncStatusSummary represents a summary of sync statuses
type SyncStatusSummary struct {
	Total           int
	FullyInSync     int
	PartiallyInSync int
	OutOfSync       int
	CaddyOnly       int
}

// SyncStatusDashboard is a stub for the dashboard
type SyncStatusDashboard struct {
	statuses      []SyncStatus
	caddyServerIP string
	filters       StatusFilters
}

// NewSyncStatusDashboard creates a new dashboard (stub)
func NewSyncStatusDashboard(caddyServerIP string) *SyncStatusDashboard {
	return &SyncStatusDashboard{
		caddyServerIP: caddyServerIP,
		statuses:      []SyncStatus{},
		filters:       StatusFilters{},
	}
}

// LoadSyncData loads data (stub)
func (d *SyncStatusDashboard) LoadSyncData(
	caddyClient *api.CaddyClient,
	unboundClient *api.Client,
	adguardClient *api.AdguardClient,
	dnsmasqClient *api.DNSMasqClient,
) error {
	// TODO: Implement this properly when TUI is rebuilt
	return nil
}

// LoadSyncDataWithProgress loads data (stub)
func (d *SyncStatusDashboard) LoadSyncDataWithProgress(
	caddyClient *api.CaddyClient,
	unboundClient *api.Client,
	adguardClient *api.AdguardClient,
	dnsmasqClient *api.DNSMasqClient,
	progressCallback func(phase string, current, total int),
) error {
	// TODO: Implement this properly when TUI is rebuilt
	return nil
}

// GetFilteredStatuses returns filtered statuses (stub)
func (d *SyncStatusDashboard) GetFilteredStatuses() []SyncStatus {
	return d.statuses
}

// GetCurrentFilters returns current filters (stub)
func (d *SyncStatusDashboard) GetCurrentFilters() StatusFilters {
	return d.filters
}

// SetFilters sets filters (stub)
func (d *SyncStatusDashboard) SetFilters(filters StatusFilters) {
	d.filters = filters
}

// GetSummary returns a summary (stub)
func (d *SyncStatusDashboard) GetSummary() SyncStatusSummary {
	return SyncStatusSummary{
		Total: len(d.statuses),
	}
}

// GetStatusIcon returns a status icon (stub)
func GetStatusIcon(status ServiceStatus, hasInCaddy bool) string {
	if !hasInCaddy {
		if !status.Present {
			return "✅"
		}
		return "🗑️"
	}
	if !status.Present {
		return "❌"
	}
	if status.InSync {
		return "✅"
	}
	return "⚠️"
}

// GetOverallStatusIcon returns an overall status icon (stub)
func GetOverallStatusIcon(status OverallSyncStatus) string {
	switch status {
	case FullyInSync:
		return "✅"
	case PartiallyInSync:
		return "⚠️"
	case OutOfSync:
		return "❌"
	case CaddyOnly:
		return "📋"
	default:
		return "?"
	}
}

// StyleConfig represents UI styling configuration
type StyleConfig struct {
	SuccessColor lipgloss.Color
	ErrorColor   lipgloss.Color
	WarningColor lipgloss.Color
	InfoColor    lipgloss.Color
	DimColor     lipgloss.Color

	// Lipgloss styles
	Header      lipgloss.Style
	Section     lipgloss.Style
	Dimmed      lipgloss.Style
	Hostname    lipgloss.Style
	Warning     lipgloss.Style
	Success     lipgloss.Style
	Error       lipgloss.Style
	Info        lipgloss.Style
	Count       lipgloss.Style
	Add         lipgloss.Style
	Update      lipgloss.Style
	Remove      lipgloss.Style
	DryRun      lipgloss.Style
	IP          lipgloss.Style
	Description lipgloss.Style
}

// DefaultStyles returns default styles
func DefaultStyles() StyleConfig {
	successColor := lipgloss.Color("#9ece6a")
	errorColor := lipgloss.Color("#f7768e")
	warningColor := lipgloss.Color("#e0af68")
	infoColor := lipgloss.Color("#7aa2f7")
	dimColor := lipgloss.Color("#565f89")
	cyanColor := lipgloss.Color("#7dcfff")
	purpleColor := lipgloss.Color("#bb9af7")

	return StyleConfig{
		SuccessColor: successColor,
		ErrorColor:   errorColor,
		WarningColor: warningColor,
		InfoColor:    infoColor,
		DimColor:     dimColor,

		Header:      lipgloss.NewStyle().Bold(true).Foreground(infoColor),
		Section:     lipgloss.NewStyle().Foreground(infoColor),
		Dimmed:      lipgloss.NewStyle().Foreground(dimColor),
		Hostname:    lipgloss.NewStyle().Bold(true).Foreground(successColor),
		Warning:     lipgloss.NewStyle().Foreground(warningColor),
		Success:     lipgloss.NewStyle().Foreground(successColor),
		Error:       lipgloss.NewStyle().Foreground(errorColor),
		Info:        lipgloss.NewStyle().Foreground(infoColor),
		Count:       lipgloss.NewStyle().Bold(true).Foreground(cyanColor),
		Add:         lipgloss.NewStyle().Foreground(successColor),
		Update:      lipgloss.NewStyle().Foreground(warningColor),
		Remove:      lipgloss.NewStyle().Foreground(errorColor),
		DryRun:      lipgloss.NewStyle().Bold(true).Foreground(purpleColor).Background(lipgloss.Color("#1a1b26")),
		IP:          lipgloss.NewStyle().Foreground(cyanColor),
		Description: lipgloss.NewStyle().Foreground(dimColor).Italic(true),
	}
}

// UI represents a UI helper (stub)
type UI struct{}

// NewUI creates a new UI (stub)
func NewUI() *UI {
	return &UI{}
}

// RenderError renders an error (stub)
func (u *UI) RenderError(err error) string {
	return "Error: " + err.Error()
}

// RenderSuccess renders a success message (stub)
func (u *UI) RenderSuccess(message string) string {
	return "Success: " + message
}

// RenderInfo renders an info message (stub)
func (u *UI) RenderInfo(message string) string {
	return "Info: " + message
}

// SyncStatusRenderer is a stub renderer
type SyncStatusRenderer struct {
	dashboard *SyncStatusDashboard
	width     int
	showIPs   bool
}

// NewSyncStatusRenderer creates a new renderer (stub)
func NewSyncStatusRenderer(dashboard *SyncStatusDashboard) *SyncStatusRenderer {
	return &SyncStatusRenderer{
		dashboard: dashboard,
		width:     80,
		showIPs:   false,
	}
}

// SetWidth sets the width (stub)
func (r *SyncStatusRenderer) SetWidth(width int) {
	r.width = width
}

// ShowIPs returns whether to show IPs (stub)
func (r *SyncStatusRenderer) ShowIPs() bool {
	return r.showIPs
}

// SetShowIPs sets whether to show IPs (stub)
func (r *SyncStatusRenderer) SetShowIPs(show bool) {
	r.showIPs = show
}

// RenderCompactSummary renders a compact summary (stub)
func (r *SyncStatusRenderer) RenderCompactSummary() string {
	if r.dashboard == nil {
		return "No data"
	}
	summary := r.dashboard.GetSummary()
	return lipgloss.NewStyle().Render(
		lipgloss.JoinVertical(lipgloss.Left,
			"Sync Status Summary:",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")).Render(
				"  ✓ "+string(rune(summary.FullyInSync))+" synced",
			),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")).Render(
				"  ⚠ "+string(rune(summary.PartiallyInSync))+" partial",
			),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e")).Render(
				"  ✗ "+string(rune(summary.OutOfSync))+" out of sync",
			),
		),
	)
}

// RenderDashboard renders the full dashboard (stub)
func (r *SyncStatusRenderer) RenderDashboard() string {
	if r.dashboard == nil {
		return "No data loaded"
	}
	return "Dashboard view (TUI removed - to be rebuilt)"
}
