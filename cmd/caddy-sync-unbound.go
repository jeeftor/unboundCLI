package cmd

import (
	"fmt"
	"os"

	sync2 "github.com/jeeftor/unboundCLI/internal/exec/sync"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/config"
	"github.com/jeeftor/unboundCLI/internal/logging"
	"github.com/spf13/cobra"
)

var (
	dryRun             bool
	caddyServerIP      string
	caddyServerPort    int
	entryDescription   string
	legacyDescriptions []string
	unboundPrompt      bool
)

// caddySyncUnboundCmd represents the caddy-sync-unbound command
var caddySyncUnboundCmd = &cobra.Command{
	Use:   "caddy-sync-unbound",
	Short: "Synchronize UnboundDNS entries with Caddy server",
	Long: `Synchronize DNS entries in Unbound with hostnames from a Caddy server.

This command queries the Caddy server for its configuration, extracts all
hostnames from the routes, and ensures that corresponding DNS entries exist
in Unbound. It will add missing entries, update changed ones with the correct
description, and remove entries that were previously created by this command
but are no longer present in Caddy.`,
	Run: func(cmd *cobra.Command, args []string) {
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
		unboundClient.Prompt = unboundPrompt

		// Create sync UI
		syncUI := sync2.NewSyncUI()

		// Setup sync options
		options := sync2.CaddySyncOptions{
			DryRun:             dryRun,
			CaddyServerIP:      caddyServerIP,
			CaddyServerPort:    caddyServerPort,
			EntryDescription:   entryDescription,
			LegacyDescriptions: legacyDescriptions,
			Verbose:            verbose,
		}

		// Print header
		fmt.Print(syncUI.RenderHeader())

		// Fetch and process data
		fmt.Println(syncUI.RenderFetchingMessage(caddyServerIP, caddyServerPort))

		// Perform the sync operation
		result, err := sync2.SyncCaddyWithUnbound(unboundClient, options)
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
		fmt.Print(syncUI.RenderSummary(result))

		// If dry run, just print what would happen and exit
		if dryRun {
			fmt.Print(syncUI.RenderDryRunOutput(result, entryDescription))
			return
		}

		// Display changes as they are applied
		fmt.Print(syncUI.RenderChanges(result, entryDescription))
	},
}

func init() {
	rootCmd.AddCommand(caddySyncUnboundCmd)

	// Add flags
	caddySyncUnboundCmd.Flags().
		BoolVar(&dryRun, "dry-run", false, "Show what would be done without making any changes")
	caddySyncUnboundCmd.Flags().
		StringVar(&caddyServerIP, "caddy-ip", "192.168.1.15", "IP address of the Caddy server")
	caddySyncUnboundCmd.Flags().
		IntVar(&caddyServerPort, "caddy-port", 2019, "Admin port of the Caddy server")
	caddySyncUnboundCmd.Flags().
		StringVar(&entryDescription, "description", "Entry created by unboundCLI caddy-sync-unbound",
			"Description to use for created entries")
	caddySyncUnboundCmd.Flags().
		StringSliceVar(&legacyDescriptions, "legacy-desc", []string{"Route via Caddy"},
			"Legacy descriptions to consider as created by sync")
	caddySyncUnboundCmd.Flags().
		BoolVar(&unboundPrompt, "prompt", false, "Prompt before each API call (useful for debugging)")
}
