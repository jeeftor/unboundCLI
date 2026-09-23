package syncplan

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/ownership"
)

func TestApplyUpdatesUnboundByFullHostnameAndRestartsOnce(t *testing.T) {
	unbound := &fakeUnboundClient{
		overrides: []api.DNSOverride{
			{UUID: "uuid-1", Host: "app", Domain: "example.com", Server: "10.0.0.99", Description: "Managed by caddy-dns-sync"},
		},
	}

	result := Apply(context.Background(), Clients{Unbound: unbound}, Plan{Actions: []Action{
		{
			Type:     "update",
			Service:  "unbound",
			Hostname: "app.example.com",
			OldIP:    "10.0.0.99",
			NewIP:    "10.0.0.15",
			Enabled:  true,
		},
	}}, ApplyOptions{})

	if !result.Success {
		t.Fatalf("expected success, got errors: %#v", result.Errors)
	}
	if result.ItemsUpdated != 1 {
		t.Fatalf("expected one update, got %d", result.ItemsUpdated)
	}
	if len(unbound.updated) != 1 {
		t.Fatalf("expected one Unbound update, got %d", len(unbound.updated))
	}
	if unbound.updated[0].UUID != "uuid-1" {
		t.Fatalf("expected update UUID uuid-1, got %q", unbound.updated[0].UUID)
	}
	if unbound.applyCalls != 1 {
		t.Fatalf("expected one Unbound apply, got %d", unbound.applyCalls)
	}
	if len(result.ActionResults) != 1 || !result.ActionResults[0].Success {
		t.Fatalf("expected successful per-action result, got %#v", result.ActionResults)
	}
}

func TestApplyProtectsManualUnboundOverride(t *testing.T) {
	unbound := &fakeUnboundClient{overrides: []api.DNSOverride{{
		UUID: "manual", Host: "app", Domain: "example.com", Server: "10.0.0.99", Description: "manual record",
	}}}
	result := Apply(context.Background(), Clients{Unbound: unbound}, Plan{Actions: []Action{{
		Type: "delete", Service: "unbound", Hostname: "app.example.com", OldIP: "10.0.0.99", Enabled: true,
	}}}, ApplyOptions{})
	if result.Success || len(unbound.deleted) != 0 {
		t.Fatalf("manual override must be protected, result=%#v deleted=%#v", result, unbound.deleted)
	}
}

func TestApplyRecordsPerActionFailures(t *testing.T) {
	result := Apply(context.Background(), Clients{}, Plan{Actions: []Action{
		{
			Type:     "add",
			Service:  "unbound",
			Hostname: "missing.example.com",
			NewIP:    "10.0.0.15",
			Enabled:  true,
		},
		{
			Type:     "add",
			Service:  "adguard",
			Hostname: "missing.example.com",
			NewIP:    "10.0.0.15",
			Enabled:  false,
		},
	}}, ApplyOptions{})

	if result.Success {
		t.Fatal("expected apply failure")
	}
	if len(result.ActionResults) != 2 {
		t.Fatalf("expected two per-action results, got %d", len(result.ActionResults))
	}
	if result.ActionResults[0].Success {
		t.Fatalf("expected first action to fail: %#v", result.ActionResults[0])
	}
	if result.ActionResults[0].Error == "" {
		t.Fatal("expected first action error")
	}
	if !result.ActionResults[1].Skipped {
		t.Fatalf("expected disabled action to be marked skipped: %#v", result.ActionResults[1])
	}
}

func TestApplyDryRunCountsEnabledActionsWithoutMutating(t *testing.T) {
	unbound := &fakeUnboundClient{}

	result := Apply(context.Background(), Clients{Unbound: unbound}, Plan{Actions: []Action{
		{
			Type:     "add",
			Service:  "unbound",
			Hostname: "dry.example.com",
			NewIP:    "10.0.0.15",
			Enabled:  true,
		},
	}}, ApplyOptions{DryRun: true})

	if !result.Success {
		t.Fatalf("expected dry-run success, got errors: %#v", result.Errors)
	}
	if result.ItemsAdded != 1 {
		t.Fatalf("expected dry-run add count 1, got %d", result.ItemsAdded)
	}
	if len(unbound.added) != 0 || unbound.applyCalls != 0 {
		t.Fatalf("dry-run mutated Unbound: added=%d apply=%d", len(unbound.added), unbound.applyCalls)
	}
}

