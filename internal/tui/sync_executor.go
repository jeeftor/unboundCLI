package tui

import (
	"fmt"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/widgets"
)

// TUISyncExecutor executes sync actions using the API clients
// This bridges the widgets package (which can't import api) with the actual API calls
type TUISyncExecutor struct {
	unboundClient *api.Client
	adguardClient *api.AdguardClient
	dhcpClient    *api.DNSMasqClient
	dryRun        bool
}

// NewTUISyncExecutor creates a new TUI sync executor
func NewTUISyncExecutor(unboundClient *api.Client, adguardClient *api.AdguardClient, dhcpClient *api.DNSMasqClient) *TUISyncExecutor {
	return &TUISyncExecutor{
		unboundClient: unboundClient,
		adguardClient: adguardClient,
		dhcpClient:    dhcpClient,
		dryRun:        false,
	}
}

// SetDryRun sets dry run mode
func (e *TUISyncExecutor) SetDryRun(dryRun bool) {
	e.dryRun = dryRun
}

// ExecuteSyncAction executes a single sync action
func (e *TUISyncExecutor) ExecuteSyncAction(action widgets.SyncAction) error {
	if e.dryRun {
		// In dry run mode, just log what would be done
		return nil
	}

	switch action.Service {
	case "unbound":
		return e.executeUnboundAction(action)
	case "adguard":
		return e.executeAdguardAction(action)
	case "dhcp":
		return e.executeDHCPAction(action)
	default:
		return fmt.Errorf("unknown service: %s", action.Service)
	}
}

// ExecuteSyncActions executes multiple sync actions and returns a result
func (e *TUISyncExecutor) ExecuteSyncActions(actions []widgets.SyncAction) *widgets.SyncResult {
	addCount := 0
	updateCount := 0
	deleteCount := 0
	errors := []string{}
	unboundChanged := false
	adguardChanged := false

	for _, action := range actions {
		if !action.Enabled {
			continue
		}

		err := e.ExecuteSyncAction(action)
		if err != nil {
			errMsg := fmt.Sprintf("%s %s for %s: %v",
				action.Type, action.Service, action.Hostname, err)
			errors = append(errors, errMsg)
		} else {
			// Success - track which services changed
			switch action.Service {
			case "unbound":
				unboundChanged = true
			case "adguard":
				adguardChanged = true
			}

			switch action.Type {
			case "add":
				addCount++
			case "update":
				updateCount++
			case "delete":
				deleteCount++
			}
		}
	}

	// Apply changes to services that were modified
	if unboundChanged && e.unboundClient != nil {
		if err := e.unboundClient.ApplyChanges(); err != nil {
			errMsg := fmt.Sprintf("Failed to restart Unbound service: %v", err)
			errors = append(errors, errMsg)
		}
	}

	if adguardChanged && e.adguardClient != nil {
		// AdGuard doesn't require restart - changes are immediate
		// But we could add a reload call here if needed in the future
	}

	result := &widgets.SyncResult{
		Success:      len(errors) == 0,
		ItemsAdded:   addCount,
		ItemsUpdated: updateCount,
		ItemsDeleted: deleteCount,
		Errors:       errors,
		Message:      "",
	}

	if len(errors) > 0 {
		result.Message = fmt.Sprintf("Completed with %d error(s)", len(errors))
	} else {
		msg := "All operations completed successfully"
		if unboundChanged {
			msg += " (Unbound restarted)"
		}
		result.Message = msg
	}

	return result
}

// executeUnboundAction executes an Unbound sync action
func (e *TUISyncExecutor) executeUnboundAction(action widgets.SyncAction) error {
	if e.unboundClient == nil {
		return fmt.Errorf("Unbound client not available")
	}

	switch action.Type {
	case "add":
		// Add new DNS override
		// Split hostname into host and domain parts
		host, domain := splitHostname(action.Hostname)
		override := api.DNSOverride{
			Enabled:     "1",
			Host:        host,
			Domain:      domain,
			Server:      action.NewIP,
			Description: "Managed by caddy-dns-sync",
		}
		_, err := e.unboundClient.AddOverride(override)
		return err

	case "update":
		// Update existing DNS override
		// First, find the existing override to get its UUID
		overrides, err := e.unboundClient.GetOverrides()
		if err != nil {
			return fmt.Errorf("failed to get overrides: %w", err)
		}

		var targetUUID string
		for _, o := range overrides {
			if o.Host == action.Hostname || o.Domain == action.Hostname {
				targetUUID = o.UUID
				break
			}
		}

		if targetUUID == "" {
			return fmt.Errorf("override not found for %s", action.Hostname)
		}

		// Split hostname into host and domain parts
		host, domain := splitHostname(action.Hostname)
		override := api.DNSOverride{
			UUID:        targetUUID,
			Enabled:     "1",
			Host:        host,
			Domain:      domain,
			Server:      action.NewIP,
			Description: "Managed by caddy-dns-sync",
		}
		return e.unboundClient.UpdateOverride(override)

	case "delete":
		// Delete DNS override
		// First, find the existing override to get its UUID
		overrides, err := e.unboundClient.GetOverrides()
		if err != nil {
			return fmt.Errorf("failed to get overrides: %w", err)
		}

		var targetUUID string
		for _, o := range overrides {
			if o.Host == action.Hostname || o.Domain == action.Hostname {
				targetUUID = o.UUID
				break
			}
		}

		if targetUUID == "" {
			return fmt.Errorf("override not found for %s", action.Hostname)
		}

		return e.unboundClient.DeleteOverride(targetUUID)

	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// executeAdguardAction executes an AdGuard sync action
func (e *TUISyncExecutor) executeAdguardAction(action widgets.SyncAction) error {
	if e.adguardClient == nil {
		return fmt.Errorf("AdGuard client not available")
	}

	switch action.Type {
	case "add":
		// Add new DNS rewrite
		return e.adguardClient.AddRewrite(action.Hostname, action.NewIP)

	case "update":
		// Update existing DNS rewrite
		target := api.Rewrite{
			Domain: action.Hostname,
			Answer: action.OldIP,
		}
		update := api.Rewrite{
			Domain: action.Hostname,
			Answer: action.NewIP,
		}
		return e.adguardClient.UpdateRewrite(target, update)

	case "delete":
		// Delete DNS rewrite
		return e.adguardClient.DeleteRewrite(action.Hostname, action.OldIP)

	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// splitHostname splits a fully qualified domain name into host and domain parts
// Matches the behavior in internal/exec/sync/caddy.go
// e.g., "pve1.example.com" -> ("pve1", "example.com")
// e.g., "esphome.example.com" -> ("esphome", "example.com")
func splitHostname(fqdn string) (host, domain string) {
	// Use SplitN with limit of 2 to split on first dot only
	parts := strings.SplitN(fqdn, ".", 2)
	if len(parts) != 2 {
		// Invalid hostname - no dot found
		// This shouldn't happen with Caddy hostnames, but handle gracefully
		return fqdn, "local"
	}

	return parts[0], parts[1]
}

// executeDHCPAction executes a DHCP sync action
func (e *TUISyncExecutor) executeDHCPAction(action widgets.SyncAction) error {
	if e.dhcpClient == nil {
		return fmt.Errorf("DHCP client not available")
	}

	// DHCP sync is typically for creating static leases
	// This would need to be implemented based on the DHCP client API
	return fmt.Errorf("DHCP sync not yet implemented")
}
