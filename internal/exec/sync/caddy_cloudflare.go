package sync

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
)

// CaddyCloudflareSyncOptions contains options for the Caddy-Cloudflare sync operation
type CaddyCloudflareSyncOptions struct {
	BaseSyncOptions
	DirectSubdomain string
	CaddySubdomain  string
	SyncDirect      bool
	SyncCaddy       bool
}

// CaddyCloudflareSyncResult contains the result of the Caddy-Cloudflare sync operation
type CaddyCloudflareSyncResult struct {
	HostnameMap    map[string]string
	DirectEntries  map[string]string // hostname -> service IP
	CaddyEntries   map[string]string // hostname -> caddy IP
	ToAdd          []CloudflareEntry
	ToUpdate       []CloudflareEntry
	ToRemove       []api.DNSOverride
	ChangesApplied bool
	SyncOverrides  map[string]api.DNSOverride
	OtherOverrides map[string]api.DNSOverride
}

// CloudflareEntry represents a DNS entry to be created for Cloudflare routing
type CloudflareEntry struct {
	Hostname    string
	Domain      string
	IP          string
	Description string
	Mode        string // "direct" or "caddy"
}

// SyncCaddyWithCloudflare synchronizes DNS entries for dual-mode Cloudflare tunnel routing
func SyncCaddyWithCloudflare(unboundClient *api.Client, options CaddyCloudflareSyncOptions) (*CaddyCloudflareSyncResult, error) {
	// Fetch hostname map from Caddy
	caddyClient := api.NewCaddyClient(options.CaddyServerIP, options.CaddyServerPort)
	hostnameMap, err := caddyClient.GetHostnameMap()
	if err != nil {
		logging.Error("Error fetching Caddy hostnames", "error", err)
		return nil, fmt.Errorf("error fetching Caddy hostnames: %w", err)
	}

	if len(hostnameMap) == 0 {
		logging.Warn("No hostnames found in Caddy config")
		return &CaddyCloudflareSyncResult{HostnameMap: hostnameMap}, nil
	}

	// Get existing overrides from Unbound
	existingOverrides, err := unboundClient.GetOverrides()
	if err != nil {
		logging.Error("Error fetching overrides", "error", err)
		return nil, fmt.Errorf("error fetching overrides: %w", err)
	}

	// Organize overrides for easier processing
	syncCreatedOverrides, otherOverrides := OrganizeOverridesByOwnership(
		existingOverrides, options.EntryDescription, options.LegacyDescriptions,
	)
	remainingSyncOverrides := make(map[string]api.DNSOverride, len(syncCreatedOverrides))
	for key, override := range syncCreatedOverrides {
		remainingSyncOverrides[key] = override
	}

	// Create dual-mode entries for each Caddy hostname
	directEntries := make(map[string]string)
	caddyEntries := make(map[string]string)

	for hostname, serviceIP := range hostnameMap {
		// Extract base service name (remove domain suffix if present)
		serviceName := hostname
		if idx := strings.Index(hostname, "."); idx != -1 {
			serviceName = hostname[:idx]
		}

		// Create direct access hostname (service.dev.example.com)
		if options.SyncDirect {
			directHostname := fmt.Sprintf("%s.%s.example.com", serviceName, options.DirectSubdomain)
			directEntries[directHostname] = serviceIP
		}

		// Create Caddy proxy hostname (service.caddy.example.com)
		if options.SyncCaddy {
			caddyHostname := fmt.Sprintf("%s.%s.example.com", serviceName, options.CaddySubdomain)
			caddyEntries[caddyHostname] = options.CaddyServerIP
		}
	}

	// Determine which entries need to be added, updated, or removed
	var toAdd []CloudflareEntry
	var toUpdate []CloudflareEntry
	var toRemove []api.DNSOverride

	// Check direct entries
	for hostname, ip := range directEntries {
		entry := parseHostnameToDNSEntry(hostname, ip, options.EntryDescription, "direct")
		key := fmt.Sprintf("%s.%s", entry.Hostname, entry.Domain)

		if existing, exists := syncCreatedOverrides[key]; exists {
			if existing.Server != ip {
				toUpdate = append(toUpdate, entry)
			}
			// Remove from sync overrides so we know it's still needed
			delete(remainingSyncOverrides, key)
		} else {
			toAdd = append(toAdd, entry)
		}
	}

	// Check Caddy entries
	for hostname, ip := range caddyEntries {
		entry := parseHostnameToDNSEntry(hostname, ip, options.EntryDescription, "caddy")
		key := fmt.Sprintf("%s.%s", entry.Hostname, entry.Domain)

		if existing, exists := syncCreatedOverrides[key]; exists {
			if existing.Server != ip {
				toUpdate = append(toUpdate, entry)
			}
			// Remove from sync overrides so we know it's still needed
			delete(remainingSyncOverrides, key)
		} else {
			toAdd = append(toAdd, entry)
		}
	}

	// Any remaining sync overrides should be removed
	for _, override := range remainingSyncOverrides {
		toRemove = append(toRemove, override)
	}

	result := &CaddyCloudflareSyncResult{
		HostnameMap:    hostnameMap,
		DirectEntries:  directEntries,
		CaddyEntries:   caddyEntries,
		ToAdd:          toAdd,
		ToUpdate:       toUpdate,
		ToRemove:       toRemove,
		SyncOverrides:  make(map[string]api.DNSOverride),
		OtherOverrides: otherOverrides,
	}

	// If dry run, don't make actual changes
	if options.DryRun {
		return result, nil
	}

	// Stage every provider write before activation. A failed write leaves the
	// provider in an unresolved state, so report it as an error rather than
	// presenting the partial result as a successful sync.
	var writeErrors []error

	// Add new entries
	for _, entry := range toAdd {
		override := api.DNSOverride{
			Enabled:     "1",
			Host:        entry.Hostname,
			Domain:      entry.Domain,
			Server:      entry.IP,
			Description: entry.Description,
		}
		_, err := unboundClient.AddOverride(override)
		if err != nil {
			logging.Error("Error adding DNS override",
				"hostname", entry.Hostname,
				"domain", entry.Domain,
				"ip", entry.IP,
				"error", err)
			writeErrors = append(writeErrors, fmt.Errorf("add %s.%s: %w", entry.Hostname, entry.Domain, err))
		} else {
			logging.Debug("Added DNS override",
				"hostname", entry.Hostname,
				"domain", entry.Domain,
				"ip", entry.IP,
				"mode", entry.Mode)
		}
	}

	// Update existing entries
	for _, entry := range toUpdate {
		key := fmt.Sprintf("%s.%s", entry.Hostname, entry.Domain)
		if existing, exists := syncCreatedOverrides[key]; exists {
			override := api.DNSOverride{
				UUID:        existing.UUID,
				Enabled:     "1",
				Host:        entry.Hostname,
				Domain:      entry.Domain,
				Server:      entry.IP,
				Description: entry.Description,
			}
			err := unboundClient.UpdateOverride(override)
			if err != nil {
				logging.Error("Error updating DNS override",
					"uuid", existing.UUID,
					"hostname", entry.Hostname,
					"domain", entry.Domain,
					"ip", entry.IP,
					"error", err)
				writeErrors = append(writeErrors, fmt.Errorf("update %s.%s: %w", entry.Hostname, entry.Domain, err))
			} else {
				logging.Debug("Updated DNS override",
					"uuid", existing.UUID,
					"hostname", entry.Hostname,
					"domain", entry.Domain,
					"ip", entry.IP,
					"mode", entry.Mode)
			}
		}
	}

	// Remove obsolete entries
	for _, override := range toRemove {
		err := unboundClient.DeleteOverride(override.UUID)
		if err != nil {
			logging.Error("Error removing DNS override",
				"uuid", override.UUID,
				"hostname", override.Host,
				"domain", override.Domain,
				"error", err)
			writeErrors = append(writeErrors, fmt.Errorf("delete %s.%s: %w", override.Host, override.Domain, err))
		} else {
			logging.Debug("Removed DNS override",
				"uuid", override.UUID,
				"hostname", override.Host,
				"domain", override.Domain)
		}
	}

	if len(writeErrors) > 0 {
		return result, fmt.Errorf("sync has unresolved provider writes; review provider state before retrying: %w", errors.Join(writeErrors...))
	}

	if len(toAdd)+len(toUpdate)+len(toRemove) == 0 {
		result.ChangesApplied = true
		return result, nil
	}

	if err := unboundClient.ApplyChanges(); err != nil {
		return result, fmt.Errorf("activate Unbound changes: %w; provider state may be partially updated", err)
	}
	if err := verifyCaddyCloudflareChanges(unboundClient, toAdd, toUpdate, toRemove); err != nil {
		return result, err
	}

	result.ChangesApplied = true
	return result, nil
}

