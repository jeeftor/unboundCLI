package config

import (
	"testing"
)

// clearConfigEnvVars unsets all config-related env vars for the duration of the
// test using t.Setenv, which automatically restores them when the test ends.
// This replaces the old os.Clearenv() pattern that wiped the entire process
// environment and could pollute parallel tests.
func clearConfigEnvVars(t *testing.T) {
	t.Helper()
	vars := []string{
		EnvAPIKey, EnvAPISecret, EnvBaseURL, EnvInsecure,
		EnvAPIKeyDeprecated, EnvAPISecretDeprecated, EnvBaseURLDeprecated, EnvInsecureDeprecated,
		EnvAdguardEnabled, EnvAdguardUsername, EnvAdguardPassword, EnvAdguardBaseURL, EnvAdguardInsecure,
		EnvCFEnabled, EnvCFAPIToken, EnvCFAccountID, EnvCFZoneID, EnvCFTunnelID, EnvCFCaddyServiceURL,
		EnvAuthentikEnabled, EnvAuthentikAPIToken, EnvAuthentikBaseURL, EnvAuthentikInsecure,
	}
	for _, v := range vars {
		t.Setenv(v, "")
	}
}

func TestAdguardConfigDefaults(t *testing.T) {
	clearConfigEnvVars(t)

	config, err := LoadAdguardConfig()
	if err != nil {
		t.Fatalf("Failed to load AdguardHome config: %v", err)
	}

	// Test defaults
	if config.Enabled {
		t.Errorf("Expected default Enabled=false, got %v", config.Enabled)
	}

	expectedDesc := "Entry created by caddy-dns-sync adguard-sync"
	if config.Description != expectedDesc {
		t.Errorf("Expected default Description='%s', got '%s'", expectedDesc, config.Description)
	}
}

func TestLoadConfig_NewEnvVars(t *testing.T) {
	clearConfigEnvVars(t)

	t.Setenv(EnvAPIKey, "new-key")
	t.Setenv(EnvAPISecret, "new-secret")
	t.Setenv(EnvBaseURL, "https://192.168.1.1")
	t.Setenv(EnvInsecure, "true")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.APIKey != "new-key" {
		t.Errorf("Expected APIKey='new-key', got '%s'", cfg.APIKey)
	}
	if cfg.APISecret != "new-secret" {
		t.Errorf("Expected APISecret='new-secret', got '%s'", cfg.APISecret)
	}
	if cfg.BaseURL != "https://192.168.1.1" {
		t.Errorf("Expected BaseURL='https://192.168.1.1', got '%s'", cfg.BaseURL)
	}
	if !cfg.Insecure {
		t.Errorf("Expected Insecure=true, got %v", cfg.Insecure)
	}
}

func TestLoadConfig_DeprecatedEnvVarFallback(t *testing.T) {
	clearConfigEnvVars(t)

	// Set ONLY the deprecated UNBOUND_CLI_* names — new names unset.
	t.Setenv(EnvAPIKeyDeprecated, "old-key")
	t.Setenv(EnvAPISecretDeprecated, "old-secret")
	t.Setenv(EnvBaseURLDeprecated, "https://10.0.0.1")
	t.Setenv(EnvInsecureDeprecated, "1")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.APIKey != "old-key" {
		t.Errorf("Expected fallback APIKey='old-key', got '%s'", cfg.APIKey)
	}
	if cfg.APISecret != "old-secret" {
		t.Errorf("Expected fallback APISecret='old-secret', got '%s'", cfg.APISecret)
	}
	if cfg.BaseURL != "https://10.0.0.1" {
		t.Errorf("Expected fallback BaseURL='https://10.0.0.1', got '%s'", cfg.BaseURL)
	}
	if !cfg.Insecure {
		t.Errorf("Expected fallback Insecure=true, got %v", cfg.Insecure)
	}
}

func TestLoadConfig_NewEnvVarsTakePrecedenceOverDeprecated(t *testing.T) {
	clearConfigEnvVars(t)

	// Set both new and deprecated; new must win.
	t.Setenv(EnvAPIKey, "new-key")
	t.Setenv(EnvAPIKeyDeprecated, "old-key")
	t.Setenv(EnvAPISecret, "new-secret")
	t.Setenv(EnvAPISecretDeprecated, "old-secret")
	t.Setenv(EnvBaseURL, "https://new.example")
	t.Setenv(EnvBaseURLDeprecated, "https://old.example")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.APIKey != "new-key" {
		t.Errorf("Expected new APIKey to win, got '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "https://new.example" {
		t.Errorf("Expected new BaseURL to win, got '%s'", cfg.BaseURL)
	}
}

func TestAdguardConfigEnvironmentVariables(t *testing.T) {
	clearConfigEnvVars(t)

	t.Setenv(EnvAdguardEnabled, "true")
	t.Setenv(EnvAdguardUsername, "test-user")
	t.Setenv(EnvAdguardPassword, "test-pass")
	t.Setenv(EnvAdguardBaseURL, "http://192.168.0.1:3000")
	t.Setenv(EnvAdguardInsecure, "true")

	config, err := LoadAdguardConfig()
	if err != nil {
		t.Fatalf("Failed to load AdguardHome config: %v", err)
	}

	if !config.Enabled {
		t.Errorf("Expected Enabled=true from env var, got %v", config.Enabled)
	}
	if config.Username != "test-user" {
		t.Errorf("Expected Username='test-user', got '%s'", config.Username)
	}
	if config.Password != "test-pass" {
		t.Errorf("Expected Password='test-pass', got '%s'", config.Password)
	}
	if config.BaseURL != "http://192.168.0.1:3000" {
		t.Errorf("Expected BaseURL='http://192.168.0.1:3000', got '%s'", config.BaseURL)
	}
	if !config.Insecure {
		t.Errorf("Expected Insecure=true from env var, got %v", config.Insecure)
	}
}

func TestAdguardConfigFallbackToMainConfig(t *testing.T) {
	clearConfigEnvVars(t)

	t.Setenv(EnvAdguardEnabled, "1")
	t.Setenv(EnvAPIKey, "main-key")
	t.Setenv(EnvAPISecret, "main-secret")

	config, err := LoadAdguardConfig()
	if err != nil {
		t.Fatalf("Failed to load AdguardHome config: %v", err)
	}

	if config.Username != "main-key" {
		t.Errorf("Expected fallback Username='main-key', got '%s'", config.Username)
	}
	if config.Password != "main-secret" {
		t.Errorf("Expected fallback Password='main-secret', got '%s'", config.Password)
	}
}

func TestGetAdguardAPIConfig(t *testing.T) {
	adguardConfig := AdguardConfig{
		Enabled:  true,
		Username: "test-user",
		Password: "test-pass",
		BaseURL:  "http://192.168.0.1:3000",
		Insecure: true,
	}

	apiConfig := adguardConfig.GetAdguardAPIConfig()

	if apiConfig.Username != "test-user" {
		t.Errorf("Expected Username='test-user', got '%s'", apiConfig.Username)
	}
	if apiConfig.Password != "test-pass" {
		t.Errorf("Expected Password='test-pass', got '%s'", apiConfig.Password)
	}
	if apiConfig.BaseURL != "http://192.168.0.1:3000" {
		t.Errorf("Expected BaseURL='http://192.168.0.1:3000', got '%s'", apiConfig.BaseURL)
	}
	if !apiConfig.Insecure {
		t.Errorf("Expected Insecure=true, got %v", apiConfig.Insecure)
	}
	if !apiConfig.Enabled {
		t.Errorf("Expected Enabled=true, got %v", apiConfig.Enabled)
	}
}
