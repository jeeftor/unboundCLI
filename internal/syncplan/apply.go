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
	DeleteTunnelRuleAtPath(hostname, tunnelID, path string) error
	EnsureDNSRecord(hostname string) error
	EnsureDNSRecordInTunnel(hostname, tunnelID string) error
	DeleteDNSRecord(hostname string) error
	DeleteDNSRecordByID(recordID string) error
	FindTunnelIngress(tunnelID, hostname, path string) (api.CloudflareIngressEntry, bool, error)
	FindTunnelDNSRecord(hostname string) (api.CloudflareDNSRecord, bool, error)
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
	var cloudflareOwnership *cloudflareOwnership
	var cloudflareOwnershipErr error
	if !options.DryRun && hasEnabledAction(actions, "adguard") {
		adguardOwnership, adguardOwnershipErr = loadAdguardOwnership(options.OwnershipPath)
	}
	if !options.DryRun && hasEnabledAction(actions, "cloudflare") {
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

func applyAction(clients Clients, action Action, adguardOwnership *adguardOwnership, adguardOwnershipErr error, cloudflareOwnership *cloudflareOwnership, cloudflareOwnershipErr error) error {
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

func applyCloudflareAction(client CloudflareClient, action Action, owned *cloudflareOwnership) error {
	if client == nil {
		return fmt.Errorf("Cloudflare client not available")
	}
	if owned == nil {
		return fmt.Errorf("Cloudflare ownership state is required for mutations")
	}

	switch action.Type {
	case "add":
		if action.TunnelID == "" {
			return fmt.Errorf("Cloudflare add for %s has no explicit tunnel identity; preview again with a configured tunnel", action.Hostname)
		}
		ingressID := cloudflareIngressResourceID(action)
		if err := owned.requireAbsent("ingress", ingressID); err != nil {
			return err
		}
		if _, found, err := client.FindTunnelIngress(action.TunnelID, action.Hostname, action.Path); err != nil {
			return fmt.Errorf("read Cloudflare ingress %s: %w", action.Hostname, err)
		} else if found {
			return fmt.Errorf("Cloudflare ingress %s already exists; explicit adoption is required", action.Hostname)
		}
		if record, found, err := client.FindTunnelDNSRecord(action.Hostname); err != nil {
			return fmt.Errorf("read Cloudflare DNS record for %s: %w", action.Hostname, err)
		} else if found && !owned.state.Owns("cloudflare", "dns", record.ID) {
			return fmt.Errorf("Cloudflare DNS record for %s is not explicitly owned; adoption is required", action.Hostname)
		}
		if err := owned.begin("add", "ingress", ingressID, cloudflareActionIngressValue(action)); err != nil {
			return err
		}
		if err := client.UpdateTunnelRule(api.IngressRuleSpec{
			Hostname:                  action.Hostname,
			Path:                      action.Path,
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
		if err := owned.verifyIngress(client, action, true); err != nil {
			return err
		}
		pendingDNSID := cloudflarePendingDNSResourceID(action)
		if err := owned.begin("add", "dns", pendingDNSID, action.TunnelID+".cfargotunnel.com"); err != nil {
			return err
		}
		if err := client.EnsureDNSRecordInTunnel(action.Hostname, action.TunnelID); err != nil {
			return err
		}
		return owned.verifyDNS(client, action, pendingDNSID, true)
	case "update":
		if err := owned.requireCurrentIngress(client, action); err != nil {
			return err
		}
		if err := owned.begin("update", "ingress", cloudflareIngressResourceID(action), cloudflareActionIngressValue(action)); err != nil {
			return err
		}
		if err := client.UpdateTunnelRule(api.IngressRuleSpec{
			Hostname:                  action.Hostname,
			Path:                      action.Path,
			Service:                   action.NewService,
			HTTPHostHeader:            action.NewHTTPHostHeader,
			OriginServerName:          action.OriginServerName,
			SetOriginServerName:       true, // always write on update
			NoTLSVerify:               action.NoTLSVerify,
			SetNoTLSVerify:            true,
			DisableChunkedEncoding:    action.DisableChunkedEncoding,
			SetDisableChunkedEncoding: true,
			TunnelID:                  action.TunnelID,
		}); err != nil {
			return err
		}
		return owned.verifyIngress(client, action, true)
	case "delete":
		if err := owned.requireCurrentIngress(client, action); err != nil {
			return err
		}
		if err := owned.requireCurrentDNS(client, action); err != nil {
			return err
		}
		ingressID := cloudflareIngressResourceID(action)
		if err := owned.begin("delete", "ingress", ingressID, action.OldService); err != nil {
			return err
		}
		if err := owned.begin("delete", "dns", action.CloudflareDNSRecordID, action.Hostname); err != nil {
			return err
		}
		if err := client.DeleteTunnelRuleAtPath(action.Hostname, action.TunnelID, action.Path); err != nil {
			return err
		}
		if _, found, err := client.FindTunnelIngress(action.TunnelID, action.Hostname, action.Path); err != nil {
			return fmt.Errorf("read back deleted Cloudflare ingress %s: %w", action.Hostname, err)
		} else if found {
			return fmt.Errorf("readback still found Cloudflare ingress %s", action.Hostname)
		}
		if err := client.DeleteDNSRecordByID(action.CloudflareDNSRecordID); err != nil {
			return err
		}
		if _, found, err := client.FindTunnelDNSRecord(action.Hostname); err != nil {
			return fmt.Errorf("read back deleted Cloudflare DNS record for %s: %w", action.Hostname, err)
		} else if found {
			return fmt.Errorf("readback still found Cloudflare DNS record for %s", action.Hostname)
		}
		owned.state.Forget("cloudflare", "ingress", ingressID)
		owned.state.Forget("cloudflare", "dns", action.CloudflareDNSRecordID)
		owned.state.Resolve("cloudflare", "ingress", ingressID)
		owned.state.Resolve("cloudflare", "dns", action.CloudflareDNSRecordID)
		return ownership.Save(owned.path, owned.state)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

type cloudflareOwnership struct {
	path  string
	state ownership.State
}

func loadCloudflareOwnership(path string) (*cloudflareOwnership, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("Cloudflare ownership state path is required for mutations")
	}
	state, err := ownership.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load Cloudflare ownership state: %w", err)
	}
	return &cloudflareOwnership{path: path, state: state}, nil
}

func (o *cloudflareOwnership) requireAbsent(kind, id string) error {
	if id == "" {
		return fmt.Errorf("Cloudflare %s identity is required", kind)
	}
	if o.state.HasIntent("cloudflare", kind, id) {
		return fmt.Errorf("Cloudflare %s %s has an unresolved operation; review recovery state before retrying", kind, id)
	}
	if o.state.Owns("cloudflare", kind, id) {
		return fmt.Errorf("Cloudflare %s %s is already owned; preview again before changing it", kind, id)
	}
	return nil
}

func (o *cloudflareOwnership) begin(operation, kind, id, expected string) error {
	if id == "" {
		return fmt.Errorf("Cloudflare %s identity is required", kind)
	}
	if o.state.HasIntent("cloudflare", kind, id) {
		return fmt.Errorf("Cloudflare %s %s has an unresolved operation; review recovery state before retrying", kind, id)
	}
	if err := o.state.Begin(ownership.Intent{Operation: operation, Provider: "cloudflare", Kind: kind, ID: id, Expected: ownership.Fingerprint(expected)}); err != nil {
		return err
	}
	return ownership.Save(o.path, o.state)
}

func (o *cloudflareOwnership) requireCurrentIngress(client CloudflareClient, action Action) error {
	id := cloudflareIngressResourceID(action)
	resource, ok := o.state.Resources[ownership.Key("cloudflare", "ingress", id)]
	if !ok || !o.state.Owns("cloudflare", "ingress", id) || resource.Expected == "" {
		return fmt.Errorf("Cloudflare ingress %s is not explicitly owned; adoption is required", action.Hostname)
	}
	entry, found, err := client.FindTunnelIngress(action.TunnelID, action.Hostname, action.Path)
	if err != nil {
		return fmt.Errorf("read Cloudflare ingress %s: %w", action.Hostname, err)
	}
	if !found || resource.Expected != ownership.Fingerprint(cloudflareIngressValue(entry)) {
		return fmt.Errorf("Cloudflare ingress %s changed outside this tool; preview adoption again", action.Hostname)
	}
	return nil
}

func (o *cloudflareOwnership) requireCurrentDNS(client CloudflareClient, action Action) error {
	if action.CloudflareDNSRecordID == "" {
		return fmt.Errorf("Cloudflare DNS record identity is required for %s", action.Hostname)
	}
	resource, ok := o.state.Resources[ownership.Key("cloudflare", "dns", action.CloudflareDNSRecordID)]
	if !ok || !o.state.Owns("cloudflare", "dns", action.CloudflareDNSRecordID) || resource.Expected == "" {
		return fmt.Errorf("Cloudflare DNS record for %s is not explicitly owned; adoption is required", action.Hostname)
	}
	record, found, err := client.FindTunnelDNSRecord(action.Hostname)
	if err != nil {
		return fmt.Errorf("read Cloudflare DNS record for %s: %w", action.Hostname, err)
	}
	if !found || record.ID != action.CloudflareDNSRecordID || resource.Expected != ownership.Fingerprint(record.Target) {
		return fmt.Errorf("Cloudflare DNS record for %s changed outside this tool; preview adoption again", action.Hostname)
	}
	return nil
}

func (o *cloudflareOwnership) verifyIngress(client CloudflareClient, action Action, resolve bool) error {
	entry, found, err := client.FindTunnelIngress(action.TunnelID, action.Hostname, action.Path)
	if err != nil {
		return fmt.Errorf("read back Cloudflare ingress %s: %w", action.Hostname, err)
	}
	if !found || entry.Service != action.NewService || entry.HTTPHostHeader != action.NewHTTPHostHeader || entry.OriginServerName != action.OriginServerName || entry.NoTLSVerify != action.NoTLSVerify {
		return fmt.Errorf("Cloudflare ingress %s did not match the requested value after update", action.Hostname)
	}
	id := cloudflareIngressResourceID(action)
	if err := o.state.Record(ownership.Resource{Provider: "cloudflare", Kind: "ingress", ID: id, Expected: ownership.Fingerprint(cloudflareIngressValue(entry))}); err != nil {
		return err
	}
	if resolve {
		o.state.Resolve("cloudflare", "ingress", id)
	}
	return ownership.Save(o.path, o.state)
}

func (o *cloudflareOwnership) verifyDNS(client CloudflareClient, action Action, pendingID string, resolve bool) error {
	record, found, err := client.FindTunnelDNSRecord(action.Hostname)
	if err != nil {
		return fmt.Errorf("read back Cloudflare DNS record for %s: %w", action.Hostname, err)
	}
	expected := action.TunnelID + ".cfargotunnel.com"
	if !found || record.Target != expected {
		return fmt.Errorf("Cloudflare DNS record for %s did not match tunnel %s after update", action.Hostname, action.TunnelID)
	}
	if err := o.state.Record(ownership.Resource{Provider: "cloudflare", Kind: "dns", ID: record.ID, Expected: ownership.Fingerprint(record.Target)}); err != nil {
		return err
	}
	if resolve {
		o.state.Resolve("cloudflare", "dns", pendingID)
	}
	return ownership.Save(o.path, o.state)
}

func cloudflareIngressResourceID(action Action) string {
	if action.TunnelID == "" || action.Hostname == "" {
		return ""
	}
	return action.TunnelID + ":" + action.Hostname + ":" + action.Path
}

func cloudflarePendingDNSResourceID(action Action) string {
	return "pending:" + action.TunnelID + ":" + action.Hostname
}

func cloudflareActionIngressValue(action Action) string {
	return action.TunnelID + "|" + action.Hostname + "|" + action.Path + "|" + action.NewService + "|" + action.NewHTTPHostHeader + "|" + action.OriginServerName
}

func cloudflareIngressValue(entry api.CloudflareIngressEntry) string {
	return entry.TunnelID + "|" + entry.Hostname + "|" + entry.Path + "|" + entry.Service + "|" + entry.HTTPHostHeader + "|" + entry.OriginServerName
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
