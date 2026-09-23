package sync

import (
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/ownership"
)

// CaddyAdguardSyncOptions contains options for the Caddy to AdguardHome sync operation
type CaddyAdguardSyncOptions struct {
	BaseSyncOptions
	// OwnershipPath stores explicit AdGuard rewrite identities. Empty uses the
	// selected configuration file's adjacent ownership state.
	OwnershipPath string
}

// AdguardSyncResult contains the results of the AdguardHome sync operation
type AdguardSyncResult struct {
	HostnameMap     map[string]string
	ToAdd           []string
	ToUpdate        []string
	ToRemove        []string
	ChangesApplied  bool
	SyncRewrites    []api.Rewrite
	OtherRewrites   []api.Rewrite
	ExistingCount   int
	FailedHostnames []string // hostnames that failed during apply
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

	// DNS rewrites must point to the Caddy server IP, not the upstream backend.
	NormalizeHostnameMapToCaddyIP(hostnameMap, options.CaddyServerIP)

	if len(hostnameMap) == 0 {
		logging.Warn("No hostnames found in Caddy config")
		return &AdguardSyncResult{HostnameMap: hostnameMap}, nil
	}
	return syncCaddyWithAdguardInternal(adguardClient, options, hostnameMap)
}

// applyAdguardChanges applies the changes to the AdguardHome DNS rewrites.
// Returns (changesApplied, failedHostnames).
func applyAdguardChanges(
	client *api.AdguardClient,
	options CaddyAdguardSyncOptions,
	hostnameMap map[string]string,
	syncRewriteMap map[string]api.Rewrite,
	toAdd, toUpdate, toRemove []string,
	owned *adguardOwnership,
) (bool, []string) {
	changesApplied := false
	var failed []string

	// Add new rewrites
	for _, hostname := range toAdd {
		serverIP := hostnameMap[hostname]
		if err := owned.begin("add", hostname, serverIP); err != nil {
			failed = append(failed, hostname)
			logging.Error("Failed to persist AdGuard ownership intent", "error", err, "domain", hostname)
			continue
		}

		logging.Info("Adding DNS rewrite", "domain", hostname, "answer", serverIP)

		err := client.AddRewrite(hostname, serverIP)
		if err != nil {
			logging.Error(
				"Failed to add DNS rewrite",
				"error", err,
				"domain", hostname,
				"answer", serverIP,
			)
			failed = append(failed, hostname)
			continue
		}

		changesApplied = true
		if err := owned.verifyAndRecord(client, hostname, serverIP); err != nil {
			failed = append(failed, hostname)
			logging.Error("AdGuard rewrite outcome is unresolved", "error", err, "domain", hostname)
		}
	}

	// Update existing rewrites
	for _, hostname := range toUpdate {
		existingRewrite := syncRewriteMap[hostname]
		newServerIP := hostnameMap[hostname]
		if err := owned.begin("update", hostname, newServerIP); err != nil {
			failed = append(failed, hostname)
			logging.Error("Failed to persist AdGuard ownership intent", "error", err, "domain", hostname)
			continue
		}

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
			failed = append(failed, hostname)
			continue
		}

		changesApplied = true
		if err := owned.verifyAndRecord(client, hostname, newServerIP); err != nil {
			failed = append(failed, hostname)
			logging.Error("AdGuard rewrite outcome is unresolved", "error", err, "domain", hostname)
		}
	}

	// Remove stale rewrites
	for _, hostname := range toRemove {
		existingRewrite := syncRewriteMap[hostname]
		if err := owned.begin("delete", hostname, existingRewrite.Answer); err != nil {
			failed = append(failed, hostname)
			logging.Error("Failed to persist AdGuard ownership intent", "error", err, "domain", hostname)
			continue
		}

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
			failed = append(failed, hostname)
			continue
		}

		changesApplied = true
		if err := owned.verifyDeleted(client, hostname); err != nil {
			failed = append(failed, hostname)
			logging.Error("AdGuard rewrite deletion is unresolved", "error", err, "domain", hostname)
		}
	}

	if changesApplied {
		logging.Info("AdguardHome DNS rewrites updated successfully")
	} else {
		logging.Info("No changes were needed - AdguardHome rewrites are in sync")
	}

	return changesApplied, failed
}

type adguardOwnership struct {
	path  string
	state ownership.State
}

func loadAdguardOwnership(path string) (*adguardOwnership, error) {
	if path == "" {
		configPath, err := config.SelectedConfigPath("")
		if err != nil {
			return nil, fmt.Errorf("resolve ownership state path: %w", err)
		}
		path = ownership.PathForConfig(configPath)
	}
	state, err := ownership.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load AdGuard ownership state: %w", err)
	}
	return &adguardOwnership{path: path, state: state}, nil
}

func classifyAdguardRewrites(rewrites []api.Rewrite, state ownership.State) ([]api.Rewrite, []api.Rewrite, map[string]api.Rewrite) {
	owned := make([]api.Rewrite, 0)
	other := make([]api.Rewrite, 0)
	ownedByDomain := make(map[string]api.Rewrite)
	for _, rewrite := range rewrites {
		key := ownership.Key("adguard", "rewrite", rewrite.Domain)
		resource, known := state.Resources[key]
		if known && state.Owns("adguard", "rewrite", rewrite.Domain) && resource.Expected == ownership.Fingerprint(rewrite.Answer) {
			owned = append(owned, rewrite)
			ownedByDomain[rewrite.Domain] = rewrite
			continue
		}
		other = append(other, rewrite)
	}
	return owned, other, ownedByDomain
}

func (o *adguardOwnership) begin(operation, domain, expected string) error {
	if err := o.state.Begin(ownership.Intent{Operation: operation, Provider: "adguard", Kind: "rewrite", ID: domain, Expected: ownership.Fingerprint(expected)}); err != nil {
		return err
	}
	return ownership.Save(o.path, o.state)
}

func (o *adguardOwnership) verifyAndRecord(client *api.AdguardClient, domain, answer string) error {
	rewrites, err := client.ListRewrites()
	if err != nil {
		return fmt.Errorf("read back rewrites: %w", err)
	}
	for _, rewrite := range rewrites {
		if rewrite.Domain == domain && rewrite.Answer == answer {
			if err := o.state.Record(ownership.Resource{Provider: "adguard", Kind: "rewrite", ID: domain, Expected: ownership.Fingerprint(answer)}); err != nil {
				return err
			}
			o.state.Resolve("adguard", "rewrite", domain)
			return ownership.Save(o.path, o.state)
		}
	}
	return fmt.Errorf("readback did not find %s -> %s", domain, answer)
}

func (o *adguardOwnership) verifyDeleted(client *api.AdguardClient, domain string) error {
	rewrites, err := client.ListRewrites()
	if err != nil {
		return fmt.Errorf("read back rewrites: %w", err)
	}
	for _, rewrite := range rewrites {
		if rewrite.Domain == domain {
			return fmt.Errorf("readback still found %s -> %s", domain, rewrite.Answer)
		}
	}
	o.state.Forget("adguard", "rewrite", domain)
	o.state.Resolve("adguard", "rewrite", domain)
	return ownership.Save(o.path, o.state)
}
