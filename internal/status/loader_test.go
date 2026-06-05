package status

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/syncplan"
)

func TestLoadEntriesReportsMissingCaddyClient(t *testing.T) {
	entries, report, err := LoadEntries(context.Background(), app.ClientSet{}, Options{
		CaddyServerIP: "192.168.1.15",
	})
	if err == nil {
		t.Fatal("expected missing Caddy client error")
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(entries))
	}
	caddy := report.Services[ServiceCaddy]
	if caddy.Status != ServiceFailed {
		t.Fatalf("expected Caddy failed status, got %s", caddy.Status)
	}
	if caddy.Error == "" {
		t.Fatal("expected Caddy error message in report")
	}
	if report.Services[ServiceDNS].Status == ServicePending {
		t.Fatal("expected DNS report to have terminal status after Caddy failure")
	}
}

func TestLoadEntriesUsesLocalCaddyFixtureAndReportsOptionalServices(t *testing.T) {
	caddy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config/" {
			t.Fatalf("unexpected Caddy path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"apps": {
				"http": {
					"servers": {
						"srv0": {
							"routes": [
								{
									"match": [{"host": ["app.example.test"]}],
									"handle": [
										{
											"handler": "reverse_proxy",
											"upstreams": [{"dial": "10.0.0.5:8080"}]
										}
									]
								}
							]
						}
					}
				}
			}
		}`)
	}))
	defer caddy.Close()

	host, port := splitServerHostPort(t, caddy.URL)
	entries, report, err := LoadEntries(context.Background(), app.ClientSet{
		Caddy: api.NewCaddyClient(host, port),
	}, Options{
		CaddyServerIP: "10.0.0.1",
	})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	if entries[0].Hostname != "app.example.test" {
		t.Fatalf("expected app.example.test, got %q", entries[0].Hostname)
	}
	if report.Services[ServiceCaddy].Status != ServiceLoaded {
		t.Fatalf("expected Caddy loaded status, got %s", report.Services[ServiceCaddy].Status)
	}
	if report.Services[ServiceAdguard].Status != ServiceSkipped {
		t.Fatalf("expected missing AdGuard client to be skipped, got %s", report.Services[ServiceAdguard].Status)
	}
}

func TestLoadPlanApplyWithNoLANFixtures(t *testing.T) {
	caddy := httptest.NewServer(fixtureHandler(t, map[string]string{
		"/config/": "testdata/caddy_config.json",
	}))
	defer caddy.Close()

	opnsense := httptest.NewTLSServer(fixtureHandler(t, map[string]string{
		"/api/unbound/settings/searchHostOverride": "testdata/unbound_overrides.json",
		"/api/dnsmasq/leases/search":               "testdata/dhcp_leases.json",
	}))
	defer opnsense.Close()

	adguard := httptest.NewServer(fixtureHandler(t, map[string]string{
		"/control/rewrite/list": "testdata/adguard_rewrites.json",
	}))
	defer adguard.Close()

	caddyHost, caddyPort := splitServerHostPort(t, caddy.URL)
	var events []ProgressEvent
	entries, report, err := LoadEntries(context.Background(), app.ClientSet{
		Caddy: api.NewCaddyClient(caddyHost, caddyPort),
		Unbound: api.NewClient(api.Config{
			APIKey:    "fixture-key",
			APISecret: "fixture-secret",
			BaseURL:   opnsense.URL,
			Insecure:  true,
		}),
		DNSMasq: api.NewDNSMasqClient(api.Config{
			APIKey:    "fixture-key",
			APISecret: "fixture-secret",
			BaseURL:   opnsense.URL,
			Insecure:  true,
		}),
		Adguard: api.NewAdguardClient(api.AdguardConfig{
			BaseURL:  adguard.URL,
			Username: "fixture-user",
			Password: "fixture-pass",
		}),
	}, Options{
		CaddyServerIP: "10.0.0.1",
		Progress: func(event ProgressEvent) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected three entries from fixtures, got %d", len(entries))
	}
	for _, service := range []ServiceName{ServiceCaddy, ServiceUnbound, ServiceAdguard, ServiceDHCP, ServiceDNS} {
		if report.Services[service].Status != ServiceLoaded {
			t.Fatalf("expected %s loaded status, got %#v", service, report.Services[service])
		}
	}
	if report.Services[ServiceDHCP].Count != 2 {
		t.Fatalf("expected DHCP report to count raw leases, got %#v", report.Services[ServiceDHCP])
	}
	assertProgressEvent(t, events, ServiceCaddy, ServiceLoaded)
	assertProgressEvent(t, events, ServiceDNS, ServiceLoaded)

	unboundPlan := syncplan.BuildPlan(entries, syncplan.Options{
		Service:       "unbound",
		CaddyServerIP: "10.0.0.1",
	})
	if len(unboundPlan.Actions) != 2 {
		t.Fatalf("expected two unbound actions, got %#v", unboundPlan.Actions)
	}
	unbound := &fixtureApplyUnbound{
		overrides: []api.DNSOverride{
			{UUID: "uuid-app", Host: "app", Domain: "example.test", Server: "10.0.0.1"},
			{UUID: "uuid-old", Host: "old", Domain: "example.test", Server: "10.0.0.1"},
		},
	}
	unboundResult := syncplan.Apply(context.Background(), syncplan.Clients{Unbound: unbound}, unboundPlan, syncplan.ApplyOptions{})
	if !unboundResult.Success {
		t.Fatalf("expected unbound apply success, got %#v", unboundResult)
	}
	if len(unbound.added) != 1 || len(unbound.deleted) != 1 || !unbound.reconfigured {
		t.Fatalf("unexpected unbound mutations: added=%#v deleted=%#v reconfigured=%t", unbound.added, unbound.deleted, unbound.reconfigured)
	}

	adguardPlan := syncplan.BuildPlan(entries, syncplan.Options{
		Service:       "adguard",
		CaddyServerIP: "10.0.0.1",
	})
	if len(adguardPlan.Actions) != 1 {
		t.Fatalf("expected one adguard action, got %#v", adguardPlan.Actions)
	}
	adguardApply := &fixtureApplyAdguard{}
	adguardResult := syncplan.Apply(context.Background(), syncplan.Clients{Adguard: adguardApply}, adguardPlan, syncplan.ApplyOptions{})
	if !adguardResult.Success {
		t.Fatalf("expected adguard apply success, got %#v", adguardResult)
	}
	if len(adguardApply.updated) != 1 {
		t.Fatalf("expected one adguard update, got %#v", adguardApply.updated)
	}
}

func TestLoadEntriesHonorsCancelledContextBeforeLoading(t *testing.T) {
	var requests atomic.Int32
	caddy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer caddy.Close()

	host, port := splitServerHostPort(t, caddy.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	entries, _, err := LoadEntries(ctx, app.ClientSet{
		Caddy: api.NewCaddyClient(host, port),
	}, Options{
		CaddyServerIP: "10.0.0.1",
	})
	if err == nil {
		t.Fatal("expected cancelled context error")
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries after cancelled context, got %d", len(entries))
	}
	if requests.Load() != 0 {
		t.Fatalf("expected no Caddy request after cancelled context, got %d", requests.Load())
	}
}

func splitServerHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}
	host, portString, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("failed to split server host/port: %v", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatalf("failed to parse server port: %v", err)
	}
	return host, port
}

func fixtureHandler(t *testing.T, fixtures map[string]string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixturePath, ok := fixtures[r.URL.Path]
		if !ok {
			t.Fatalf("unexpected fixture request path %s", r.URL.Path)
		}
		body, err := os.ReadFile(fixturePath)
		if err != nil {
			t.Fatalf("failed to read fixture %s: %v", fixturePath, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
}

func assertProgressEvent(t *testing.T, events []ProgressEvent, service ServiceName, status ServiceState) {
	t.Helper()
	for _, event := range events {
		if event.Service == service && event.Status == status {
			return
		}
	}
	t.Fatalf("missing progress event %s/%s in %#v", service, status, events)
}

type fixtureApplyUnbound struct {
	overrides    []api.DNSOverride
	added        []api.DNSOverride
	updated      []api.DNSOverride
	deleted      []string
	reconfigured bool
}

func (f *fixtureApplyUnbound) GetOverrides() ([]api.DNSOverride, error) {
	return f.overrides, nil
}

func (f *fixtureApplyUnbound) AddOverride(override api.DNSOverride) (string, error) {
	f.added = append(f.added, override)
	return "new-uuid", nil
}

func (f *fixtureApplyUnbound) UpdateOverride(override api.DNSOverride) error {
	f.updated = append(f.updated, override)
	return nil
}

func (f *fixtureApplyUnbound) DeleteOverride(uuid string) error {
	f.deleted = append(f.deleted, uuid)
	return nil
}

func (f *fixtureApplyUnbound) ApplyChanges() error {
	f.reconfigured = true
	return nil
}

type fixtureRewriteUpdate struct {
	target api.Rewrite
	update api.Rewrite
}

type fixtureApplyAdguard struct {
	added   []api.Rewrite
	updated []fixtureRewriteUpdate
	deleted []api.Rewrite
}

func (f *fixtureApplyAdguard) AddRewrite(domain, answer string) error {
	f.added = append(f.added, api.Rewrite{Domain: domain, Answer: answer})
	return nil
}

func (f *fixtureApplyAdguard) UpdateRewrite(target, update api.Rewrite) error {
	f.updated = append(f.updated, fixtureRewriteUpdate{target: target, update: update})
	return nil
}

func (f *fixtureApplyAdguard) DeleteRewrite(domain, answer string) error {
	f.deleted = append(f.deleted, api.Rewrite{Domain: domain, Answer: answer})
	return nil
}
