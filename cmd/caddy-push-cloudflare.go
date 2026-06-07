package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	sync2 "github.com/jeeftor/caddy-dns-sync/internal/exec/sync"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/spf13/cobra"
)

var (
	cpCFDryRun          bool
	cpCFCaddyIP         string
	cpCFCaddyPort       int
	cpCFCaddyServiceURL string
	cpCFHostFilter      []string
	cpCFVerbose         bool
)

var caddyPushCloudflareCmd = &cobra.Command{
	Use:   "caddy-push-cloudflare",
	Short: "Push Caddy hostnames into a Cloudflare tunnel and create DNS CNAME records",
	Long: `Reads hostnames from the Caddy Admin API and synchronizes them into the configured
Cloudflare tunnel as ingress rules, creating or updating the corresponding DNS CNAME
records in the Cloudflare zone.

Hostnames found in other tunnels in the same account are skipped (reported only).
Hostnames in the default tunnel that are no longer in Caddy are removed.

Configuration is loaded from ~/.caddy-dns-sync.json (cloudflare section) or environment
variables (CF_API_TOKEN, CF_ACCOUNT_ID, CF_ZONE_ID, CF_TUNNEL_ID, CF_CADDY_SERVICE_URL).`,
	RunE: runCaddyPushCloudflare,
}

func runCaddyPushCloudflare(cmd *cobra.Command, args []string) error {
	cfCfg, err := config.LoadCloudflareConfig()
	if err != nil {
		logging.Error("Error loading Cloudflare configuration", "error", err)
		return fmt.Errorf("error loading Cloudflare configuration: %w\nRun 'config-tui' or 'cloudflare-setup' to configure Cloudflare", err)
	}

	if !cfCfg.Enabled {
		return fmt.Errorf("Cloudflare integration is disabled. Set enabled=true in config or CF_ENABLED=true")
	}

	caddyIP := cpCFCaddyIP
	caddyPort := cpCFCaddyPort
	serviceURL := cpCFCaddyServiceURL

	if caddyIP == "" || caddyPort == 0 {
		extCfg, extErr := config.LoadExtendedConfig()
		if extErr == nil {
			if caddyIP == "" && extCfg.Caddy.ServerIP != "" {
				caddyIP = extCfg.Caddy.ServerIP
			}
			if caddyPort == 0 && extCfg.Caddy.ServerPort != 0 {
				caddyPort = extCfg.Caddy.ServerPort
			}
		}
	}
	if caddyPort == 0 {
		caddyPort = 2019
	}

	if serviceURL == "" {
		serviceURL = cfCfg.CaddyServiceURL
	}
	if serviceURL == "" {
		return fmt.Errorf("Caddy service URL is required (--caddy-service-url or caddy_service_url in config)")
	}

	cfClient, err := api.NewCloudflareClient(cfCfg.GetCloudflareAPIConfig())
	if err != nil {
		logging.Error("Error creating Cloudflare client", "error", err)
		return fmt.Errorf("error creating Cloudflare client: %w", err)
	}

	caddyClient := api.NewCaddyClient(caddyIP, caddyPort)

	options := sync2.CaddyToCloudflareSyncOptions{
		DryRun:          cpCFDryRun,
		CaddyServiceURL: serviceURL,
		HostFilter:      cpCFHostFilter,
		Verbose:         cpCFVerbose,
	}

	if cpCFDryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "DRY RUN - no changes will be applied")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Fetching hostnames from Caddy at %s:%d...\n", caddyIP, caddyPort)

	result, err := sync2.SyncCaddyToCloudflare(caddyClient, cfClient, options)
	if err != nil {
		logging.Error("Error during sync", "error", err)
		return fmt.Errorf("error during sync: %w", err)
	}

	renderCaddyPushCloudflareResult(cmd, result)
	return nil
}

