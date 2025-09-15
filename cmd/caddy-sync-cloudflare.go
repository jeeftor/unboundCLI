package cmd

import (
	"fmt"
	"os"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/config"
	sync2 "github.com/jeeftor/unboundCLI/internal/exec/sync"
	"github.com/jeeftor/unboundCLI/internal/logging"
	"github.com/spf13/cobra"
)

var (
	cfDryRun             bool
	cfCaddyServerIP      string
	cfCaddyServerPort    int
	cfEntryDescription   string
	cfLegacyDescriptions []string
	cfDirectSubdomain    string
	cfCaddySubdomain     string
	cfDirectOnly         bool
	cfCaddyOnly          bool
	cfPrompt             bool
)

// caddySyncCloudflareCmd represents the caddy-sync-cloudflare command
var caddySyncCloudflareCmd = &cobra.Command{
	Use:   "caddy-sync-cloudflare",
	Short: "Synchronize UnboundDNS entries with Caddy server for Cloudflare tunnel routing",
	Long: `Synchronize DNS entries in Unbound with hostnames from a Caddy server for dual-mode Cloudflare tunnel routing.

This command queries the Caddy server for its configuration, extracts all hostnames from the routes,
and creates DNS entries with two modes:

1. Direct Mode (service.dev.vookie.net): Points directly to the service IP for LAN access
2. Caddy Mode (service.caddy.vookie.net): Points to Caddy server for reverse proxy access

This enables flexible routing where services can be accessed either directly or through Caddy,
supporting both LAN optimization and external Cloudflare tunnel access patterns.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate flag combinations
		if cfDirectOnly && cfCaddyOnly {
			fmt.Println("Error: Cannot specify both --direct-only and --caddy-only")
			os.Exit(1)
		}

		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			logging.Error("Error loading configuration", "error", err)
			// Create sync UI for error message
			syncUI := sync2.NewSyncUI()
			fmt.Println(
				syncUI.RenderError(
					fmt.Errorf(
						"error loading configuration: %v\nPlease run 'config' command to set up API access",
						err,
					),
				),
			)
			os.Exit(1)
		}

		// Create unbound client
		unboundClient := api.NewClient(cfg)
		unboundClient.Prompt = cfPrompt

		// Create sync UI
		syncUI := sync2.NewSyncUI()

		// Determine which modes to sync
		syncDirect := !cfCaddyOnly
		syncCaddy := !cfDirectOnly

		// Setup sync options
		options := sync2.CaddyCloudflareSyncOptions{
			DryRun:             cfDryRun,
			CaddyServerIP:      cfCaddyServerIP,
			CaddyServerPort:    cfCaddyServerPort,
			EntryDescription:   cfEntryDescription,
			LegacyDescriptions: cfLegacyDescriptions,
			DirectSubdomain:    cfDirectSubdomain,
			CaddySubdomain:     cfCaddySubdomain,
			SyncDirect:         syncDirect,
			SyncCaddy:          syncCaddy,
			Verbose:            verbose,
		}

		// Print header
		fmt.Print(syncUI.RenderCloudflareHeader(syncDirect, syncCaddy))

		// Print sync targets
		fmt.Print(syncUI.RenderCloudflareSyncTargets(syncDirect, syncCaddy, cfDirectSubdomain, cfCaddySubdomain))

		// Fetch and process data
		fmt.Println(syncUI.RenderFetchingMessage(cfCaddyServerIP, cfCaddyServerPort))

		// Perform the sync operation
		result, err := sync2.SyncCaddyWithCloudflare(unboundClient, options)
		if err != nil {
			logging.Error("Error during sync operation", "error", err)
			fmt.Println(
				syncUI.RenderError(
					fmt.Errorf("error during sync operation: %v", err),
				),
			)
			os.Exit(1)
		}

		if len(result.HostnameMap) == 0 {
			fmt.Println(syncUI.RenderWarning("No hostnames found in Caddy config"))
			return
		}

		// Display hostname count
		fmt.Print(syncUI.RenderHostnameCount(len(result.HostnameMap)))

		// Display hostnames if verbose
		if verbose {
			fmt.Println()

			// Convert map keys to slice for rendering
			hostnames := make([]string, 0, len(result.HostnameMap))
			for hostname := range result.HostnameMap {
				hostnames = append(hostnames, hostname)
			}

			fmt.Print(syncUI.RenderHostnameList(hostnames))
		}

		// Print summary of changes
		fmt.Print(syncUI.RenderCloudflareSummary(result))

		// If dry run, just print what would happen and exit
		if cfDryRun {
			fmt.Print(syncUI.RenderCloudflareDryRunOutput(result, cfEntryDescription))
			return
		}

		// Display changes as they are applied
		fmt.Print(syncUI.RenderCloudflareChanges(result, cfEntryDescription))
	},
}

func init() {
	rootCmd.AddCommand(caddySyncCloudflareCmd)

	// Add flags
	caddySyncCloudflareCmd.Flags().
		BoolVar(&cfDryRun, "dry-run", false, "Show what would be done without making any changes")
	caddySyncCloudflareCmd.Flags().
		StringVar(&cfCaddyServerIP, "caddy-ip", "192.168.1.15", "IP address of the Caddy server")
	caddySyncCloudflareCmd.Flags().
		IntVar(&cfCaddyServerPort, "caddy-port", 2019, "Admin port of the Caddy server")
	caddySyncCloudflareCmd.Flags().
		StringVar(&cfEntryDescription, "description", "Entry created by unboundCLI caddy-sync-cloudflare",
			"Description to use for created entries")
	caddySyncCloudflareCmd.Flags().
		StringSliceVar(&cfLegacyDescriptions, "legacy-desc", []string{"Route via Cloudflare"},
			"Legacy descriptions to consider as created by sync")
	caddySyncCloudflareCmd.Flags().
		StringVar(&cfDirectSubdomain, "direct-subdomain", "dev", "Subdomain for direct service access (e.g., 'dev' for service.dev.vookie.net)")
	caddySyncCloudflareCmd.Flags().
		StringVar(&cfCaddySubdomain, "caddy-subdomain", "caddy", "Subdomain for Caddy proxy access (e.g., 'caddy' for service.caddy.vookie.net)")
	caddySyncCloudflareCmd.Flags().
		BoolVar(&cfDirectOnly, "direct-only", false, "Sync only direct access entries (skip Caddy proxy entries)")
	caddySyncCloudflareCmd.Flags().
		BoolVar(&cfCaddyOnly, "caddy-only", false, "Sync only Caddy proxy entries (skip direct access entries)")
	caddySyncCloudflareCmd.Flags().
		BoolVar(&cfPrompt, "prompt", false, "Prompt before each API call (useful for debugging)")
}
