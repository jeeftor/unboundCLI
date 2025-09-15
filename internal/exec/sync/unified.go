package sync

import (
	"fmt"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/logging"
)

// CommonSyncOptions contains common options for unified sync operations
type CommonSyncOptions struct {
	DryRun             bool
	CaddyServerIP      string
	CaddyServerPort    int
	EntryDescription   string
	LegacyDescriptions []string
	Verbose            bool
}

// UnifiedSyncResult contains the results of syncing to both systems
type UnifiedSyncResult struct {
	HostnameMap     map[string]string
	UnboundResult   *SyncResult
	AdguardResult   *AdguardSyncResult
	SyncedToUnbound bool
	SyncedToAdguard bool
	UnboundError    error
	AdguardError    error
}

// UnifiedCaddySync performs sync to both UnboundDNS and AdguardHome
func UnifiedCaddySync(
	unboundClient *api.Client,
	adguardClient *api.AdguardClient,
	options CommonSyncOptions,
) (*UnifiedSyncResult, error) {
	result := &UnifiedSyncResult{
		SyncedToUnbound: unboundClient != nil,
		SyncedToAdguard: adguardClient != nil,
	}

	// Fetch hostname map from Caddy once (shared by both syncs)
	caddyClient := api.NewCaddyClient(options.CaddyServerIP, options.CaddyServerPort)
	hostnameMap, err := caddyClient.GetHostnameMap()
	if err != nil {
		logging.Error("Error fetching Caddy hostnames", "error", err)
		return nil, fmt.Errorf("error fetching Caddy hostnames: %w", err)
	}

	result.HostnameMap = hostnameMap

	if len(hostnameMap) == 0 {
		logging.Warn("No hostnames found in Caddy config")
		return result, nil
	}

	// Sync to UnboundDNS if client provided
	if unboundClient != nil {
		logging.Info("Starting UnboundDNS sync", "hostnames", len(hostnameMap))

		unboundOptions := CaddySyncOptions{
			DryRun:             options.DryRun,
			CaddyServerIP:      options.CaddyServerIP,
			CaddyServerPort:    options.CaddyServerPort,
			EntryDescription:   options.EntryDescription,
			LegacyDescriptions: options.LegacyDescriptions,
			Verbose:            options.Verbose,
		}

		// Use existing sync function but with pre-fetched hostname map
		unboundResult, err := syncCaddyWithUnboundInternal(unboundClient, unboundOptions, hostnameMap)
		if err != nil {
			logging.Error("UnboundDNS sync failed", "error", err)
			result.UnboundError = err
		} else {
			result.UnboundResult = unboundResult
			logging.Info("UnboundDNS sync completed successfully",
				"added", len(unboundResult.ToAdd),
				"updated", len(unboundResult.ToUpdate),
				"removed", len(unboundResult.ToRemove))
		}
	}

	// Sync to AdguardHome if client provided
	if adguardClient != nil {
		logging.Info("Starting AdguardHome sync", "hostnames", len(hostnameMap))

		adguardOptions := CaddyAdguardSyncOptions{
			DryRun:             options.DryRun,
			CaddyServerIP:      options.CaddyServerIP,
			CaddyServerPort:    options.CaddyServerPort,
			EntryDescription:   options.EntryDescription,
			LegacyDescriptions: options.LegacyDescriptions,
			Verbose:            options.Verbose,
		}

		// Use existing sync function but with pre-fetched hostname map
		adguardResult, err := syncCaddyWithAdguardInternal(adguardClient, adguardOptions, hostnameMap)
		if err != nil {
			logging.Error("AdguardHome sync failed", "error", err)
			result.AdguardError = err
		} else {
			result.AdguardResult = adguardResult
			logging.Info("AdguardHome sync completed successfully",
				"added", len(adguardResult.ToAdd),
				"updated", len(adguardResult.ToUpdate),
				"removed", len(adguardResult.ToRemove))
		}
	}

	// Check for any errors
	if result.UnboundError != nil || result.AdguardError != nil {
		var errorMsg string
		if result.UnboundError != nil && result.AdguardError != nil {
			errorMsg = fmt.Sprintf("both UnboundDNS (%v) and AdguardHome (%v) sync failed",
				result.UnboundError, result.AdguardError)
		} else if result.UnboundError != nil {
			errorMsg = fmt.Sprintf("UnboundDNS sync failed: %v", result.UnboundError)
		} else {
			errorMsg = fmt.Sprintf("AdguardHome sync failed: %v", result.AdguardError)
		}
		return result, fmt.Errorf(errorMsg)
	}

	return result, nil
}

