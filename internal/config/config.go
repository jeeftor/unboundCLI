package config

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/caddyeditor"
	"github.com/spf13/viper"
)

// Revision returns a stable opaque revision for optimistic configuration saves.
// A missing file has its own revision so concurrent first-run saves conflict
// rather than silently overwriting one another.
func Revision(path string) (string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "missing", nil
	}
	if err != nil {
		return "", fmt.Errorf("read config revision: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:]), nil
}

const (
	// DefaultConfigFileName is the default name for the config file
	DefaultConfigFileName = ".caddy-dns-sync.json"

	// Primary environment variable names (preferred).
	// The UNBOUND_CLI_* equivalents below remain supported as deprecated
	// fallbacks for users with existing shell configs.
	EnvAPIKey    = "CADDY_DNS_SYNC_API_KEY"
	EnvAPISecret = "CADDY_DNS_SYNC_API_SECRET"
	EnvBaseURL   = "CADDY_DNS_SYNC_BASE_URL"
	EnvInsecure  = "CADDY_DNS_SYNC_INSECURE"

	// Deprecated: kept as fallback aliases for backwards compatibility.
	// These will be removed in a future release.
	EnvAPIKeyDeprecated    = "UNBOUND_CLI_API_KEY"
	EnvAPISecretDeprecated = "UNBOUND_CLI_API_SECRET"
	EnvBaseURLDeprecated   = "UNBOUND_CLI_BASE_URL"
	EnvInsecureDeprecated  = "UNBOUND_CLI_INSECURE"

	// AdguardHome specific environment variables
	EnvAdguardEnabled  = "ADGUARD_ENABLED"
	EnvAdguardUsername = "ADGUARD_USERNAME"
	EnvAdguardPassword = "ADGUARD_PASSWORD"
	EnvAdguardBaseURL  = "ADGUARD_BASE_URL"
	EnvAdguardInsecure = "ADGUARD_INSECURE"

	// Cloudflare specific environment variables
	EnvCFEnabled         = "CF_ENABLED"
	EnvCFAPIToken        = "CF_API_TOKEN"
	EnvCFAccountID       = "CF_ACCOUNT_ID"
	EnvCFZoneID          = "CF_ZONE_ID"
	EnvCFTunnelID        = "CF_TUNNEL_ID"
	EnvCFCaddyServiceURL = "CF_CADDY_SERVICE_URL"
	EnvCFInsecure        = "CF_INSECURE"

	// Authentik specific environment variables
	EnvAuthentikEnabled  = "AUTHENTIK_ENABLED"
	EnvAuthentikAPIToken = "AUTHENTIK_API_TOKEN"
	EnvAuthentikBaseURL  = "AUTHENTIK_BASE_URL"
	EnvAuthentikInsecure = "AUTHENTIK_INSECURE"
)

// envOr returns the value of the primary env var if set, otherwise the
// fallback (deprecated) env var. Used to migrate from UNBOUND_CLI_* to
// CADDY_DNS_SYNC_* without breaking existing user setups.
func envOr(primary, fallback string) string {
	if v := os.Getenv(primary); v != "" {
		return v
	}
	return os.Getenv(fallback)
}

// envBoolOr returns the bool value of the primary env var if set, otherwise
// the fallback (deprecated) env var. "true" or "1" are treated as true.
func envBoolOr(primary, fallback string) bool {
	v := envOr(primary, fallback)
	return v == "true" || v == "1"
}

// CaddyConfig represents configuration specific to Caddy server integration
type CaddyConfig struct {
	ServerIP   string `json:"server_ip,omitempty" mapstructure:"server_ip"`
	ServerPort int    `json:"server_port,omitempty" mapstructure:"server_port"`
	// AdminHost overrides the host used to reach the Caddy admin API.
	// Useful when caddy-sync runs on the same machine as Caddy (use "127.0.0.1")
	// while ServerIP holds the LAN IP that DNS entries should resolve to.
	// Defaults to ServerIP when empty.
	AdminHost string `json:"admin_host,omitempty" mapstructure:"admin_host"`
}

// AdguardConfig represents configuration specific to AdguardHome integration
type AdguardConfig struct {
	Enabled     bool   `json:"enabled" mapstructure:"enabled"`
	Username    string `json:"username,omitempty" mapstructure:"username"`
	Password    string `json:"password,omitempty" mapstructure:"password"`
	BaseURL     string `json:"base_url,omitempty" mapstructure:"base_url"`
	Insecure    bool   `json:"insecure" mapstructure:"insecure"`
	Description string `json:"description" mapstructure:"description"`
}

// CloudflareConfig represents configuration specific to Cloudflare integration
type CloudflareConfig struct {
	Enabled         bool   `json:"enabled" mapstructure:"enabled"`
	APIToken        string `json:"api_token,omitempty" mapstructure:"api_token"`
	AccountID       string `json:"account_id,omitempty" mapstructure:"account_id"`
	ZoneID          string `json:"zone_id,omitempty" mapstructure:"zone_id"`
	TunnelID        string `json:"tunnel_id,omitempty" mapstructure:"tunnel_id"`
	Insecure        bool   `json:"insecure" mapstructure:"insecure"`
	CaddyServiceURL string `json:"caddy_service_url,omitempty" mapstructure:"caddy_service_url"`
}

// GetCloudflareAPIConfig creates a CloudflareConfig suitable for API client use
func (c CloudflareConfig) GetCloudflareAPIConfig() api.CloudflareConfig {
	return api.CloudflareConfig{
		APIToken:  c.APIToken,
		AccountID: c.AccountID,
		ZoneID:    c.ZoneID,
		TunnelID:  c.TunnelID,
		Insecure:  c.Insecure,
	}
}

// AuthentikConfig represents configuration specific to Authentik integration.
// Authentik is the identity provider used for Caddy forward_auth and OIDC.
// The API token is an admin-level token created via the Authentik admin UI
// (Directory → Tokens → Create Token, intent=API Access Token).
type AuthentikConfig struct {
	Enabled  bool   `json:"enabled" mapstructure:"enabled"`
	APIToken string `json:"api_token,omitempty" mapstructure:"api_token"`
	BaseURL  string `json:"base_url,omitempty" mapstructure:"base_url"`
	Insecure bool   `json:"insecure" mapstructure:"insecure"`
}

// GetAuthentikAPIConfig creates an api.AuthentikConfig suitable for API client use
func (c AuthentikConfig) GetAuthentikAPIConfig() api.AuthentikConfig {
	return api.AuthentikConfig{
		APIToken: c.APIToken,
		BaseURL:  c.BaseURL,
		Insecure: c.Insecure,
	}
}

// ExtendedConfig represents the full application configuration including AdguardHome and Caddy
type ExtendedConfig struct {
	api.Config  `json:",inline" mapstructure:",squash"`
	Caddy       CaddyConfig              `json:"caddy" mapstructure:"caddy"`
	Adguard     AdguardConfig            `json:"adguard" mapstructure:"adguard"`
	Cloudflare  CloudflareConfig         `json:"cloudflare" mapstructure:"cloudflare"`
	Authentik   AuthentikConfig          `json:"authentik" mapstructure:"authentik"`
	CaddyEditor caddyeditor.EditorConfig `json:"caddy_editor" mapstructure:"caddy_editor"`
}

// GetDefaultConfigPath returns the default path for the config file
func GetDefaultConfigPath() (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homedir, DefaultConfigFileName), nil
}

