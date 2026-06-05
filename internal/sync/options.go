package sync

import "fmt"

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

// DefaultSyncOptions returns sync options with default values
func DefaultSyncOptions() *SyncOptions {
	return &SyncOptions{
		DryRun:           false,
		CaddyServerIP:    "192.168.1.15",
		CaddyServerPort:  2019,
		EntryDescription: "Entry created by CaddySync",
		Verbose:          false,
		UnboundOnly:      false,
		AdguardOnly:      false,
	}
}
