package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/tui"
	"github.com/spf13/cobra"
)

var (
	dashboardJsonOutput bool
	dashboardCompact    bool
	dashboardEmojis     bool
)

// Styles for the 3-way list
var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Margin(1, 0).
			Align(lipgloss.Center)

	summaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Margin(0, 2)

	statusInSyncStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46"))

	statusPartialStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("226"))

	statusOutOfSyncStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

	statusCaddyOnlyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("33"))
)

type listModel struct {
	dashboard *tui.SyncStatusDashboard
	renderer  *tui.SyncStatusRenderer
	table     table.Model
	width     int
	height    int
	compact   bool
	loaded    bool
	err       error
}

type dataLoadedMsg struct {
	dashboard *tui.SyncStatusDashboard
	err       error
}

func initialListModel(compact bool) listModel {
	return listModel{
		compact: compact,
		loaded:  false,
	}
}

func (m listModel) Init() tea.Cmd {
	return loadSyncData
}

func loadSyncData() tea.Msg {
	// Load main config first
	cfg, err := config.LoadConfig()
	if err != nil {
		return dataLoadedMsg{err: fmt.Errorf("error loading configuration: %w", err)}
	}

	// Load AdguardHome config
	adguardConfig, err := config.LoadAdguardConfig()
	if err != nil {
		return dataLoadedMsg{err: fmt.Errorf("error loading AdguardHome configuration: %w", err)}
	}

	// Create clients
	unboundClient := api.NewClient(cfg)

	var adguardClient *api.AdguardClient
	if adguardConfig.Enabled && adguardConfig.BaseURL != "" && adguardConfig.Username != "" && adguardConfig.Password != "" {
		adguardClient = api.NewAdguardClient(adguardConfig.GetAdguardAPIConfig())
	}

	// Create Caddy client - use default config from Caddy sync commands
	caddyServerIP := "192.168.1.15"
	caddyClient := api.NewCaddyClient(caddyServerIP, 2019)

	// Create DNSMasq client
	dnsmasqClient := api.NewDNSMasqClient(cfg)

	// Create dashboard and load data
	dashboard := tui.NewSyncStatusDashboard(caddyServerIP)
	err = dashboard.LoadSyncData(caddyClient, unboundClient, adguardClient, dnsmasqClient)
	if err != nil {
		return dataLoadedMsg{err: fmt.Errorf("error loading sync data: %w", err)}
	}

	return dataLoadedMsg{dashboard: dashboard}
}

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.renderer != nil {
			m.renderer.SetWidth(m.width)
		}
		if m.loaded {
			m.updateTable()
		}

	case dataLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.dashboard = msg.dashboard
		m.renderer = tui.NewSyncStatusRenderer(msg.dashboard)
		m.renderer.SetWidth(m.width)
		m.loaded = true
		m.updateTable()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"))):
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			m.loaded = false
			return m, loadSyncData
		case key.Matches(msg, key.NewBinding(key.WithKeys("i"))):
			if m.renderer != nil {
				m.renderer.SetShowIPs(!m.renderer.ShowIPs())
				// Toggle between compact and detailed view based on IP display
				m.compact = !m.renderer.ShowIPs()
				m.updateTable()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("o"))):
			if m.dashboard != nil {
				filters := m.dashboard.GetCurrentFilters()
				filters.ShowOnlyOutOfSync = !filters.ShowOnlyOutOfSync
				m.dashboard.SetFilters(filters)
				m.updateTable()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
			// Toggle compact mode
			m.compact = !m.compact
			m.updateTable()
		case key.Matches(msg, key.NewBinding(key.WithKeys("e"))):
			// Toggle emoji mode
			dashboardEmojis = !dashboardEmojis
			m.updateTable()
		}
	}

	if m.loaded {
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}

