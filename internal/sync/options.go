package sync

import (
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/app"
)

// SyncOptions contains all sync configuration
type SyncOptions struct {
	// Common options
	DryRun             bool
	CaddyServerIP      string
	CaddyServerPort    int
	EntryDescription   string
	LegacyDescriptions []string
	Verbose            bool

	// Target selection for unified sync
	UnboundOnly bool
	AdguardOnly bool
}

// Validate ensures required fields are set
func (o *SyncOptions) Validate() error {
	if o.CaddyServerIP == "" {
		return fmt.Errorf("caddy server IP is required")
	}
	if o.CaddyServerPort == 0 {
		return fmt.Errorf("caddy server port is required")
	}
	return nil
}

// DefaultSyncOptions returns sync options whose endpoint is resolved by the
// shared runtime from the selected config file or built-in fallback.
func DefaultSyncOptions() *SyncOptions {
	return &SyncOptions{
		DryRun:           false,
		EntryDescription: app.CurrentUnboundDescription,
		Verbose:          false,
		UnboundOnly:      false,
		AdguardOnly:      false,
	}
}
