package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/tables"
	"github.com/jeeftor/caddy-dns-sync/internal/tui"
)

// UnboundDataSource implements ListDataSource for Unbound DNS
type UnboundDataSource struct {
	client    *api.Client
	overrides []api.DNSOverride
}

// NewUnboundDataSource creates a new Unbound data source
func NewUnboundDataSource() *UnboundDataSource {
	return &UnboundDataSource{}
}

func (s *UnboundDataSource) Initialize() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	s.client = api.NewClient(cfg)
	return nil
}

func (s *UnboundDataSource) FetchData() (interface{}, error) {
	overrides, err := s.client.GetOverrides()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Unbound overrides: %w", err)
	}
	s.overrides = overrides
	return overrides, nil
}

func (s *UnboundDataSource) FormatAsTable() tables.TableConfig {
	headers := []string{"UUID", "Host", "Domain", "IP Address", "Description", "Enabled"}
	rows := [][]string{}

	for _, o := range s.overrides {
		enabled := "No"
		if o.Enabled == "1" {
			enabled = "Yes"
		}
		rows = append(rows, []string{
			o.UUID,
			o.Host,
			o.Domain,
			o.Server,
			o.Description,
			enabled,
		})
	}

	return tables.TableConfig{
		Title:   "UNBOUND DNS OVERRIDES",
		Headers: headers,
		Rows:    rows,
		Summary: fmt.Sprintf("Total: %d overrides", len(s.overrides)),
	}
}

func (s *UnboundDataSource) FormatAsJSON() ([]byte, error) {
	return json.MarshalIndent(s.overrides, "", "  ")
}

func (s *UnboundDataSource) EmptyMessage() string {
	return "No DNS overrides found."
}

// AdguardDataSource implements ListDataSource for AdguardHome
type AdguardDataSource struct {
	client   *api.AdguardClient
	rewrites []api.Rewrite
}

// NewAdguardDataSource creates a new Adguard data source
func NewAdguardDataSource() *AdguardDataSource {
	return &AdguardDataSource{}
}

func (s *AdguardDataSource) Initialize() error {
	cfg, err := config.LoadExtendedConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.Adguard.Enabled {
		return fmt.Errorf("AdguardHome is not enabled in configuration")
	}

	s.client = api.NewAdguardClient(cfg.Adguard.GetAdguardAPIConfig())
	return nil
}

func (s *AdguardDataSource) FetchData() (interface{}, error) {
	rewrites, err := s.client.ListRewrites()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Adguard rewrites: %w", err)
	}
	s.rewrites = rewrites
	return rewrites, nil
}

func (s *AdguardDataSource) FormatAsTable() tables.TableConfig {
	headers := []string{"Domain", "Answer (IP)"}
	rows := [][]string{}

	for _, r := range s.rewrites {
		rows = append(rows, []string{
			r.Domain,
			r.Answer,
		})
	}

	return tables.TableConfig{
		Title:   "ADGUARD DNS REWRITES",
		Headers: headers,
		Rows:    rows,
		Summary: fmt.Sprintf("Total: %d rewrites", len(s.rewrites)),
	}
}

func (s *AdguardDataSource) FormatAsJSON() ([]byte, error) {
	return json.MarshalIndent(s.rewrites, "", "  ")
}

func (s *AdguardDataSource) EmptyMessage() string {
	return "No DNS rewrites found."
}

// DHCPDataSource implements ListDataSource for DHCP/DNSMasq
type DHCPDataSource struct {
	client *api.DNSMasqClient
	leases []api.DNSMasqLease
}

// NewDHCPDataSource creates a new DHCP data source
func NewDHCPDataSource() *DHCPDataSource {
	return &DHCPDataSource{}
}

func (s *DHCPDataSource) Initialize() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	s.client = api.NewDNSMasqClient(cfg)
	return nil
}

func (s *DHCPDataSource) FetchData() (interface{}, error) {
	leases, err := s.client.GetLeases()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DHCP leases: %w", err)
	}
	s.leases = leases
	return leases, nil
}

