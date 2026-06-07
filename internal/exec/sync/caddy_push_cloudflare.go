package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
	"github.com/jeeftor/caddy-dns-sync/internal/syncplan"
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
	TunnelUpdated  []string          // updated in default tunnel
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

	// 2. Fetch all tunnels' ingress details (read-only account scan)
	allCFHosts, err := cfClient.GetAllTunnelsDetails()
	if err != nil {
		return nil, fmt.Errorf("error scanning account tunnels: %w", err)
	}

	entries := cloudflareSyncEntries(caddyHosts, allCFHosts)
	plan := syncplan.BuildPlan(entries, syncplan.Options{
		Service:           "cloudflare",
		CaddyServiceURL:   options.CaddyServiceURL,
		IncludeCloudflare: true,
	})
	for _, action := range plan.Actions {
		switch action.Type {
		case "add":
			result.TunnelAdded = append(result.TunnelAdded, action.Hostname)
		case "update":
			result.TunnelUpdated = append(result.TunnelUpdated, action.Hostname)
		case "delete":
			result.TunnelRemoved = append(result.TunnelRemoved, action.Hostname)
		}
	}
	for hostname, entry := range allCFHosts {
		_, inCaddy := caddyHosts[hostname]
		if entry.IsDefaultTunnel {
			continue
		}
		if inCaddy {
			result.AlreadyCovered = append(result.AlreadyCovered, hostname)
			logging.Info("Hostname already covered by another tunnel",
				"hostname", hostname,
				"tunnel", entry.TunnelName)
		} else {
			result.StaleElsewhere[hostname] = entry.TunnelName
		}
	}

	if options.Verbose {
		logging.Info("Sync diff computed",
			"toAdd", len(result.TunnelAdded),
			"toUpdate", len(result.TunnelUpdated),
			"toRemove", len(result.TunnelRemoved),
			"alreadyCovered", len(result.AlreadyCovered),
			"staleElsewhere", len(result.StaleElsewhere),
		)
	}

	if !options.DryRun {
		applyResult := syncplan.Apply(context.Background(), syncplan.Clients{Cloudflare: cfClient}, plan, syncplan.ApplyOptions{})
		if !applyResult.Success {
			return result, fmt.Errorf("error updating Cloudflare tunnel: %s", strings.Join(applyResult.Errors, "; "))
		}
		for _, h := range result.TunnelAdded {
			result.DNSAdded = append(result.DNSAdded, h)
		}
		for _, h := range result.TunnelRemoved {
			result.DNSRemoved = append(result.DNSRemoved, h)
		}
	}

	return result, nil
}

func cloudflareSyncEntries(
	caddyHosts map[string]string,
	cfHosts map[string]api.CloudflareIngressEntry,
) []*models.Entry {
	entries := make([]*models.Entry, 0, len(caddyHosts)+len(cfHosts))
	seen := make(map[string]bool, len(caddyHosts)+len(cfHosts))

	for hostname, upstream := range caddyHosts {
		entry := &models.Entry{
			Hostname:      hostname,
			CaddyUpstream: upstream,
		}
		if cfEntry, ok := cfHosts[hostname]; ok {
			entry.CloudflareStatus = cloudflareStatusFromIngress(cfEntry)
		}
		entries = append(entries, entry)
		seen[hostname] = true
	}

	for hostname, cfEntry := range cfHosts {
		if seen[hostname] {
			continue
		}
		entries = append(entries, &models.Entry{
			Hostname:         hostname,
			CloudflareStatus: cloudflareStatusFromIngress(cfEntry),
		})
	}

	return entries
}

func cloudflareStatusFromIngress(cfEntry api.CloudflareIngressEntry) models.CloudflareStatus {
	return models.CloudflareStatus{
		Configured:      true,
		TunnelName:      cfEntry.TunnelName,
		TunnelID:        cfEntry.TunnelID,
		Service:         cfEntry.Service,
		Path:            cfEntry.Path,
		IsDefaultTunnel: cfEntry.IsDefaultTunnel,
		HTTPHostHeader:  cfEntry.HTTPHostHeader,
		NoTLSVerify:     cfEntry.NoTLSVerify,
		Http2Origin:     cfEntry.Http2Origin,
		HasAccessPolicy: cfEntry.HasAccessPolicy,
	}
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