func (m *listModel) updateTable() {
	if m.dashboard == nil {
		return
	}

	statuses := m.dashboard.GetFilteredStatuses()

	var columns []table.Column
	var rows []table.Row

	if m.compact {
		// Compact view with icons - use fixed widths for better alignment
		columns = []table.Column{
			{Title: "HOSTNAME", Width: 35},
			{Title: "CADDY", Width: 8},
			{Title: "UNBOUND", Width: 10},
			{Title: "ADGUARD", Width: 10},
			{Title: "STATUS", Width: 12},
		}

		for _, status := range statuses {
			var caddyText, unboundText, adguardText, overallText string

			if dashboardEmojis {
				// Use emoji indicators
				caddyText = "❌"
				if status.CaddyIP != "" {
					caddyText = "✅"
				}
				unboundText = tui.GetStatusIcon(status.UnboundStatus, status.CaddyIP != "")
				adguardText = tui.GetStatusIcon(status.AdguardStatus, status.CaddyIP != "")
				overallText = tui.GetOverallStatusIcon(status.Overall)
			} else {
				// Use simple text indicators for better alignment
				caddyText = "NO"
				if status.CaddyIP != "" {
					caddyText = "YES"
				}
				unboundText = getSimpleStatusText(status.UnboundStatus, status.CaddyIP != "")
				adguardText = getSimpleStatusText(status.AdguardStatus, status.CaddyIP != "")
				overallText = getSimpleOverallText(status.Overall)
			}

			rows = append(rows, table.Row{
				status.Hostname,
				caddyText,
				unboundText,
				adguardText,
				overallText,
			})
		}
	} else {
		// Detailed view with IP addresses - use fixed widths for better alignment
		columns = []table.Column{
			{Title: "HOSTNAME", Width: 30},
			{Title: "CADDY CONFIG", Width: 18},
			{Title: "UNBOUNDDNS", Width: 18},
			{Title: "ADGUARDHOME", Width: 18},
			{Title: "STATUS", Width: 15},
		}

		for _, status := range statuses {
			var caddyInfo, unboundInfo, adguardInfo, overallStatus string

			if dashboardEmojis {
				// Use emoji indicators with IP addresses
				caddyInfo = "❌ Not configured"
				if status.CaddyIP != "" {
					caddyInfo = "✅ " + status.CaddyIP
				}
				unboundInfo = formatServiceInfo(status.UnboundStatus, status.CaddyIP != "")
				adguardInfo = formatServiceInfo(status.AdguardStatus, status.CaddyIP != "")
				overallStatus = formatOverallStatus(status.Overall)
			} else {
				// Use simple text indicators with IP addresses for better alignment
				if status.CaddyIP != "" {
					caddyInfo = "YES " + status.CaddyIP
				} else {
					caddyInfo = "NO"
				}
				unboundInfo = formatServiceInfoText(status.UnboundStatus, status.CaddyIP != "")
				adguardInfo = formatServiceInfoText(status.AdguardStatus, status.CaddyIP != "")
				overallStatus = getSimpleOverallText(status.Overall)
			}

			rows = append(rows, table.Row{
				status.Hostname,
				caddyInfo,
				unboundInfo,
				adguardInfo,
				overallStatus,
			})
		}
	}

	// Create table with better styling
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(m.height-8), // Leave room for header and controls
		table.WithFocused(true),      // Enable keyboard navigation
	)

	// Simple table styling - minimal formatting for better alignment
	s := table.DefaultStyles()

	// Header styling
	s.Header = s.Header.
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("57")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true)

	// Row selection styling
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("33")).
		Bold(false)

	t.SetStyles(s)
	m.table = t
}

func formatServiceInfo(status tui.ServiceStatus, hasInCaddy bool) string {
	if !hasInCaddy {
		if !status.Present {
			return "✅ Not present"
		}
		return "❌ Should not exist"
	}

	if !status.Present {
		return "❌ Missing"
	}

	icon := "⚠️"
	if status.InSync {
		icon = "✅"
	}

	return fmt.Sprintf("%s %s", icon, status.IP)
}

func formatOverallStatus(status tui.OverallSyncStatus) string {
	switch status {
	case tui.FullyInSync:
		return statusInSyncStyle.Render("✅ Synced")
	case tui.PartiallyInSync:
		return statusPartialStyle.Render("⚠️ Partial")
	case tui.OutOfSync:
		return statusOutOfSyncStyle.Render("❌ Out of Sync")
	case tui.CaddyOnly:
		return statusCaddyOnlyStyle.Render("📋 Caddy Only")
	default:
		return "❓ Unknown"
	}
}

// Simple text status functions for better alignment
func getSimpleStatusText(status tui.ServiceStatus, hasInCaddy bool) string {
	if !hasInCaddy {
		if !status.Present {
			return "OK" // Correctly absent
		}
		return "EXTRA" // Should not be present
	}

	if !status.Present {
		return "MISSING" // Missing
	}
	if status.InSync {
		return "SYNC" // In sync
	}
	return "WRONG" // Present but wrong IP
}

func getSimpleOverallText(status tui.OverallSyncStatus) string {
	switch status {
	case tui.FullyInSync:
		return "SYNCED"
	case tui.PartiallyInSync:
		return "PARTIAL"
	case tui.OutOfSync:
		return "OUT-OF-SYNC"
	case tui.CaddyOnly:
		return "CADDY-ONLY"
	default:
		return "UNKNOWN"
	}
}

func formatServiceInfoText(status tui.ServiceStatus, hasInCaddy bool) string {
	if !hasInCaddy {
		if !status.Present {
			return "OK (not present)"
		}
		return "EXTRA (should not exist)"
	}

	if !status.Present {
		return "MISSING"
	}

	if status.InSync {
		return "SYNC " + status.IP
	}
	return "WRONG " + status.IP
}

