package config

import (
	"os"
	"testing"
)

func TestAdguardConfigDefaults(t *testing.T) {
	// Clear environment variables to test defaults
	os.Clearenv()
	// Set HOME to prevent error in test environment
	os.Setenv("HOME", "/tmp")

	config, err := LoadAdguardConfig()
	if err != nil {
		t.Fatalf("Failed to load AdguardHome config: %v", err)
	}

	// Test defaults
	if config.Enabled {
		t.Errorf("Expected default Enabled=false, got %v", config.Enabled)
	}

	expectedDesc := "Entry created by unboundCLI adguard-sync"
	if config.Description != expectedDesc {
		t.Errorf("Expected default Description='%s', got '%s'", expectedDesc, config.Description)
	}
}

func TestAdguardConfigEnvironmentVariables(t *testing.T) {
	// Clear environment first
	os.Clearenv()

	// Set environment variables
	os.Setenv(EnvAdguardEnabled, "true")
	os.Setenv(EnvAdguardUsername, "test-user")
	os.Setenv(EnvAdguardPassword, "test-pass")
	os.Setenv(EnvAdguardBaseURL, "http://192.168.0.1:3000")
	os.Setenv(EnvAdguardInsecure, "true")

	config, err := LoadAdguardConfig()
	if err != nil {
		t.Fatalf("Failed to load AdguardHome config: %v", err)
	}

	// Test environment variable loading
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

	// Clean up
	os.Clearenv()
}

func TestAdguardConfigFallbackToMainConfig(t *testing.T) {
	// Clear environment first
	os.Clearenv()

	// Set main config environment variables
	os.Setenv(EnvAdguardEnabled, "1")
	os.Setenv(EnvAPIKey, "main-key")
	os.Setenv(EnvAPISecret, "main-secret")

	config, err := LoadAdguardConfig()
	if err != nil {
		t.Fatalf("Failed to load AdguardHome config: %v", err)
	}

	// Should fall back to main config environment variables
	if config.Username != "main-key" {
		t.Errorf("Expected fallback Username='main-key', got '%s'", config.Username)
	}

	if config.Password != "main-secret" {
		t.Errorf("Expected fallback Password='main-secret', got '%s'", config.Password)
	}

	// Clean up
	os.Clearenv()
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
