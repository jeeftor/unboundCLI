package sync

import (
	"fmt"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
)

// CaddyToCloudflareSyncOptions contains options for the Caddy-to-Cloudflare push sync.
type CaddyToCloudflareSyncOptions struct {
	DryRun          bool
	CaddyServiceURL string   // target service URL for new ingress rules, e.g. "http://192.168.1.15:80"
	HostFilter      []string // optional: only sync hostnames matching these domain suffixes
	Verbose         bool
}

// CaddyToCloudflareSyncResult holds the outcome of a Caddy-to-Cloudflare push sync.
type CaddyToCloudflareSyncResult struct {
	CaddyHostnames []string
	TunnelAdded    []string          // added to default tunnel
	TunnelRemoved  []string          // removed from default tunnel
	AlreadyCovered []string          // found in other tunnels, skipped
	StaleElsewhere map[string]string // hostname → tunnelName, in another tunnel but not in Caddy (report only)
	DNSAdded       []string
	DNSRemoved     []string
	DryRun         bool
}

// SyncCaddyToCloudflare reads hostnames from Caddy and synchronizes them into the
// default Cloudflare tunnel (cfClient.tunnelID), while treating all other account
// tunnels as read-only. DNS CNAME records are created/removed to match.
func SyncCaddyToCloudflare(
	caddyClient *api.CaddyClient,
	cfClient *api.CloudflareClient,
	options CaddyToCloudflareSyncOptions,
) (*CaddyToCloudflareSyncResult, error) {
	result := &CaddyToCloudflareSyncResult{
		DryRun:         options.DryRun,
		StaleElsewhere: make(map[string]string),
	}

	// 1. Fetch hostnames from Caddy
	caddyHosts, err := caddyClient.GetHostnameMap()
	if err != nil {
		return nil, fmt.Errorf("error fetching Caddy hostnames: %w", err)
	}

	// Apply host filter if set
	if len(options.HostFilter) > 0 {
		filtered := make(map[string]string, len(caddyHosts))
		for hostname, svc := range caddyHosts {
			if matchesFilter(hostname, options.HostFilter) {
				filtered[hostname] = svc
			} else if options.Verbose {
				logging.Info("Skipping hostname (does not match filter)", "hostname", hostname)
			}
		}
		caddyHosts = filtered
	}

	for h := range caddyHosts {
		result.CaddyHostnames = append(result.CaddyHostnames, h)
	}

	// 2. Fetch all tunnels' hostnames (read-only scan)
	allCFHosts, err := cfClient.GetAllTunnelsHostnames()
	if err != nil {
		return nil, fmt.Errorf("error scanning account tunnels: %w", err)
	}

	// 3. Fetch default tunnel's hostnames (writeable)
	defaultHosts, err := cfClient.GetTunnelHostnames()
	if err != nil {
		return nil, fmt.Errorf("error fetching default tunnel hostnames: %w", err)
	}

	// 4. Compute diff
	// toAdd: in Caddy but not anywhere in CF at all
	// alreadyCovered: in Caddy AND in another (non-default) tunnel
	// toRemove: in default tunnel but not in Caddy
	// staleElsewhere: in another tunnel AND not in Caddy (report only)

	for hostname := range caddyHosts {
		entry, inAllCF := allCFHosts[hostname]
		_, inDefault := defaultHosts[hostname]

		if !inAllCF {
			result.TunnelAdded = append(result.TunnelAdded, hostname)
		} else if !inDefault {
			// Present in another tunnel — skip, report only
			result.AlreadyCovered = append(result.AlreadyCovered, hostname)
			logging.Info("Hostname already covered by another tunnel",
				"hostname", hostname,
				"tunnel", entry.TunnelName)
		}
		// If inDefault: already correct, nothing to do
	}

	for hostname := range defaultHosts {
		if _, inCaddy := caddyHosts[hostname]; !inCaddy {
			result.TunnelRemoved = append(result.TunnelRemoved, hostname)
		}
	}

	for hostname, entry := range allCFHosts {
		_, inDefault := defaultHosts[hostname]
		_, inCaddy := caddyHosts[hostname]
		if !inDefault && !inCaddy {
			result.StaleElsewhere[hostname] = entry.TunnelName
		}
	}

	if options.Verbose {
		logging.Info("Sync diff computed",
			"toAdd", len(result.TunnelAdded),
			"toRemove", len(result.TunnelRemoved),
			"alreadyCovered", len(result.AlreadyCovered),
			"staleElsewhere", len(result.StaleElsewhere),
		)
	}

	// 5. Apply (when not dry-run)
	if !options.DryRun {
		// Build new ingress map: defaultHosts - toRemove + toAdd
		newIngress := make(map[string]string, len(defaultHosts))
		for h, svc := range defaultHosts {
			newIngress[h] = svc
		}
		for _, h := range result.TunnelRemoved {
			delete(newIngress, h)
		}
		for _, h := range result.TunnelAdded {
			newIngress[h] = options.CaddyServiceURL
		}

		if err := cfClient.SetTunnelIngress(newIngress); err != nil {
			return result, fmt.Errorf("error updating tunnel ingress: %w", err)
		}

		for _, h := range result.TunnelAdded {
			if err := cfClient.EnsureDNSRecord(h); err != nil {
				logging.Error("Failed to create DNS record", "hostname", h, "error", err)
				continue
			}
			result.DNSAdded = append(result.DNSAdded, h)
		}

		for _, h := range result.TunnelRemoved {
			if err := cfClient.DeleteDNSRecord(h); err != nil {
				logging.Error("Failed to delete DNS record", "hostname", h, "error", err)
				continue
			}
			result.DNSRemoved = append(result.DNSRemoved, h)
		}
	}

	return result, nil
}

// matchesFilter returns true if hostname ends with any of the given domain suffixes.
func matchesFilter(hostname string, filters []string) bool {
	for _, suffix := range filters {
		if strings.HasSuffix(hostname, suffix) {
			return true
		}
	}
	return false
}
