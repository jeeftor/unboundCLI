package cmd

import (
	"fmt"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	runtimeapp "github.com/jeeftor/caddy-dns-sync/internal/app"
	execsync "github.com/jeeftor/caddy-dns-sync/internal/exec/sync"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/sync"
	"github.com/spf13/cobra"
)

var (
	syncDryRun             bool
	syncCaddyServerIP      string
	syncCaddyServerPort    int
	syncEntryDescription   string
	syncLegacyDescriptions string
	syncUnboundOnly        bool
	syncAdguardOnly        bool
	syncPrompt             bool
)

// syncCmd is the parent command for sync operations
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize Caddy routes to DNS services",
	Long: `Sync Caddy reverse proxy routes to Unbound, Adguard, or both.

Available subcommands:
  all      - Sync to both Unbound and Adguard
  unbound  - Sync to Unbound only
  adguard  - Sync to Adguard only`,
}

// syncAllCmd syncs to all DNS services
var syncAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Sync Caddy routes to all DNS services",
	Long: `Synchronize DNS entries in both UnboundDNS and AdguardHome with hostnames from Caddy.

This unified command queries the Caddy server for its configuration, extracts all
hostnames from the routes, and ensures that corresponding DNS entries exist in
both UnboundDNS (host overrides) and AdguardHome (DNS rewrites).

The command can target specific systems using --unbound-only or --adguard-only flags,
or sync to both systems simultaneously (default behavior).`,
	RunE: runSyncAll,
}

// syncUnboundCmd syncs to Unbound only
var syncUnboundCmd = &cobra.Command{
	Use:   "unbound",
	Short: "Sync Caddy routes to Unbound",
	Long: `Synchronize DNS entries in UnboundDNS with hostnames from Caddy.

This command queries the Caddy server for its configuration, extracts all
hostnames from the routes, and ensures that corresponding DNS host override
entries exist in UnboundDNS pointing to the Caddy server.`,
	RunE: runSyncUnbound,
}

// syncAdguardCmd syncs to Adguard only
var syncAdguardCmd = &cobra.Command{
	Use:   "adguard",
	Short: "Sync Caddy routes to Adguard",
	Long: `Synchronize DNS entries in AdguardHome with hostnames from Caddy.

This command queries the Caddy server for its configuration, extracts all
hostnames from the routes, and ensures that corresponding DNS rewrite
entries exist in AdguardHome pointing to the Caddy server.`,
	RunE: runSyncAdguard,
}

// buildSyncOptions creates SyncOptions from command flags
func buildSyncOptions() *sync.SyncOptions {
	opts := sync.DefaultSyncOptions()
	opts.DryRun = syncDryRun
	opts.CaddyServerIP = syncCaddyServerIP
	opts.CaddyServerPort = syncCaddyServerPort
	opts.EntryDescription = syncEntryDescription
	opts.Verbose = verbose
	opts.UnboundOnly = syncUnboundOnly
	opts.AdguardOnly = syncAdguardOnly

	// Parse legacy descriptions
	if syncLegacyDescriptions != "" {
		opts.LegacyDescriptions = strings.Split(syncLegacyDescriptions, ",")
		// Trim whitespace from each description
		for i := range opts.LegacyDescriptions {
			opts.LegacyDescriptions[i] = strings.TrimSpace(opts.LegacyDescriptions[i])
		}
	}

	return opts
}

