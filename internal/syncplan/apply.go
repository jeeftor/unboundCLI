package syncplan

import (
	"context"
	"fmt"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
)

type UnboundClient interface {
	GetOverrides() ([]api.DNSOverride, error)
	AddOverride(api.DNSOverride) (string, error)
	UpdateOverride(api.DNSOverride) error
	DeleteOverride(string) error
	ApplyChanges() error
}

type AdguardClient interface {
	AddRewrite(domain, answer string) error
	UpdateRewrite(target, update api.Rewrite) error
	DeleteRewrite(domain, answer string) error
}

// Clients contains service clients used to apply a sync plan.
type Clients struct {
	Unbound UnboundClient
	Adguard AdguardClient
}

// ApplyOptions controls sync plan application.
type ApplyOptions struct {
	DryRun bool
}

type ActionError string

func (e ActionError) Error() string {
	return string(e)
}

// Apply executes enabled plan actions and returns aggregate and per-action results.
func Apply(ctx context.Context, clients Clients, plan Plan, options ApplyOptions) *Result {
	actions := plan.Actions
	result := &Result{
		Success:       true,
		ActionResults: make([]ActionResult, 0, len(actions)),
	}

	unboundChanged := false
	adguardChanged := false

	for _, action := range actions {
		actionResult := ActionResult{Action: action}
		if !action.Enabled {
			actionResult.Skipped = true
			result.ActionResults = append(result.ActionResults, actionResult)
			continue
		}

		if err := ctx.Err(); err != nil {
			recordActionError(result, actionResult, fmt.Errorf("context cancelled: %w", err))
			continue
		}

		var err error
		if !options.DryRun {
			err = applyAction(clients, action)
		}
		if err != nil {
			recordActionError(result, actionResult, err)
			continue
		}

		actionResult.Success = true
		result.ActionResults = append(result.ActionResults, actionResult)
		switch action.Service {
		case "unbound":
			unboundChanged = true
		case "adguard":
			adguardChanged = true
		}
		incrementResultCounts(result, action)
	}

	if !options.DryRun && unboundChanged && clients.Unbound != nil {
		if err := clients.Unbound.ApplyChanges(); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to restart Unbound service: %v", err))
		}
	}

	if adguardChanged {
		// AdGuard rewrites are applied immediately.
	}

	result.Success = len(result.Errors) == 0
	if result.Success {
		result.Message = "All operations completed successfully"
		if unboundChanged && !options.DryRun {
			result.Message += " (Unbound restarted)"
		}
	} else {
		result.Message = fmt.Sprintf("Completed with %d error(s)", len(result.Errors))
	}

	return result
}

// ApplyActions executes an action list through a temporary plan.
func ApplyActions(ctx context.Context, clients Clients, actions []Action, options ApplyOptions) *Result {
	return Apply(ctx, clients, Plan{Actions: actions}, options)
}

func applyAction(clients Clients, action Action) error {
	switch action.Service {
	case "unbound":
		return applyUnboundAction(clients.Unbound, action)
	case "adguard":
		return applyAdguardAction(clients.Adguard, action)
	case "dhcp":
		return fmt.Errorf("DHCP sync not yet implemented")
	default:
		return fmt.Errorf("unknown service: %s", action.Service)
	}
}

func applyUnboundAction(client UnboundClient, action Action) error {
	if client == nil {
		return fmt.Errorf("Unbound client not available")
	}

	switch action.Type {
	case "add":
		host, domain := SplitHostname(action.Hostname)
		_, err := client.AddOverride(api.DNSOverride{
			Enabled:     "1",
			Host:        host,
			Domain:      domain,
			Server:      action.NewIP,
			Description: "Managed by caddy-dns-sync",
		})
		return err
	case "update":
		uuid, err := findUnboundOverrideUUID(client, action.Hostname)
		if err != nil {
			return err
		}
		host, domain := SplitHostname(action.Hostname)
		return client.UpdateOverride(api.DNSOverride{
			UUID:        uuid,
			Enabled:     "1",
			Host:        host,
			Domain:      domain,
			Server:      action.NewIP,
			Description: "Managed by caddy-dns-sync",
		})
	case "delete":
		uuid, err := findUnboundOverrideUUID(client, action.Hostname)
		if err != nil {
			return err
		}
		return client.DeleteOverride(uuid)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

func applyAdguardAction(client AdguardClient, action Action) error {
	if client == nil {
		return fmt.Errorf("AdGuard client not available")
	}

	switch action.Type {
	case "add":
		return client.AddRewrite(action.Hostname, action.NewIP)
	case "update":
		return client.UpdateRewrite(
			api.Rewrite{Domain: action.Hostname, Answer: action.OldIP},
			api.Rewrite{Domain: action.Hostname, Answer: action.NewIP},
		)
	case "delete":
		return client.DeleteRewrite(action.Hostname, action.OldIP)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

func findUnboundOverrideUUID(client UnboundClient, hostname string) (string, error) {
	overrides, err := client.GetOverrides()
	if err != nil {
		return "", fmt.Errorf("failed to get overrides: %w", err)
	}
	for _, override := range overrides {
		if joinHostname(override.Host, override.Domain) == hostname {
			return override.UUID, nil
		}
	}
	return "", fmt.Errorf("override not found for %s", hostname)
}

// SplitHostname splits a fully qualified hostname into host and domain parts.
func SplitHostname(fqdn string) (host, domain string) {
	parts := strings.SplitN(fqdn, ".", 2)
	if len(parts) != 2 {
		return fqdn, "local"
	}
	return parts[0], parts[1]
}

func joinHostname(host, domain string) string {
	if domain == "" {
		return host
	}
	return host + "." + domain
}

func recordActionError(result *Result, actionResult ActionResult, err error) {
	errMsg := fmt.Sprintf("%s %s for %s: %v",
		actionResult.Action.Type,
		actionResult.Action.Service,
		actionResult.Action.Hostname,
		err,
	)
	actionResult.Error = errMsg
	result.ActionResults = append(result.ActionResults, actionResult)
	result.Errors = append(result.Errors, errMsg)
}

func incrementResultCounts(result *Result, action Action) {
	switch action.Type {
	case "add":
		result.ItemsAdded++
	case "update":
		result.ItemsUpdated++
	case "delete":
		result.ItemsDeleted++
	}
}