func (s *DHCPDataSource) FormatAsTable() tables.TableConfig {
	// Tokyo Night color scheme
	staticColor := lipgloss.Color("#9ece6a")  // Green
	dynamicColor := lipgloss.Color("#7aa2f7") // Blue
	dimColor := lipgloss.Color("#565f89")     // Dim
	warnColor := lipgloss.Color("#e0af68")    // Yellow

	// Styles
	staticStyle := lipgloss.NewStyle().Foreground(staticColor)
	dynamicStyle := lipgloss.NewStyle().Foreground(dynamicColor)
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)
	warnStyle := lipgloss.NewStyle().Foreground(warnColor)

	// Count static and dynamic leases
	var staticCount, dynamicCount int
	for _, lease := range s.leases {
		if lease.Type == "static" {
			staticCount++
		} else {
			dynamicCount++
		}
	}

	headers := []string{"Hostname", "IP Address", "MAC Address", "Expires", "Type"}
	rows := [][]string{}

	for _, lease := range s.leases {
		// Hostname - color based on type
		hostname := lease.Hostname
		if hostname == "*" {
			hostname = dimStyle.Render("(unnamed)")
		} else if lease.Type == "static" {
			hostname = staticStyle.Render(hostname)
		}

		// MAC address
		mac := lease.MACAddress
		if mac == "" {
			mac = dimStyle.Render("—")
		}

		// Expiration
		expires := dimStyle.Render("—")
		if lease.Expires > 0 {
			expiresTime := time.Unix(lease.Expires, 0)
			timeUntil := time.Until(expiresTime)
			expiresStr := expiresTime.Format("Jan 02 15:04")

			// Color code based on time until expiration
			if timeUntil < 1*time.Hour {
				expires = warnStyle.Render(expiresStr + " ⚠️")
			} else if timeUntil < 24*time.Hour {
				expires = warnStyle.Render(expiresStr)
			} else {
				expires = dimStyle.Render(expiresStr)
			}
		}

		// Type
		var typeStr string
		if lease.Type == "static" {
			typeStr = staticStyle.Render("Static")
		} else {
			typeStr = dynamicStyle.Render("Dynamic")
		}

		rows = append(rows, []string{
			hostname,
			lease.IPAddress,
			mac,
			expires,
			typeStr,
		})
	}

	summaryLine := fmt.Sprintf("Total: %d  │  %s  │  %s",
		len(s.leases),
		staticStyle.Render(fmt.Sprintf("%d Static", staticCount)),
		dynamicStyle.Render(fmt.Sprintf("%d Dynamic", dynamicCount)))

	return tables.TableConfig{
		Title:   "DHCP LEASES",
		Headers: headers,
		Rows:    rows,
		Summary: summaryLine,
	}
}

func (s *DHCPDataSource) FormatAsJSON() ([]byte, error) {
	return json.MarshalIndent(s.leases, "", "  ")
}

func (s *DHCPDataSource) EmptyMessage() string {
	return "No DHCP leases found."
}

// CaddyDataSource implements ListDataSource for Caddy
type CaddyDataSource struct {
	client    *api.CaddyClient
	hostnames map[string]string
}

// NewCaddyDataSource creates a new Caddy data source
func NewCaddyDataSource() *CaddyDataSource {
	return &CaddyDataSource{}
}

func (s *CaddyDataSource) Initialize() error {
	// Use default Caddy configuration (IP and port can be made configurable later)
	s.client = api.NewCaddyClient("192.168.1.15", 2019)
	return nil
}

func (s *CaddyDataSource) FetchData() (interface{}, error) {
	hostnames, err := s.client.GetHostnameMap()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Caddy routes: %w", err)
	}
	s.hostnames = hostnames
	return hostnames, nil
}

func (s *CaddyDataSource) FormatAsTable() tables.TableConfig {
	headers := []string{"Hostname", "Upstream IP:Port"}
	rows := [][]string{}

	// Sort hostnames for consistent output
	sortedHostnames := make([]string, 0, len(s.hostnames))
	for hostname := range s.hostnames {
		sortedHostnames = append(sortedHostnames, hostname)
	}
	sort.Strings(sortedHostnames)

	for _, hostname := range sortedHostnames {
		upstream := s.hostnames[hostname]
		rows = append(rows, []string{hostname, upstream})
	}

	return tables.TableConfig{
		Title:   "CADDY REVERSE PROXY ROUTES",
		Headers: headers,
		Rows:    rows,
		Summary: fmt.Sprintf("Total: %d routes", len(s.hostnames)),
	}
}

func (s *CaddyDataSource) FormatAsJSON() ([]byte, error) {
	return json.MarshalIndent(s.hostnames, "", "  ")
}

