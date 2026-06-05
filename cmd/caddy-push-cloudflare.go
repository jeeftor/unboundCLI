package cmd

import (
	"fmt"
	"os"
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
	Run: func(cmd *cobra.Command, args []string) {
		// Load Cloudflare config
		cfCfg, err := config.LoadCloudflareConfig()
		if err != nil {
			logging.Error("Error loading Cloudflare configuration", "error", err)
			fmt.Fprintf(os.Stderr, "Error loading Cloudflare configuration: %v\n", err)
			fmt.Fprintln(os.Stderr, "Run 'config-tui' or 'cloudflare-setup' to configure Cloudflare.")
			os.Exit(1)
		}

		if !cfCfg.Enabled {
			fmt.Fprintln(os.Stderr, "Cloudflare integration is disabled. Set enabled=true in config or CF_ENABLED=true.")
			os.Exit(1)
		}

		// Flags override config for Caddy coordinates
		caddyIP := cpCFCaddyIP
		caddyPort := cpCFCaddyPort
		serviceURL := cpCFCaddyServiceURL

		// Fall back to extended config for Caddy IP/port
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
			fmt.Fprintln(os.Stderr, "Caddy service URL is required (--caddy-service-url or caddy_service_url in config).")
			os.Exit(1)
		}

		// Create Cloudflare client
		cfClient, err := api.NewCloudflareClient(cfCfg.GetCloudflareAPIConfig())
		if err != nil {
			logging.Error("Error creating Cloudflare client", "error", err)
			fmt.Fprintf(os.Stderr, "Error creating Cloudflare client: %v\n", err)
			os.Exit(1)
		}

		caddyClient := api.NewCaddyClient(caddyIP, caddyPort)

		options := sync2.CaddyToCloudflareSyncOptions{
			DryRun:          cpCFDryRun,
			CaddyServiceURL: serviceURL,
			HostFilter:      cpCFHostFilter,
			Verbose:         cpCFVerbose,
		}

		if cpCFDryRun {
			fmt.Println("DRY RUN — no changes will be applied")
		}
		fmt.Printf("Fetching hostnames from Caddy at %s:%d...\n", caddyIP, caddyPort)

		result, err := sync2.SyncCaddyToCloudflare(caddyClient, cfClient, options)
		if err != nil {
			logging.Error("Error during sync", "error", err)
			fmt.Fprintf(os.Stderr, "Error during sync: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nCaddy hostnames found: %d\n", len(result.CaddyHostnames))

		if len(result.TunnelAdded) > 0 {
			sort.Strings(result.TunnelAdded)
			if cpCFDryRun {
				fmt.Printf("  [dry-run] Would add %d tunnel rule(s):\n", len(result.TunnelAdded))
			} else {
				fmt.Printf("  Added %d tunnel rule(s):\n", len(result.TunnelAdded))
			}
			for _, h := range result.TunnelAdded {
				fmt.Printf("    + %s\n", h)
			}
		}

		if len(result.TunnelRemoved) > 0 {
			sort.Strings(result.TunnelRemoved)
			if cpCFDryRun {
				fmt.Printf("  [dry-run] Would remove %d tunnel rule(s):\n", len(result.TunnelRemoved))
			} else {
				fmt.Printf("  Removed %d tunnel rule(s):\n", len(result.TunnelRemoved))
			}
			for _, h := range result.TunnelRemoved {
				fmt.Printf("    - %s\n", h)
			}
		}

		if cpCFVerbose && len(result.AlreadyCovered) > 0 {
			sort.Strings(result.AlreadyCovered)
			fmt.Printf("  Skipped %d hostname(s) covered by other tunnels:\n", len(result.AlreadyCovered))
			for _, h := range result.AlreadyCovered {
				fmt.Printf("    ~ %s\n", h)
			}
		}

		if len(result.StaleElsewhere) > 0 {
			stale := make([]string, 0, len(result.StaleElsewhere))
			for h := range result.StaleElsewhere {
				stale = append(stale, h)
			}
			sort.Strings(stale)
			fmt.Printf("  Stale in other tunnels (not in Caddy): %d\n", len(stale))
			if cpCFVerbose {
				for _, h := range stale {
					fmt.Printf("    ? %s (tunnel: %s)\n", h, result.StaleElsewhere[h])
				}
			}
		}

		if !cpCFDryRun {
			if len(result.DNSAdded) > 0 {
				sort.Strings(result.DNSAdded)
				fmt.Printf("  DNS records created: %s\n", strings.Join(result.DNSAdded, ", "))
			}
			if len(result.DNSRemoved) > 0 {
				sort.Strings(result.DNSRemoved)
				fmt.Printf("  DNS records removed: %s\n", strings.Join(result.DNSRemoved, ", "))
			}
			if len(result.TunnelAdded) == 0 && len(result.TunnelRemoved) == 0 {
				fmt.Println("  No changes needed — tunnel is already in sync.")
			}
		}
	},
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
