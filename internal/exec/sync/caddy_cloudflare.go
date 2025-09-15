package sync

import (
	"fmt"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/logging"
)

// CaddyCloudflareSyncOptions contains options for the Caddy-Cloudflare sync operation
type CaddyCloudflareSyncOptions struct {
	DryRun             bool
	CaddyServerIP      string
	CaddyServerPort    int
	EntryDescription   string
	LegacyDescriptions []string
	DirectSubdomain    string
	CaddySubdomain     string
	SyncDirect         bool
	SyncCaddy          bool
	Verbose            bool
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
	syncCreatedOverrides := make(map[string]api.DNSOverride)
	otherOverrides := make(map[string]api.DNSOverride)

	for _, override := range existingOverrides {
		key := fmt.Sprintf("%s.%s", override.Host, override.Domain)
		if override.Description == options.EntryDescription ||
			isLegacyDescription(override.Description, options.LegacyDescriptions) {
			syncCreatedOverrides[key] = override
		} else {
			otherOverrides[key] = override
		}
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

		// Create direct access hostname (service.dev.vookie.net)
		if options.SyncDirect {
			directHostname := fmt.Sprintf("%s.%s.vookie.net", serviceName, options.DirectSubdomain)
			directEntries[directHostname] = serviceIP
		}

		// Create Caddy proxy hostname (service.caddy.vookie.net)
		if options.SyncCaddy {
			caddyHostname := fmt.Sprintf("%s.%s.vookie.net", serviceName, options.CaddySubdomain)
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
			delete(syncCreatedOverrides, key)
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
			delete(syncCreatedOverrides, key)
		} else {
			toAdd = append(toAdd, entry)
		}
	}

	// Any remaining sync overrides should be removed
	for _, override := range syncCreatedOverrides {
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

	// Apply changes
	changesApplied := true

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
			changesApplied = false
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
				changesApplied = false
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
			changesApplied = false
		} else {
			logging.Debug("Removed DNS override",
				"uuid", override.UUID,
				"hostname", override.Host,
				"domain", override.Domain)
		}
	}

	result.ChangesApplied = changesApplied
	return result, nil
}

// parseHostnameToDNSEntry converts a hostname to a CloudflareEntry
func parseHostnameToDNSEntry(hostname, ip, description, mode string) CloudflareEntry {
	// Split hostname into host and domain parts
	parts := strings.SplitN(hostname, ".", 2)
	host := parts[0]
	domain := "local" // default

	if len(parts) > 1 {
		domain = parts[1]
	}

	return CloudflareEntry{
		Hostname:    host,
		Domain:      domain,
		IP:          ip,
		Description: description,
		Mode:        mode,
	}
}
