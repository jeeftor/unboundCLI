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
	adguardDryRun             bool
	adguardCaddyServerIP      string
	adguardCaddyServerPort    int
	adguardEntryDescription   string
	adguardLegacyDescriptions []string
	adguardPrompt             bool
)

// caddySyncAdguardCmd represents the caddy-sync-adguard command
var caddySyncAdguardCmd = &cobra.Command{
	Use:   "caddy-sync-adguard",
	Short: "Synchronize AdguardHome DNS rewrites with Caddy server",
	Long: `Synchronize DNS rewrites in AdguardHome with hostnames from a Caddy server.

This command queries the Caddy server for its configuration, extracts all
hostnames from the routes, and ensures that corresponding DNS rewrites exist
in AdguardHome. It will add missing rewrites, update changed ones with the correct
IP address, and remove rewrites that were previously created by this command
but are no longer present in Caddy.

This creates split-horizon DNS where LAN clients using AdguardHome will resolve
*.vookie.net domains to the Caddy server IP for internal access.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load main config first (may be needed for fallback credentials)
		_, err := config.LoadConfig()
		if err != nil {
			logging.Error("Error loading main configuration", "error", err)
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

		// Load AdguardHome config
		adguardConfig, err := config.LoadAdguardConfig()
		if err != nil {
			logging.Error("Error loading AdguardHome configuration", "error", err)
			syncUI := sync2.NewSyncUI()
			fmt.Println(
				syncUI.RenderError(
					fmt.Errorf(
						"error loading AdguardHome configuration: %v",
						err,
					),
				),
			)
			os.Exit(1)
		}

		// Check if AdguardHome is enabled
		if !adguardConfig.Enabled {
			logging.Warn("AdguardHome sync is disabled in configuration")
			fmt.Println("AdguardHome sync is disabled. Set ADGUARD_ENABLED=true or enable in config file.")
			return
		}

		// Validate required AdguardHome config
		if adguardConfig.BaseURL == "" || adguardConfig.Username == "" || adguardConfig.Password == "" {
			logging.Error("Incomplete AdguardHome configuration")
			fmt.Println("AdguardHome configuration missing required fields (BaseURL, Username, Password)")
			fmt.Println("Set environment variables:")
			fmt.Println("  ADGUARD_BASE_URL=http://192.168.0.1:3000")
			fmt.Println("  ADGUARD_USERNAME=admin")
			fmt.Println("  ADGUARD_PASSWORD=password")
			os.Exit(1)
		}

		// Create AdguardHome client
		adguardClient := api.NewAdguardClient(adguardConfig.GetAdguardAPIConfig())
		adguardClient.Prompt = adguardPrompt

		// Create sync UI
		syncUI := sync2.NewSyncUI()

		// Setup sync options
		options := sync2.CaddyAdguardSyncOptions{
			DryRun:             adguardDryRun,
			CaddyServerIP:      adguardCaddyServerIP,
			CaddyServerPort:    adguardCaddyServerPort,
			EntryDescription:   adguardEntryDescription,
			LegacyDescriptions: adguardLegacyDescriptions,
			Verbose:            verbose,
		}

		// Print header
		fmt.Print(syncUI.RenderHeader())

		// Fetch and process data
		fmt.Println(syncUI.RenderFetchingMessage(adguardCaddyServerIP, adguardCaddyServerPort))

		// Perform the sync operation
		result, err := sync2.SyncCaddyWithAdguard(adguardClient, options)
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
		fmt.Print(syncUI.RenderAdguardSummary(result))

		// If dry run, just print what would happen and exit
		if adguardDryRun {
			fmt.Print(syncUI.RenderAdguardDryRunOutput(result, adguardEntryDescription))
			return
		}

		// Display changes as they are applied
		fmt.Print(syncUI.RenderAdguardChanges(result, adguardEntryDescription))
	},
}

func init() {
	rootCmd.AddCommand(caddySyncAdguardCmd)

	// Add flags
	caddySyncAdguardCmd.Flags().
		BoolVar(&adguardDryRun, "dry-run", false, "Show what would be done without making any changes")
	caddySyncAdguardCmd.Flags().
		StringVar(&adguardCaddyServerIP, "caddy-ip", "192.168.1.15", "IP address of the Caddy server")
	caddySyncAdguardCmd.Flags().
		IntVar(&adguardCaddyServerPort, "caddy-port", 2019, "Admin port of the Caddy server")
	caddySyncAdguardCmd.Flags().
		StringVar(&adguardEntryDescription, "description", "Entry created by unboundCLI caddy-sync-adguard",
			"Description to use for created rewrites (stored in AdguardHome comments)")
	caddySyncAdguardCmd.Flags().
		StringSliceVar(&adguardLegacyDescriptions, "legacy-desc", []string{"Route via Caddy"},
			"Legacy descriptions to consider as created by sync")
	caddySyncAdguardCmd.Flags().
		BoolVar(&adguardPrompt, "prompt", false, "Prompt before each API call (useful for debugging)")
}
