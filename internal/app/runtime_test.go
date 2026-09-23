package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
)

func TestNewRuntimeFromConfigsBuildsCoreClientsWithDefaults(t *testing.T) {
	runtime, err := NewRuntimeFromConfigs(api.Config{
		APIKey:    "key",
		APISecret: "secret",
		BaseURL:   "https://opnsense.example",
	}, config.AdguardConfig{}, config.CloudflareConfig{}, config.AuthentikConfig{}, RuntimeOptions{
		IncludeUnbound: true,
		IncludeDNSMasq: true,
	})
	if err != nil {
		t.Fatalf("NewRuntimeFromConfigs failed: %v", err)
	}

	if runtime.Clients.Caddy == nil {
		t.Fatal("expected Caddy client")
	}
	if runtime.Clients.Unbound == nil {
		t.Fatal("expected Unbound client")
	}
	if runtime.Clients.DNSMasq == nil {
		t.Fatal("expected DNSMasq client")
	}
	if runtime.CaddyEndpoint.ServerIP != DefaultCaddyServerIP {
		t.Fatalf("expected default Caddy IP %q, got %q", DefaultCaddyServerIP, runtime.CaddyEndpoint.ServerIP)
	}
	if runtime.CaddyEndpoint.ServerPort != DefaultCaddyServerPort {
		t.Fatalf("expected default Caddy port %d, got %d", DefaultCaddyServerPort, runtime.CaddyEndpoint.ServerPort)
	}
	if runtime.CaddyServiceURL != "http://10.0.0.15:80" {
		t.Fatalf("unexpected Caddy service URL %q", runtime.CaddyServiceURL)
	}
}

func TestLoadRuntimeUsesExplicitSelectedConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "selected.json")
	data := []byte(`{"api_key":"selected-key","api_secret":"selected-secret","base_url":"https://selected.example.test","caddy":{"server_ip":"10.10.0.8","server_port":2022,"admin_host":"caddy-admin.example.test"}}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write selected config: %v", err)
	}
	t.Setenv(config.EnvAPIKey, "")
	t.Setenv(config.EnvAPISecret, "")
	t.Setenv(config.EnvBaseURL, "")
	t.Setenv(config.EnvAPIKeyDeprecated, "")
	t.Setenv(config.EnvAPISecretDeprecated, "")
	t.Setenv(config.EnvBaseURLDeprecated, "")

	runtime, err := LoadRuntime(RuntimeOptions{ConfigPath: path, IncludeUnbound: true})
	if err != nil {
		t.Fatalf("load runtime: %v", err)
	}
	if runtime.Clients.Unbound == nil || runtime.CaddyEndpoint.ServerIP != "10.10.0.8" || runtime.CaddyEndpoint.ServerPort != 2022 || runtime.CaddyEndpoint.AdminHost != "caddy-admin.example.test" {
		t.Fatalf("runtime did not use selected file: %#v", runtime)
	}
}

func TestNewRuntimeFromConfigsDoesNotContactUnboundAtStartup(t *testing.T) {
	// Client construction must remain pure. Ownership migration happens only in
	// explicit, reviewed operations, never while displaying status or dry-run.
	runtime, err := NewRuntimeFromConfigs(api.Config{
		APIKey: "key", APISecret: "secret", BaseURL: "https://127.0.0.1:1",
	}, config.AdguardConfig{}, config.CloudflareConfig{}, config.AuthentikConfig{}, RuntimeOptions{IncludeUnbound: true})
	if err != nil {
		t.Fatalf("construct runtime: %v", err)
	}
	if runtime.Clients.Unbound == nil {
		t.Fatal("expected configured Unbound client")
	}
}

func TestNewRuntimeFromConfigsUsesCaddyOverridesAndCloudflareServiceURL(t *testing.T) {
	runtime, err := NewRuntimeFromConfigs(api.Config{}, config.AdguardConfig{}, config.CloudflareConfig{
		CaddyServiceURL: "http://caddy.internal:8080",
	}, config.AuthentikConfig{}, RuntimeOptions{
		CaddyServerIP:   "10.0.0.10",
		CaddyServerPort: 2020,
	})
	if err != nil {
		t.Fatalf("NewRuntimeFromConfigs failed: %v", err)
	}

	if runtime.CaddyEndpoint.ServerIP != "10.0.0.10" {
		t.Fatalf("expected Caddy IP override, got %q", runtime.CaddyEndpoint.ServerIP)
	}
	if runtime.CaddyEndpoint.ServerPort != 2020 {
		t.Fatalf("expected Caddy port override, got %d", runtime.CaddyEndpoint.ServerPort)
	}
	if runtime.CaddyServiceURL != "http://caddy.internal:8080" {
		t.Fatalf("expected configured Caddy service URL, got %q", runtime.CaddyServiceURL)
	}
}

func TestNewRuntimeFromConfigsBuildsOptionalAdguardWhenComplete(t *testing.T) {
	runtime, err := NewRuntimeFromConfigs(api.Config{}, config.AdguardConfig{
		Enabled:  true,
		BaseURL:  "http://adguard.example",
		Username: "user",
		Password: "pass",
	}, config.CloudflareConfig{}, config.AuthentikConfig{}, RuntimeOptions{
		IncludeAdguard: true,
	})
	if err != nil {
		t.Fatalf("NewRuntimeFromConfigs failed: %v", err)
	}

	if runtime.Clients.Adguard == nil {
		t.Fatal("expected Adguard client")
	}
}

func TestNewRuntimeFromConfigsRequiresAdguardWhenRequested(t *testing.T) {
	_, err := NewRuntimeFromConfigs(api.Config{}, config.AdguardConfig{
		Enabled: true,
		BaseURL: "http://adguard.example",
	}, config.CloudflareConfig{}, config.AuthentikConfig{}, RuntimeOptions{
		IncludeAdguard: true,
		RequireAdguard: true,
	})
	if err == nil {
		t.Fatal("expected error for incomplete required Adguard config")
	}
}

func TestNewRuntimeFromConfigsBuildsCloudflareFromCredentials(t *testing.T) {
	runtime, err := NewRuntimeFromConfigs(api.Config{}, config.AdguardConfig{}, config.CloudflareConfig{
		Enabled:   true,
		APIToken:  "token",
		AccountID: "account-id",
		ZoneID:    "zone-id",
		TunnelID:  "tunnel-id",
	}, config.AuthentikConfig{}, RuntimeOptions{
		IncludeCloudflare: true,
	})
	if err != nil {
		t.Fatalf("NewRuntimeFromConfigs failed: %v", err)
	}

	if runtime.Clients.Cloudflare == nil {
		t.Fatal("expected Cloudflare client")
	}
}

func TestNewRuntimeFromConfigsSkipsDisabledCloudflare(t *testing.T) {
	runtime, err := NewRuntimeFromConfigs(api.Config{}, config.AdguardConfig{}, config.CloudflareConfig{
		Enabled:   false,
		APIToken:  "token",
		AccountID: "account-id",
		ZoneID:    "zone-id",
		TunnelID:  "tunnel-id",
	}, config.AuthentikConfig{}, RuntimeOptions{
		IncludeCloudflare: true,
	})
	if err != nil {
		t.Fatalf("NewRuntimeFromConfigs failed: %v", err)
	}

	if runtime.Clients.Cloudflare != nil {
		t.Fatal("expected disabled Cloudflare config to skip client creation")
	}
}
