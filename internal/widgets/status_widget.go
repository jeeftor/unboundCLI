package widgets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

// ServiceLoadStatus tracks which services have been loaded
type ServiceLoadStatus struct {
	Caddy    bool
	Unbound  bool
	AdGuard  bool
	DHCP     bool
	Complete bool
}

// LoadingPhase represents the current loading phase
type LoadingPhase string

const (
	PhaseIdle       LoadingPhase = "idle"
	PhaseCaddy      LoadingPhase = "caddy"
	PhaseUnbound    LoadingPhase = "unbound"
	PhaseAdGuard    LoadingPhase = "adguard"
	PhaseDHCP       LoadingPhase = "dhcp"
	PhaseDNSResolve LoadingPhase = "dns-resolve"
	PhaseComplete   LoadingPhase = "complete"
)

// StatusSummary contains aggregated statistics
type StatusSummary struct {
	Total           int
	FullyInSync     int
	PartiallyInSync int
	OutOfSync       int
	CaddyOnly       int
	Stale           int
	DHCPMismatches  int
}

// SystemDistribution tracks how entries are distributed across systems
type SystemDistribution struct {
	AllThreeSystems int // Caddy + Unbound + AdGuard
	CaddyUnbound    int // Caddy + Unbound only
	CaddyAdGuard    int // Caddy + AdGuard only
	CaddyOnly       int // Only in Caddy
	DHCPMismatches  int // DHCP IP doesn't match
}

// StatusWidget displays loading progress and summary statistics at the top of the screen
type StatusWidget struct {
	BaseWidget

	// Data
	entries      []*models.Entry
	summary      StatusSummary
	distribution SystemDistribution

	// Loading state
	loading       bool
	loadingPhase  LoadingPhase
	serviceStatus ServiceLoadStatus
	dnsTotal      int
	dnsCompleted  int

	// Components
	spinner  spinner.Model
	progress progress.Model

	// Layout
	compactMode bool // When true, show only summary (2-3 lines)
	theme       *Theme
}

// NewStatusWidget creates a new status widget
func NewStatusWidget() *StatusWidget {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(CurrentTheme.ColorInfo)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	return &StatusWidget{
		BaseWidget:    NewBaseWidget(),
		loading:       false,
		loadingPhase:  PhaseIdle,
		serviceStatus: ServiceLoadStatus{},
		spinner:       s,
		progress:      p,
		compactMode:   true,
		theme:         CurrentTheme,
	}
}

// Init initializes the status widget
func (w *StatusWidget) Init() tea.Cmd {
	return w.spinner.Tick
}

// Update handles messages
func (w *StatusWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	var cmds []tea.Cmd

	// Update spinner
	var cmd tea.Cmd
	w.spinner, cmd = w.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return w, tea.Batch(cmds...)
}

// View renders the status widget
func (w *StatusWidget) View() string {
	if w.width == 0 {
		return ""
	}

	var sections []string

	// If loading, show full view with loading banner
	if w.loading {
		sections = append(sections, w.renderLoadingBanner())
		sections = append(sections, w.renderServiceStatus())
		sections = append(sections, w.renderProgressBar())
		sections = append(sections, "")
	}

	// Always show summary stats
	sections = append(sections, w.renderSummaryStats())

	// Show distribution breakdown if not in compact mode
	if !w.compactMode || w.loading {
		sections = append(sections, w.renderSystemDistribution())
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Add border with simple title above
	title := w.theme.Header.Render("╭─ Status ─")

	style := lipgloss.NewStyle().
		Border(lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "", // Title will provide this
			TopRight:    "╮",
			BottomLeft:  "╰",
			BottomRight: "╯",
		}).
		BorderForeground(w.theme.ColorInfo).
		Padding(0, 1).
		Width(w.width - 4)

	bordered := style.Render(content)

	// Prepend title line
	lines := strings.Split(bordered, "\n")
	if len(lines) > 0 {
		// Calculate remaining border width
		remainingWidth := w.width - lipgloss.Width(title) - 1
		if remainingWidth < 0 {
			remainingWidth = 0
		}
		titleLine := title + w.theme.Header.Render(strings.Repeat("─", remainingWidth)+"╮")
		lines[0] = titleLine
	}

	return strings.Join(lines, "\n")
}

