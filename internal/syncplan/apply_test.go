package syncplan

import (
	"context"
	"errors"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
)

func TestApplyUpdatesUnboundByFullHostnameAndRestartsOnce(t *testing.T) {
	unbound := &fakeUnboundClient{
		overrides: []api.DNSOverride{
			{UUID: "uuid-1", Host: "app", Domain: "example.com", Server: "10.0.0.99"},
		},
	}

	result := Apply(context.Background(), Clients{Unbound: unbound}, Plan{Actions: []Action{
		{
			Type:     "update",
			Service:  "unbound",
			Hostname: "app.example.com",
			OldIP:    "10.0.0.99",
			NewIP:    "192.168.1.15",
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

func TestApplyRecordsPerActionFailures(t *testing.T) {
	result := Apply(context.Background(), Clients{}, Plan{Actions: []Action{
		{
			Type:     "add",
			Service:  "unbound",
			Hostname: "missing.example.com",
			NewIP:    "192.168.1.15",
			Enabled:  true,
		},
		{
			Type:     "add",
			Service:  "adguard",
			Hostname: "missing.example.com",
			NewIP:    "192.168.1.15",
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
			NewIP:    "192.168.1.15",
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
			NewIP:    "192.168.1.15",
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
	adguard := &fakeAdguardClient{}

	result := Apply(context.Background(), Clients{Adguard: adguard}, Plan{Actions: []Action{
		{
			Type:     "add",
			Service:  "adguard",
			Hostname: "new.example.com",
			NewIP:    "192.168.1.15",
			Enabled:  true,
		},
		{
			Type:     "update",
			Service:  "adguard",
			Hostname: "update.example.com",
			OldIP:    "192.168.1.10",
			NewIP:    "192.168.1.15",
			Enabled:  true,
		},
		{
			Type:     "delete",
			Service:  "adguard",
			Hostname: "old.example.com",
			OldIP:    "192.168.1.10",
			Enabled:  true,
		},
	}}, ApplyOptions{})

	if !result.Success {
		t.Fatalf("expected success, got %#v", result.Errors)
	}
	if result.ItemsAdded != 1 || result.ItemsUpdated != 1 || result.ItemsDeleted != 1 {
		t.Fatalf("unexpected result counts: %#v", result)
	}
	if len(adguard.added) != 1 || len(adguard.updated) != 1 || len(adguard.deleted) != 1 {
		t.Fatalf("unexpected adguard mutations: added=%#v updated=%#v deleted=%#v", adguard.added, adguard.updated, adguard.deleted)
	}
	if adguard.updated[0].target.Answer != "192.168.1.10" || adguard.updated[0].update.Answer != "192.168.1.15" {
		t.Fatalf("unexpected update payload: %#v", adguard.updated[0])
	}
}

func TestApplyCloudflareAddUpdateAndDeleteActions(t *testing.T) {
	cloudflare := &fakeCloudflareClient{}

	result := Apply(context.Background(), Clients{Cloudflare: cloudflare}, Plan{Actions: []Action{
		{
			Type:              "add",
			Service:           "cloudflare",
			Hostname:          "new.example.com",
			NewService:        "http://192.168.1.15:80",
			NewHTTPHostHeader: "new.example.com",
			Enabled:           true,
		},
		{
			Type:              "update",
			Service:           "cloudflare",
			Hostname:          "update.example.com",
			NewService:        "http://192.168.1.15:80",
			NewHTTPHostHeader: "update.example.com",
			Enabled:           true,
		},
		{
			Type:     "delete",
			Service:  "cloudflare",
			Hostname: "old.example.com",
			Enabled:  true,
		},
	}}, ApplyOptions{})

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
		cloudflare.updatedRules[0].Service != "http://192.168.1.15:80" ||
		cloudflare.updatedRules[0].HTTPHostHeader != "new.example.com" {
		t.Fatalf("unexpected add rule spec: %#v", cloudflare.updatedRules[0])
	}
	if len(cloudflare.ensuredDNS) != 1 || cloudflare.ensuredDNS[0] != "new.example.com" {
		t.Fatalf("expected DNS ensure for add only, got %#v", cloudflare.ensuredDNS)
	}
	if len(cloudflare.deletedRules) != 1 || cloudflare.deletedRules[0] != "old.example.com" {
		t.Fatalf("expected one deleted rule, got %#v", cloudflare.deletedRules)
	}
	if len(cloudflare.deletedDNS) != 1 || cloudflare.deletedDNS[0] != "old.example.com" {
		t.Fatalf("expected one deleted DNS record, got %#v", cloudflare.deletedDNS)
	}
}

func TestApplyCloudflareDryRunDoesNotMutate(t *testing.T) {
	cloudflare := &fakeCloudflareClient{}

	result := Apply(context.Background(), Clients{Cloudflare: cloudflare}, Plan{Actions: []Action{
		{
			Type:              "add",
			Service:           "cloudflare",
			Hostname:          "dry.example.com",
			NewService:        "http://192.168.1.15:80",
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
	added   []api.Rewrite
	updated []fakeAdguardUpdate
	deleted []api.Rewrite
}

func (f *fakeAdguardClient) AddRewrite(domain, answer string) error {
	f.added = append(f.added, api.Rewrite{Domain: domain, Answer: answer})
	return nil
}

func (f *fakeAdguardClient) UpdateRewrite(target, update api.Rewrite) error {
	f.updated = append(f.updated, fakeAdguardUpdate{target: target, update: update})
	return nil
}

func (f *fakeAdguardClient) DeleteRewrite(domain, answer string) error {
	f.deleted = append(f.deleted, api.Rewrite{Domain: domain, Answer: answer})
	return nil
}

type fakeCloudflareClient struct {
	updatedRules []api.IngressRuleSpec
	deletedRules []string
	ensuredDNS   []string
	deletedDNS   []string
}

func (f *fakeCloudflareClient) UpdateTunnelRule(spec api.IngressRuleSpec) error {
	f.updatedRules = append(f.updatedRules, spec)
	return nil
}

func (f *fakeCloudflareClient) DeleteTunnelRule(hostname string) error {
	f.deletedRules = append(f.deletedRules, hostname)
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
