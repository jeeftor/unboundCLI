package sync

import (
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
)

// CaddySyncOptions contains options for the Caddy sync operation
type CaddySyncOptions struct {
	BaseSyncOptions
}

// SyncResult contains the results of the sync operation
type SyncResult struct {
	HostnameMap     map[string]string
	ToAdd           []string
	ToUpdate        []string
	ToUpdateDesc    []string
	ToRemove        []string
	ChangesApplied  bool
	SyncOverrides   map[string]api.DNSOverride
	OtherOverrides  map[string]api.DNSOverride
	ExistingCount   int
	FailedHostnames []string // hostnames that failed during apply
	ApplyFailed     bool     // true if ApplyChanges (service restart) failed
}

// SyncCaddyWithUnbound synchronizes DNS entries between Caddy and Unbound
func SyncCaddyWithUnbound(
	unboundClient *api.Client,
	options CaddySyncOptions,
) (*SyncResult, error) {
	caddyClient := api.NewCaddyClient(options.CaddyServerIP, options.CaddyServerPort)

	// Fetch hostname map from Caddy
	hostnameMap, err := caddyClient.GetHostnameMap()
	if err != nil {
		logging.Error("Error fetching Caddy hostnames", "error", err)
		return nil, fmt.Errorf("error fetching Caddy hostnames: %w", err)
	}

	return syncHostnamesWithUnbound(unboundClient, hostnameMap, unboundSyncOptions{
		Source:             "Caddy config",
		EntryDescription:   options.EntryDescription,
		LegacyDescriptions: options.LegacyDescriptions,
		DryRun:             options.DryRun,
		Verbose:            options.Verbose,
	})
}

type unboundSyncOptions struct {
	Source             string
	EntryDescription   string
	LegacyDescriptions []string
	DryRun             bool
	Verbose            bool
}

// syncHostnamesWithUnbound plans and optionally applies an Unbound sync from a hostname map.
func syncHostnamesWithUnbound(
	unboundClient *api.Client,
	hostnameMap map[string]string,
	options unboundSyncOptions,
) (*SyncResult, error) {
	if len(hostnameMap) == 0 {
		logging.Warn("No hostnames found", "source", options.Source)
		return &SyncResult{HostnameMap: hostnameMap}, nil
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

	// Process each hostname from the source.
	var toAdd, toUpdate, toUpdateDesc []string

	for hostname, serverIP := range hostnameMap {
		// Skip if hostname doesn't contain a dot (not a FQDN)
		if !IsFQDN(hostname) {
			continue
		}

		// Split hostname into host and domain parts
		_, domain := SplitHostname(hostname)
		if domain == "" {
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

	// Find entries to remove (in sync but no longer in the source).
	var toRemove []string
	for hostname := range syncCreatedOverrides {
		if _, exists := hostnameMap[hostname]; !exists {
			toRemove = append(toRemove, hostname)
		}
	}

	// If not a dry run, perform the actual changes
	changesApplied := false
	var applyFailed bool
	var failedHostnames []string
	if !options.DryRun {
		changesApplied, applyFailed, failedHostnames = applyUnboundChanges(
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
		HostnameMap:     hostnameMap,
		ToAdd:           toAdd,
		ToUpdate:        toUpdate,
		ToUpdateDesc:    toUpdateDesc,
		ToRemove:        toRemove,
		ChangesApplied:  changesApplied,
		SyncOverrides:   syncCreatedOverrides,
		OtherOverrides:  otherOverrides,
		ExistingCount:   len(existingOverrides),
		FailedHostnames: failedHostnames,
		ApplyFailed:     applyFailed,
	}, nil
}

// applyUnboundChanges applies planned changes to the Unbound DNS server.
// Returns (changesApplied, applyFailed, failedHostnames).
func applyUnboundChanges(
	client *api.Client,
	options unboundSyncOptions,
	hostnameMap map[string]string,
	syncCreatedOverrides map[string]api.DNSOverride,
	toAdd, toUpdate, toUpdateDesc, toRemove []string,
) (bool, bool, []string) {
	changesApplied := false
	var failed []string

	// Add new entries
	for _, hostname := range toAdd {
		host, domain := SplitHostname(hostname)
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
			failed = append(failed, hostname)
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
			failed = append(failed, hostname)
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
			failed = append(failed, hostname)
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
			failed = append(failed, hostname)
			continue
		}

		changesApplied = true
	}

	// Apply changes if needed (service restart).
	// Even if some individual operations failed, we still need to apply
	// the ones that succeeded.
	applyFailed := false
	if changesApplied {
		logging.Info("Applying changes to Unbound")
		err := client.ApplyChanges()
		if err != nil {
			logging.Error("Failed to apply changes (service restart)", "error", err)
			applyFailed = true
		} else {
			logging.Info("Changes applied successfully")
		}
	} else {
		logging.Info("No changes were needed - everything is in sync")
	}

	return changesApplied, applyFailed, failed
}
