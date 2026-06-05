package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/tui"
	"github.com/spf13/cobra"
)

var (
	tuiCaddyServerIP   string
	tuiCaddyServerPort int
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI dashboard",
	Long: `Launch an interactive terminal user interface (TUI) for monitoring
and managing DNS sync status across all services.

The TUI provides:
  - Real-time sync status for Caddy, Unbound, AdGuard, and DHCP
  - Interactive table with filtering and search
  - Sync operations with preview
  - Configuration editor

Keyboard shortcuts:
  ↑/↓, j/k      Navigate entries
  enter/space   Toggle selection (✓ mark appears)
  s             Open sync dialog (all or selected)
  o             Sync current entry
  r             Refresh data
  f             Cycle filters (All/Out of Sync/Caddy Only/Stale/Mismatches)
  c             Clear filter
  /             Search (type to filter, Enter/Esc to exit)
  t             Cycle sort (Hostname/IP/Status)
  i             Show Cloudflare detail overlay
  e             Edit CF tunnel settings for selected entry
  v             View full entry details
  C             Open config editor
  l             Toggle logs
  ?             Toggle help
  q             Quit`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)

	tuiCmd.Flags().StringVar(&tuiCaddyServerIP, "caddy-server-ip", "192.168.1.15",
		"IP address of the Caddy server (source of truth)")
	tuiCmd.Flags().IntVar(&tuiCaddyServerPort, "caddy-server-port", 2019,
		"Port number for Caddy admin API")
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Silence logging initially (will be redirected to TUI once it starts)
	logging.SetCustomHandler(func(level, message string) {
		// Discard logs during initialization
	})

	// Load main config (required for Unbound)
	cfg, err := config.LoadConfig()
	if err != nil {
		logging.ResetToStderr()
		fmt.Printf("Error loading configuration: %v\n", err)
		fmt.Println("Please run 'config' command to set up API access")
		return err
	}

	// Load AdguardHome config (optional)
	adguardConfig, err := config.LoadAdguardConfig()
	if err != nil {
		// Continue without AdGuard - not critical
		adguardConfig = config.AdguardConfig{Enabled: false}
	}

	// Create API clients
	unboundClient := api.NewClient(cfg)

	var adguardClient *api.AdguardClient
	if adguardConfig.Enabled && adguardConfig.BaseURL != "" &&
		adguardConfig.Username != "" && adguardConfig.Password != "" {
		adguardClient = api.NewAdguardClient(adguardConfig.GetAdguardAPIConfig())
	}

	// Create Caddy client
	caddyClient := api.NewCaddyClient(tuiCaddyServerIP, tuiCaddyServerPort)

	// Create DNSMasq client
	dnsmasqClient := api.NewDNSMasqClient(cfg)

	// Load Cloudflare config (optional — TUI works without it).
	// The "enabled" flag gates write operations (sync commands); the TUI only reads
	// CF data, so we create a client whenever credentials are present.
	var cfClient *api.CloudflareClient
	cfConfig, cfErr := config.LoadCloudflareConfig()
	if cfErr == nil && cfConfig.APIToken != "" && cfConfig.AccountID != "" {
		if c, err := api.NewCloudflareClient(cfConfig.GetCloudflareAPIConfig()); err == nil {
			cfClient = c
		} else {
			fmt.Fprintf(os.Stderr, "Warning: could not create Cloudflare client: %v\n", err)
		}
	}

	// Determine Caddy service URL for CF edit quick-fill.
	// Prefer configured CF_CADDY_SERVICE_URL; fall back to http://<caddyIP>:80.
	caddyServiceURL := ""
	if cfErr == nil {
		caddyServiceURL = cfConfig.CaddyServiceURL
	}
	if caddyServiceURL == "" {
		caddyServiceURL = fmt.Sprintf("http://%s:80", tuiCaddyServerIP)
	}

	// Create TUI application
	app := tui.NewAppModel(
		caddyClient,
		unboundClient,
		adguardClient,
		dnsmasqClient,
		tuiCaddyServerIP,
		cfClient,
		caddyServiceURL,
	)

	// NOW redirect logging to TUI log widget
	logging.SetCustomHandler(func(level, message string) {
		app.AddLog(level, message)
	})

	// Add initialization logs retroactively
	if adguardClient != nil {
		app.AddLog("INFO", "AdGuard client initialized")
	} else {
		app.AddLog("INFO", "AdGuard client not available (disabled or not configured)")
	}
	if cfClient != nil {
		app.AddLog("INFO", "Cloudflare client initialized")
	}

	// Reset logging to stderr when TUI exits
	defer logging.ResetToStderr()

	// Run the Bubble Tea program
	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	finalModel, err := p.Run()
	if err != nil {
		logging.Error("Error running TUI", "error", err)
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check if there were any errors in the final model
	if finalApp, ok := finalModel.(*tui.AppModel); ok {
		if finalApp.Error() != nil {
			logging.Error("TUI exited with error", "error", finalApp.Error())
			return finalApp.Error()
		}
	}

	return nil
}