func runSyncAll(cmd *cobra.Command, args []string) error {
	// Validate flag combinations
	if syncUnboundOnly && syncAdguardOnly {
		return fmt.Errorf("cannot specify both --unbound-only and --adguard-only")
	}

	opts := buildSyncOptions()
	syncUI := execsync.NewSyncUI()

	runtime, err := runtimeapp.LoadRuntime(runtimeapp.RuntimeOptions{
		CaddyServerIP:   opts.CaddyServerIP,
		CaddyServerPort: opts.CaddyServerPort,
		IncludeUnbound:  !syncAdguardOnly,
		IncludeAdguard:  !syncUnboundOnly,
		RequireAdguard:  syncAdguardOnly,
	})
	if err != nil {
		logging.Error("Error loading sync runtime", "error", err)
		return fmt.Errorf("error loading sync runtime: %w", err)
	}

	// Check what systems we'll sync to
	syncToUnbound := !syncAdguardOnly
	syncToAdguard := !syncUnboundOnly && runtime.Clients.Adguard != nil

	// Validate that we have at least one target
	if !syncToUnbound && !syncToAdguard {
		fmt.Println("No sync targets available:")
		if syncAdguardOnly {
			fmt.Println("  - AdguardHome sync was requested but AdguardHome is not enabled")
			fmt.Println("  - Set ADGUARD_ENABLED=true and configure credentials")
		} else if runtime.Clients.Adguard == nil {
			fmt.Println("  - UnboundDNS: Disabled by --adguard-only flag")
			fmt.Println("  - AdguardHome: Not enabled (set ADGUARD_ENABLED=true)")
		}
		return fmt.Errorf("no sync targets available")
	}

	// Create executor and set clients
	executor := sync.NewSyncExecutor(opts)
	unboundClient, adguardClient := syncCommandClients(runtime, !syncAdguardOnly, !syncUnboundOnly && runtime.Clients.Adguard != nil)
	executor.SetClients(runtime.Clients.Caddy, unboundClient, adguardClient)

	// Print header
	fmt.Print(syncUI.RenderUnifiedHeader(syncToUnbound, syncToAdguard))
	fmt.Print(syncUI.RenderSyncTargets(syncToUnbound, syncToAdguard))
	fmt.Println(syncUI.RenderFetchingMessage(opts.CaddyServerIP, opts.CaddyServerPort))

	// Perform unified sync
	result, err := executor.SyncAll()
	if err != nil {
		logging.Error("Error during unified sync operation", "error", err)
		return fmt.Errorf("error during unified sync operation: %w", err)
	}

	if len(result.HostnameMap) == 0 {
		fmt.Println(syncUI.RenderWarning("No hostnames found in Caddy config"))
		return nil
	}

	// Display results
	fmt.Print(syncUI.RenderHostnameCount(len(result.HostnameMap)))

	if verbose {
		fmt.Println()
		hostnames := make([]string, 0, len(result.HostnameMap))
		for hostname := range result.HostnameMap {
			hostnames = append(hostnames, hostname)
		}
		fmt.Print(syncUI.RenderHostnameList(hostnames))
	}

	fmt.Print(syncUI.RenderUnifiedSummary(result))

	if syncDryRun {
		fmt.Print(syncUI.RenderUnifiedDryRunOutput(result, opts.EntryDescription))
	} else {
		fmt.Print(syncUI.RenderUnifiedChanges(result, opts.EntryDescription))
	}

	return nil
}

func runSyncUnbound(cmd *cobra.Command, args []string) error {
	opts := buildSyncOptions()
	syncUI := execsync.NewSyncUI()

	runtime, err := runtimeapp.LoadRuntime(runtimeapp.RuntimeOptions{
		CaddyServerIP:   opts.CaddyServerIP,
		CaddyServerPort: opts.CaddyServerPort,
		IncludeUnbound:  true,
	})
	if err != nil {
		logging.Error("Error loading configuration", "error", err)
		return fmt.Errorf("error loading configuration: %w\nPlease run 'config' command to set up API access", err)
	}

	// Create executor
	executor := sync.NewSyncExecutor(opts)
	unboundClient, _ := syncCommandClients(runtime, true, false)
	executor.SetClients(runtime.Clients.Caddy, unboundClient, nil)

	// Print header
	fmt.Print(syncUI.RenderHeader())
	fmt.Println(syncUI.RenderFetchingMessage(opts.CaddyServerIP, opts.CaddyServerPort))

	// Perform sync
	result, err := executor.SyncToUnbound()
	if err != nil {
		logging.Error("Error during Unbound sync", "error", err)
		return fmt.Errorf("error during Unbound sync: %w", err)
	}

	if len(result.HostnameMap) == 0 {
		fmt.Println(syncUI.RenderWarning("No hostnames found in Caddy config"))
		return nil
	}

	// Display results
	fmt.Print(syncUI.RenderHostnameCount(len(result.HostnameMap)))

	if verbose {
		fmt.Println()
		hostnames := make([]string, 0, len(result.HostnameMap))
		for hostname := range result.HostnameMap {
			hostnames = append(hostnames, hostname)
		}
		fmt.Print(syncUI.RenderHostnameList(hostnames))
	}

	fmt.Print(syncUI.RenderSummary(result))

	if syncDryRun {
		fmt.Print(syncUI.RenderDryRunOutput(result, opts.EntryDescription))
	} else {
		fmt.Print(syncUI.RenderChanges(result, opts.EntryDescription))
	}

	return nil
}

