package sync

import (
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
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

	return syncHostnamesWithUnbound(unboundClient, hostnameMap, unboundSyncOptions{
		Source:             "Cloudflare tunnel config",
		EntryDescription:   options.EntryDescription,
		LegacyDescriptions: options.LegacyDescriptions,
		DryRun:             options.DryRun,
		Verbose:            options.Verbose,
	})
}
