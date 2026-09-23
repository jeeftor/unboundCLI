package syncplan

import (
	"context"
	"fmt"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/ownership"
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
	ListRewrites() ([]api.Rewrite, error)
}

type CloudflareClient interface {
	UpdateTunnelRule(api.IngressRuleSpec) error
	DeleteTunnelRule(hostname string) error
	DeleteTunnelRuleInTunnel(hostname, tunnelID string) error
	EnsureDNSRecord(hostname string) error
	DeleteDNSRecord(hostname string) error
}

// Clients contains service clients used to apply a sync plan.
type Clients struct {
	Unbound    UnboundClient
	Adguard    AdguardClient
	Cloudflare CloudflareClient
}

// ApplyOptions controls sync plan application.
type ApplyOptions struct {
	DryRun        bool
	OwnershipPath string
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
	var adguardOwnership *adguardOwnership
	var adguardOwnershipErr error
	var cloudflareOwnership ownership.State
	var cloudflareOwnershipErr error
	if !options.DryRun && hasEnabledAction(actions, "adguard") {
		adguardOwnership, adguardOwnershipErr = loadAdguardOwnership(options.OwnershipPath)
	}
	if !options.DryRun && hasEnabledCloudflareExistingAction(actions) {
		cloudflareOwnership, cloudflareOwnershipErr = loadCloudflareOwnership(options.OwnershipPath)
	}

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
			err = applyAction(clients, action, adguardOwnership, adguardOwnershipErr, cloudflareOwnership, cloudflareOwnershipErr)
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

func applyAction(clients Clients, action Action, adguardOwnership *adguardOwnership, adguardOwnershipErr error, cloudflareOwnership ownership.State, cloudflareOwnershipErr error) error {
	switch action.Service {
	case "unbound":
		return applyUnboundAction(clients.Unbound, action)
	case "adguard":
		if adguardOwnershipErr != nil {
			return adguardOwnershipErr
		}
		return applyAdguardAction(clients.Adguard, action, adguardOwnership)
	case "cloudflare":
		if cloudflareOwnershipErr != nil {
			return cloudflareOwnershipErr
		}
		return applyCloudflareAction(clients.Cloudflare, action, cloudflareOwnership)
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
		if err := ensureUnboundHostnameUnused(client, action.Hostname); err != nil {
			return err
		}
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
		override, err := findManagedUnboundOverride(client, action.Hostname, action.OldIP)
		if err != nil {
			return err
		}
		override.Enabled = "1"
		override.Server = action.NewIP
		return client.UpdateOverride(override)
	case "delete":
		override, err := findManagedUnboundOverride(client, action.Hostname, action.OldIP)
		if err != nil {
			return err
		}
		return client.DeleteOverride(override.UUID)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

func applyAdguardAction(client AdguardClient, action Action, owned *adguardOwnership) error {
	if client == nil {
		return fmt.Errorf("AdGuard client not available")
	}
	if owned == nil {
		return fmt.Errorf("AdGuard ownership state is required for mutations")
	}
	if owned.state.HasIntent("adguard", "rewrite", action.Hostname) {
		return fmt.Errorf("AdGuard rewrite %s has an unresolved operation; review recovery state before retrying", action.Hostname)
	}

	switch action.Type {
	case "add":
		rewrites, err := client.ListRewrites()
		if err != nil {
			return fmt.Errorf("list AdGuard rewrites: %w", err)
		}
		if _, err := exactAdguardRewrite(rewrites, action.Hostname, ""); err == nil {
			return fmt.Errorf("AdGuard rewrite already exists for %s; explicit adoption is required", action.Hostname)
		} else if !isAdguardRewriteNotFound(err) {
			return err
		}
		if err := owned.begin("add", action.Hostname, action.NewIP); err != nil {
			return err
		}
		if err := client.AddRewrite(action.Hostname, action.NewIP); err != nil {
			return err
		}
		return owned.verifyAndRecord(client, action.Hostname, action.NewIP)
	case "update":
		if err := owned.require(action.Hostname, action.OldIP); err != nil {
			return err
		}
		existing, err := owned.current(client, action.Hostname, action.OldIP)
		if err != nil {
			return err
		}
		if err := owned.begin("update", action.Hostname, action.NewIP); err != nil {
			return err
		}
		if err := client.UpdateRewrite(existing, api.Rewrite{Domain: action.Hostname, Answer: action.NewIP}); err != nil {
			return err
		}
		return owned.verifyAndRecord(client, action.Hostname, action.NewIP)
	case "delete":
		if err := owned.require(action.Hostname, action.OldIP); err != nil {
			return err
		}
		if _, err := owned.current(client, action.Hostname, action.OldIP); err != nil {
			return err
		}
		if err := owned.begin("delete", action.Hostname, action.OldIP); err != nil {
			return err
		}
		if err := client.DeleteRewrite(action.Hostname, action.OldIP); err != nil {
			return err
		}
		return owned.verifyDeleted(client, action.Hostname)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

func hasEnabledAction(actions []Action, service string) bool {
	for _, action := range actions {
		if action.Enabled && action.Service == service {
			return true
		}
	}
	return false
}

func hasEnabledCloudflareExistingAction(actions []Action) bool {
	for _, action := range actions {
		if action.Enabled && action.Service == "cloudflare" && (action.Type == "update" || action.Type == "delete") {
			return true
		}
	}
	return false
}

type adguardOwnership struct {
	path  string
	state ownership.State
}

func loadAdguardOwnership(path string) (*adguardOwnership, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("AdGuard ownership state path is required for mutations")
	}
	state, err := ownership.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load AdGuard ownership state: %w", err)
	}
	return &adguardOwnership{path: path, state: state}, nil
}

func (o *adguardOwnership) begin(operation, domain, expected string) error {
	if err := o.state.Begin(ownership.Intent{Operation: operation, Provider: "adguard", Kind: "rewrite", ID: domain, Expected: ownership.Fingerprint(expected)}); err != nil {
		return err
	}
	return ownership.Save(o.path, o.state)
}

func (o *adguardOwnership) require(domain, expected string) error {
	resource, ok := o.state.Resources[ownership.Key("adguard", "rewrite", domain)]
	if !ok || !o.state.Owns("adguard", "rewrite", domain) || resource.Expected != ownership.Fingerprint(expected) {
		return fmt.Errorf("AdGuard rewrite %s is not an owned resource with the expected value; explicit adoption is required", domain)
	}
	return nil
}

func (o *adguardOwnership) current(client AdguardClient, domain, expected string) (api.Rewrite, error) {
	rewrites, err := client.ListRewrites()
	if err != nil {
		return api.Rewrite{}, fmt.Errorf("list AdGuard rewrites: %w", err)
	}
	return exactAdguardRewrite(rewrites, domain, expected)
}

func (o *adguardOwnership) verifyAndRecord(client AdguardClient, domain, answer string) error {
	current, err := o.current(client, domain, answer)
	if err != nil {
		return fmt.Errorf("read back AdGuard rewrite: %w", err)
	}
	if err := o.state.Record(ownership.Resource{Provider: "adguard", Kind: "rewrite", ID: current.Domain, Expected: ownership.Fingerprint(current.Answer)}); err != nil {
		return err
	}
	o.state.Resolve("adguard", "rewrite", domain)
	return ownership.Save(o.path, o.state)
}

func (o *adguardOwnership) verifyDeleted(client AdguardClient, domain string) error {
	rewrites, err := client.ListRewrites()
	if err != nil {
		return fmt.Errorf("read back AdGuard rewrites: %w", err)
	}
	if _, err := exactAdguardRewrite(rewrites, domain, ""); err == nil {
		return fmt.Errorf("readback still found AdGuard rewrite %s", domain)
	} else if !isAdguardRewriteNotFound(err) {
		return err
	}
	o.state.Forget("adguard", "rewrite", domain)
	o.state.Resolve("adguard", "rewrite", domain)
	return ownership.Save(o.path, o.state)
}

func exactAdguardRewrite(rewrites []api.Rewrite, domain, expected string) (api.Rewrite, error) {
	var match *api.Rewrite
	for index := range rewrites {
		rewrite := &rewrites[index]
		if rewrite.Domain != domain {
			continue
		}
		if match != nil {
			return api.Rewrite{}, fmt.Errorf("ambiguous AdGuard rewrites for %s", domain)
		}
		match = rewrite
	}
	if match == nil {
		return api.Rewrite{}, fmt.Errorf("no matching AdGuard rewrite for %s", domain)
	}
	if expected != "" && match.Answer != expected {
		return api.Rewrite{}, fmt.Errorf("AdGuard rewrite %s no longer has the expected value", domain)
	}
	return *match, nil
}

func isAdguardRewriteNotFound(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "no matching AdGuard rewrite for ")
}

func applyCloudflareAction(client CloudflareClient, action Action, state ownership.State) error {
	if client == nil {
		return fmt.Errorf("Cloudflare client not available")
	}
	if action.Type == "update" || action.Type == "delete" {
		ingressID := cloudflareIngressResourceID(action)
		if ingressID == "" || !state.Owns("cloudflare", "ingress", ingressID) {
			return fmt.Errorf("Cloudflare ingress %s is not explicitly owned; adoption is required", action.Hostname)
		}
		if action.Type == "delete" {
			if action.CloudflareDNSRecordID == "" || !state.Owns("cloudflare", "dns", action.CloudflareDNSRecordID) {
				return fmt.Errorf("Cloudflare DNS record for %s is not explicitly owned; adoption is required", action.Hostname)
			}
		}
	}

	switch action.Type {
	case "add":
		if action.TunnelID == "" {
			return fmt.Errorf("Cloudflare add for %s has no explicit tunnel identity; preview again with a configured tunnel", action.Hostname)
		}
		if err := client.UpdateTunnelRule(api.IngressRuleSpec{
			Hostname:                  action.Hostname,
			Service:                   action.NewService,
			HTTPHostHeader:            action.NewHTTPHostHeader,
			OriginServerName:          action.OriginServerName,
			SetOriginServerName:       action.OriginServerName != "",
			NoTLSVerify:               action.NoTLSVerify,
			SetNoTLSVerify:            action.NoTLSVerify,
			DisableChunkedEncoding:    action.DisableChunkedEncoding,
			SetDisableChunkedEncoding: action.DisableChunkedEncoding,
			TunnelID:                  action.TunnelID,
		}); err != nil {
			return err
		}
		return client.EnsureDNSRecord(action.Hostname)
	case "update":
		return client.UpdateTunnelRule(api.IngressRuleSpec{
			Hostname:                  action.Hostname,
			Service:                   action.NewService,
			HTTPHostHeader:            action.NewHTTPHostHeader,
			OriginServerName:          action.OriginServerName,
			SetOriginServerName:       true, // always write on update
			NoTLSVerify:               action.NoTLSVerify,
			SetNoTLSVerify:            true,
			DisableChunkedEncoding:    action.DisableChunkedEncoding,
			SetDisableChunkedEncoding: true,
			TunnelID:                  action.TunnelID,
		})
	case "delete":
		if err := client.DeleteTunnelRuleInTunnel(action.Hostname, action.TunnelID); err != nil {
			return err
		}
		return client.DeleteDNSRecord(action.Hostname)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

func loadCloudflareOwnership(path string) (ownership.State, error) {
	if strings.TrimSpace(path) == "" {
		return ownership.State{}, fmt.Errorf("Cloudflare ownership state path is required for mutations")
	}
	state, err := ownership.Load(path)
	if err != nil {
		return ownership.State{}, fmt.Errorf("load Cloudflare ownership state: %w", err)
	}
	return state, nil
}

func cloudflareIngressResourceID(action Action) string {
	if action.TunnelID == "" || action.Hostname == "" {
		return ""
	}
	return action.TunnelID + ":" + action.Hostname + ":" + action.Path
}

func ensureUnboundHostnameUnused(client UnboundClient, hostname string) error {
	overrides, err := client.GetOverrides()
	if err != nil {
		return fmt.Errorf("get Unbound overrides: %w", err)
	}
	for _, override := range overrides {
		if joinHostname(override.Host, override.Domain) == hostname {
			return fmt.Errorf("Unbound override already exists for %s; explicit adoption is required", hostname)
		}
	}
	return nil
}

func findManagedUnboundOverride(client UnboundClient, hostname, expectedIP string) (api.DNSOverride, error) {
	overrides, err := client.GetOverrides()
	if err != nil {
		return api.DNSOverride{}, fmt.Errorf("get Unbound overrides: %w", err)
	}
	var match *api.DNSOverride
	for index := range overrides {
		override := &overrides[index]
		if joinHostname(override.Host, override.Domain) != hostname {
			continue
		}
		if !isManagedUnboundDescription(override.Description) {
			continue
		}
		if expectedIP != "" && override.Server != expectedIP {
			continue
		}
		if match != nil {
			return api.DNSOverride{}, fmt.Errorf("ambiguous managed Unbound overrides for %s", hostname)
		}
		match = override
	}
	if match == nil {
		return api.DNSOverride{}, fmt.Errorf("no matching managed Unbound override for %s; explicit adoption is required", hostname)
	}
	return *match, nil
}

func isManagedUnboundDescription(description string) bool {
	if description == app.CurrentUnboundDescription {
		return true
	}
	for _, legacy := range app.LegacyUnboundDescriptions {
		if description == legacy {
			return true
		}
	}
	return false
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