func verifyCaddyCloudflareChanges(client *api.Client, added, updated []CloudflareEntry, removed []api.DNSOverride) error {
	overrides, err := client.GetOverrides()
	if err != nil {
		return fmt.Errorf("read back Unbound changes: %w", err)
	}
	byHostname := make(map[string]api.DNSOverride, len(overrides))
	for _, override := range overrides {
		byHostname[fmt.Sprintf("%s.%s", override.Host, override.Domain)] = override
	}
	for _, entry := range append(added, updated...) {
		key := fmt.Sprintf("%s.%s", entry.Hostname, entry.Domain)
		override, ok := byHostname[key]
		if !ok || override.Server != entry.IP || override.Description != entry.Description {
			return fmt.Errorf("verify Unbound change for %s: expected managed record pointing to %s", key, entry.IP)
		}
	}
	for _, override := range removed {
		if _, ok := byHostname[fmt.Sprintf("%s.%s", override.Host, override.Domain)]; ok {
			return fmt.Errorf("verify Unbound deletion for %s.%s: record remains", override.Host, override.Domain)
		}
	}
	return nil
}

// parseHostnameToDNSEntry converts a hostname to a CloudflareEntry
func parseHostnameToDNSEntry(hostname, ip, description, mode string) CloudflareEntry {
	host, domain := SplitHostname(hostname)
	if domain == "" {
		domain = "local" // default
	}

	return CloudflareEntry{
		Hostname:    host,
		Domain:      domain,
		IP:          ip,
		Description: description,
		Mode:        mode,
	}
}
