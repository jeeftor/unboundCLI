package sync

import (
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	execsync "github.com/jeeftor/caddy-dns-sync/internal/exec/sync"
)

// SyncExecutor handles sync operations with a unified interface
type SyncExecutor struct {
	options       *SyncOptions
	caddyClient   *api.CaddyClient
	unboundClient *api.Client
	adguardClient *api.AdguardClient
}

// NewSyncExecutor creates a new sync executor
func NewSyncExecutor(options *SyncOptions) *SyncExecutor {
	return &SyncExecutor{
		options: options,
	}
}

// SetClients sets the API clients for the executor
func (e *SyncExecutor) SetClients(caddy *api.CaddyClient, unbound *api.Client, adguard *api.AdguardClient) {
	e.caddyClient = caddy
	e.unboundClient = unbound
	e.adguardClient = adguard
}

// SyncToUnbound executes Caddy → Unbound sync
func (e *SyncExecutor) SyncToUnbound() (*execsync.SyncResult, error) {
	if e.unboundClient == nil {
		return nil, fmt.Errorf("unbound client not set")
	}

	// Convert options to the format expected by exec/sync
	opts := execsync.CaddySyncOptions{
		DryRun:             e.options.DryRun,
		CaddyServerIP:      e.options.CaddyServerIP,
		CaddyServerPort:    e.options.CaddyServerPort,
		EntryDescription:   e.options.EntryDescription,
		LegacyDescriptions: e.options.LegacyDescriptions,
		Verbose:            e.options.Verbose,
	}

	// Set prompt mode if requested
	if e.options.UnboundPrompt {
		e.unboundClient.Prompt = true
	}

	return execsync.SyncCaddyWithUnbound(e.unboundClient, opts)
}

// SyncToAdguard executes Caddy → Adguard sync
func (e *SyncExecutor) SyncToAdguard() (*execsync.AdguardSyncResult, error) {
	if e.adguardClient == nil {
		return nil, fmt.Errorf("adguard client not set")
	}

	// Convert options to the format expected by exec/sync
	opts := execsync.CaddyAdguardSyncOptions{
		DryRun:             e.options.DryRun,
		CaddyServerIP:      e.options.CaddyServerIP,
		CaddyServerPort:    e.options.CaddyServerPort,
		EntryDescription:   e.options.EntryDescription,
		LegacyDescriptions: e.options.LegacyDescriptions,
		Verbose:            e.options.Verbose,
	}

	// Set prompt mode if requested
	if e.options.AdguardPrompt {
		e.adguardClient.Prompt = true
	}

	return execsync.SyncCaddyWithAdguard(e.adguardClient, opts)
}

// SyncAll executes unified sync to both Unbound and Adguard
func (e *SyncExecutor) SyncAll() (*execsync.UnifiedSyncResult, error) {
	// Determine which clients to sync to based on options
	var unboundClient *api.Client
	var adguardClient *api.AdguardClient

	if !e.options.AdguardOnly && e.unboundClient != nil {
		unboundClient = e.unboundClient
		if e.options.UnboundPrompt {
			unboundClient.Prompt = true
		}
	}

	if !e.options.UnboundOnly && e.adguardClient != nil {
		adguardClient = e.adguardClient
		if e.options.AdguardPrompt {
			adguardClient.Prompt = true
		}
	}

	// Convert options to the format expected by exec/sync
	opts := execsync.CommonSyncOptions{
		DryRun:             e.options.DryRun,
		CaddyServerIP:      e.options.CaddyServerIP,
		CaddyServerPort:    e.options.CaddyServerPort,
		EntryDescription:   e.options.EntryDescription,
		LegacyDescriptions: e.options.LegacyDescriptions,
		Verbose:            e.options.Verbose,
	}

	return execsync.UnifiedCaddySync(unboundClient, adguardClient, opts)
}
