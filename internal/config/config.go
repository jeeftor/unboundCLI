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
)

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
