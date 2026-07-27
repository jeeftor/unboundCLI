package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
)

func TestMigrateUnboundDescriptions_NilClientIsNoOp(t *testing.T) {
	// Should not panic on nil client.
	MigrateUnboundDescriptions(nil)
}

func TestMigrateUnboundDescriptions_RewritesLegacyDescriptions(t *testing.T) {
	var (
		mu       sync.Mutex
		updates  []map[string]api.DNSOverride
		setPaths []string
	)

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/unbound/settings/searchHostOverride":
			// Return a mix of legacy, current, and foreign descriptions.
			overrides := []api.DNSOverride{
				{UUID: "uuid-legacy-1", Host: "app", Domain: "example.com", Server: "10.0.0.1", Description: "Entry created by unboundCLI caddy-sync-all"},
				{UUID: "uuid-legacy-2", Host: "wiki", Domain: "example.com", Server: "10.0.0.2", Description: "Entry created by CaddySync"},
				{UUID: "uuid-current", Host: "grafana", Domain: "example.com", Server: "10.0.0.3", Description: "Managed by caddy-dns-sync"},
				{UUID: "uuid-foreign", Host: "manual", Domain: "example.com", Server: "10.0.0.4", Description: "Manually configured entry"},
			}
			rows, _ := json.Marshal(overrides)
			fmt.Fprintf(w, `{"rows":%s}`, string(rows))

		case "/api/unbound/settings/setHostOverride/uuid-legacy-1":
			mu.Lock()
			setPaths = append(setPaths, r.URL.Path)
			var body map[string]api.DNSOverride
			_ = json.NewDecoder(r.Body).Decode(&body)
			updates = append(updates, body)
			mu.Unlock()
			fmt.Fprint(w, `{"status":"ok"}`)

		case "/api/unbound/settings/setHostOverride/uuid-legacy-2":
			mu.Lock()
			setPaths = append(setPaths, r.URL.Path)
			var body map[string]api.DNSOverride
			_ = json.NewDecoder(r.Body).Decode(&body)
			updates = append(updates, body)
			mu.Unlock()
			fmt.Fprint(w, `{"status":"ok"}`)

		default:
			// setHostOverride for current/foreign entries should NOT be called.
			t.Errorf("unexpected request to %s", r.URL.Path)
			fmt.Fprint(w, `{"status":"ok"}`)
		}
	}))
	defer server.Close()

	client := api.NewClient(api.Config{
		APIKey:    "test-key",
		APISecret: "test-secret",
		BaseURL:   server.URL,
		Insecure:  true,
	})

	MigrateUnboundDescriptions(client)

	mu.Lock()
	defer mu.Unlock()

	if len(updates) != 2 {
		t.Fatalf("expected 2 update calls (legacy entries only), got %d", len(updates))
	}
	for _, u := range updates {
		entry, ok := u["host"]
		if !ok {
			t.Fatalf("update body missing 'host' key: %#v", u)
		}
		if entry.Description != CurrentUnboundDescription {
			t.Errorf("expected migrated description %q, got %q", CurrentUnboundDescription, entry.Description)
		}
	}
}

func TestIsLegacyUnboundDescription(t *testing.T) {
	tests := []struct {
		desc string
		want bool
	}{
		{"Entry created by unboundCLI caddy-sync-all", true},
		{"Entry created by unboundCLI sync", true},
		{"Entry created by unboundCLI caddy-sync-unbound", true},
		{"Entry created by CaddySync", true},
		{"Route via Caddy", true},
		{"Managed by caddy-dns-sync", false},
		{"Manually configured entry", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isLegacyUnboundDescription(tt.desc)
		if got != tt.want {
			t.Errorf("isLegacyUnboundDescription(%q) = %v, want %v", tt.desc, got, tt.want)
		}
	}
}