// renderLoadingBanner renders the loading banner at the top
func (w *StatusWidget) renderLoadingBanner() string {
	var message string
	switch w.loadingPhase {
	case PhaseCaddy:
		message = "LOADING CADDY CONFIGURATION"
	case PhaseUnbound:
		message = "LOADING UNBOUND DNS OVERRIDES"
	case PhaseAdGuard:
		message = "LOADING ADGUARD REWRITES"
	case PhaseDHCP:
		message = "LOADING DHCP LEASES"
	case PhaseDNSResolve:
		message = "RESOLVING DNS HOSTNAMES"
	default:
		message = "LOADING SERVICES IN BACKGROUND"
	}

	banner := fmt.Sprintf("%s %s", w.spinner.View(), message)

	style := lipgloss.NewStyle().
		Width(w.width).
		Background(w.theme.ColorInfo).
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true).
		Padding(0, 2)

	return style.Render(banner)
}

// renderServiceStatus renders the service loading checkmarks
func (w *StatusWidget) renderServiceStatus() string {
	var parts []string

	// Caddy
	if w.serviceStatus.Caddy {
		parts = append(parts, w.theme.Success.Render("✓ Caddy"))
	} else {
		parts = append(parts, w.theme.Dimmed.Render("○ Caddy"))
	}

	// Unbound
	if w.serviceStatus.Unbound {
		parts = append(parts, w.theme.Success.Render("✓ Unbound"))
	} else {
		parts = append(parts, w.theme.Dimmed.Render("○ Unbound"))
	}

	// AdGuard
	if w.serviceStatus.AdGuard {
		parts = append(parts, w.theme.Success.Render("✓ AdGuard"))
	} else {
		parts = append(parts, w.theme.Dimmed.Render("○ AdGuard"))
	}

	// DHCP
	if w.serviceStatus.DHCP {
		parts = append(parts, w.theme.Success.Render("✓ DHCP"))
	} else {
		parts = append(parts, w.theme.Dimmed.Render("○ DHCP"))
	}

	return strings.Join(parts, "  ")
}

// renderProgressBar renders the loading progress bar
func (w *StatusWidget) renderProgressBar() string {
	if w.dnsTotal == 0 {
		return ""
	}

	percent := float64(w.dnsCompleted) / float64(w.dnsTotal)
	progressBar := w.progress.ViewAs(percent)

	label := fmt.Sprintf("Processing DNS entries: %d/%d", w.dnsCompleted, w.dnsTotal)

	return fmt.Sprintf("%s\n%s", label, progressBar)
}

// renderSummaryStats renders the overall statistics
func (w *StatusWidget) renderSummaryStats() string {
	if w.summary.Total == 0 {
		return w.theme.Dimmed.Render("No services loaded")
	}

	var parts []string

	// Total
	parts = append(parts, w.theme.Info.Render(fmt.Sprintf("Total: %d services", w.summary.Total)))

	// Fully in sync
	if w.summary.FullyInSync > 0 {
		parts = append(parts, w.theme.Success.Render(fmt.Sprintf("✅ %d synced", w.summary.FullyInSync)))
	}

	// Partially in sync
	if w.summary.PartiallyInSync > 0 {
		parts = append(parts, w.theme.Warning.Render(fmt.Sprintf("⚠️ %d partial", w.summary.PartiallyInSync)))
	}

	// Out of sync
	if w.summary.OutOfSync > 0 {
		parts = append(parts, w.theme.Error.Render(fmt.Sprintf("❌ %d out of sync", w.summary.OutOfSync)))
	}

	// Caddy only
	if w.summary.CaddyOnly > 0 {
		parts = append(parts, w.theme.Warning.Render(fmt.Sprintf("📋 %d Caddy only", w.summary.CaddyOnly)))
	}

	// Stale
	if w.summary.Stale > 0 {
		parts = append(parts, w.theme.Error.Render(fmt.Sprintf("🗑 %d stale", w.summary.Stale)))
	}

	return strings.Join(parts, "  •  ")
}

// renderSystemDistribution renders the system distribution breakdown
func (w *StatusWidget) renderSystemDistribution() string {
	if w.summary.Total == 0 {
		return ""
	}

	title := w.theme.Section.Render("System Distribution:")
	var parts []string

	// DHCP mismatches (highest priority warning)
	if w.distribution.DHCPMismatches > 0 {
		parts = append(parts,
			fmt.Sprintf("  %s DHCP Mismatches:  %s",
				w.theme.Warning.Render("⚠️"),
				w.theme.Count.Render(fmt.Sprintf("%d services", w.distribution.DHCPMismatches)),
			),
		)
	}

	// All three systems (ideal state)
	if w.distribution.AllThreeSystems > 0 {
		parts = append(parts,
			fmt.Sprintf("  %s All 3 Systems:    %s",
				w.theme.Success.Render("✅"),
				w.theme.Count.Render(fmt.Sprintf("%d services", w.distribution.AllThreeSystems)),
			),
		)
	}

	// Caddy + Unbound only
	if w.distribution.CaddyUnbound > 0 {
		parts = append(parts,
			fmt.Sprintf("  %s Caddy + Unbound:  %s",
				w.theme.Warning.Render("⚠️"),
				w.theme.Count.Render(fmt.Sprintf("%d services", w.distribution.CaddyUnbound)),
			),
		)
	}

	// Caddy + AdGuard only
	if w.distribution.CaddyAdGuard > 0 {
		parts = append(parts,
			fmt.Sprintf("  %s Caddy + AdGuard:  %s",
				w.theme.Warning.Render("⚠️"),
				w.theme.Count.Render(fmt.Sprintf("%d services", w.distribution.CaddyAdGuard)),
			),
		)
	}

	// Caddy only (needs sync)
	if w.distribution.CaddyOnly > 0 {
		parts = append(parts,
			fmt.Sprintf("  %s Caddy Only:       %s",
				w.theme.Warning.Render("⚠️"),
				w.theme.Count.Render(fmt.Sprintf("%d services", w.distribution.CaddyOnly)),
			),
		)
	}

	if len(parts) == 0 {
		return ""
	}

	return title + "\n" + strings.Join(parts, "\n")
}