// SelectedConfigPath resolves the one configuration file used by every
// interface. An explicit path wins, followed by the root command's --config
// selection, then the conventional default path.
func SelectedConfigPath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if configured := viper.GetString("config_path"); configured != "" {
		return configured, nil
	}
	if used := viper.ConfigFileUsed(); used != "" {
		return used, nil
	}
	return GetDefaultConfigPath()
}

// LoadExtendedConfigAt loads the selected JSON configuration file without
// applying environment overrides. A missing file is reported to the caller.
func LoadExtendedConfigAt(path string) (ExtendedConfig, error) {
	var cfg ExtendedConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config file: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config file: %w", err)
	}
	return cfg, nil
}

// LoadEffectiveConfig resolves preferred environment variables, deprecated
// aliases, and then the selected file. It never writes configuration or
// contacts providers, so status and dry-run setup stay side-effect free.
func LoadEffectiveConfig(explicitPath string) (ExtendedConfig, string, error) {
	path, err := SelectedConfigPath(explicitPath)
	if err != nil {
		return ExtendedConfig{}, "", err
	}
	cfg := ExtendedConfig{}
	if fileCfg, err := LoadExtendedConfigAt(path); err == nil {
		cfg = fileCfg
	} else if !errors.Is(err, os.ErrNotExist) {
		return ExtendedConfig{}, path, err
	}
	applyEnvironmentOverrides(&cfg)
	if cfg.Adguard.Description == "" {
		cfg.Adguard.Description = "Entry created by caddy-dns-sync adguard-sync"
	}
	return cfg, path, nil
}

