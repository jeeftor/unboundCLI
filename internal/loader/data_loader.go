package loader

import (
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
)

// SyncDataLoader loads and compares data from all services
type SyncDataLoader struct {
	caddyClient   *api.CaddyClient
	unboundClient *api.Client
	adguardClient *api.AdguardClient
	dnsmasqClient *api.DNSMasqClient
}

// NewSyncDataLoader creates a new sync data loader
func NewSyncDataLoader(
	caddyClient *api.CaddyClient,
	unboundClient *api.Client,
	adguardClient *api.AdguardClient,
	dnsmasqClient *api.DNSMasqClient,
) *SyncDataLoader {
	return &SyncDataLoader{
		caddyClient:   caddyClient,
		unboundClient: unboundClient,
		adguardClient: adguardClient,
		dnsmasqClient: dnsmasqClient,
	}
}

// SyncData contains loaded data from all services
type SyncData struct {
	CaddyHostnames   map[string]string
	UnboundOverrides []api.DNSOverride
	AdguardRewrites  []api.Rewrite
	DHCPLeases       []api.DNSMasqLease
}

// LoadData fetches data from all services
func (l *SyncDataLoader) LoadData() (*SyncData, error) {
	return l.LoadDataWithProgress(nil)
}

// LoadDataWithProgress fetches data and reports progress via callback
func (l *SyncDataLoader) LoadDataWithProgress(progressCallback func(service string, data *SyncData)) (*SyncData, error) {
	data := &SyncData{
		CaddyHostnames:   make(map[string]string),
		UnboundOverrides: []api.DNSOverride{},
		AdguardRewrites:  []api.Rewrite{},
		DHCPLeases:       []api.DNSMasqLease{},
	}

	// Fetch from Caddy
	if l.caddyClient != nil {
		caddyHostnames, err := l.caddyClient.GetHostnameMap()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Caddy data: %w", err)
		}
		data.CaddyHostnames = caddyHostnames
		if progressCallback != nil {
			progressCallback("caddy", data)
		}
	}

	// Fetch from Unbound
	if l.unboundClient != nil {
		unboundOverrides, err := l.unboundClient.GetOverrides()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Unbound data: %w", err)
		}
		data.UnboundOverrides = unboundOverrides
		if progressCallback != nil {
			progressCallback("unbound", data)
		}
	}

	// Fetch from Adguard (if enabled)
	if l.adguardClient != nil {
		adguardRewrites, err := l.adguardClient.ListRewrites()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Adguard data: %w", err)
		}
		data.AdguardRewrites = adguardRewrites
		if progressCallback != nil {
			progressCallback("adguard", data)
		}
	}

	// Fetch from DNSMasq (if available)
	if l.dnsmasqClient != nil {
		dhcpLeases, err := l.dnsmasqClient.GetLeases()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch DHCP data: %w", err)
		}
		data.DHCPLeases = dhcpLeases
		if progressCallback != nil {
			progressCallback("dhcp", data)
		}
	}

	return data, nil
}
