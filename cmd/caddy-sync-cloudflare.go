package cmd

import (
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	sync2 "github.com/jeeftor/caddy-dns-sync/internal/exec/sync"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
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
	Short: "Synchronize UnboundDNS overrides for Cloudflare-style Caddy hostnames",
	Long: `Synchronize DNS override entries in Unbound with hostnames from a Caddy server for dual-mode Cloudflare-style naming.

This command does not modify Cloudflare tunnel ingress rules. Use caddy-push-cloudflare
to sync Caddy hostnames into the configured Cloudflare tunnel.

This command queries the Caddy server for its configuration, extracts all hostnames from the routes,
and creates DNS entries with two modes:

1. Direct Mode (service.dev.example.com): Points directly to the service IP for LAN access
2. Caddy Mode (service.caddy.example.com): Points to Caddy server for reverse proxy access

This enables flexible routing where services can be accessed either directly or through Caddy,
supporting both LAN optimization and external Cloudflare tunnel access patterns.`,
	RunE: runCaddySyncCloudflare,
}

func runCaddySyncCloudflare(cmd *cobra.Command, args []string) error {
	if cfDirectOnly && cfCaddyOnly {
		return fmt.Errorf("cannot specify both --direct-only and --caddy-only")
	}

	runtime, err := app.LoadRuntime(app.RuntimeOptions{
		ConfigPath:      cfgFile,
		CaddyServerIP:   cfCaddyServerIP,
		CaddyServerPort: cfCaddyServerPort,
		IncludeUnbound:  true,
	})
	if err != nil {
		logging.Error("Error loading effective configuration", "error", err)
		return fmt.Errorf("error loading configuration: %w\nPlease run 'config' command to set up API access", err)
	}
	if runtime.Clients.Unbound == nil {
		return fmt.Errorf("Unbound configuration missing required API key, API secret, or base URL")
	}
	unboundClient := runtime.Clients.Unbound
	if cfPrompt {
		unboundClient = api.NewClient(runtime.UnboundConfig)
		unboundClient.Prompt = true
	}

	syncUI := sync2.NewSyncUI()
	syncDirect := !cfCaddyOnly
	syncCaddy := !cfDirectOnly

	options := sync2.CaddyCloudflareSyncOptions{
		BaseSyncOptions: sync2.BaseSyncOptions{
			DryRun:             cfDryRun,
			CaddyServerIP:      runtime.CaddyEndpoint.ServerIP,
			CaddyServerPort:    runtime.CaddyEndpoint.ServerPort,
			EntryDescription:   cfEntryDescription,
			LegacyDescriptions: cfLegacyDescriptions,
			Verbose:            verbose,
		},
		DirectSubdomain: cfDirectSubdomain,
		CaddySubdomain:  cfCaddySubdomain,
		SyncDirect:      syncDirect,
		SyncCaddy:       syncCaddy,
	}

	fmt.Fprint(cmd.OutOrStdout(), syncUI.RenderCloudflareHeader(syncDirect, syncCaddy))
	fmt.Fprint(cmd.OutOrStdout(), syncUI.RenderCloudflareSyncTargets(syncDirect, syncCaddy, cfDirectSubdomain, cfCaddySubdomain))
	fmt.Fprintln(cmd.OutOrStdout(), syncUI.RenderFetchingMessage(runtime.CaddyEndpoint.ServerIP, runtime.CaddyEndpoint.ServerPort))

	result, err := sync2.SyncCaddyWithCloudflare(unboundClient, options)
	if err != nil {
		logging.Error("Error during sync operation", "error", err)
		return fmt.Errorf("error during sync operation: %w", err)
	}

	if len(result.HostnameMap) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), syncUI.RenderWarning("No hostnames found in Caddy config"))
		return nil
	}

	fmt.Fprint(cmd.OutOrStdout(), syncUI.RenderHostnameCount(len(result.HostnameMap)))
	if verbose {
		fmt.Fprintln(cmd.OutOrStdout())

		hostnames := make([]string, 0, len(result.HostnameMap))
		for hostname := range result.HostnameMap {
			hostnames = append(hostnames, hostname)
		}

		fmt.Fprint(cmd.OutOrStdout(), syncUI.RenderHostnameList(hostnames))
	}

	fmt.Fprint(cmd.OutOrStdout(), syncUI.RenderCloudflareSummary(result))
	if cfDryRun {
		fmt.Fprint(cmd.OutOrStdout(), syncUI.RenderCloudflareDryRunOutput(result, cfEntryDescription))
		return nil
	}

	fmt.Fprint(cmd.OutOrStdout(), syncUI.RenderCloudflareChanges(result, cfEntryDescription))
	return nil
}

func init() {
	rootCmd.AddCommand(caddySyncCloudflareCmd)

	// Add flags
	caddySyncCloudflareCmd.Flags().
		BoolVar(&cfDryRun, "dry-run", false, "Show what would be done without making any changes")
	caddySyncCloudflareCmd.Flags().
		StringVar(&cfCaddyServerIP, "caddy-ip", "", "Caddy server IP (defaults to selected config)")
	caddySyncCloudflareCmd.Flags().
		IntVar(&cfCaddyServerPort, "caddy-port", 0, "Caddy admin API port (defaults to selected config)")
	caddySyncCloudflareCmd.Flags().
		StringVar(&cfEntryDescription, "description", "Entry created by caddy-dns-sync caddy-sync-cloudflare",
			"Description to use for created entries")
	caddySyncCloudflareCmd.Flags().
		StringSliceVar(&cfLegacyDescriptions, "legacy-desc", []string{"Route via Cloudflare"},
			"Legacy descriptions to consider as created by sync")
	caddySyncCloudflareCmd.Flags().
		StringVar(&cfDirectSubdomain, "direct-subdomain", "dev", "Subdomain for direct service access (e.g., 'dev' for service.dev.example.com)")
	caddySyncCloudflareCmd.Flags().
		StringVar(&cfCaddySubdomain, "caddy-subdomain", "caddy", "Subdomain for Caddy proxy access (e.g., 'caddy' for service.caddy.example.com)")
	caddySyncCloudflareCmd.Flags().
		BoolVar(&cfDirectOnly, "direct-only", false, "Sync only direct access entries (skip Caddy proxy entries)")
	caddySyncCloudflareCmd.Flags().
		BoolVar(&cfCaddyOnly, "caddy-only", false, "Sync only Caddy proxy entries (skip direct access entries)")
	caddySyncCloudflareCmd.Flags().
		BoolVar(&cfPrompt, "prompt", false, "Prompt before each API call (useful for debugging)")
}