func TestApplyReportsUnboundRestartFailure(t *testing.T) {
	unbound := &fakeUnboundClient{applyErr: errors.New("restart failed")}

	result := Apply(context.Background(), Clients{Unbound: unbound}, Plan{Actions: []Action{
		{
			Type:     "add",
			Service:  "unbound",
			Hostname: "restart.example.com",
			NewIP:    "10.0.0.15",
			Enabled:  true,
		},
	}}, ApplyOptions{})

	if result.Success {
		t.Fatal("expected restart failure to fail result")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected one error, got %#v", result.Errors)
	}
}

func TestApplyAdguardActions(t *testing.T) {
	adguard := &fakeAdguardClient{rewrites: []api.Rewrite{
		{Domain: "update.example.com", Answer: "10.0.0.10"},
		{Domain: "old.example.com", Answer: "10.0.0.10"},
	}}
	ownershipPath := adguardOwnershipPath(t, map[string]string{
		"update.example.com": "10.0.0.10",
		"old.example.com":    "10.0.0.10",
	})

	result := Apply(context.Background(), Clients{Adguard: adguard}, Plan{Actions: []Action{
		{
			Type:     "add",
			Service:  "adguard",
			Hostname: "new.example.com",
			NewIP:    "10.0.0.15",
			Enabled:  true,
		},
		{
			Type:     "update",
			Service:  "adguard",
			Hostname: "update.example.com",
			OldIP:    "10.0.0.10",
			NewIP:    "10.0.0.15",
			Enabled:  true,
		},
		{
			Type:     "delete",
			Service:  "adguard",
			Hostname: "old.example.com",
			OldIP:    "10.0.0.10",
			Enabled:  true,
		},
	}}, ApplyOptions{OwnershipPath: ownershipPath})

	if !result.Success {
		t.Fatalf("expected success, got %#v", result.Errors)
	}
	if result.ItemsAdded != 1 || result.ItemsUpdated != 1 || result.ItemsDeleted != 1 {
		t.Fatalf("unexpected result counts: %#v", result)
	}
	if len(adguard.added) != 1 || len(adguard.updated) != 1 || len(adguard.deleted) != 1 {
		t.Fatalf("unexpected adguard mutations: added=%#v updated=%#v deleted=%#v", adguard.added, adguard.updated, adguard.deleted)
	}
	if adguard.updated[0].target.Answer != "10.0.0.10" || adguard.updated[0].update.Answer != "10.0.0.15" {
		t.Fatalf("unexpected update payload: %#v", adguard.updated[0])
	}
}

func TestApplyProtectsUnownedAdguardRewrite(t *testing.T) {
	adguard := &fakeAdguardClient{rewrites: []api.Rewrite{{Domain: "manual.example.com", Answer: "10.0.0.10"}}}
	result := Apply(context.Background(), Clients{Adguard: adguard}, Plan{Actions: []Action{{
		Type: "delete", Service: "adguard", Hostname: "manual.example.com", OldIP: "10.0.0.10", Enabled: true,
	}}}, ApplyOptions{OwnershipPath: adguardOwnershipPath(t, nil)})
	if result.Success || len(adguard.deleted) != 0 {
		t.Fatalf("manual AdGuard rewrite must be protected, result=%#v deleted=%#v", result, adguard.deleted)
	}
}

func TestApplyRejectsDuplicateAdguardRewrite(t *testing.T) {
	adguard := &fakeAdguardClient{rewrites: []api.Rewrite{
		{Domain: "duplicate.example.com", Answer: "10.0.0.10"},
		{Domain: "duplicate.example.com", Answer: "10.0.0.99"},
	}}
	result := Apply(context.Background(), Clients{Adguard: adguard}, Plan{Actions: []Action{{
		Type: "update", Service: "adguard", Hostname: "duplicate.example.com", OldIP: "10.0.0.10", NewIP: "10.0.0.15", Enabled: true,
	}}}, ApplyOptions{OwnershipPath: adguardOwnershipPath(t, map[string]string{"duplicate.example.com": "10.0.0.10"})})
	if result.Success || len(adguard.updated) != 0 {
		t.Fatalf("duplicate AdGuard rewrites must be protected, result=%#v updates=%#v", result, adguard.updated)
	}
}

func TestApplyAdguardPersistsIntentBeforeWriteFailure(t *testing.T) {
	adguard := &fakeAdguardClient{addErr: errors.New("provider unavailable")}
	path := adguardOwnershipPath(t, nil)
	result := Apply(context.Background(), Clients{Adguard: adguard}, Plan{Actions: []Action{{
		Type: "add", Service: "adguard", Hostname: "new.example.com", NewIP: "10.0.0.15", Enabled: true,
	}}}, ApplyOptions{OwnershipPath: path})
	if result.Success {
		t.Fatal("expected provider failure")
	}
	state, err := ownership.Load(path)
	if err != nil || !state.HasIntent("adguard", "rewrite", "new.example.com") {
		t.Fatalf("expected unresolved intent after provider failure, state=%#v err=%v", state, err)
	}
}

