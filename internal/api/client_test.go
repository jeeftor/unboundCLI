package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestGenerateCurlCommandRedactsSensitiveHeaders(t *testing.T) {
	curl := generateCurlCommand("GET", "https://opnsense.example/api/test", map[string]string{
		"Authorization": "Basic super-secret-token",
		"X-Api-Key":     "also-secret",
		"Content-Type":  "application/json",
	}, nil)

	for _, secret := range []string{"super-secret-token", "also-secret"} {
		if strings.Contains(curl, secret) {
			t.Fatalf("curl command leaked secret %q: %s", secret, curl)
		}
	}

	for _, want := range []string{
		"Authorization: <redacted>",
		"X-Api-Key: <redacted>",
		"Content-Type: application/json",
	} {
		if !strings.Contains(curl, want) {
			t.Fatalf("curl command missing %q: %s", want, curl)
		}
	}
}

func TestClientGetOverridesUsesFixtureRows(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/unbound/settings/searchHostOverride" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeFixture(t, w, "../status/testdata/unbound_overrides.json")
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:    "fixture-key",
		APISecret: "fixture-secret",
		BaseURL:   server.URL,
		Insecure:  true,
	})
	overrides, err := client.GetOverrides()
	if err != nil {
		t.Fatalf("GetOverrides failed: %v", err)
	}
	if len(overrides) != 2 {
		t.Fatalf("expected two overrides, got %d", len(overrides))
	}
	if overrides[0].UUID != "uuid-app" || overrides[0].Host != "app" {
		t.Fatalf("unexpected first override: %#v", overrides[0])
	}
}

func TestClientMutatingEndpointsUseExpectedPaths(t *testing.T) {
	var paths []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/unbound/settings/searchHostOverride":
			fmt.Fprint(w, `{"rows":[]}`)
		case "/api/unbound/settings/addHostOverride":
			fmt.Fprint(w, `{"result":"saved","uuid":"uuid-new"}`)
		case "/api/unbound/settings/setHostOverride/uuid-existing":
			fmt.Fprint(w, `{"status":"ok"}`)
		case "/api/unbound/settings/delHostOverride/uuid-existing":
			fmt.Fprint(w, `{"result":"deleted"}`)
		case "/api/unbound/service/reconfigure":
			fmt.Fprint(w, `{"status":"ok"}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:    "fixture-key",
		APISecret: "fixture-secret",
		BaseURL:   server.URL,
		Insecure:  true,
	})
	uuid, err := client.AddOverride(DNSOverride{
		Enabled:     "1",
		Host:        "new",
		Domain:      "example.test",
		Server:      "10.0.0.1",
		Description: "fixture",
	})
	if err != nil {
		t.Fatalf("AddOverride failed: %v", err)
	}
	if uuid != "uuid-new" {
		t.Fatalf("expected uuid-new, got %q", uuid)
	}
	if err := client.UpdateOverride(DNSOverride{
		UUID:    "uuid-existing",
		Enabled: "1",
		Host:    "app",
		Domain:  "example.test",
		Server:  "10.0.0.2",
	}); err != nil {
		t.Fatalf("UpdateOverride failed: %v", err)
	}
	if err := client.DeleteOverride("uuid-existing"); err != nil {
		t.Fatalf("DeleteOverride failed: %v", err)
	}
	if err := client.ApplyChanges(); err != nil {
		t.Fatalf("ApplyChanges failed: %v", err)
	}

	for _, want := range []string{
		"GET /api/unbound/settings/searchHostOverride",
		"POST /api/unbound/settings/addHostOverride",
		"POST /api/unbound/settings/setHostOverride/uuid-existing",
		"POST /api/unbound/settings/delHostOverride/uuid-existing",
		"POST /api/unbound/service/reconfigure",
	} {
		if !contains(paths, want) {
			t.Fatalf("missing request %q in %#v", want, paths)
		}
	}
}

func writeFixture(t *testing.T, w http.ResponseWriter, path string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", path, err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