func (m listModel) View() string {
	if m.err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Render(fmt.Sprintf("Error: %v", m.err))
	}

	if !m.loaded {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Render("Loading sync status...")
	}

	// Title section
	title := titleStyle.Render("🚀 CADDY-CENTRIC DNS SYNC STATUS 🚀")

	// Summary section
	summary := m.dashboard.GetSummary()
	summaryText := fmt.Sprintf(
		"📊 %d services: %s%d✅ %s%d⚠️ %s%d❌ %s%d📋",
		summary.Total,
		statusInSyncStyle.Render(""),
		summary.FullyInSync,
		statusPartialStyle.Render(""),
		summary.PartiallyInSync,
		statusOutOfSyncStyle.Render(""),
		summary.OutOfSync,
		statusCaddyOnlyStyle.Render(""),
		summary.CaddyOnly,
	)
	summarySection := summaryStyle.Render(summaryText)

	// Controls section
	var controlsText strings.Builder
	controlsText.WriteString("Controls: ")
	controlsText.WriteString("[q]uit • [r]efresh • [↑↓] navigate • [c]ompact • [e]moji")

	if m.renderer != nil {
		if m.renderer.ShowIPs() {
			controlsText.WriteString(" • [i] hide IPs")
		} else {
			controlsText.WriteString(" • [i] show IPs")
		}
	}

	filters := m.dashboard.GetCurrentFilters()
	if filters.ShowOnlyOutOfSync {
		controlsText.WriteString(" • [o] show all")
	} else {
		controlsText.WriteString(" • [o] out-of-sync only")
	}

	// Show current view mode
	if m.compact {
		controlsText.WriteString(" | Compact")
	} else {
		controlsText.WriteString(" | Detailed")
	}

	if dashboardEmojis {
		controlsText.WriteString(" | Emoji")
	} else {
		controlsText.WriteString(" | Text")
	}

	controlsSection := summaryStyle.Render(controlsText.String())

	// Combine all sections with proper spacing
	sections := []string{
		title,
		"",
		summarySection,
		"",
		m.table.View(),
		"",
		controlsSection,
	}

	return strings.Join(sections, "\n")
}

// dashboardCmd represents the dashboard command
var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Aliases: []string{"dash"},
	Short:   "Interactive 3-way sync status dashboard",
	Long: `Interactive 3-way sync status dashboard showing DNS entries across Caddy, UnboundDNS, and AdguardHome.

This command displays a live, interactive table showing the sync status of all hostnames
across your DNS infrastructure. Use keyboard shortcuts to navigate and filter the data.

Controls:
  q/Ctrl+C/Esc  - Quit
  r             - Refresh data
  ↑↓            - Navigate table
  c             - Toggle compact/detailed view
  e             - Toggle emoji/text mode
  i             - Toggle IP address display
  o             - Toggle out-of-sync filter`,
	Run: func(cmd *cobra.Command, args []string) {
		if dashboardJsonOutput {
			// For JSON output, use the status command logic but output JSON
			runStatusJSON()
			return
		}

		// Run the interactive TUI
		model := initialListModel(dashboardCompact)
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			logging.Error("Error running interactive list", "error", err)
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runStatusJSON() {
	// Load config and data (similar to status command)
	cfg, err := config.LoadConfig()
	if err != nil {
		logging.Error("Error loading configuration", "error", err)
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	adguardConfig, err := config.LoadAdguardConfig()
	if err != nil {
		logging.Error("Error loading AdguardHome configuration", "error", err)
		fmt.Printf("Error loading AdguardHome configuration: %v\n", err)
		os.Exit(1)
	}

	unboundClient := api.NewClient(cfg)

	var adguardClient *api.AdguardClient
	if adguardConfig.Enabled && adguardConfig.BaseURL != "" && adguardConfig.Username != "" && adguardConfig.Password != "" {
		adguardClient = api.NewAdguardClient(adguardConfig.GetAdguardAPIConfig())
	}

	caddyServerIP := "192.168.1.15"
	caddyClient := api.NewCaddyClient(caddyServerIP, 2019)

	// Create DNSMasq client
	dnsmasqClient := api.NewDNSMasqClient(cfg)

	dashboard := tui.NewSyncStatusDashboard(caddyServerIP)
	err = dashboard.LoadSyncData(caddyClient, unboundClient, adguardClient, dnsmasqClient)
	if err != nil {
		logging.Error("Error loading sync data", "error", err)
		fmt.Printf("Error loading sync data: %v\n", err)
		os.Exit(1)
	}

	// Output JSON format
	statuses := dashboard.GetFilteredStatuses()
	jsonData := map[string]interface{}{
		"summary":  dashboard.GetSummary(),
		"statuses": statuses,
	}

	// Print JSON
	jsonBytes, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		logging.Error("Error formatting JSON", "error", err)
		fmt.Printf("Error formatting JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonBytes))
}

func init() {
	rootCmd.AddCommand(dashboardCmd)

	// Add flags
	dashboardCmd.Flags().BoolVar(&dashboardJsonOutput, "json", false, "Output in JSON format (non-interactive)")
	dashboardCmd.Flags().BoolVar(&dashboardCompact, "compact", false, "Use compact view (icons only)")
	dashboardCmd.Flags().BoolVar(&dashboardEmojis, "emojis", false, "Use emojis instead of text (may have alignment issues)")
}
