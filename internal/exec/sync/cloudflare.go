package sync

import (
	"fmt"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/logging"
)

// CloudflareSyncOptions contains options for the Cloudflare sync operation
type CloudflareSyncOptions struct {
	DryRun             bool
	TunnelID           string
	EntryDescription   string
	LegacyDescriptions []string
	Verbose            bool
}

// SyncCloudflareWithUnbound synchronizes DNS entries between Cloudflare tunnel and Unbound
func SyncCloudflareWithUnbound(
	unboundClient *api.Client,
	cfClient *api.CloudflareClient,
	options CloudflareSyncOptions,
) (*SyncResult, error) {
	// Fetch hostname map from Cloudflare tunnel
	hostnameMap, err := cfClient.GetTunnelHostnames()
	if err != nil {
		logging.Error("Error fetching Cloudflare tunnel hostnames", "error", err)
		return nil, fmt.Errorf("error fetching Cloudflare tunnel hostnames: %w", err)
	}

	if len(hostnameMap) == 0 {
		logging.Warn("No hostnames found in Cloudflare tunnel config")
		return &SyncResult{HostnameMap: hostnameMap}, nil
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
		// Create a key from host and domain for easier comparison
		key := fmt.Sprintf("%s.%s", override.Host, override.Domain)

		// Check if this was created by sync command or has one of the legacy descriptions
		if override.Description == options.EntryDescription ||
			isLegacyDescription(override.Description, options.LegacyDescriptions) {
			syncCreatedOverrides[key] = override
		} else {
			otherOverrides[key] = override
		}
	}

	// Process each hostname from Cloudflare
	var toAdd, toUpdate, toUpdateDesc []string

	for hostname, serverIP := range hostnameMap {
		// Skip if hostname doesn't contain a dot (not a FQDN)
		if !strings.Contains(hostname, ".") {
			continue
		}

		// Split hostname into host and domain parts
		parts := strings.SplitN(hostname, ".", 2)
		if len(parts) != 2 {
			logging.Warn("Skipping invalid hostname", "hostname", hostname)
			continue
		}

		// Check if this hostname already exists in Unbound
		override, existsInSync := syncCreatedOverrides[hostname]
		_, existsOther := otherOverrides[hostname]

		if !existsInSync && !existsOther {
			// Need to add this hostname
			toAdd = append(toAdd, hostname)
		} else if !existsInSync && existsOther {
			// Exists but not created by sync - leave it alone
			if options.Verbose {
				logging.Info("Hostname already exists (not created by sync)", "hostname", hostname)
			}
		} else if existsInSync {
			// Created by sync, check if it needs updating
			needsUpdate := false
			needsDescUpdate := false

			// Check if server IP needs updating
			if override.Server != serverIP {
				needsUpdate = true
			}

			// Check if description needs updating
			if override.Description != options.EntryDescription {
				needsDescUpdate = true
			}

			if needsUpdate {
				toUpdate = append(toUpdate, hostname)
			} else if needsDescUpdate {
				toUpdateDesc = append(toUpdateDesc, hostname)
			}
		}
	}

	// Find entries to remove (in sync but not in Cloudflare)
	var toRemove []string
	for hostname := range syncCreatedOverrides {
		if _, exists := hostnameMap[hostname]; !exists {
			toRemove = append(toRemove, hostname)
		}
	}

	// If not a dry run, perform the actual changes
	changesApplied := false
	if !options.DryRun {
		changesApplied = applyCloudflareChanges(
			unboundClient,
			options,
			hostnameMap,
			syncCreatedOverrides,
			toAdd,
			toUpdate,
			toUpdateDesc,
			toRemove,
		)
	}

	return &SyncResult{
		HostnameMap:    hostnameMap,
		ToAdd:          toAdd,
		ToUpdate:       toUpdate,
		ToUpdateDesc:   toUpdateDesc,
		ToRemove:       toRemove,
		ChangesApplied: changesApplied,
		SyncOverrides:  syncCreatedOverrides,
		OtherOverrides: otherOverrides,
		ExistingCount:  len(existingOverrides),
	}, nil
}

// applyCloudflareChanges applies the changes to the Unbound DNS server
func applyCloudflareChanges(
	client *api.Client,
	options CloudflareSyncOptions,
	hostnameMap map[string]string,
	syncCreatedOverrides map[string]api.DNSOverride,
	toAdd, toUpdate, toUpdateDesc, toRemove []string,
) bool {
	changesApplied := false

	// Add new entries
	for _, hostname := range toAdd {
		parts := strings.SplitN(hostname, ".", 2)
		host, domain := parts[0], parts[1]
		serverIP := hostnameMap[hostname]

		logging.Info("Adding DNS override", "host", host, "domain", domain, "ip", serverIP)

		override := api.DNSOverride{
			Enabled:     "1",
			Host:        host,
			Domain:      domain,
			Server:      serverIP,
			Description: options.EntryDescription,
		}

		_, err := client.AddOverride(override)
		if err != nil {
			logging.Error(
				"Failed to add DNS override",
				"error",
				err,
				"host",
				host,
				"domain",
				domain,
			)
			continue
		}

		changesApplied = true
	}

	// Update existing entries (IP changes)
	for _, hostname := range toUpdate {
		override := syncCreatedOverrides[hostname]
		serverIP := hostnameMap[hostname]

		logging.Info("Updating DNS override IP",
			"host", override.Host,
			"domain", override.Domain,
			"old_ip", override.Server,
			"new_ip", serverIP)

		override.Server = serverIP
		override.Description = options.EntryDescription // Also update description

		err := client.UpdateOverride(override)
		if err != nil {
			logging.Error(
				"Failed to update DNS override",
				"error",
				err,
				"host",
				override.Host,
				"domain",
				override.Domain,
			)
			continue
		}

		changesApplied = true
	}

	// Update existing entries (description only)
	for _, hostname := range toUpdateDesc {
		override := syncCreatedOverrides[hostname]

		logging.Info("Updating DNS override description",
			"host", override.Host,
			"domain", override.Domain,
			"old_desc", override.Description,
			"new_desc", options.EntryDescription)

		override.Description = options.EntryDescription

		err := client.UpdateOverride(override)
		if err != nil {
			logging.Error(
				"Failed to update DNS override description",
				"error",
				err,
				"host",
				override.Host,
				"domain",
				override.Domain,
			)
			continue
		}

		changesApplied = true
	}

	// Remove stale entries
	for _, hostname := range toRemove {
		override := syncCreatedOverrides[hostname]

		logging.Info("Removing DNS override",
			"host", override.Host,
			"domain", override.Domain,
			"ip", override.Server)

		err := client.DeleteOverride(override.UUID)
		if err != nil {
			logging.Error(
				"Failed to remove DNS override",
				"error",
				err,
				"host",
				override.Host,
				"domain",
				override.Domain,
			)
			continue
		}

		changesApplied = true
	}

	// Apply changes if needed
	if changesApplied {
		logging.Info("Applying changes to Unbound")
		err := client.ApplyChanges()
		if err != nil {
			logging.Error("Failed to apply changes", "error", err)
			return false
		}
		logging.Info("Changes applied successfully")
	} else {
		logging.Info("No changes were needed - everything is in sync")
	}

	return changesApplied
}
