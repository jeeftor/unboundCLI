package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	runtimeapp "github.com/jeeftor/caddy-dns-sync/internal/app"
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

	tuiCmd.Flags().StringVar(&tuiCaddyServerIP, "caddy-server-ip", runtimeapp.DefaultCaddyServerIP,
		"IP address of the Caddy server (source of truth)")
	tuiCmd.Flags().IntVar(&tuiCaddyServerPort, "caddy-server-port", runtimeapp.DefaultCaddyServerPort,
		"Port number for Caddy admin API")
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Silence logging initially (will be redirected to TUI once it starts)
	logging.SetCustomHandler(func(level, message string) {
		// Discard logs during initialization
	})

	runtime, err := runtimeapp.LoadRuntime(runtimeapp.RuntimeOptions{
		CaddyServerIP:     tuiCaddyServerIP,
		CaddyServerPort:   tuiCaddyServerPort,
		IncludeUnbound:    true,
		IncludeDNSMasq:    true,
		IncludeAdguard:    true,
		IncludeCloudflare: true,
	})
	if err != nil {
		logging.ResetToStderr()
		fmt.Printf("Error loading configuration: %v\n", err)
		fmt.Println("Please run 'config' command to set up API access")
		return err
	}

	// Create TUI application
	tuiApp := tui.NewAppModel(
		runtime.Clients.Caddy,
		runtime.Clients.Unbound,
		runtime.Clients.Adguard,
		runtime.Clients.DNSMasq,
		runtime.CaddyEndpoint.ServerIP,
		runtime.Clients.Cloudflare,
		runtime.CaddyServiceURL,
	)

	// NOW redirect logging to TUI log widget
	logging.SetCustomHandler(func(level, message string) {
		tuiApp.AddLog(level, message)
	})

	// Add initialization logs retroactively
	if runtime.Clients.Adguard != nil {
		tuiApp.AddLog("INFO", "AdGuard client initialized")
	} else {
		tuiApp.AddLog("INFO", "AdGuard client not available (disabled or not configured)")
	}
	if runtime.Clients.Cloudflare != nil {
		tuiApp.AddLog("INFO", "Cloudflare client initialized")
	}

	// Reset logging to stderr when TUI exits
	defer logging.ResetToStderr()

	// Run the Bubble Tea program
	p := tea.NewProgram(
		tuiApp,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	finalModel, err := p.Run()
	if err != nil {
		logging.Error("Error running TUI", "error", err)
		fmt.Printf("Error: %v\n", err)
		return err
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