func (s *CaddyDataSource) EmptyMessage() string {
	return "No Caddy routes found."
}

// AllDataSource implements ListDataSource for all services sync status
type AllDataSource struct {
	dashboard     *tui.SyncStatusDashboard
	caddyServerIP string
}

// NewAllDataSource creates a new all services data source
func NewAllDataSource() *AllDataSource {
	return &AllDataSource{
		caddyServerIP: "192.168.1.15", // Default Caddy IP
	}
}

func (s *AllDataSource) Initialize() error {
	// Dashboard will be initialized in FetchData
	return nil
}

func (s *AllDataSource) FetchData() (interface{}, error) {
	// Load configs
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Load AdguardHome config (optional)
	var adguardConfig config.AdguardConfig
	adguardConfig, _ = config.LoadAdguardConfig()

	// Create clients
	unboundClient := api.NewClient(cfg)

	var adguardClient *api.AdguardClient
	if adguardConfig.Enabled && adguardConfig.BaseURL != "" &&
		adguardConfig.Username != "" && adguardConfig.Password != "" {
		adguardClient = api.NewAdguardClient(adguardConfig.GetAdguardAPIConfig())
	}

	caddyClient := api.NewCaddyClient(s.caddyServerIP, 2019)
	dnsmasqClient := api.NewDNSMasqClient(cfg)

	// Create dashboard and load data with progress reporting
	s.dashboard = tui.NewSyncStatusDashboard(s.caddyServerIP)

	// Progress callback for CLI (for DNS queries, not service loading)
	var lastPhase string
	progressCallback := func(phase string, current, total int) {
		if phase != lastPhase {
			// New phase started
			if lastPhase != "" {
				fmt.Print("\r\033[K") // Clear line
			}
			lastPhase = phase
		}

		if current == 0 && total == 0 {
			// Phase just started, no count yet
			fmt.Printf("\r%s...", phase)
		} else {
			// Show progress with count
			fmt.Printf("\r%s... (%d/%d)", phase, current, total)
		}
	}

	err = s.dashboard.LoadSyncDataWithProgress(caddyClient, unboundClient, adguardClient, dnsmasqClient, progressCallback)

	// Clear the progress line
	fmt.Print("\r\033[K")

	if err != nil {
		return nil, fmt.Errorf("failed to load sync data: %w", err)
	}

	statuses := s.dashboard.GetFilteredStatuses()
	return statuses, nil
}

