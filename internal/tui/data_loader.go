package tui

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

// DataLoader handles loading data from all API clients and building unified Entry models
type DataLoader struct {
	caddyClient   *api.CaddyClient
	unboundClient *api.Client
	adguardClient *api.AdguardClient
	dnsmasqClient *api.DNSMasqClient
	cfClient      *api.CloudflareClient
	caddyServerIP string
}

// WithCloudflareClient sets an optional Cloudflare client for loading CF tunnel data.
// If nil, CF data is skipped and the TUI works without it.
func (d *DataLoader) WithCloudflareClient(c *api.CloudflareClient) {
	d.cfClient = c
}

// NewDataLoader creates a new data loader
func NewDataLoader(
	caddyClient *api.CaddyClient,
	unboundClient *api.Client,
	adguardClient *api.AdguardClient,
	dnsmasqClient *api.DNSMasqClient,
	caddyServerIP string,
) *DataLoader {
	return &DataLoader{
		caddyClient:   caddyClient,
		unboundClient: unboundClient,
		adguardClient: adguardClient,
		dnsmasqClient: dnsmasqClient,
		caddyServerIP: caddyServerIP,
	}
}

// LoadData loads all data from API clients concurrently and builds Entry models.
// Caddy, Unbound, AdGuard, DHCP, and Cloudflare are fetched in parallel.
// DNS resolution for all entries is also parallelized with a worker pool.
func (d *DataLoader) LoadData() ([]*models.Entry, error) {
	// --- Phase 1: parallel API fetches ---
	var (
		caddyHostnames   map[string]models.CaddyRouteInfo
		unboundOverrides map[string]*api.DNSOverride
		adguardRewrites  map[string]*api.Rewrite
		dhcpLeases       map[string]*api.DNSMasqLease
		cfDetails        map[string]api.CloudflareIngressEntry

		caddyErr   error
		unboundErr error
		adguardErr error
		dhcpErr    error
		cfErr      error

		wg sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		logging.Info("Loading Caddy configuration...")
		caddyHostnames, caddyErr = d.loadCaddyHostnames()
		if caddyErr != nil {
			logging.Error("Failed to load Caddy hostnames", "error", caddyErr)
		} else {
			logging.Info("Loaded Caddy hostnames", "count", len(caddyHostnames))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logging.Info("Loading Unbound DNS overrides...")
		unboundOverrides, unboundErr = d.loadUnboundOverrides()
		if unboundErr != nil {
			logging.Warn("Failed to load Unbound overrides", "error", unboundErr)
			unboundOverrides = make(map[string]*api.DNSOverride)
		} else {
			logging.Info("Loaded Unbound overrides", "count", len(unboundOverrides))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if d.adguardClient == nil {
			adguardRewrites = make(map[string]*api.Rewrite)
			return
		}
		logging.Info("Loading AdGuard DNS rewrites...")
		adguardRewrites, adguardErr = d.loadAdguardRewrites()
		if adguardErr != nil {
			logging.Warn("Failed to load AdGuard rewrites", "error", adguardErr)
			adguardRewrites = make(map[string]*api.Rewrite)
		} else {
			logging.Info("Loaded AdGuard rewrites", "count", len(adguardRewrites))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logging.Info("Loading DHCP leases...")
		dhcpLeases, dhcpErr = d.loadDHCPLeases()
		if dhcpErr != nil {
			logging.Warn("Failed to load DHCP leases", "error", dhcpErr)
			dhcpLeases = make(map[string]*api.DNSMasqLease)
		} else {
			logging.Info("Loaded DHCP leases", "count", len(dhcpLeases))
		}
	}()

	if d.cfClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logging.Info("Loading Cloudflare tunnel details...")
			cfDetails, cfErr = d.cfClient.GetAllTunnelsDetails()
			if cfErr != nil {
				logging.Warn("Failed to load Cloudflare tunnel details", "error", cfErr)
			} else {
				logging.Info("Loaded Cloudflare tunnel details", "count", len(cfDetails))
			}
		}()
	}

	wg.Wait()

	if caddyErr != nil {
		return nil, fmt.Errorf("failed to load Caddy hostnames: %w", caddyErr)
	}

	// --- Phase 2: build entry models ---
	logging.Info("Building unified entry models...")
	entries := d.buildEntries(caddyHostnames, unboundOverrides, adguardRewrites, dhcpLeases)
	logging.Info("Built entry models", "count", len(entries))

	// --- Phase 3: parallel DNS resolution ---
	logging.Info("Resolving DNS hostnames in parallel...")
	d.resolveAllDNS(entries)

	// --- Phase 4: enrich with Cloudflare data ---
	if cfDetails != nil {
		hostIndex := make(map[string]int, len(entries))
		for i, e := range entries {
			hostIndex[e.Hostname] = i
		}
		for hostname, cfEntry := range cfDetails {
			cfStatus := models.CloudflareStatus{
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
			if idx, ok := hostIndex[hostname]; ok {
				entries[idx].CloudflareStatus = cfStatus
			} else {
				entries = append(entries, &models.Entry{
					Hostname:         hostname,
					DataSource:       "CloudFlare",
					CloudflareStatus: cfStatus,
				})
			}
		}
	}

	// Recompute overall status now that CF data is merged
	for _, e := range entries {
		e.OverallStatus = models.ComputeSyncStatus(e, d.caddyServerIP)
	}

	return entries, nil
}

// resolveAllDNS resolves DNS for all entries in parallel using a worker pool.
func (d *DataLoader) resolveAllDNS(entries []*models.Entry) {
	const workers = 20
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, e := range entries {
		e := e
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			e.DNSResolved = d.resolveDNS(e.Hostname)
		}()
	}
	wg.Wait()
}

// loadCaddyHostnames loads full route details (hostname -> CaddyRouteInfo) from Caddy.
func (d *DataLoader) loadCaddyHostnames() (map[string]models.CaddyRouteInfo, error) {
	if d.caddyClient == nil {
		return nil, fmt.Errorf("Caddy client not initialized")
	}
	return d.caddyClient.GetHostnameDetails()
}

// loadUnboundOverrides loads DNS overrides from Unbound
func (d *DataLoader) loadUnboundOverrides() (map[string]*api.DNSOverride, error) {
	if d.unboundClient == nil {
		return nil, fmt.Errorf("Unbound client not initialized")
	}

	overrides, err := d.unboundClient.GetOverrides()
	if err != nil {
		return nil, err
	}

	// Index by hostname
	overrideMap := make(map[string]*api.DNSOverride)
	for i := range overrides {
		hostname := overrides[i].Host + "." + overrides[i].Domain
		overrideMap[hostname] = &overrides[i]
	}

	return overrideMap, nil
}

// loadAdguardRewrites loads DNS rewrites from AdGuard
func (d *DataLoader) loadAdguardRewrites() (map[string]*api.Rewrite, error) {
	if d.adguardClient == nil {
		return nil, fmt.Errorf("AdGuard client not initialized")
	}

	rewrites, err := d.adguardClient.ListRewrites()
	if err != nil {
		return nil, err
	}

	// Index by domain
	rewriteMap := make(map[string]*api.Rewrite)
	for i := range rewrites {
		rewriteMap[rewrites[i].Domain] = &rewrites[i]
	}

	return rewriteMap, nil
}

// loadDHCPLeases loads DHCP leases from DNSMasq.
// Returns a map keyed by BOTH short hostname (when present) AND IP address so
// that static reservations without a hostname can still be matched by IP.
func (d *DataLoader) loadDHCPLeases() (map[string]*api.DNSMasqLease, error) {
	if d.dnsmasqClient == nil {
		return nil, fmt.Errorf("DNSMasq client not initialized")
	}

	leases, err := d.dnsmasqClient.GetLeases()
	if err != nil {
		return nil, err
	}

	leaseMap := make(map[string]*api.DNSMasqLease)
	for i := range leases {
		// Index by short hostname (primary lookup path)
		if leases[i].Hostname != "" {
			leaseMap[leases[i].Hostname] = &leases[i]
		}
		// Also index by IP address so we can fall back when hostname is empty
		// (common for static DHCP reservations in OPNSense)
		if leases[i].IPAddress != "" {
			leaseMap[leases[i].IPAddress] = &leases[i]
		}
	}

	return leaseMap, nil
}

// buildEntries builds unified Entry models from all data sources
func (d *DataLoader) buildEntries(
	caddyHostnames map[string]models.CaddyRouteInfo,
	unboundOverrides map[string]*api.DNSOverride,
	adguardRewrites map[string]*api.Rewrite,
	dhcpLeases map[string]*api.DNSMasqLease,
) []*models.Entry {
	// Collect all unique hostnames from all sources
	hostnameSet := make(map[string]bool)

	// Add Caddy hostnames
	for hostname := range caddyHostnames {
		hostnameSet[hostname] = true
	}

	// Add Unbound hostnames
	for hostname := range unboundOverrides {
		hostnameSet[hostname] = true
	}

	// Add AdGuard hostnames
	for hostname := range adguardRewrites {
		hostnameSet[hostname] = true
	}

	// Build entries
	entries := make([]*models.Entry, 0, len(hostnameSet))

	for hostname := range hostnameSet {
		entry := d.buildEntry(hostname, caddyHostnames, unboundOverrides, adguardRewrites, dhcpLeases)
		entries = append(entries, entry)
	}

	return entries
}

// buildEntry builds a single Entry model for a hostname
func (d *DataLoader) buildEntry(
	hostname string,
	caddyHostnames map[string]models.CaddyRouteInfo,
	unboundOverrides map[string]*api.DNSOverride,
	adguardRewrites map[string]*api.Rewrite,
	dhcpLeases map[string]*api.DNSMasqLease,
) *models.Entry {
	entry := &models.Entry{
		Hostname: hostname,
	}

	// Caddy data (source of truth)
	if routeInfo, exists := caddyHostnames[hostname]; exists {
		entry.CaddyUpstream = routeInfo.Upstream
		entry.CaddyRoute = routeInfo
		entry.DataSource = "Caddy"

		// Extract IP and port from upstream
		// Upstream format: "192.168.1.112:8096" or "192.168.1.112"
		if parts := strings.Split(routeInfo.Upstream, ":"); len(parts) >= 1 {
			entry.CaddyIP = parts[0]
			if len(parts) == 2 {
				entry.CaddyPort = parts[1]
			}
		}
	}

	// Unbound data
	if override, exists := unboundOverrides[hostname]; exists {
		configured := override.Server != ""
		inSync := configured && override.Server == d.caddyServerIP
		entry.UnboundStatus = models.NewServiceStatus(configured, override.Server, inSync)

		if entry.DataSource == "" {
			entry.DataSource = "Unbound"
		}
	} else {
		entry.UnboundStatus = models.NotConfigured()
	}

	// AdGuard data
	if rewrite, exists := adguardRewrites[hostname]; exists {
		configured := rewrite.Answer != ""
		inSync := configured && rewrite.Answer == d.caddyServerIP
		entry.AdguardStatus = models.NewServiceStatus(configured, rewrite.Answer, inSync)

		if entry.DataSource == "" {
			entry.DataSource = "AdGuard"
		}
	} else {
		entry.AdguardStatus = models.NotConfigured()
	}

	// DHCP data
	// Try to find DHCP lease by: (1) short hostname, (2) IP from Caddy upstream.
	// Static reservations in OPNSense often have no hostname, only an IP.
	baseHostname := hostname
	if dotIdx := strings.Index(hostname, "."); dotIdx > 0 {
		baseHostname = hostname[:dotIdx]
	}
	var dhcpLease *api.DNSMasqLease
	if l, ok := dhcpLeases[baseHostname]; ok {
		dhcpLease = l
	} else if entry.CaddyIP != "" {
		dhcpLease = dhcpLeases[entry.CaddyIP] // may be nil — that's fine
	}

	if lease := dhcpLease; lease != nil {
		leaseType := "dynamic"
		if lease.Type == "static" {
			leaseType = "static"
		}

		inSync := lease.IPAddress == entry.CaddyIP
		entry.DHCPStatus = models.NewDHCPStatus(
			true,
			leaseType,
			lease.IPAddress,
			lease.MACAddress,
			lease.Hostname,
			inSync,
		)

		if entry.DataSource == "" {
			entry.DataSource = "DHCP"
		}
	} else {
		entry.DHCPStatus = models.NoDHCP()
	}

	// Compute overall sync status (DNS resolution happens separately in parallel)
	entry.OverallStatus = models.ComputeSyncStatus(entry, d.caddyServerIP)

	return entry
}

// resolveDNS performs a DNS lookup for the hostname and returns the first IP address
func (d *DataLoader) resolveDNS(hostname string) string {
	// Perform DNS lookup
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		// DNS resolution failed
		return "FAIL"
	}

	if len(addrs) == 0 {
		// No addresses returned
		return "NONE"
	}

	// Return the first IP address
	return addrs[0]
}
