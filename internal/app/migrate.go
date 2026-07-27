package app

import (
	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
)

// LegacyUnboundDescriptions lists the old description strings that sync
// operations may have stamped on Unbound DNS overrides in prior releases.
// These are rewritten to CurrentUnboundDescription by MigrateUnboundDescriptions.
var LegacyUnboundDescriptions = []string{
	"Entry created by unboundCLI caddy-sync-all",
	"Entry created by unboundCLI sync",
	"Entry created by unboundCLI caddy-sync-unbound",
	"Entry created by unboundCLI caddy-sync-cloudflare",
	"Entry created by CaddySync",
	"Route via Caddy",
}

// CurrentUnboundDescription is the description stamp used by current sync
// operations. Old entries are migrated to this value.
const CurrentUnboundDescription = "Managed by caddy-dns-sync"

// MigrateUnboundDescriptions scans Unbound DNS overrides for entries whose
// description matches a known legacy string and rewrites them to
// CurrentUnboundDescription. This runs best-effort at startup so that the
// description-based ownership model continues to work after the rename from
// unboundCLI to caddy-dns-sync.
//
// Failures are logged but not returned: migration is a convenience, not a
// prerequisite for the caller to proceed.
func MigrateUnboundDescriptions(client *api.Client) {
	if client == nil {
		return
	}

	overrides, err := client.GetOverrides()
	if err != nil {
		logging.Debug("Unbound description migration skipped: cannot fetch overrides", "error", err)
		return
	}

	migrated := 0
	for _, override := range overrides {
		if override.Description == CurrentUnboundDescription {
			continue
		}
		if !isLegacyUnboundDescription(override.Description) {
			continue
		}
		override.Description = CurrentUnboundDescription
		if err := client.UpdateOverride(override); err != nil {
			logging.Warn("Failed to migrate Unbound override description",
				"uuid", override.UUID,
				"host", override.Host,
				"domain", override.Domain,
				"error", err,
			)
			continue
		}
		migrated++
	}

	if migrated > 0 {
		logging.Info("Migrated legacy Unbound override descriptions", "count", migrated)
	}
}

func isLegacyUnboundDescription(desc string) bool {
	for _, legacy := range LegacyUnboundDescriptions {
		if desc == legacy {
			return true
		}
	}
	return false
}
