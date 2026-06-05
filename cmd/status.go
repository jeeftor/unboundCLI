package cmd

import (
	"fmt"

	runtimeapp "github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/tui"
	"github.com/spf13/cobra"
)

var (
	statusCaddyServerIP   string
	statusCaddyServerPort int
	statusShowIPs         bool
	statusOutOfSyncOnly   bool
	statusHostnameFilter  string
	statusCompact         bool
	statusSkipAdguard     bool
	statusSkipUnbound     bool
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show 3-way sync status across Caddy, UnboundDNS, and AdguardHome",
	Long: `Display a comprehensive sync status dashboard showing which services are configured
in Caddy and whether they are properly synchronized to UnboundDNS and AdguardHome.

This command fetches data from all three systems and compares them to show:
- Which services are configured in Caddy (source of truth)
- Whether UnboundDNS has matching host overrides
- Whether AdguardHome has matching DNS rewrites
- Overall sync status for each service

	Use this command to quickly identify services that need synchronization.`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	runtime, err := runtimeapp.LoadRuntime(runtimeapp.RuntimeOptions{
		CaddyServerIP:   statusCaddyServerIP,
		CaddyServerPort: statusCaddyServerPort,
		IncludeUnbound:  !statusSkipUnbound,
		IncludeDNSMasq:  true,
		IncludeAdguard:  !statusSkipAdguard,
	})
	if err != nil {
		logging.Error("Error loading configuration", "error", err)
		return fmt.Errorf("error loading configuration: %w\nPlease run 'config' command to set up API access", err)
	}

	dashboard := tui.NewSyncStatusDashboard(runtime.CaddyEndpoint.ServerIP)
	dashboard.SetFilters(tui.StatusFilters{
		ShowOnlyOutOfSync: statusOutOfSyncOnly,
		HostnameFilter:    statusHostnameFilter,
	})

	if !statusCompact {
		fmt.Fprint(cmd.OutOrStdout(), "🔍 Fetching sync status from all systems...")
	}

	err = dashboard.LoadSyncData(
		runtime.Clients.Caddy,
		runtime.Clients.Unbound,
		runtime.Clients.Adguard,
		runtime.Clients.DNSMasq,
	)
	if err != nil {
		return fmt.Errorf("error loading sync data: %w", err)
	}

	if !statusCompact {
		fmt.Fprintln(cmd.OutOrStdout(), " Done!")
	}

	renderer := tui.NewSyncStatusRenderer(dashboard)
	renderer.SetShowIPs(statusShowIPs)

	if statusCompact {
		fmt.Fprintln(cmd.OutOrStdout(), renderer.RenderCompactSummary())
	} else {
		fmt.Fprint(cmd.OutOrStdout(), renderer.RenderDashboard())
	}

	summary := dashboard.GetSummary()
	if summary.OutOfSync > 0 || summary.PartiallyInSync > 0 {
		if !statusCompact {
			fmt.Fprintln(cmd.OutOrStdout(), "\n💡 TIP: Run sync commands to fix out-of-sync services:")
			fmt.Fprintln(cmd.OutOrStdout(), "   caddy-dns-sync caddy-sync-all --dry-run  # Preview changes")
			fmt.Fprintln(cmd.OutOrStdout(), "   caddy-dns-sync caddy-sync-all            # Apply changes")
		}
		return exitCode(1)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Add flags
	statusCmd.Flags().
		StringVar(&statusCaddyServerIP, "caddy-ip", runtimeapp.DefaultCaddyServerIP, "IP address of the Caddy server")
	statusCmd.Flags().
		IntVar(&statusCaddyServerPort, "caddy-port", runtimeapp.DefaultCaddyServerPort, "Admin port of the Caddy server")
	statusCmd.Flags().
		BoolVar(&statusShowIPs, "show-ips", false, "Show IP addresses in the table")
	statusCmd.Flags().
		BoolVar(&statusOutOfSyncOnly, "out-of-sync-only", false, "Show only services that are out of sync")
	statusCmd.Flags().
		StringVar(&statusHostnameFilter, "hostname", "", "Filter by hostname (partial match)")
	statusCmd.Flags().
		BoolVar(&statusCompact, "compact", false, "Show compact one-line summary only")
	statusCmd.Flags().
		BoolVar(&statusSkipAdguard, "skip-adguard", false, "Skip AdguardHome status check")
	statusCmd.Flags().
		BoolVar(&statusSkipUnbound, "skip-unbound", false, "Skip UnboundDNS status check")
}
