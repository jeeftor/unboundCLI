package cmd

import (
	"fmt"
	"os"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/config"
	"github.com/jeeftor/unboundCLI/internal/logging"
	"github.com/jeeftor/unboundCLI/internal/tui"
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
	Run: func(cmd *cobra.Command, args []string) {
		// Load main config
		cfg, err := config.LoadConfig()
		if err != nil {
			logging.Error("Error loading configuration", "error", err)
			fmt.Printf("❌ Error loading configuration: %v\n", err)
			fmt.Println("Please run 'config' command to set up API access")
			os.Exit(1)
		}

		// Create clients
		var unboundClient *api.Client
		var adguardClient *api.AdguardClient

		// Create UnboundDNS client (unless skipped)
		if !statusSkipUnbound {
			unboundClient = api.NewClient(cfg)
		}

		// Create AdguardHome client (unless skipped)
		if !statusSkipAdguard {
			adguardConfig, err := config.LoadAdguardConfig()
			if err != nil {
				if !statusCompact {
					fmt.Printf("⚠️  Warning: Could not load AdguardHome config: %v\n", err)
					fmt.Println("AdguardHome status will be skipped. Use --skip-adguard to suppress this warning.")
				}
			} else if adguardConfig.Enabled {
				// Validate AdguardHome config
				if adguardConfig.BaseURL != "" && adguardConfig.Username != "" && adguardConfig.Password != "" {
					adguardClient = api.NewAdguardClient(adguardConfig.GetAdguardAPIConfig())
				} else {
					if !statusCompact {
						fmt.Println("⚠️  Warning: AdguardHome configuration incomplete")
						fmt.Println("Set ADGUARD_BASE_URL, ADGUARD_USERNAME, and ADGUARD_PASSWORD")
					}
				}
			}
		}

		// Create Caddy client
		caddyClient := api.NewCaddyClient(statusCaddyServerIP, statusCaddyServerPort)

		// Create dashboard
		dashboard := tui.NewSyncStatusDashboard()

		// Set up filters
		filters := tui.StatusFilters{
			ShowOnlyOutOfSync: statusOutOfSyncOnly,
			HostnameFilter:    statusHostnameFilter,
		}
		dashboard.SetFilters(filters)

		// Load data from all systems
		if !statusCompact {
			fmt.Print("🔍 Fetching sync status from all systems...")
		}

		err = dashboard.LoadSyncData(caddyClient, unboundClient, adguardClient)
		if err != nil {
			fmt.Printf("\n❌ Error loading sync data: %v\n", err)
			os.Exit(1)
		}

		if !statusCompact {
			fmt.Println(" Done!")
		}

		// Create renderer
		renderer := tui.NewSyncStatusRenderer(dashboard)
		renderer.SetShowIPs(statusShowIPs)

		// Render output
		if statusCompact {
			fmt.Println(renderer.RenderCompactSummary())
		} else {
			fmt.Print(renderer.RenderDashboard())
		}

		// Check if there are any issues and exit with appropriate code
		summary := dashboard.GetSummary()
		if summary.OutOfSync > 0 || summary.PartiallyInSync > 0 {
			if !statusCompact {
				fmt.Printf("\n💡 TIP: Run sync commands to fix out-of-sync services:\n")
				fmt.Printf("   unboundCLI caddy-sync-all --dry-run  # Preview changes\n")
				fmt.Printf("   unboundCLI caddy-sync-all            # Apply changes\n")
			}
			os.Exit(1) // Exit with error code for scripting
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Add flags
	statusCmd.Flags().
		StringVar(&statusCaddyServerIP, "caddy-ip", "192.168.1.15", "IP address of the Caddy server")
	statusCmd.Flags().
		IntVar(&statusCaddyServerPort, "caddy-port", 2019, "Admin port of the Caddy server")
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
