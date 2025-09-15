package sync

import (
	"fmt"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/logging"
)

// CaddyAdguardSyncOptions contains options for the Caddy to AdguardHome sync operation
type CaddyAdguardSyncOptions struct {
	DryRun             bool
	CaddyServerIP      string
	CaddyServerPort    int
	EntryDescription   string
	LegacyDescriptions []string
	Verbose            bool
}

// AdguardSyncResult contains the results of the AdguardHome sync operation
type AdguardSyncResult struct {
	HostnameMap    map[string]string
	ToAdd          []string
	ToUpdate       []string
	ToRemove       []string
	ChangesApplied bool
	SyncRewrites   []api.Rewrite
	OtherRewrites  []api.Rewrite
	ExistingCount  int
}

// SyncCaddyWithAdguard synchronizes DNS rewrites between Caddy and AdguardHome
func SyncCaddyWithAdguard(
	adguardClient *api.AdguardClient,
	options CaddyAdguardSyncOptions,
) (*AdguardSyncResult, error) {
	caddyClient := api.NewCaddyClient(options.CaddyServerIP, options.CaddyServerPort)

	// Fetch hostname map from Caddy
	hostnameMap, err := caddyClient.GetHostnameMap()
	if err != nil {
		logging.Error("Error fetching Caddy hostnames", "error", err)
		return nil, fmt.Errorf("error fetching Caddy hostnames: %w", err)
	}

	if len(hostnameMap) == 0 {
		logging.Warn("No hostnames found in Caddy config")
		return &AdguardSyncResult{HostnameMap: hostnameMap}, nil
	}

	// Get existing rewrites from AdguardHome
	existingRewrites, err := adguardClient.ListRewrites()
	if err != nil {
		logging.Error("Error fetching AdguardHome rewrites", "error", err)
		return nil, fmt.Errorf("error fetching AdguardHome rewrites: %w", err)
	}

	if options.Verbose {
		logging.Info("AdguardHome sync analysis",
			"totalRewrites", len(existingRewrites),
			"caddyHostnames", len(hostnameMap),
			"caddyServerIP", options.CaddyServerIP)
	}

	// Organize rewrites for easier processing
	var syncCreatedRewrites []api.Rewrite
	var otherRewrites []api.Rewrite
	syncRewriteMap := make(map[string]api.Rewrite)

	// Get the current hostnames from Caddy to determine what we should manage
	caddyHostnameSet := make(map[string]bool)
	for hostname := range hostnameMap {
		// Skip non-FQDN hostnames
		if strings.Contains(hostname, ".") {
			caddyHostnameSet[hostname] = true
		}
	}

	for _, rewrite := range existingRewrites {
		// More precise logic: Only consider rewrites that:
		// 1. Point to our Caddy server IP AND
		// 2. Have domains that are currently in Caddy config
		// This prevents us from managing manually created rewrites that happen to use the same IP
		if rewrite.Answer == options.CaddyServerIP && caddyHostnameSet[rewrite.Domain] {
			syncCreatedRewrites = append(syncCreatedRewrites, rewrite)
			syncRewriteMap[rewrite.Domain] = rewrite
		} else {
			otherRewrites = append(otherRewrites, rewrite)
		}
	}

	if options.Verbose {
		logging.Info("AdguardHome rewrite categorization",
			"syncManaged", len(syncCreatedRewrites),
			"otherRewrites", len(otherRewrites))
	}

	// Process each hostname from Caddy
	var toAdd, toUpdate []string
	addedDomains := make(map[string]bool) // Track domains we've already processed

	for hostname, serverIP := range hostnameMap {
		// Skip if hostname doesn't contain a dot (not a FQDN)
		if !strings.Contains(hostname, ".") {
			continue
		}

		// Skip if we've already processed this domain (avoid duplicates)
		if addedDomains[hostname] {
			continue
		}
		addedDomains[hostname] = true

		// Check if this hostname already exists in AdguardHome rewrites
		existingRewrite, existsInSync := syncRewriteMap[hostname]

		// Check if this domain exists in other rewrites (not pointing to Caddy)
		existsOther := false
		for _, rewrite := range otherRewrites {
			if rewrite.Domain == hostname {
				existsOther = true
				break
			}
		}

		if !existsInSync && !existsOther {
			// Need to add this hostname
			toAdd = append(toAdd, hostname)
		} else if !existsInSync && existsOther {
			// Exists but not pointing to Caddy - leave it alone
			if options.Verbose {
				logging.Info("Hostname already exists in AdguardHome (not pointing to Caddy)",
					"hostname", hostname)
			}
		} else if existsInSync {
			// Created by sync, check if it needs updating
			// Note: Since we only include rewrites pointing to CaddyServerIP in syncRewriteMap,
			// and serverIP is always CaddyServerIP, this should rarely trigger unless there's
			// an edge case like IP address changes
			if existingRewrite.Answer != serverIP {
				if options.Verbose {
					logging.Info("AdguardHome rewrite needs IP update",
						"hostname", hostname,
						"existingIP", existingRewrite.Answer,
						"newIP", serverIP)
				}
				toUpdate = append(toUpdate, hostname)
			}
		}
	}

	// Find entries to remove (in sync but not in Caddy)
	var toRemove []string
	for domain := range syncRewriteMap {
		if _, exists := hostnameMap[domain]; !exists {
			toRemove = append(toRemove, domain)
		}
	}

	// If not a dry run, perform the actual changes
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

// applyAdguardChanges applies the changes to the AdguardHome DNS rewrites
func applyAdguardChanges(
	client *api.AdguardClient,
	options CaddyAdguardSyncOptions,
	hostnameMap map[string]string,
	syncRewriteMap map[string]api.Rewrite,
	toAdd, toUpdate, toRemove []string,
) bool {
	changesApplied := false

	// Add new rewrites
	for _, hostname := range toAdd {
		serverIP := hostnameMap[hostname]

		logging.Info("Adding DNS rewrite", "domain", hostname, "answer", serverIP)

		err := client.AddRewrite(hostname, serverIP)
		if err != nil {
			logging.Error(
				"Failed to add DNS rewrite",
				"error", err,
				"domain", hostname,
				"answer", serverIP,
			)
			continue
		}

		changesApplied = true
	}

	// Update existing rewrites
	for _, hostname := range toUpdate {
		existingRewrite := syncRewriteMap[hostname]
		newServerIP := hostnameMap[hostname]

		logging.Info("Updating DNS rewrite",
			"domain", hostname,
			"old_answer", existingRewrite.Answer,
			"new_answer", newServerIP)

		// Create updated rewrite
		updatedRewrite := api.Rewrite{
			Domain: hostname,
			Answer: newServerIP,
		}

		err := client.UpdateRewrite(existingRewrite, updatedRewrite)
		if err != nil {
			logging.Error(
				"Failed to update DNS rewrite",
				"error", err,
				"domain", hostname,
			)
			continue
		}

		changesApplied = true
	}

	// Remove stale rewrites
	for _, hostname := range toRemove {
		existingRewrite := syncRewriteMap[hostname]

		logging.Info("Removing DNS rewrite",
			"domain", hostname,
			"answer", existingRewrite.Answer)

		err := client.DeleteRewrite(hostname, existingRewrite.Answer)
		if err != nil {
			logging.Error(
				"Failed to remove DNS rewrite",
				"error", err,
				"domain", hostname,
			)
			continue
		}

		changesApplied = true
	}

	if changesApplied {
		logging.Info("AdguardHome DNS rewrites updated successfully")
	} else {
		logging.Info("No changes were needed - AdguardHome rewrites are in sync")
	}

	return changesApplied
}
