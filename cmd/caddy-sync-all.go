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
	allDryRun             bool
	allCaddyServerIP      string
	allCaddyServerPort    int
	allEntryDescription   string
	allLegacyDescriptions []string
	allUnboundOnly        bool
	allAdguardOnly        bool
	allPrompt             bool
)

// caddySyncAllCmd represents the caddy-sync-all command
var caddySyncAllCmd = &cobra.Command{
	Use:   "caddy-sync-all",
	Short: "Synchronize Caddy hostnames to both UnboundDNS and AdguardHome",
	Long: `Synchronize DNS entries in both UnboundDNS and AdguardHome with hostnames from a Caddy server.

This unified command queries the Caddy server for its configuration, extracts all
hostnames from the routes, and ensures that corresponding DNS entries exist in
both UnboundDNS (host overrides) and AdguardHome (DNS rewrites).

This creates comprehensive split-horizon DNS where:
- UnboundDNS handles router-level DNS resolution
- AdguardHome handles client-level DNS resolution and ad blocking
- Both point *.vookie.net domains to the Caddy server for internal access

The command can target specific systems using --unbound-only or --adguard-only flags,
or sync to both systems simultaneously (default behavior).`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate flag combinations
		if allUnboundOnly && allAdguardOnly {
			fmt.Println("Error: Cannot specify both --unbound-only and --adguard-only")
			os.Exit(1)
		}

		// Load main config (required for UnboundDNS)
		mainConfig, err := config.LoadConfig()
		if err != nil && !allAdguardOnly {
			logging.Error("Error loading main configuration", "error", err)
			syncUI := sync2.NewSyncUI()
			fmt.Println(
				syncUI.RenderError(
					fmt.Errorf(
						"error loading main configuration: %v\nPlease run 'config' command to set up API access",
						err,
					),
				),
			)
			os.Exit(1)
		}

		// Load AdguardHome config (optional unless adguard-only)
		var adguardConfig config.AdguardConfig
		var adguardEnabled bool
		if !allUnboundOnly {
			adguardConfig, err = config.LoadAdguardConfig()
			if err != nil {
				logging.Error("Error loading AdguardHome configuration", "error", err)
				if allAdguardOnly {
					syncUI := sync2.NewSyncUI()
					fmt.Println(
						syncUI.RenderError(
							fmt.Errorf("error loading AdguardHome configuration: %v", err),
						),
					)
					os.Exit(1)
				}
				// If not adguard-only, continue without AdguardHome
				adguardEnabled = false
			} else {
				adguardEnabled = adguardConfig.Enabled
			}
		}

		// Check what systems we'll actually sync to
		syncToUnbound := !allAdguardOnly
		syncToAdguard := !allUnboundOnly && adguardEnabled

		// Validate that we have at least one target
		if !syncToUnbound && !syncToAdguard {
			fmt.Println("No sync targets available:")
			if allAdguardOnly {
				fmt.Println("  - AdguardHome sync was requested but AdguardHome is not enabled")
				fmt.Println("  - Set ADGUARD_ENABLED=true and configure credentials")
			} else if !adguardEnabled {
				fmt.Println("  - UnboundDNS: Disabled by --adguard-only flag")
				fmt.Println("  - AdguardHome: Not enabled (set ADGUARD_ENABLED=true)")
			}
			os.Exit(1)
		}

		// Validate AdguardHome config if we're going to use it
		if syncToAdguard {
			if adguardConfig.BaseURL == "" || adguardConfig.Username == "" || adguardConfig.Password == "" {
				fmt.Println("AdguardHome configuration missing required fields (BaseURL, Username, Password)")
				fmt.Println("Set environment variables:")
				fmt.Println("  ADGUARD_BASE_URL=http://192.168.0.1:3000")
				fmt.Println("  ADGUARD_USERNAME=admin")
				fmt.Println("  ADGUARD_PASSWORD=password")
				os.Exit(1)
			}
		}

		// Create clients
		var unboundClient *api.Client
		var adguardClient *api.AdguardClient

		if syncToUnbound {
			unboundClient = api.NewClient(mainConfig)
		}
		if syncToAdguard {
			adguardClient = api.NewAdguardClient(adguardConfig.GetAdguardAPIConfig())
			adguardClient.Prompt = allPrompt
		}

		// Create sync UI
		syncUI := sync2.NewSyncUI()

		// Print header
		fmt.Print(syncUI.RenderUnifiedHeader(syncToUnbound, syncToAdguard))

		// Print target systems
		fmt.Print(syncUI.RenderSyncTargets(syncToUnbound, syncToAdguard))

		// Fetch and process data
		fmt.Println(syncUI.RenderFetchingMessage(allCaddyServerIP, allCaddyServerPort))

		// Setup common sync options
		commonOptions := sync2.CommonSyncOptions{
			DryRun:             allDryRun,
			CaddyServerIP:      allCaddyServerIP,
			CaddyServerPort:    allCaddyServerPort,
			EntryDescription:   allEntryDescription,
			LegacyDescriptions: allLegacyDescriptions,
			Verbose:            verbose,
		}

		// Perform unified sync
		result, err := sync2.UnifiedCaddySync(unboundClient, adguardClient, commonOptions)
		if err != nil {
			logging.Error("Error during unified sync operation", "error", err)
			fmt.Println(
				syncUI.RenderError(
					fmt.Errorf("error during unified sync operation: %v", err),
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
			hostnames := make([]string, 0, len(result.HostnameMap))
			for hostname := range result.HostnameMap {
				hostnames = append(hostnames, hostname)
			}
			fmt.Print(syncUI.RenderHostnameList(hostnames))
		}

		// Print summary of changes for both systems
		fmt.Print(syncUI.RenderUnifiedSummary(result))

		// If dry run, just print what would happen and exit
		if allDryRun {
			fmt.Print(syncUI.RenderUnifiedDryRunOutput(result, allEntryDescription))
			return
		}

		// Display changes as they are applied
		fmt.Print(syncUI.RenderUnifiedChanges(result, allEntryDescription))
	},
}

func init() {
	rootCmd.AddCommand(caddySyncAllCmd)

	// Add flags
	caddySyncAllCmd.Flags().
		BoolVar(&allDryRun, "dry-run", false, "Show what would be done without making any changes")
	caddySyncAllCmd.Flags().
		StringVar(&allCaddyServerIP, "caddy-ip", "192.168.1.15", "IP address of the Caddy server")
	caddySyncAllCmd.Flags().
		IntVar(&allCaddyServerPort, "caddy-port", 2019, "Admin port of the Caddy server")
	caddySyncAllCmd.Flags().
		StringVar(&allEntryDescription, "description", "Entry created by unboundCLI caddy-sync-all",
			"Description to use for created entries")
	caddySyncAllCmd.Flags().
		StringSliceVar(&allLegacyDescriptions, "legacy-desc", []string{"Route via Caddy"},
			"Legacy descriptions to consider as created by sync")
	caddySyncAllCmd.Flags().
		BoolVar(&allUnboundOnly, "unbound-only", false, "Sync only to UnboundDNS (skip AdguardHome)")
	caddySyncAllCmd.Flags().
		BoolVar(&allAdguardOnly, "adguard-only", false, "Sync only to AdguardHome (skip UnboundDNS)")
	caddySyncAllCmd.Flags().
		BoolVar(&allPrompt, "prompt", false, "Prompt before each API call (useful for debugging)")
}