func renderCaddyPushCloudflareResult(cmd *cobra.Command, result *sync2.CaddyToCloudflareSyncResult) {
	fmt.Fprintf(cmd.OutOrStdout(), "\nCaddy hostnames found: %d\n", len(result.CaddyHostnames))

	if len(result.TunnelAdded) > 0 {
		sort.Strings(result.TunnelAdded)
		if cpCFDryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "  [dry-run] Would add %d tunnel rule(s):\n", len(result.TunnelAdded))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  Added %d tunnel rule(s):\n", len(result.TunnelAdded))
		}
		for _, h := range result.TunnelAdded {
			fmt.Fprintf(cmd.OutOrStdout(), "    + %s\n", h)
		}
	}

	if len(result.TunnelRemoved) > 0 {
		sort.Strings(result.TunnelRemoved)
		if cpCFDryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "  [dry-run] Would remove %d tunnel rule(s):\n", len(result.TunnelRemoved))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  Removed %d tunnel rule(s):\n", len(result.TunnelRemoved))
		}
		for _, h := range result.TunnelRemoved {
			fmt.Fprintf(cmd.OutOrStdout(), "    - %s\n", h)
		}
	}

	if len(result.TunnelUpdated) > 0 {
		sort.Strings(result.TunnelUpdated)
		if cpCFDryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "  [dry-run] Would update %d tunnel rule(s):\n", len(result.TunnelUpdated))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  Updated %d tunnel rule(s):\n", len(result.TunnelUpdated))
		}
		for _, h := range result.TunnelUpdated {
			fmt.Fprintf(cmd.OutOrStdout(), "    ~ %s\n", h)
		}
	}

	if cpCFVerbose && len(result.AlreadyCovered) > 0 {
		sort.Strings(result.AlreadyCovered)
		fmt.Fprintf(cmd.OutOrStdout(), "  Skipped %d hostname(s) covered by other tunnels:\n", len(result.AlreadyCovered))
		for _, h := range result.AlreadyCovered {
			fmt.Fprintf(cmd.OutOrStdout(), "    ~ %s\n", h)
		}
	}

	if len(result.StaleElsewhere) > 0 {
		stale := make([]string, 0, len(result.StaleElsewhere))
		for h := range result.StaleElsewhere {
			stale = append(stale, h)
		}
		sort.Strings(stale)
		fmt.Fprintf(cmd.OutOrStdout(), "  Stale in other tunnels (not in Caddy): %d\n", len(stale))
		if cpCFVerbose {
			for _, h := range stale {
				fmt.Fprintf(cmd.OutOrStdout(), "    ? %s (tunnel: %s)\n", h, result.StaleElsewhere[h])
			}
		}
	}

	if !cpCFDryRun {
		if len(result.DNSAdded) > 0 {
			sort.Strings(result.DNSAdded)
			fmt.Fprintf(cmd.OutOrStdout(), "  DNS records created: %s\n", strings.Join(result.DNSAdded, ", "))
		}
		if len(result.DNSRemoved) > 0 {
			sort.Strings(result.DNSRemoved)
			fmt.Fprintf(cmd.OutOrStdout(), "  DNS records removed: %s\n", strings.Join(result.DNSRemoved, ", "))
		}
		if len(result.TunnelAdded) == 0 && len(result.TunnelUpdated) == 0 && len(result.TunnelRemoved) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "  No changes needed - tunnel is already in sync.")
		}
	}
}

func init() {
	rootCmd.AddCommand(caddyPushCloudflareCmd)

	caddyPushCloudflareCmd.Flags().
		BoolVar(&cpCFDryRun, "dry-run", false, "Show changes without applying them")
	caddyPushCloudflareCmd.Flags().
		StringVar(&cpCFCaddyIP, "caddy-ip", "", "Caddy Admin API IP (default from config)")
	caddyPushCloudflareCmd.Flags().
		IntVar(&cpCFCaddyPort, "caddy-port", 0, "Caddy Admin API port (default 2019)")
	caddyPushCloudflareCmd.Flags().
		StringVar(&cpCFCaddyServiceURL, "caddy-service-url", "",
			"Internal service URL for tunnel ingress rules (e.g. http://192.168.1.15:80)")
	caddyPushCloudflareCmd.Flags().
		StringSliceVar(&cpCFHostFilter, "host-filter", nil,
			"Only sync hostnames matching these domain suffixes (repeatable)")
	caddyPushCloudflareCmd.Flags().
		BoolVar(&cpCFVerbose, "verbose", false, "Show additional detail including skipped hostnames")
}
