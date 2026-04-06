package config

import (
	"os"
	"testing"
)

func TestLoadCloudflareConfig_FromEnv(t *testing.T) {
	os.Clearenv()
	os.Setenv("HOME", "/tmp")

	os.Setenv(EnvCFEnabled, "true")
	os.Setenv(EnvCFAPIToken, "test-token")
	os.Setenv(EnvCFAccountID, "test-account-id")
	os.Setenv(EnvCFZoneID, "test-zone-id")
	os.Setenv(EnvCFTunnelID, "test-tunnel-id")
	os.Setenv(EnvCFCaddyServiceURL, "http://192.168.1.15:80")
	defer os.Clearenv()

	cfg, err := LoadCloudflareConfig()
	if err != nil {
		t.Fatalf("LoadCloudflareConfig failed: %v", err)
	}

	if !cfg.Enabled {
		t.Errorf("Expected Enabled=true, got %v", cfg.Enabled)
	}
	if cfg.APIToken != "test-token" {
		t.Errorf("Expected APIToken='test-token', got '%s'", cfg.APIToken)
	}
	if cfg.AccountID != "test-account-id" {
		t.Errorf("Expected AccountID='test-account-id', got '%s'", cfg.AccountID)
	}
	if cfg.ZoneID != "test-zone-id" {
		t.Errorf("Expected ZoneID='test-zone-id', got '%s'", cfg.ZoneID)
	}
	if cfg.TunnelID != "test-tunnel-id" {
		t.Errorf("Expected TunnelID='test-tunnel-id', got '%s'", cfg.TunnelID)
	}
	if cfg.CaddyServiceURL != "http://192.168.1.15:80" {
		t.Errorf("Expected CaddyServiceURL='http://192.168.1.15:80', got '%s'", cfg.CaddyServiceURL)
	}
}

func TestLoadCloudflareConfig_Defaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("HOME", "/tmp")
	defer os.Clearenv()

	cfg, err := LoadCloudflareConfig()
	if err != nil {
		t.Fatalf("LoadCloudflareConfig failed: %v", err)
	}

	if cfg.Enabled {
		t.Errorf("Expected default Enabled=false, got %v", cfg.Enabled)
	}
	if cfg.APIToken != "" {
		t.Errorf("Expected empty APIToken, got '%s'", cfg.APIToken)
	}
	if cfg.AccountID != "" {
		t.Errorf("Expected empty AccountID, got '%s'", cfg.AccountID)
	}
	if cfg.ZoneID != "" {
		t.Errorf("Expected empty ZoneID, got '%s'", cfg.ZoneID)
	}
	if cfg.TunnelID != "" {
		t.Errorf("Expected empty TunnelID, got '%s'", cfg.TunnelID)
	}
	if cfg.CaddyServiceURL != "" {
		t.Errorf("Expected empty CaddyServiceURL, got '%s'", cfg.CaddyServiceURL)
	}
}

func TestCloudflareConfigToAPIConfig(t *testing.T) {
	cfgCF := CloudflareConfig{
		Enabled:         true,
		APIToken:        "my-token",
		AccountID:       "my-account",
		ZoneID:          "my-zone",
		TunnelID:        "my-tunnel",
		CaddyServiceURL: "http://caddy:80",
	}

	apiCfg := cfgCF.GetCloudflareAPIConfig()

	if apiCfg.APIToken != "my-token" {
		t.Errorf("Expected APIToken='my-token', got '%s'", apiCfg.APIToken)
	}
	if apiCfg.AccountID != "my-account" {
		t.Errorf("Expected AccountID='my-account', got '%s'", apiCfg.AccountID)
	}
	if apiCfg.ZoneID != "my-zone" {
		t.Errorf("Expected ZoneID='my-zone', got '%s'", apiCfg.ZoneID)
	}
	if apiCfg.TunnelID != "my-tunnel" {
		t.Errorf("Expected TunnelID='my-tunnel', got '%s'", apiCfg.TunnelID)
	}
}
