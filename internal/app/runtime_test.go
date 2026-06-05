package app

import (
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
)

func TestNewRuntimeFromConfigsBuildsCoreClientsWithDefaults(t *testing.T) {
	runtime, err := NewRuntimeFromConfigs(api.Config{
		APIKey:    "key",
		APISecret: "secret",
		BaseURL:   "https://opnsense.example",
	}, config.AdguardConfig{}, config.CloudflareConfig{}, RuntimeOptions{
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
	if runtime.CaddyServiceURL != "http://192.168.1.15:80" {
		t.Fatalf("unexpected Caddy service URL %q", runtime.CaddyServiceURL)
	}
}

func TestNewRuntimeFromConfigsUsesCaddyOverridesAndCloudflareServiceURL(t *testing.T) {
	runtime, err := NewRuntimeFromConfigs(api.Config{}, config.AdguardConfig{}, config.CloudflareConfig{
		CaddyServiceURL: "http://caddy.internal:8080",
	}, RuntimeOptions{
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
	}, config.CloudflareConfig{}, RuntimeOptions{
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
	}, config.CloudflareConfig{}, RuntimeOptions{
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
	}, RuntimeOptions{
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
	}, RuntimeOptions{
		IncludeCloudflare: true,
	})
	if err != nil {
		t.Fatalf("NewRuntimeFromConfigs failed: %v", err)
	}

	if runtime.Clients.Cloudflare != nil {
		t.Fatal("expected disabled Cloudflare config to skip client creation")
	}
}
