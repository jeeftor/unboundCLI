package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
	statussvc "github.com/jeeftor/caddy-dns-sync/internal/status"
)

// OverallSyncStatus represents the overall sync state
type OverallSyncStatus int

const (
	FullyInSync OverallSyncStatus = iota
	PartiallyInSync
	OutOfSync
	CaddyOnly
	Stale
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

// SyncStatusDashboard holds status data for dashboard-style CLI views.
type SyncStatusDashboard struct {
	statuses      []SyncStatus
	caddyServerIP string
	filters       StatusFilters
}

// NewSyncStatusDashboard creates a new dashboard.
func NewSyncStatusDashboard(caddyServerIP string) *SyncStatusDashboard {
	return &SyncStatusDashboard{
		caddyServerIP: caddyServerIP,
		statuses:      []SyncStatus{},
		filters:       StatusFilters{},
	}
}

// LoadSyncData loads data from service clients.
func (d *SyncStatusDashboard) LoadSyncData(
	caddyClient *api.CaddyClient,
	unboundClient *api.Client,
	adguardClient *api.AdguardClient,
	dnsmasqClient *api.DNSMasqClient,
) error {
	entries, _, err := statussvc.LoadEntries(context.Background(), app.ClientSet{
		Caddy:   caddyClient,
		Unbound: unboundClient,
		Adguard: adguardClient,
		DNSMasq: dnsmasqClient,
	}, statussvc.Options{CaddyServerIP: d.caddyServerIP})
	if err != nil {
		return err
	}
	d.statuses = syncStatusesFromEntries(entries)
	return nil
}

// LoadSyncDataWithProgress loads data from service clients and reports coarse progress.
func (d *SyncStatusDashboard) LoadSyncDataWithProgress(
	caddyClient *api.CaddyClient,
	unboundClient *api.Client,
	adguardClient *api.AdguardClient,
	dnsmasqClient *api.DNSMasqClient,
	progressCallback func(phase string, current, total int),
) error {
	if progressCallback != nil {
		progressCallback("loading", 0, 1)
	}
	err := d.LoadSyncData(caddyClient, unboundClient, adguardClient, dnsmasqClient)
	if progressCallback != nil {
		progressCallback("loaded", 1, 1)
	}
	return err
}

// GetFilteredStatuses returns filtered statuses.
func (d *SyncStatusDashboard) GetFilteredStatuses() []SyncStatus {
	filtered := make([]SyncStatus, 0, len(d.statuses))
	for _, status := range d.statuses {
		if d.matchesFilters(status) {
			filtered = append(filtered, status)
		}
	}
	return filtered
}

// GetCurrentFilters returns current filters.
func (d *SyncStatusDashboard) GetCurrentFilters() StatusFilters {
	return d.filters
}

// SetFilters sets filters.
func (d *SyncStatusDashboard) SetFilters(filters StatusFilters) {
	d.filters = filters
}

// GetSummary returns a summary.
func (d *SyncStatusDashboard) GetSummary() SyncStatusSummary {
	summary := SyncStatusSummary{Total: len(d.GetFilteredStatuses())}
	for _, status := range d.GetFilteredStatuses() {
		switch status.Overall {
		case FullyInSync:
			summary.FullyInSync++
		case PartiallyInSync:
			summary.PartiallyInSync++
		case OutOfSync, Stale:
			summary.OutOfSync++
		case CaddyOnly:
			summary.CaddyOnly++
		}
	}
	return summary
}

func (d *SyncStatusDashboard) matchesFilters(status SyncStatus) bool {
	if d.filters.HostnameFilter != "" &&
		!strings.Contains(strings.ToLower(status.Hostname), strings.ToLower(d.filters.HostnameFilter)) {
		return false
	}
	if d.filters.ShowOnlyOutOfSync && status.Overall != OutOfSync && status.Overall != Stale {
		return false
	}
	if d.filters.ShowOutOfSync && status.Overall != OutOfSync {
		return false
	}
	if d.filters.ShowCaddyOnly && status.Overall != CaddyOnly {
		return false
	}
	if d.filters.ShowStale && status.Overall != Stale {
		return false
	}
	if d.filters.ShowMismatches && !status.DHCPMismatch {
		return false
	}
	return true
}

func syncStatusesFromEntries(entries []*models.Entry) []SyncStatus {
	statuses := make([]SyncStatus, 0, len(entries))
	for _, entry := range entries {
		statuses = append(statuses, syncStatusFromEntry(entry))
	}
	return statuses
}

func syncStatusFromEntry(entry *models.Entry) SyncStatus {
	return SyncStatus{
		Hostname:      entry.Hostname,
		DataSource:    entry.DataSource,
		CaddyIP:       entry.CaddyIP,
		DNSResolvedIP: entry.DNSResolved,
		UpstreamIP:    entry.CaddyUpstream,
		DHCPLeaseIP:   entry.DHCPStatus.IP,
		DHCPLeaseType: entry.DHCPStatus.Type,
		DHCPMismatch:  entry.OverallStatus == models.DHCPMismatch,
		UnboundStatus: serviceStatusFromModel(entry.UnboundStatus),
		AdguardStatus: serviceStatusFromModel(entry.AdguardStatus),
		Overall:       overallStatusFromModel(entry.OverallStatus),
	}
}

func serviceStatusFromModel(status models.ServiceStatus) ServiceStatus {
	return ServiceStatus{
		Present: status.Configured,
		InSync:  status.InSync,
		IP:      status.IP,
	}
}

func overallStatusFromModel(status models.SyncStatus) OverallSyncStatus {
	switch status {
	case models.FullyInSync:
		return FullyInSync
	case models.PartiallyInSync:
		return PartiallyInSync
	case models.OutOfSync, models.DHCPMismatch:
		return OutOfSync
	case models.CaddyOnly:
		return CaddyOnly
	case models.Stale:
		return Stale
	default:
		return CaddyOnly
	}
}

// GetStatusIcon returns a status icon.
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

// GetOverallStatusIcon returns an overall status icon.
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
	case Stale:
		return "🗑️"
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

// UI represents a UI helper.
type UI struct{}

// NewUI creates a new UI.
func NewUI() *UI {
	return &UI{}
}

// RenderError renders an error.
func (u *UI) RenderError(err error) string {
	return "Error: " + err.Error()
}

// RenderSuccess renders a success message.
func (u *UI) RenderSuccess(message string) string {
	return "Success: " + message
}

// RenderInfo renders an info message.
func (u *UI) RenderInfo(message string) string {
	return "Info: " + message
}

// SyncStatusRenderer renders dashboard data for CLI views.
type SyncStatusRenderer struct {
	dashboard *SyncStatusDashboard
	width     int
	showIPs   bool
}

// NewSyncStatusRenderer creates a new renderer.
func NewSyncStatusRenderer(dashboard *SyncStatusDashboard) *SyncStatusRenderer {
	return &SyncStatusRenderer{
		dashboard: dashboard,
		width:     80,
		showIPs:   false,
	}
}

// SetWidth sets the width.
func (r *SyncStatusRenderer) SetWidth(width int) {
	r.width = width
}

// ShowIPs returns whether to show IPs.
func (r *SyncStatusRenderer) ShowIPs() bool {
	return r.showIPs
}

// SetShowIPs sets whether to show IPs.
func (r *SyncStatusRenderer) SetShowIPs(show bool) {
	r.showIPs = show
}

// RenderCompactSummary renders a compact summary.
func (r *SyncStatusRenderer) RenderCompactSummary() string {
	if r.dashboard == nil {
		return "No data"
	}
	summary := r.dashboard.GetSummary()
	return lipgloss.JoinVertical(lipgloss.Left,
		"Sync Status Summary:",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")).Render(
			fmt.Sprintf("  ✓ %d synced", summary.FullyInSync),
		),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")).Render(
			fmt.Sprintf("  ⚠ %d partial", summary.PartiallyInSync),
		),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e")).Render(
			fmt.Sprintf("  ✗ %d out of sync", summary.OutOfSync),
		),
		fmt.Sprintf("  + %d caddy only", summary.CaddyOnly),
	)
}

// RenderDashboard renders the full dashboard.
func (r *SyncStatusRenderer) RenderDashboard() string {
	if r.dashboard == nil {
		return "No data loaded"
	}
	statuses := r.dashboard.GetFilteredStatuses()
	lines := []string{"Sync Status Dashboard"}
	if len(statuses) == 0 {
		lines = append(lines, "No matching services")
		return strings.Join(lines, "\n")
	}
	for _, status := range statuses {
		line := fmt.Sprintf("%s %s [%s]", GetOverallStatusIcon(status.Overall), status.Hostname, status.DataSource)
		if r.showIPs {
			line += fmt.Sprintf(" caddy=%s dns=%s", status.CaddyIP, status.DNSResolvedIP)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
