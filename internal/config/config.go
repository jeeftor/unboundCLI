package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/spf13/viper"
)

const (
	// DefaultConfigFileName is the default name for the config file
	DefaultConfigFileName = ".unboundCLI.json"

	// Environment variable names
	EnvAPIKey    = "UNBOUND_CLI_API_KEY"
	EnvAPISecret = "UNBOUND_CLI_API_SECRET"
	EnvBaseURL   = "UNBOUND_CLI_BASE_URL"
	EnvInsecure  = "UNBOUND_CLI_INSECURE"

	// AdguardHome specific environment variables
	EnvAdguardEnabled  = "ADGUARD_ENABLED"
	EnvAdguardUsername = "ADGUARD_USERNAME"
	EnvAdguardPassword = "ADGUARD_PASSWORD"
	EnvAdguardBaseURL  = "ADGUARD_BASE_URL"
	EnvAdguardInsecure = "ADGUARD_INSECURE"
)

// AdguardConfig represents configuration specific to AdguardHome integration
type AdguardConfig struct {
	Enabled     bool   `json:"enabled" mapstructure:"enabled"`
	Username    string `json:"username,omitempty" mapstructure:"username"`
	Password    string `json:"password,omitempty" mapstructure:"password"`
	BaseURL     string `json:"base_url,omitempty" mapstructure:"base_url"`
	Insecure    bool   `json:"insecure" mapstructure:"insecure"`
	Description string `json:"description" mapstructure:"description"`
}

// ExtendedConfig represents the full application configuration including AdguardHome
type ExtendedConfig struct {
	api.Config `json:",inline" mapstructure:",squash"`
	Adguard    AdguardConfig `json:"adguard" mapstructure:"adguard"`
}

// GetDefaultConfigPath returns the default path for the config file
func GetDefaultConfigPath() (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homedir, DefaultConfigFileName), nil
}

// SaveConfig saves the API configuration to a file
func SaveConfig(config api.Config, filePath string) error {
	// If no path is provided, use the default
	if filePath == "" {
		var err error
		filePath, err = GetDefaultConfigPath()
		if err != nil {
			return err
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, jsonData, 0o600); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// LoadConfig loads the API configuration from environment variables, Viper, or a file
func LoadConfig() (api.Config, error) {
	var config api.Config

	// First check environment variables
	if apiKey := os.Getenv(EnvAPIKey); apiKey != "" {
		config.APIKey = apiKey
		config.APISecret = os.Getenv(EnvAPISecret)
		config.BaseURL = os.Getenv(EnvBaseURL)
		config.Insecure = os.Getenv(EnvInsecure) == "true" || os.Getenv(EnvInsecure) == "1"

		// Validate required fields
		if config.APISecret != "" && config.BaseURL != "" {
			return config, nil
		}
	}

	// Then try to load from viper
	if viper.IsSet("api_key") && viper.IsSet("api_secret") && viper.IsSet("base_url") {
		config.APIKey = viper.GetString("api_key")
		config.APISecret = viper.GetString("api_secret")
		config.BaseURL = viper.GetString("base_url")
		config.Insecure = viper.GetBool("insecure")
		return config, nil
	}

	// Finally try to load from JSON file
	configPath, err := GetDefaultConfigPath()
	if err != nil {
		return config, err
	}

	// Check if the config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, fmt.Errorf(
			"no configuration found, please run 'config' command or set environment variables",
		)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("error reading config file: %w", err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("error parsing config file: %w", err)
	}

	// Store in viper for future use
	viper.Set("api_key", config.APIKey)
	viper.Set("api_secret", config.APISecret)
	viper.Set("base_url", config.BaseURL)
	viper.Set("insecure", config.Insecure)

	return config, nil
}

// LoadAdguardConfig loads AdguardHome-specific configuration from environment variables, viper, or config file
func LoadAdguardConfig() (AdguardConfig, error) {
	var config AdguardConfig

	// Set defaults
	config.Enabled = false
	config.Description = "Entry created by unboundCLI adguard-sync"

	// Check environment variables first
	if enabledEnv := os.Getenv(EnvAdguardEnabled); enabledEnv != "" {
		config.Enabled = enabledEnv == "true" || enabledEnv == "1"

		// Load AdguardHome-specific credentials or fall back to main config
		if username := os.Getenv(EnvAdguardUsername); username != "" {
			config.Username = username
		} else {
			config.Username = os.Getenv(EnvAPIKey) // Fallback to main API key
		}

		if password := os.Getenv(EnvAdguardPassword); password != "" {
			config.Password = password
		} else {
			config.Password = os.Getenv(EnvAPISecret) // Fallback to main API secret
		}

		if baseURL := os.Getenv(EnvAdguardBaseURL); baseURL != "" {
			config.BaseURL = baseURL
		}

		if insecure := os.Getenv(EnvAdguardInsecure); insecure != "" {
			config.Insecure = insecure == "true" || insecure == "1"
		}

		return config, nil
	}

	// Try to load from viper
	if viper.IsSet("adguard") {
		if err := viper.UnmarshalKey("adguard", &config); err != nil {
			return config, fmt.Errorf("error parsing AdguardHome config from viper: %w", err)
		}
		return config, nil
	}

	// Try to load from config file
	configPath, err := GetDefaultConfigPath()
	if err != nil {
		return config, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config file doesn't exist, return defaults
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("error reading config file: %w", err)
	}

	var extendedConfig ExtendedConfig
	if err := json.Unmarshal(data, &extendedConfig); err != nil {
		return config, fmt.Errorf("error parsing extended config file: %w", err)
	}

	config = extendedConfig.Adguard

	// Store in viper for future use
	viper.Set("adguard", config)

	return config, nil
}

// GetAdguardAPIConfig creates an AdguardConfig from the configuration suitable for API client use
func (a AdguardConfig) GetAdguardAPIConfig() api.AdguardConfig {
	return api.AdguardConfig{
		BaseURL:  a.BaseURL,
		Username: a.Username,
		Password: a.Password,
		Insecure: a.Insecure,
		Enabled:  a.Enabled,
	}
}

// SaveExtendedConfig saves the extended configuration (including AdguardHome) to a file
func SaveExtendedConfig(cfg ExtendedConfig, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling extended config: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}
