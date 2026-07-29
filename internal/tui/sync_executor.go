package tui

import (
	"context"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/syncplan"
)

// TUISyncExecutor adapts TUI sync callbacks to the UI-neutral syncplan applier.
type TUISyncExecutor struct {
	clients syncplan.Clients
	dryRun  bool
	ctx     context.Context
}

// WithContext sets the context for sync operations (enables Ctrl+C cancellation).
func (e *TUISyncExecutor) WithContext(ctx context.Context) *TUISyncExecutor {
	e.ctx = ctx
	return e
}

// NewTUISyncExecutor creates a new TUI sync executor.
func NewTUISyncExecutor(
	unboundClient *api.Client,
	adguardClient *api.AdguardClient,
	dhcpClient *api.DNSMasqClient,
	cfClient *api.CloudflareClient,
) *TUISyncExecutor {
	_ = dhcpClient
	executor := &TUISyncExecutor{}
	if unboundClient != nil {
		executor.clients.Unbound = unboundClient
	}
	if adguardClient != nil {
		executor.clients.Adguard = adguardClient
	}
	if cfClient != nil {
		executor.clients.Cloudflare = cfClient
	}
	return executor
}

// SetDryRun sets dry run mode.
func (e *TUISyncExecutor) SetDryRun(dryRun bool) {
	e.dryRun = dryRun
}

// ExecuteSyncAction executes a single sync action.
func (e *TUISyncExecutor) ExecuteSyncAction(action syncplan.Action) error {
	action.Enabled = true
	ctx := e.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	result := syncplan.Apply(ctx, e.clients, syncplan.Plan{Actions: []syncplan.Action{action}}, syncplan.ApplyOptions{
		DryRun: e.dryRun,
	})
	if result.Success {
		return nil
	}
	if len(result.ActionResults) > 0 && result.ActionResults[0].Error != "" {
		return syncplan.ActionError(result.ActionResults[0].Error)
	}
	if len(result.Errors) > 0 {
		return syncplan.ActionError(result.Errors[0])
	}
	return nil
}

// ExecuteSyncActions executes multiple sync actions and returns a result.
func (e *TUISyncExecutor) ExecuteSyncActions(actions []syncplan.Action) *syncplan.Result {
	ctx := e.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return syncplan.Apply(ctx, e.clients, syncplan.Plan{Actions: actions}, syncplan.ApplyOptions{
		DryRun: e.dryRun,
	})
}
