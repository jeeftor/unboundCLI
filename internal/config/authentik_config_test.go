package config

import (
	"testing"
)

func TestLoadAuthentikConfig_FromEnv(t *testing.T) {
	clearConfigEnvVars(t)

	t.Setenv(EnvAuthentikEnabled, "true")
	t.Setenv(EnvAuthentikAPIToken, "test-token")
	t.Setenv(EnvAuthentikBaseURL, "https://auth.example.com")
	t.Setenv(EnvAuthentikInsecure, "false")

	cfg, err := LoadAuthentikConfig()
	if err != nil {
		t.Fatalf("LoadAuthentikConfig failed: %v", err)
	}

	if !cfg.Enabled {
		t.Errorf("Expected Enabled=true, got %v", cfg.Enabled)
	}
	if cfg.APIToken != "test-token" {
		t.Errorf("Expected APIToken='test-token', got '%s'", cfg.APIToken)
	}
	if cfg.BaseURL != "https://auth.example.com" {
		t.Errorf("Expected BaseURL='https://auth.example.com', got '%s'", cfg.BaseURL)
	}
	if cfg.Insecure {
		t.Errorf("Expected Insecure=false, got %v", cfg.Insecure)
	}
}

func TestLoadAuthentikConfig_Defaults(t *testing.T) {
	clearConfigEnvVars(t)

	// Also clear Authentik-specific env vars explicitly
	t.Setenv(EnvAuthentikEnabled, "")
	t.Setenv(EnvAuthentikAPIToken, "")
	t.Setenv(EnvAuthentikBaseURL, "")
	t.Setenv(EnvAuthentikInsecure, "")

	cfg, err := LoadAuthentikConfig()
	if err != nil {
		t.Fatalf("LoadAuthentikConfig failed: %v", err)
	}

	if cfg.Enabled {
		t.Errorf("Expected default Enabled=false, got %v", cfg.Enabled)
	}
	if cfg.APIToken != "" {
		t.Errorf("Expected empty APIToken, got '%s'", cfg.APIToken)
	}
	if cfg.BaseURL != "" {
		t.Errorf("Expected empty BaseURL, got '%s'", cfg.BaseURL)
	}
	if cfg.Insecure {
		t.Errorf("Expected default Insecure=false, got %v", cfg.Insecure)
	}
}

func TestAuthentikConfigToAPIConfig(t *testing.T) {
	cfgAuth := AuthentikConfig{
		Enabled:  true,
		APIToken: "my-token",
		BaseURL:  "https://auth.example.com",
		Insecure: true,
	}

	apiCfg := cfgAuth.GetAuthentikAPIConfig()

	if apiCfg.APIToken != "my-token" {
		t.Errorf("Expected APIToken='my-token', got '%s'", apiCfg.APIToken)
	}
	if apiCfg.BaseURL != "https://auth.example.com" {
		t.Errorf("Expected BaseURL='https://auth.example.com', got '%s'", apiCfg.BaseURL)
	}
	if !apiCfg.Insecure {
		t.Errorf("Expected Insecure=true, got %v", apiCfg.Insecure)
	}
}