func runSyncAdguard(cmd *cobra.Command, args []string) error {
	opts := buildSyncOptions()
	syncUI := execsync.NewSyncUI()

	runtime, err := runtimeapp.LoadRuntime(runtimeapp.RuntimeOptions{
		CaddyServerIP:   opts.CaddyServerIP,
		CaddyServerPort: opts.CaddyServerPort,
		IncludeAdguard:  true,
		RequireAdguard:  true,
	})
	if err != nil {
		logging.Error("Error loading AdguardHome runtime", "error", err)
		return fmt.Errorf("error loading AdguardHome runtime: %w", err)
	}

	// Create executor
	executor := sync.NewSyncExecutor(opts)
	_, adguardClient := syncCommandClients(runtime, false, true)
	executor.SetClients(runtime.Clients.Caddy, nil, adguardClient)

	// Print header
	fmt.Print(syncUI.RenderHeader())
	fmt.Println(syncUI.RenderFetchingMessage(opts.CaddyServerIP, opts.CaddyServerPort))

	// Perform sync
	result, err := executor.SyncToAdguard()
	if err != nil {
		logging.Error("Error during Adguard sync", "error", err)
		return fmt.Errorf("error during Adguard sync: %w", err)
	}

	if len(result.HostnameMap) == 0 {
		fmt.Println(syncUI.RenderWarning("No hostnames found in Caddy config"))
		return nil
	}

	// Display results
	fmt.Print(syncUI.RenderHostnameCount(len(result.HostnameMap)))

	if verbose {
		fmt.Println()
		hostnames := make([]string, 0, len(result.HostnameMap))
		for hostname := range result.HostnameMap {
			hostnames = append(hostnames, hostname)
		}
		fmt.Print(syncUI.RenderHostnameList(hostnames))
	}

	fmt.Print(syncUI.RenderAdguardSummary(result))

	if syncDryRun {
		fmt.Print(syncUI.RenderAdguardDryRunOutput(result, opts.EntryDescription))
	} else {
		fmt.Print(syncUI.RenderAdguardChanges(result, opts.EntryDescription))
	}

	return nil
}

func syncCommandClients(runtime *runtimeapp.Runtime, useUnbound, useAdguard bool) (*api.Client, *api.AdguardClient) {
	var unboundClient *api.Client
	if useUnbound {
		unboundClient = runtime.Clients.Unbound
		if syncPrompt && runtime.Clients.Unbound != nil {
			unboundClient = api.NewClient(runtime.UnboundConfig)
			unboundClient.Prompt = true
		}
	}

	var adguardClient *api.AdguardClient
	if useAdguard {
		adguardClient = runtime.Clients.Adguard
		if syncPrompt && runtime.Clients.Adguard != nil {
			adguardClient = api.NewAdguardClient(runtime.AdguardConfig.GetAdguardAPIConfig())
			adguardClient.Prompt = true
		}
	}

	return unboundClient, adguardClient
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Add subcommands
	syncCmd.AddCommand(syncAllCmd)
	syncCmd.AddCommand(syncUnboundCmd)
	syncCmd.AddCommand(syncAdguardCmd)

	// Shared flags for all sync commands
	syncCmd.PersistentFlags().BoolVar(&syncDryRun, "dry-run", false, "Show what would be changed without applying")
	syncCmd.PersistentFlags().StringVar(&syncCaddyServerIP, "caddy-ip", runtimeapp.DefaultCaddyServerIP, "Caddy server IP")
	syncCmd.PersistentFlags().IntVar(&syncCaddyServerPort, "caddy-port", runtimeapp.DefaultCaddyServerPort, "Caddy admin API port")
	syncCmd.PersistentFlags().StringVar(&syncEntryDescription, "description", "Entry created by CaddySync", "Description for DNS entries")
	syncCmd.PersistentFlags().StringVar(&syncLegacyDescriptions, "legacy-desc", "Route via Caddy", "Comma-separated legacy descriptions")
	syncCmd.PersistentFlags().BoolVar(&syncPrompt, "prompt", false, "Prompt before each API call")

	// Target selection flags (only for 'all' subcommand)
	syncAllCmd.Flags().BoolVar(&syncUnboundOnly, "unbound-only", false, "Sync to Unbound only")
	syncAllCmd.Flags().BoolVar(&syncAdguardOnly, "adguard-only", false, "Sync to Adguard only")
}