// syncCaddyWithUnboundInternal is an internal version that accepts pre-fetched hostname map
func syncCaddyWithUnboundInternal(
	unboundClient *api.Client,
	options CaddySyncOptions,
	hostnameMap map[string]string,
) (*SyncResult, error) {
	// Get existing overrides from Unbound
	existingOverrides, err := unboundClient.GetOverrides()
	if err != nil {
		return nil, fmt.Errorf("error fetching overrides: %w", err)
	}

	// Organize overrides for easier processing (copied from caddy.go)
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

	// Process changes (logic from caddy.go)
	var toAdd, toUpdate, toUpdateDesc []string

	for hostname, serverIP := range hostnameMap {
		if !containsDot(hostname) {
			continue
		}

		override, existsInSync := syncCreatedOverrides[hostname]
		_, existsOther := otherOverrides[hostname]

		if !existsInSync && !existsOther {
			toAdd = append(toAdd, hostname)
		} else if !existsInSync && existsOther {
			if options.Verbose {
				logging.Info("Hostname already exists (not created by sync)", "hostname", hostname)
			}
		} else if existsInSync {
			needsUpdate := override.Server != serverIP
			needsDescUpdate := override.Description != options.EntryDescription

			if needsUpdate {
				toUpdate = append(toUpdate, hostname)
			} else if needsDescUpdate {
				toUpdateDesc = append(toUpdateDesc, hostname)
			}
		}
	}

	// Find entries to remove
	var toRemove []string
	for hostname := range syncCreatedOverrides {
		if _, exists := hostnameMap[hostname]; !exists {
			toRemove = append(toRemove, hostname)
		}
	}

	// Apply changes if not dry run
	changesApplied := false
	if !options.DryRun {
		changesApplied = applyChanges(
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

// syncCaddyWithAdguardInternal is an internal version that accepts pre-fetched hostname map
func syncCaddyWithAdguardInternal(
	adguardClient *api.AdguardClient,
	options CaddyAdguardSyncOptions,
	hostnameMap map[string]string,
) (*AdguardSyncResult, error) {
	// Get existing rewrites from AdguardHome
	existingRewrites, err := adguardClient.ListRewrites()
	if err != nil {
		return nil, fmt.Errorf("error fetching AdguardHome rewrites: %w", err)
	}

	// Organize rewrites (logic from caddy_adguard.go)
	var syncCreatedRewrites []api.Rewrite
	var otherRewrites []api.Rewrite
	syncRewriteMap := make(map[string]api.Rewrite)

	for _, rewrite := range existingRewrites {
		if rewrite.Answer == options.CaddyServerIP {
			syncCreatedRewrites = append(syncCreatedRewrites, rewrite)
			syncRewriteMap[rewrite.Domain] = rewrite
		} else {
			otherRewrites = append(otherRewrites, rewrite)
		}
	}

	// Process changes
	var toAdd, toUpdate []string
	addedDomains := make(map[string]bool)

	for hostname, serverIP := range hostnameMap {
		if !containsDot(hostname) || addedDomains[hostname] {
			continue
		}
		addedDomains[hostname] = true

		existingRewrite, existsInSync := syncRewriteMap[hostname]

		existsOther := false
		for _, rewrite := range otherRewrites {
			if rewrite.Domain == hostname {
				existsOther = true
				break
			}
		}

		if !existsInSync && !existsOther {
			toAdd = append(toAdd, hostname)
		} else if !existsInSync && existsOther {
			if options.Verbose {
				logging.Info("Hostname already exists in AdguardHome (not pointing to Caddy)",
					"hostname", hostname)
			}
		} else if existsInSync && existingRewrite.Answer != serverIP {
			toUpdate = append(toUpdate, hostname)
		}
	}

	// Find entries to remove
	var toRemove []string
	for domain := range syncRewriteMap {
		if _, exists := hostnameMap[domain]; !exists {
			toRemove = append(toRemove, domain)
		}
	}

	// Apply changes if not dry run
	changesApplied := false
	if !options.DryRun {
		changesApplied = applyAdguardChanges(
			adguardClient,
			options,
			hostnameMap,
			syncRewriteMap,
			toAdd,
			toUpdate,
			toRemove,
		)
	}

	return &AdguardSyncResult{
		HostnameMap:    hostnameMap,
		ToAdd:          toAdd,
		ToUpdate:       toUpdate,
		ToRemove:       toRemove,
		ChangesApplied: changesApplied,
		SyncRewrites:   syncCreatedRewrites,
		OtherRewrites:  otherRewrites,
		ExistingCount:  len(existingRewrites),
	}, nil
}

// Helper function to check if hostname contains a dot
func containsDot(hostname string) bool {
	for _, char := range hostname {
		if char == '.' {
			return true
		}
	}
	return false
}