func (s *AllDataSource) FormatAsTable() tables.TableConfig {
	if s.dashboard == nil {
		return tables.TableConfig{
			Title:   "SYNC STATUS",
			Headers: []string{"Hostname", "Caddy", "Unbound", "Adguard", "Status"},
			Rows:    [][]string{},
			Summary: "No data loaded",
		}
	}

	statuses := s.dashboard.GetFilteredStatuses()
	summary := s.dashboard.GetSummary()

	// Define colors
	syncColor := lipgloss.Color("#9ece6a")      // Green
	partialColor := lipgloss.Color("#e0af68")   // Yellow
	warnColor := lipgloss.Color("#e0af68")      // Yellow
	outOfSyncColor := lipgloss.Color("#f7768e") // Red
	caddyOnlyColor := lipgloss.Color("#7aa2f7") // Blue
	dimColor := lipgloss.Color("#565f89")       // Dim

	// Styles
	syncStyle := lipgloss.NewStyle().Foreground(syncColor)
	partialStyle := lipgloss.NewStyle().Foreground(partialColor)
	warnStyle := lipgloss.NewStyle().Foreground(warnColor)
	outOfSyncStyle := lipgloss.NewStyle().Foreground(outOfSyncColor)
	caddyOnlyStyle := lipgloss.NewStyle().Foreground(caddyOnlyColor)
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)

	headers := []string{"Hostname", "Source", "DNS", "Upstream", "DHCP", "Unbound", "Adguard", "CF-Tunnel", "Overall"}
	rows := [][]string{}

	for _, status := range statuses {
		// DNS column - shows what DNS actually resolves to
		dnsText := dimStyle.Render("—")
		if status.DNSResolvedIP != "" {
			if status.DNSResolvedIP == "FAIL" {
				dnsText = dimStyle.Render("FAIL")
			} else if status.DNSResolvedIP == "NONE" {
				dnsText = dimStyle.Render("NONE")
			} else if status.DNSResolvedIP == status.CaddyIP {
				// Resolves to Caddy IP (correct!)
				dnsText = syncStyle.Render(status.DNSResolvedIP)
			} else {
				// Resolves to something else (might be direct to upstream)
				dnsText = warnStyle.Render(status.DNSResolvedIP)
			}
		}

		// Upstream column - shows the backend service Caddy proxies to
		upstreamText := dimStyle.Render("—")
		if status.UpstreamIP != "" {
			upstreamText = status.UpstreamIP
		}

		// DHCP column - shows DHCP lease info
		dhcpText := dimStyle.Render("—")
		if status.DHCPLeaseIP != "" {
			leaseType := "D"
			if status.DHCPLeaseType == "static" {
				leaseType = "S"
			}
			if status.DHCPMismatch {
				// Mismatch between Caddy upstream and DHCP IP
				dhcpText = fmt.Sprintf("⚠️ %s (%s)", status.DHCPLeaseIP, leaseType)
			} else {
				dhcpText = fmt.Sprintf("%s (%s)", status.DHCPLeaseIP, leaseType)
			}
		}

		// Unbound column
		unboundText := formatServiceStatus(status.UnboundStatus, status.CaddyIP != "", syncStyle, partialStyle, dimStyle)

		// Adguard column
		adguardText := formatServiceStatus(status.AdguardStatus, status.CaddyIP != "", syncStyle, partialStyle, dimStyle)

		// CF-Tunnel column - placeholder for future
		cfTunnelText := dimStyle.Render("—")

		// Overall status column
		overallText := formatOverallStatusText(status.Overall, syncStyle, partialStyle, outOfSyncStyle, caddyOnlyStyle)

		rows = append(rows, []string{
			status.Hostname,
			status.DataSource,
			dnsText,
			upstreamText,
			dhcpText,
			unboundText,
			adguardText,
			cfTunnelText,
			overallText,
		})
	}

	summaryLine := fmt.Sprintf("Total: %d  │  %s  │  %s  │  %s  │  %s",
		summary.Total,
		syncStyle.Render(fmt.Sprintf("%d Synced", summary.FullyInSync)),
		partialStyle.Render(fmt.Sprintf("%d Partial", summary.PartiallyInSync)),
		outOfSyncStyle.Render(fmt.Sprintf("%d Out of Sync", summary.OutOfSync)),
		caddyOnlyStyle.Render(fmt.Sprintf("%d Caddy Only", summary.CaddyOnly)))

	return tables.TableConfig{
		Title:   "3-WAY SYNC STATUS",
		Headers: headers,
		Rows:    rows,
		Summary: summaryLine,
	}
}

func (s *AllDataSource) FormatAsJSON() ([]byte, error) {
	if s.dashboard == nil {
		return json.MarshalIndent(map[string]interface{}{
			"statuses": []interface{}{},
			"summary":  nil,
		}, "", "  ")
	}

	statuses := s.dashboard.GetFilteredStatuses()
	summary := s.dashboard.GetSummary()

	return json.MarshalIndent(map[string]interface{}{
		"statuses": statuses,
		"summary":  summary,
	}, "", "  ")
}

func (s *AllDataSource) EmptyMessage() string {
	return "No sync data found."
}

// Helper function to format service status
func formatServiceStatus(status tui.ServiceStatus, hasInCaddy bool, syncStyle, partialStyle, dimStyle lipgloss.Style) string {
	if !hasInCaddy {
		// Not in Caddy
		if !status.Present {
			return syncStyle.Render("OK")
		}
		return partialStyle.Render("EXTRA")
	}

	// In Caddy
	if !status.Present {
		return dimStyle.Render("MISSING")
	}

	if status.InSync {
		return syncStyle.Render(status.IP)
	}
	return partialStyle.Render(status.IP)
}

// Helper function to format overall status
func formatOverallStatusText(status tui.OverallSyncStatus, syncStyle, partialStyle, outOfSyncStyle, caddyOnlyStyle lipgloss.Style) string {
	switch status {
	case tui.FullyInSync:
		return syncStyle.Render("✓ Synced")
	case tui.PartiallyInSync:
		return partialStyle.Render("⚠ Partial")
	case tui.OutOfSync:
		return outOfSyncStyle.Render("✗ Out of Sync")
	case tui.CaddyOnly:
		return caddyOnlyStyle.Render("◆ Caddy Only")
	default:
		return "?"
	}
}