func adguardOwnershipPath(t *testing.T, resources map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ownership.json")
	state, err := ownership.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for domain, answer := range resources {
		if err := state.Record(ownership.Resource{Provider: "adguard", Kind: "rewrite", ID: domain, Expected: ownership.Fingerprint(answer)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := ownership.Save(path, state); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestApplyCloudflareAddUpdateAndDeleteActions(t *testing.T) {
	cloudflare := &fakeCloudflareClient{}
	path := cloudflareOwnershipPath(t, []ownership.Resource{
		{Provider: "cloudflare", Kind: "ingress", ID: "selected-tunnel:update.example.com:"},
		{Provider: "cloudflare", Kind: "ingress", ID: "selected-tunnel:old.example.com:"},
		{Provider: "cloudflare", Kind: "dns", ID: "dns-old"},
	})

	result := Apply(context.Background(), Clients{Cloudflare: cloudflare}, Plan{Actions: []Action{
		{
			Type:              "add",
			Service:           "cloudflare",
			Hostname:          "new.example.com",
			NewService:        "http://10.0.0.15:80",
			NewHTTPHostHeader: "new.example.com",
			Enabled:           true,
		},
		{
			Type:              "update",
			Service:           "cloudflare",
			Hostname:          "update.example.com",
			TunnelID:          "selected-tunnel",
			NewService:        "http://10.0.0.15:80",
			NewHTTPHostHeader: "update.example.com",
			Enabled:           true,
		},
		{
			Type:                  "delete",
			Service:               "cloudflare",
			Hostname:              "old.example.com",
			TunnelID:              "selected-tunnel",
			CloudflareDNSRecordID: "dns-old",
			Enabled:               true,
		},
	}}, ApplyOptions{OwnershipPath: path})

	if !result.Success {
		t.Fatalf("expected success, got %#v", result.Errors)
	}
	if result.ItemsAdded != 1 || result.ItemsUpdated != 1 || result.ItemsDeleted != 1 {
		t.Fatalf("unexpected result counts: %#v", result)
	}
	if len(cloudflare.updatedRules) != 2 {
		t.Fatalf("expected two Cloudflare rule updates, got %#v", cloudflare.updatedRules)
	}
	if cloudflare.updatedRules[0].Hostname != "new.example.com" ||
		cloudflare.updatedRules[0].Service != "http://10.0.0.15:80" ||
		cloudflare.updatedRules[0].HTTPHostHeader != "new.example.com" {
		t.Fatalf("unexpected add rule spec: %#v", cloudflare.updatedRules[0])
	}
	if len(cloudflare.ensuredDNS) != 1 || cloudflare.ensuredDNS[0] != "new.example.com" {
		t.Fatalf("expected DNS ensure for add only, got %#v", cloudflare.ensuredDNS)
	}
	if len(cloudflare.deletedRules) != 1 || cloudflare.deletedRules[0] != "old.example.com" {
		t.Fatalf("expected one deleted rule, got %#v", cloudflare.deletedRules)
	}
	if len(cloudflare.deleteTunnelIDs) != 1 || cloudflare.deleteTunnelIDs[0] != "selected-tunnel" {
		t.Fatalf("expected delete to use selected tunnel, got %#v", cloudflare.deleteTunnelIDs)
	}
	if len(cloudflare.deletedDNS) != 1 || cloudflare.deletedDNS[0] != "old.example.com" {
		t.Fatalf("expected one deleted DNS record, got %#v", cloudflare.deletedDNS)
	}
}

func TestApplyProtectsUnownedCloudflareResources(t *testing.T) {
	cloudflare := &fakeCloudflareClient{}
	result := Apply(context.Background(), Clients{Cloudflare: cloudflare}, Plan{Actions: []Action{{
		Type: "delete", Service: "cloudflare", Hostname: "manual.example.com", TunnelID: "selected-tunnel", CloudflareDNSRecordID: "manual-dns", Enabled: true,
	}}}, ApplyOptions{OwnershipPath: cloudflareOwnershipPath(t, nil)})
	if result.Success || len(cloudflare.deletedRules) != 0 || len(cloudflare.deletedDNS) != 0 {
		t.Fatalf("unowned Cloudflare resources must be protected, result=%#v", result)
	}
}

func cloudflareOwnershipPath(t *testing.T, resources []ownership.Resource) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ownership.json")
	state, err := ownership.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range resources {
		if err := state.Record(resource); err != nil {
			t.Fatal(err)
		}
	}
	if err := ownership.Save(path, state); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestApplyCloudflareDryRunDoesNotMutate(t *testing.T) {
	cloudflare := &fakeCloudflareClient{}

	result := Apply(context.Background(), Clients{Cloudflare: cloudflare}, Plan{Actions: []Action{
		{
			Type:              "add",
			Service:           "cloudflare",
			Hostname:          "dry.example.com",
			NewService:        "http://10.0.0.15:80",
			NewHTTPHostHeader: "dry.example.com",
			Enabled:           true,
		},
	}}, ApplyOptions{DryRun: true})

	if !result.Success {
		t.Fatalf("expected dry-run success, got %#v", result.Errors)
	}
	if result.ItemsAdded != 1 {
		t.Fatalf("expected one dry-run add count, got %#v", result)
	}
	if len(cloudflare.updatedRules) != 0 || len(cloudflare.ensuredDNS) != 0 {
		t.Fatalf("dry-run mutated Cloudflare: rules=%#v dns=%#v", cloudflare.updatedRules, cloudflare.ensuredDNS)
	}
}

type fakeUnboundClient struct {
	overrides  []api.DNSOverride
	added      []api.DNSOverride
	updated    []api.DNSOverride
	deleted    []string
	applyCalls int
	applyErr   error
}

func (f *fakeUnboundClient) GetOverrides() ([]api.DNSOverride, error) {
	return f.overrides, nil
}

func (f *fakeUnboundClient) AddOverride(override api.DNSOverride) (string, error) {
	f.added = append(f.added, override)
	return "new-uuid", nil
}

func (f *fakeUnboundClient) UpdateOverride(override api.DNSOverride) error {
	f.updated = append(f.updated, override)
	return nil
}

func (f *fakeUnboundClient) DeleteOverride(uuid string) error {
	f.deleted = append(f.deleted, uuid)
	return nil
}

func (f *fakeUnboundClient) ApplyChanges() error {
	f.applyCalls++
	return f.applyErr
}

type fakeAdguardUpdate struct {
	target api.Rewrite
	update api.Rewrite
}

type fakeAdguardClient struct {
	rewrites []api.Rewrite
	added    []api.Rewrite
	updated  []fakeAdguardUpdate
	deleted  []api.Rewrite
	addErr   error
}

func (f *fakeAdguardClient) AddRewrite(domain, answer string) error {
	if f.addErr != nil {
		return f.addErr
	}
	f.added = append(f.added, api.Rewrite{Domain: domain, Answer: answer})
	f.rewrites = append(f.rewrites, api.Rewrite{Domain: domain, Answer: answer})
	return nil
}

func (f *fakeAdguardClient) UpdateRewrite(target, update api.Rewrite) error {
	f.updated = append(f.updated, fakeAdguardUpdate{target: target, update: update})
	for index := range f.rewrites {
		if f.rewrites[index] == target {
			f.rewrites[index] = update
			break
		}
	}
	return nil
}

func (f *fakeAdguardClient) DeleteRewrite(domain, answer string) error {
	f.deleted = append(f.deleted, api.Rewrite{Domain: domain, Answer: answer})
	for index, rewrite := range f.rewrites {
		if rewrite.Domain == domain && rewrite.Answer == answer {
			f.rewrites = append(f.rewrites[:index], f.rewrites[index+1:]...)
			break
		}
	}
	return nil
}

func (f *fakeAdguardClient) ListRewrites() ([]api.Rewrite, error) {
	return append([]api.Rewrite(nil), f.rewrites...), nil
}

type fakeCloudflareClient struct {
	updatedRules    []api.IngressRuleSpec
	deletedRules    []string
	deleteTunnelIDs []string
	ensuredDNS      []string
	deletedDNS      []string
}

func (f *fakeCloudflareClient) UpdateTunnelRule(spec api.IngressRuleSpec) error {
	f.updatedRules = append(f.updatedRules, spec)
	return nil
}

func (f *fakeCloudflareClient) DeleteTunnelRule(hostname string) error {
	f.deletedRules = append(f.deletedRules, hostname)
	return nil
}

func (f *fakeCloudflareClient) DeleteTunnelRuleInTunnel(hostname, tunnelID string) error {
	f.deletedRules = append(f.deletedRules, hostname)
	f.deleteTunnelIDs = append(f.deleteTunnelIDs, tunnelID)
	return nil
}

func (f *fakeCloudflareClient) EnsureDNSRecord(hostname string) error {
	f.ensuredDNS = append(f.ensuredDNS, hostname)
	return nil
}

func (f *fakeCloudflareClient) DeleteDNSRecord(hostname string) error {
	f.deletedDNS = append(f.deletedDNS, hostname)
	return nil
}