func applyEnvironmentOverrides(cfg *ExtendedConfig) {
	if envOr(EnvAPIKey, EnvAPIKeyDeprecated) != "" {
		cfg.APIKey = envOr(EnvAPIKey, EnvAPIKeyDeprecated)
	}
	if envOr(EnvAPISecret, EnvAPISecretDeprecated) != "" {
		cfg.APISecret = envOr(EnvAPISecret, EnvAPISecretDeprecated)
	}
	if envOr(EnvBaseURL, EnvBaseURLDeprecated) != "" {
		cfg.BaseURL = envOr(EnvBaseURL, EnvBaseURLDeprecated)
	}
	if os.Getenv(EnvInsecure) != "" || os.Getenv(EnvInsecureDeprecated) != "" {
		cfg.Insecure = envBoolOr(EnvInsecure, EnvInsecureDeprecated)
	}

	if enabled := os.Getenv(EnvAdguardEnabled); enabled != "" {
		cfg.Adguard.Enabled = enabled == "true" || enabled == "1"
		if username := os.Getenv(EnvAdguardUsername); username != "" {
			cfg.Adguard.Username = username
		} else if fallback := envOr(EnvAPIKey, EnvAPIKeyDeprecated); fallback != "" {
			cfg.Adguard.Username = fallback
		}
		if password := os.Getenv(EnvAdguardPassword); password != "" {
			cfg.Adguard.Password = password
		} else if fallback := envOr(EnvAPISecret, EnvAPISecretDeprecated); fallback != "" {
			cfg.Adguard.Password = fallback
		}
		if baseURL := os.Getenv(EnvAdguardBaseURL); baseURL != "" {
			cfg.Adguard.BaseURL = baseURL
		}
		if insecure := os.Getenv(EnvAdguardInsecure); insecure != "" {
			cfg.Adguard.Insecure = insecure == "true" || insecure == "1"
		}
	}

	if enabled := os.Getenv(EnvCFEnabled); enabled != "" {
		cfg.Cloudflare.Enabled = enabled == "true" || enabled == "1"
		if value := os.Getenv(EnvCFAPIToken); value != "" {
			cfg.Cloudflare.APIToken = value
		}
		if value := os.Getenv(EnvCFAccountID); value != "" {
			cfg.Cloudflare.AccountID = value
		}
		if value := os.Getenv(EnvCFZoneID); value != "" {
			cfg.Cloudflare.ZoneID = value
		}
		if value := os.Getenv(EnvCFTunnelID); value != "" {
			cfg.Cloudflare.TunnelID = value
		}
		if value := os.Getenv(EnvCFCaddyServiceURL); value != "" {
			cfg.Cloudflare.CaddyServiceURL = value
		}
		if value := os.Getenv(EnvCFInsecure); value != "" {
			cfg.Cloudflare.Insecure = value == "true" || value == "1"
		}
	}

	if enabled := os.Getenv(EnvAuthentikEnabled); enabled != "" {
		cfg.Authentik.Enabled = enabled == "true" || enabled == "1"
		if value := os.Getenv(EnvAuthentikAPIToken); value != "" {
			cfg.Authentik.APIToken = value
		}
		if value := os.Getenv(EnvAuthentikBaseURL); value != "" {
			cfg.Authentik.BaseURL = value
		}
		if value := os.Getenv(EnvAuthentikInsecure); value != "" {
			cfg.Authentik.Insecure = value == "true" || value == "1"
		}
	}
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

	if err := atomicWriteFile(filePath, jsonData, 0o600); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// LoadConfig loads the API configuration from environment variables, Viper, or a file
func LoadConfig() (api.Config, error) {
	cfg, _, err := LoadEffectiveConfig("")
	if err != nil {
		return api.Config{}, err
	}
	if cfg.APIKey == "" || cfg.APISecret == "" || cfg.BaseURL == "" {
		return api.Config{}, fmt.Errorf("no complete Unbound configuration found; set environment variables or configure the selected file")
	}
	return cfg.Config, nil
}

// LoadAdguardConfig loads AdguardHome-specific configuration from environment variables, viper, or config file
func LoadAdguardConfig() (AdguardConfig, error) {
	cfg, _, err := LoadEffectiveConfig("")
	if err != nil {
		return AdguardConfig{}, err
	}
	return cfg.Adguard, nil
}

// LoadCloudflareConfig loads Cloudflare-specific configuration from environment variables, viper, or config file
func LoadCloudflareConfig() (CloudflareConfig, error) {
	cfg, _, err := LoadEffectiveConfig("")
	if err != nil {
		return CloudflareConfig{}, err
	}
	return cfg.Cloudflare, nil
}

// LoadAuthentikConfig loads Authentik-specific configuration from environment
// variables, viper, or config file. Authentik is optional — if not configured,
// the returned config will have Enabled=false.
func LoadAuthentikConfig() (AuthentikConfig, error) {
	cfg, _, err := LoadEffectiveConfig("")
	if err != nil {
		return AuthentikConfig{}, err
	}
	return cfg.Authentik, nil
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
	data, err := marshalExtendedConfigPreservingUnknown(cfg, path)
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	return atomicWriteFile(path, data, 0o600)
}

// LoadExtendedConfig loads the extended configuration (including Caddy and AdguardHome) from the default config file
func LoadExtendedConfig() (ExtendedConfig, error) {
	path, err := SelectedConfigPath("")
	if err != nil {
		return ExtendedConfig{}, err
	}
	return LoadExtendedConfigAt(path)
}

func marshalExtendedConfigPreservingUnknown(cfg ExtendedConfig, path string) ([]byte, error) {
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal extended config: %w", err)
	}
	var replacement map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &replacement); err != nil {
		return nil, fmt.Errorf("decode encoded config: %w", err)
	}
	current := map[string]json.RawMessage{}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &current); err != nil {
			return nil, fmt.Errorf("parse existing config: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read existing config: %w", err)
	}
	merged := mergeJSONObject(current, replacement)
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal merged config: %w", err)
	}
	return append(data, '\n'), nil
}

func mergeJSONObject(current, replacement map[string]json.RawMessage) map[string]json.RawMessage {
	merged := make(map[string]json.RawMessage, len(current)+len(replacement))
	for key, value := range current {
		merged[key] = value
	}
	for key, replacementValue := range replacement {
		var oldObject, newObject map[string]json.RawMessage
		if json.Unmarshal(merged[key], &oldObject) == nil && json.Unmarshal(replacementValue, &newObject) == nil {
			encoded, _ := json.Marshal(mergeJSONObject(oldObject, newObject))
			merged[key] = encoded
			continue
		}
		merged[key] = replacementValue
	}
	return merged
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".caddy-dns-sync-*")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set temporary config mode: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary config: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace config atomically: %w", err)
	}
	return nil
}

// WriteRawConfig validates and atomically replaces a complete JSON document.
// It is reserved for the explicit raw-editor endpoint; structured saves use
// SaveExtendedConfig so unknown fields remain intact.
func WriteRawConfig(path string, data []byte) error {
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse raw config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	pretty, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("format raw config: %w", err)
	}
	return atomicWriteFile(path, append(pretty, '\n'), 0o600)
}
