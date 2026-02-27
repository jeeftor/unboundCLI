package tui

import (
	"fmt"
	"net"
	"strings"

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
	caddyServerIP string
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

// LoadData loads all data from API clients and builds Entry models
func (d *DataLoader) LoadData() ([]*models.Entry, error) {
	// 1. Load Caddy hostnames (source of truth)
	logging.Info("Loading Caddy configuration...")
	caddyHostnames, err := d.loadCaddyHostnames()
	if err != nil {
		logging.Error("Failed to load Caddy hostnames", "error", err)
		return nil, fmt.Errorf("failed to load Caddy hostnames: %w", err)
	}
	logging.Info("Loaded Caddy hostnames", "count", len(caddyHostnames))

	// 2. Load Unbound DNS overrides
	logging.Info("Loading Unbound DNS overrides...")
	unboundOverrides, err := d.loadUnboundOverrides()
	if err != nil {
		logging.Error("Failed to load Unbound overrides", "error", err)
		// Don't fail completely - continue without Unbound data
		unboundOverrides = make(map[string]*api.DNSOverride)
	}
	logging.Info("Loaded Unbound overrides", "count", len(unboundOverrides))

	// 3. Load AdGuard DNS rewrites
	var adguardRewrites map[string]*api.Rewrite
	if d.adguardClient != nil {
		logging.Info("Loading AdGuard DNS rewrites...")
		adguardRewrites, err = d.loadAdguardRewrites()
		if err != nil {
			logging.Error("Failed to load AdGuard rewrites", "error", err)
			// Don't fail completely - continue without AdGuard data
			adguardRewrites = make(map[string]*api.Rewrite)
		}
		logging.Info("Loaded AdGuard rewrites", "count", len(adguardRewrites))
	} else {
		adguardRewrites = make(map[string]*api.Rewrite)
	}

	// 4. Load DHCP leases
	logging.Info("Loading DHCP leases...")
	dhcpLeases, err := d.loadDHCPLeases()
	if err != nil {
		logging.Error("Failed to load DHCP leases", "error", err)
		// Don't fail completely - continue without DHCP data
		dhcpLeases = make(map[string]*api.DNSMasqLease)
	}
	logging.Info("Loaded DHCP leases", "count", len(dhcpLeases))

	// 5. Build unified Entry models
	logging.Info("Building unified entry models...")
	entries := d.buildEntries(caddyHostnames, unboundOverrides, adguardRewrites, dhcpLeases)
	logging.Info("Built entry models", "count", len(entries))

	return entries, nil
}

// loadCaddyHostnames loads hostname -> upstream mappings from Caddy
func (d *DataLoader) loadCaddyHostnames() (map[string]string, error) {
	if d.caddyClient == nil {
		return nil, fmt.Errorf("Caddy client not initialized")
	}

	hostnameMap, err := d.caddyClient.GetHostnameMap()
	if err != nil {
		return nil, err
	}

	return hostnameMap, nil
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

// loadDHCPLeases loads DHCP leases from DNSMasq
func (d *DataLoader) loadDHCPLeases() (map[string]*api.DNSMasqLease, error) {
	if d.dnsmasqClient == nil {
		return nil, fmt.Errorf("DNSMasq client not initialized")
	}

	leases, err := d.dnsmasqClient.GetLeases()
	if err != nil {
		return nil, err
	}

	// Index by hostname
	leaseMap := make(map[string]*api.DNSMasqLease)
	for i := range leases {
		if leases[i].Hostname != "" {
			leaseMap[leases[i].Hostname] = &leases[i]
		}
	}

	return leaseMap, nil
}

// buildEntries builds unified Entry models from all data sources
func (d *DataLoader) buildEntries(
	caddyHostnames map[string]string,
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
	caddyHostnames map[string]string,
	unboundOverrides map[string]*api.DNSOverride,
	adguardRewrites map[string]*api.Rewrite,
	dhcpLeases map[string]*api.DNSMasqLease,
) *models.Entry {
	entry := &models.Entry{
		Hostname: hostname,
	}

	// Caddy data (source of truth)
	if upstream, exists := caddyHostnames[hostname]; exists {
		entry.CaddyUpstream = upstream
		entry.DataSource = "Caddy"

		// Extract IP and port from upstream
		// Upstream format: "192.168.1.112:8096" or "192.168.1.112"
		if parts := strings.Split(upstream, ":"); len(parts) >= 1 {
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
	// Try to find DHCP lease by extracting base hostname (without domain)
	baseHostname := hostname
	if dotIdx := strings.Index(hostname, "."); dotIdx > 0 {
		baseHostname = hostname[:dotIdx]
	}

	if lease, exists := dhcpLeases[baseHostname]; exists {
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

	// Resolve DNS to see what the hostname actually resolves to
	entry.DNSResolved = d.resolveDNS(hostname)

	// Compute overall sync status
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