// SetEntries updates the entries and recalculates statistics
func (w *StatusWidget) SetEntries(entries []*models.Entry) {
	w.entries = entries
	w.calculateStatistics()
}

// calculateStatistics computes summary and distribution from entries
func (w *StatusWidget) calculateStatistics() {
	// Reset counters
	w.summary = StatusSummary{}
	w.distribution = SystemDistribution{}

	w.summary.Total = len(w.entries)

	for _, entry := range w.entries {
		// Count by overall status
		switch entry.OverallStatus {
		case models.FullyInSync:
			w.summary.FullyInSync++
		case models.PartiallyInSync:
			w.summary.PartiallyInSync++
		case models.OutOfSync:
			w.summary.OutOfSync++
		case models.CaddyOnly:
			w.summary.CaddyOnly++
		case models.Stale:
			w.summary.Stale++
		case models.DHCPMismatch:
			w.summary.DHCPMismatches++
		}

		// Count by system distribution
		inCaddy := entry.IsConfiguredInCaddy()
		inUnbound := entry.UnboundStatus.Configured
		inAdguard := entry.AdguardStatus.Configured
		dhcpMismatch := entry.DHCPStatus.Configured && !entry.DHCPStatus.InSync

		if dhcpMismatch {
			w.distribution.DHCPMismatches++
		}

		if inCaddy && inUnbound && inAdguard {
			w.distribution.AllThreeSystems++
		} else if inCaddy && inUnbound {
			w.distribution.CaddyUnbound++
		} else if inCaddy && inAdguard {
			w.distribution.CaddyAdGuard++
		} else if inCaddy {
			w.distribution.CaddyOnly++
		}
	}
}

// SetLoading sets the loading state
func (w *StatusWidget) SetLoading(loading bool) {
	w.loading = loading
	if !loading {
		w.loadingPhase = PhaseComplete
		w.serviceStatus.Complete = true
		w.compactMode = true
	} else {
		w.compactMode = false
	}
}

// SetLoadingPhase sets the current loading phase
func (w *StatusWidget) SetLoadingPhase(phase LoadingPhase) {
	w.loadingPhase = phase

	// Update service status based on phase
	switch phase {
	case PhaseCaddy:
		w.serviceStatus.Caddy = true
	case PhaseUnbound:
		w.serviceStatus.Unbound = true
	case PhaseAdGuard:
		w.serviceStatus.AdGuard = true
	case PhaseDHCP:
		w.serviceStatus.DHCP = true
	case PhaseComplete:
		w.serviceStatus.Complete = true
	}
}

// SetProgress sets the DNS processing progress
func (w *StatusWidget) SetProgress(completed, total int) {
	w.dnsCompleted = completed
	w.dnsTotal = total
}

// SetCompactMode sets whether to show compact view (only summary) or full view
func (w *StatusWidget) SetCompactMode(compact bool) {
	w.compactMode = compact
}

// SetServiceStatus sets which services have been loaded
func (w *StatusWidget) SetServiceStatus(status ServiceLoadStatus) {
	w.serviceStatus = status
}

// IsCompactMode returns whether compact mode is enabled
func (w *StatusWidget) IsCompactMode() bool {
	return w.compactMode
}

// IsLoading returns whether the widget is in loading state
func (w *StatusWidget) IsLoading() bool {
	return w.loading
}

// GetSummary returns the current summary statistics
func (w *StatusWidget) GetSummary() StatusSummary {
	return w.summary
}

// GetDistribution returns the current distribution statistics
func (w *StatusWidget) GetDistribution() SystemDistribution {
	return w.distribution
}
