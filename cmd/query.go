package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	runtimeapp "github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/spf13/cobra"
)

var (
	queryServices string
	queryHostname string
	queryPretty   bool
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Dump raw data structures from one or more services as JSON",
	Long: `Query live data from any combination of services and print the raw internal
structures as JSON. Useful for debugging sync issues.

Services: caddy, cloudflare, unbound, adguard  (default: all)

Examples:
  caddy-dns-sync query
  caddy-dns-sync query --services cloudflare
  caddy-dns-sync query --services caddy,cloudflare --hostname openwebui.vookie.net
  caddy-dns-sync query --services unbound,adguard --pretty`,
	RunE: runQuery,
}

func runQuery(cmd *cobra.Command, args []string) error {
	want := map[string]bool{}
	for _, s := range strings.Split(queryServices, ",") {
		want[strings.TrimSpace(strings.ToLower(s))] = true
	}
	all := queryServices == "all"

	runtime, err := runtimeapp.LoadRuntime(runtimeapp.RuntimeOptions{
		CaddyServerIP:     runtimeapp.DefaultCaddyServerIP,
		CaddyServerPort:   runtimeapp.DefaultCaddyServerPort,
		IncludeUnbound:    all || want["unbound"],
		IncludeDNSMasq:    false,
		IncludeAdguard:    all || want["adguard"],
		IncludeCloudflare: all || want["cloudflare"],
	})
	if err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	result := map[string]any{}

	// ── Caddy ──────────────────────────────────────────────────────────────────
	if all || want["caddy"] {
		details, err := runtime.Clients.Caddy.GetHostnameDetails()
		if err != nil {
			result["caddy"] = map[string]any{"error": err.Error()}
		} else if queryHostname != "" {
			if entry, ok := details[queryHostname]; ok {
				result["caddy"] = map[string]any{queryHostname: entry}
			} else {
				result["caddy"] = map[string]any{"error": fmt.Sprintf("%q not found in Caddy", queryHostname)}
			}
		} else {
			result["caddy"] = details
		}
	}

	// ── Cloudflare ─────────────────────────────────────────────────────────────
	if all || want["cloudflare"] {
		if runtime.Clients.Cloudflare == nil {
			result["cloudflare"] = map[string]any{"error": "client not configured"}
		} else {
			tunnels, err := runtime.Clients.Cloudflare.GetAllTunnelsHostnames()
			if err != nil {
				result["cloudflare"] = map[string]any{"error": err.Error()}
			} else if queryHostname != "" {
				if entry, ok := tunnels[queryHostname]; ok {
					result["cloudflare"] = map[string]any{queryHostname: entry}
				} else {
					result["cloudflare"] = map[string]any{"error": fmt.Sprintf("%q not found in Cloudflare", queryHostname)}
				}
			} else {
				result["cloudflare"] = tunnels
			}
		}
	}

	// ── Unbound ────────────────────────────────────────────────────────────────
	if all || want["unbound"] {
		if runtime.Clients.Unbound == nil {
			result["unbound"] = map[string]any{"error": "client not configured"}
		} else {
			overrides, err := runtime.Clients.Unbound.GetOverrides()
			if err != nil {
				result["unbound"] = map[string]any{"error": err.Error()}
			} else if queryHostname != "" {
				var filtered []any
				for _, o := range overrides {
					full := o.Host + "." + o.Domain
					if strings.Contains(full, queryHostname) || strings.Contains(queryHostname, o.Host) {
						filtered = append(filtered, o)
					}
				}
				result["unbound"] = filtered
			} else {
				result["unbound"] = overrides
			}
		}
	}

	// ── AdGuard ────────────────────────────────────────────────────────────────
	if all || want["adguard"] {
		if runtime.Clients.Adguard == nil {
			result["adguard"] = map[string]any{"error": "client not configured"}
		} else {
			rewrites, err := runtime.Clients.Adguard.ListRewrites()
			if err != nil {
				result["adguard"] = map[string]any{"error": err.Error()}
			} else if queryHostname != "" {
				var filtered []any
				for _, r := range rewrites {
					if strings.Contains(r.Domain, queryHostname) {
						filtered = append(filtered, r)
					}
				}
				result["adguard"] = filtered
			} else {
				result["adguard"] = rewrites
			}
		}
	}

	var out []byte
	if queryPretty {
		out, err = json.MarshalIndent(result, "", "  ")
	} else {
		out, err = json.Marshal(result)
	}
	if err != nil {
		return fmt.Errorf("error marshaling output: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(out))
	return nil
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.Flags().StringVar(&queryServices, "services", "all", "Comma-separated list of services to query: caddy, cloudflare, unbound, adguard")
	queryCmd.Flags().StringVar(&queryHostname, "hostname", "", "Filter results to a specific hostname")
	queryCmd.Flags().BoolVar(&queryPretty, "pretty", false, "Pretty-print JSON output")
}
