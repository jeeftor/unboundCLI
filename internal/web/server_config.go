package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/caddyeditor"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/spf13/viper"
)

// ─── Config Types ───────────────────────────────────────────────────────────

type CaddyConfigResponse struct {
	ServerIP   string `json:"server_ip"`
	ServerPort int    `json:"server_port"`
}

type ConfigResponse struct {
	Caddy           CaddyConfigResponse      `json:"caddy"`
	Enabled         map[string]bool          `json:"enabled"`
	MutationEnabled bool                     `json:"mutation_enabled"`
	SaveTarget      string                   `json:"save_target"`
	Summary         ConfigSummary            `json:"summary"`
	CaddyEditor     caddyeditor.EditorConfig `json:"caddy_editor"`
	Version         string                   `json:"version"`
}

type ConfigSummary struct {
	Caddy      ConfigServiceSummary `json:"caddy"`
	Unbound    ConfigServiceSummary `json:"unbound"`
	Adguard    ConfigServiceSummary `json:"adguard"`
	DHCP       ConfigServiceSummary `json:"dhcp"`
	Cloudflare ConfigServiceSummary `json:"cloudflare"`
}

type ConfigServiceSummary struct {
	Label       string            `json:"label"`
	Enabled     bool              `json:"enabled"`
	ClientReady bool              `json:"client_ready"`
	Source      ConfigSource      `json:"source"`
	Endpoint    string            `json:"endpoint,omitempty"`
	Insecure    bool              `json:"insecure,omitempty"`
	Fields      map[string]bool   `json:"fields,omitempty"`
	Details     map[string]string `json:"details,omitempty"`
	Missing     []string          `json:"missing,omitempty"`
}

type ConfigSource struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Path  string `json:"path,omitempty"`
}

type ConfigUpdateRequest struct {
	Unbound     *UnboundConfigUpdate     `json:"unbound,omitempty"`
	Adguard     *AdguardConfigUpdate     `json:"adguard,omitempty"`
	Cloudflare  *CloudflareConfigUpdate  `json:"cloudflare,omitempty"`
	CaddyEditor *CaddyEditorConfigUpdate `json:"caddy_editor,omitempty"`
}

type CaddyEditorConfigUpdate struct {
	Enabled         *bool   `json:"enabled,omitempty"`
	RepoPath        *string `json:"repo_path,omitempty"`
	CaddyfilePath   *string `json:"caddyfile,omitempty"`
	DeployCommand   *string `json:"deploy_command,omitempty"`
	ValidateCommand *string `json:"validate_command,omitempty"`
	GitAutoCommit   *bool   `json:"git_auto_commit,omitempty"`
	GitAutoPush     *bool   `json:"git_auto_push,omitempty"`
	GitRemote       *string `json:"git_remote,omitempty"`
	GitBranch       *string `json:"git_branch,omitempty"`
	EntryTemplate   *string `json:"entry_template,omitempty"`
}

type ConfigTestRequest struct {
	Service string `json:"service"`
}

type ConfigTestResponse struct {
	Service string            `json:"service"`
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type UnboundConfigUpdate struct {
	APIKey    string  `json:"api_key,omitempty"`
	APISecret string  `json:"api_secret,omitempty"`
	BaseURL   *string `json:"base_url,omitempty"`
	Insecure  *bool   `json:"insecure,omitempty"`
}

type AdguardConfigUpdate struct {
	Enabled  *bool   `json:"enabled,omitempty"`
	Username string  `json:"username,omitempty"`
	Password string  `json:"password,omitempty"`
	BaseURL  *string `json:"base_url,omitempty"`
	Insecure *bool   `json:"insecure,omitempty"`
}

type CloudflareConfigUpdate struct {
	Enabled         *bool   `json:"enabled,omitempty"`
	APIToken        string  `json:"api_token,omitempty"`
	AccountID       *string `json:"account_id,omitempty"`
	ZoneID          *string `json:"zone_id,omitempty"`
	TunnelID        *string `json:"tunnel_id,omitempty"`
	Insecure        *bool   `json:"insecure,omitempty"`
	CaddyServiceURL *string `json:"caddy_service_url,omitempty"`
}

type sourceProbe string

const (
	sourceProbeUnbound    sourceProbe = "unbound"
	sourceProbeAdguard    sourceProbe = "adguard"
	sourceProbeCloudflare sourceProbe = "cloudflare"
)

// ─── Config Handlers ────────────────────────────────────────────────────────

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		resp, err := s.configResponse()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := s.allowMutation(r); err != nil {
			writeError(w, http.StatusForbidden, err)
			return
		}
		var request ConfigUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid config update request: %w", err))
			return
		}
		resp, err := s.applyConfigUpdate(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	default:
		writeMethodNotAllowed(w)
	}
}

// handleConfigRaw serves GET /api/config/raw (read) and POST /api/config/raw (write).
// The raw JSON of the config file is returned/accepted as a string field so the caller
// can display and edit it verbatim without any schema translation.
func (s *Server) handleConfigRaw(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		path, err := s.configPath()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			data = []byte("{}")
		} else if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("read config: %w", err))
			return
		}
		// Pretty-print so the editor shows indented JSON
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, data, "", "  "); err != nil {
			pretty.Write(data) // fall back to raw
		}
		writeJSON(w, http.StatusOK, map[string]string{"raw": pretty.String(), "path": path})

	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
		if err := s.allowMutation(r); err != nil {
			writeError(w, http.StatusForbidden, err)
			return
		}
		var req struct {
			Raw string `json:"raw"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
			return
		}
		// Validate: must be parseable as ExtendedConfig
		var cfg config.ExtendedConfig
		if err := json.Unmarshal([]byte(req.Raw), &cfg); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
			return
		}
		path, err := s.configPath()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		// Write pretty-printed
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, []byte(req.Raw), "", "  "); err != nil {
			pretty.WriteString(req.Raw)
		}
		if err := os.WriteFile(path, pretty.Bytes(), 0600); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("write config: %w", err))
			return
		}
		if err := s.reloadRuntimeFromConfig(cfg); err != nil {
			logging.Warn("Failed to reload runtime after config save", "error", err)
		}
		writeJSON(w, http.StatusOK, map[string]string{"path": path, "status": "saved"})

	default:
		writeMethodNotAllowed(w)
	}
}

func (s *Server) handleConfigTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := s.allowMutation(r); err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	var request ConfigTestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid config test request: %w", err))
		return
	}
	resp := s.testConfigService(strings.ToLower(strings.TrimSpace(request.Service)))
	writeJSON(w, http.StatusOK, resp)
}

// ─── Config Helpers ─────────────────────────────────────────────────────────

func (s *Server) configResponse() (ConfigResponse, error) {
	saveTarget, err := s.configPath()
	if err != nil {
		return ConfigResponse{}, err
	}
	runtime := s.runtimeSnapshot()
	return ConfigResponse{
		Caddy: CaddyConfigResponse{
			ServerIP:   runtime.CaddyEndpoint.ServerIP,
			ServerPort: runtime.CaddyEndpoint.ServerPort,
		},
		Enabled: map[string]bool{
			"caddy":      runtime.Clients.Caddy != nil,
			"unbound":    runtime.Clients.Unbound != nil,
			"adguard":    runtime.Clients.Adguard != nil,
			"dhcp":       runtime.Clients.DNSMasq != nil,
			"cloudflare": runtime.Clients.Cloudflare != nil,
		},
		MutationEnabled: s.mutationsEnabled(),
		SaveTarget:      saveTarget,
		Summary:         s.configSummary(&runtime),
		CaddyEditor:     s.loadCaddyEditorConfig(),
		Version:         s.options.Version,
	}, nil
}

func (s *Server) loadCaddyEditorConfig() caddyeditor.EditorConfig {
	path, err := s.configPath()
	if err != nil {
		return caddyeditor.DefaultEditorConfig()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return caddyeditor.DefaultEditorConfig()
	}
	var wrapper struct {
		CaddyEditor caddyeditor.EditorConfig `json:"caddy_editor"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return caddyeditor.DefaultEditorConfig()
	}
	return wrapper.CaddyEditor
}

func (s *Server) testConfigService(service string) ConfigTestResponse {
	runtime := s.runtimeSnapshot()
	switch service {
	case "caddy":
		if runtime.Clients.Caddy == nil {
			return failedConfigTest(service, "Caddy is not configured.")
		}
		cfg, err := runtime.Clients.Caddy.GetConfig()
		if err != nil {
			return failedConfigTest(service, fmt.Sprintf("Caddy test failed: %v", err))
		}
		return ConfigTestResponse{
			Service: service,
			Success: true,
			Message: "Connected to Caddy admin API.",
			Details: map[string]string{
				"endpoint": fmt.Sprintf("%s:%d", runtime.CaddyEndpoint.ServerIP, runtime.CaddyEndpoint.ServerPort),
				"sections": fmt.Sprintf("%d", len(cfg)),
			},
		}
	case "unbound":
		if runtime.Clients.Unbound == nil {
			return failedConfigTest(service, "OPNSense / Unbound is not configured.")
		}
		overrides, err := runtime.Clients.Unbound.GetOverrides()
		if err != nil {
			return failedConfigTest(service, fmt.Sprintf("OPNSense / Unbound test failed: %v", err))
		}
		return ConfigTestResponse{
			Service: service,
			Success: true,
			Message: "Connected to OPNSense Unbound API.",
			Details: map[string]string{"overrides": fmt.Sprintf("%d", len(overrides))},
		}
	case "adguard":
		if runtime.Clients.Adguard == nil {
			return failedConfigTest(service, "AdGuard is not configured.")
		}
		rewrites, err := runtime.Clients.Adguard.ListRewrites()
		if err != nil {
			return failedConfigTest(service, fmt.Sprintf("AdGuard test failed: %v", err))
		}
		return ConfigTestResponse{
			Service: service,
			Success: true,
			Message: "Connected to AdGuard rewrite API.",
			Details: map[string]string{"rewrites": fmt.Sprintf("%d", len(rewrites))},
		}
	case "cloudflare":
		if runtime.Clients.Cloudflare == nil {
			return failedConfigTest(service, "Cloudflare is not configured.")
		}
		zones, err := runtime.Clients.Cloudflare.ListZones()
		if err != nil {
			return failedConfigTest(service, fmt.Sprintf("Cloudflare test failed: %v", err))
		}
		return ConfigTestResponse{
			Service: service,
			Success: true,
			Message: "Connected to Cloudflare API.",
			Details: map[string]string{"zones": fmt.Sprintf("%d", len(zones))},
		}
	default:
		return failedConfigTest(service, fmt.Sprintf("Unknown config service %q.", service))
	}
}

func failedConfigTest(service, message string) ConfigTestResponse {
	return ConfigTestResponse{Service: service, Success: false, Message: message}
}

func (s *Server) configSummary(runtime *app.Runtime) ConfigSummary {
	configPath, _ := s.configPath()
	unboundMissing := missingFields(map[string]bool{
		"API key":    runtime.UnboundConfig.APIKey != "",
		"API secret": runtime.UnboundConfig.APISecret != "",
		"Base URL":   runtime.UnboundConfig.BaseURL != "",
	})
	adguardMissing := missingFields(map[string]bool{
		"Enabled":  runtime.AdguardConfig.Enabled,
		"Base URL": runtime.AdguardConfig.BaseURL != "",
		"Username": runtime.AdguardConfig.Username != "",
		"Password": runtime.AdguardConfig.Password != "",
	})
	cloudflareMissing := []string{}
	if runtime.CloudflareConfig.Enabled {
		cloudflareMissing = missingFields(map[string]bool{
			"API token":  runtime.CloudflareConfig.APIToken != "",
			"Account ID": runtime.CloudflareConfig.AccountID != "",
			"Zone ID":    runtime.CloudflareConfig.ZoneID != "",
			"Tunnel ID":  runtime.CloudflareConfig.TunnelID != "",
		})
	}
	caddyEndpoint := fmt.Sprintf("%s:%d", runtime.CaddyEndpoint.ServerIP, runtime.CaddyEndpoint.ServerPort)
	unboundEndpoint := sanitizeEndpoint(runtime.UnboundConfig.BaseURL)
	adguardEndpoint := sanitizeEndpoint(runtime.AdguardConfig.BaseURL)
	caddyServiceURL := sanitizeEndpoint(runtime.CaddyServiceURL)
	return ConfigSummary{
		Caddy: ConfigServiceSummary{
			Label:       "Caddy",
			Enabled:     runtime.Clients.Caddy != nil,
			ClientReady: runtime.Clients.Caddy != nil,
			Source:      ConfigSource{Kind: "cli", Label: "CLI flags/defaults"},
			Endpoint:    caddyEndpoint,
		},
		Unbound: ConfigServiceSummary{
			Label:       "OPNSense / Unbound",
			Enabled:     runtime.UnboundConfig.BaseURL != "",
			ClientReady: runtime.Clients.Unbound != nil,
			Source:      s.configSource(configPath, sourceProbeUnbound),
			Endpoint:    unboundEndpoint,
			Insecure:    runtime.UnboundConfig.Insecure,
			Fields: map[string]bool{
				"api_key_set":    runtime.UnboundConfig.APIKey != "",
				"api_secret_set": runtime.UnboundConfig.APISecret != "",
				"base_url_set":   runtime.UnboundConfig.BaseURL != "",
			},
			Missing: unboundMissing,
		},
		Adguard: ConfigServiceSummary{
			Label:       "AdGuard",
			Enabled:     runtime.AdguardConfig.Enabled,
			ClientReady: runtime.Clients.Adguard != nil,
			Source:      s.configSource(configPath, sourceProbeAdguard),
			Endpoint:    adguardEndpoint,
			Insecure:    runtime.AdguardConfig.Insecure,
			Fields: map[string]bool{
				"username_set": runtime.AdguardConfig.Username != "",
				"password_set": runtime.AdguardConfig.Password != "",
				"base_url_set": runtime.AdguardConfig.BaseURL != "",
			},
			Missing: adguardMissing,
		},
		DHCP: ConfigServiceSummary{
			Label:       "DHCP / DNSMasq",
			Enabled:     runtime.Clients.DNSMasq != nil,
			ClientReady: runtime.Clients.DNSMasq != nil,
			Source:      s.configSource(configPath, sourceProbeUnbound),
			Endpoint:    unboundEndpoint,
		},
		Cloudflare: ConfigServiceSummary{
			Label:       "Cloudflare",
			Enabled:     runtime.CloudflareConfig.Enabled,
			ClientReady: runtime.Clients.Cloudflare != nil,
			Source:      s.configSource(configPath, sourceProbeCloudflare),
			Insecure:    runtime.CloudflareConfig.Insecure,
			Fields: map[string]bool{
				"api_token_set":  runtime.CloudflareConfig.APIToken != "",
				"account_id_set": runtime.CloudflareConfig.AccountID != "",
				"zone_id_set":    runtime.CloudflareConfig.ZoneID != "",
				"tunnel_id_set":  runtime.CloudflareConfig.TunnelID != "",
			},
			Details: map[string]string{
				"caddy_service_url": caddyServiceURL,
			},
			Missing: cloudflareMissing,
		},
	}
}

func (s *Server) configSource(configPath string, probe sourceProbe) ConfigSource {
	switch probe {
	case sourceProbeUnbound:
		if os.Getenv(config.EnvAPIKey) != "" && os.Getenv(config.EnvAPISecret) != "" && os.Getenv(config.EnvBaseURL) != "" {
			return ConfigSource{Kind: "env", Label: "Environment variables"}
		}
		if viper.IsSet("api_key") && viper.IsSet("api_secret") && viper.IsSet("base_url") {
			if used := viper.ConfigFileUsed(); used != "" {
				return ConfigSource{Kind: "config-file", Label: "Viper config file", Path: used}
			}
			return ConfigSource{Kind: "cli", Label: "Viper/CLI values"}
		}
	case sourceProbeAdguard:
		if os.Getenv(config.EnvAdguardEnabled) != "" {
			return ConfigSource{Kind: "env", Label: "Environment variables"}
		}
		if viper.IsSet("adguard") {
			if used := viper.ConfigFileUsed(); used != "" {
				return ConfigSource{Kind: "config-file", Label: "Viper config file", Path: used}
			}
			return ConfigSource{Kind: "cli", Label: "Viper/CLI values"}
		}
	case sourceProbeCloudflare:
		if os.Getenv(config.EnvCFEnabled) != "" {
			return ConfigSource{Kind: "env", Label: "Environment variables"}
		}
		if viper.IsSet("cloudflare") {
			if used := viper.ConfigFileUsed(); used != "" {
				return ConfigSource{Kind: "config-file", Label: "Viper config file", Path: used}
			}
			return ConfigSource{Kind: "cli", Label: "Viper/CLI values"}
		}
	}
	if configPath != "" {
		if s.configFileHasService(configPath, probe) {
			return ConfigSource{Kind: "config-file", Label: "Config file", Path: configPath}
		}
	}
	return ConfigSource{Kind: "default", Label: "Defaults"}
}

func (s *Server) configFileHasService(configPath string, probe sourceProbe) bool {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	var cfg config.ExtendedConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false
	}
	switch probe {
	case sourceProbeUnbound:
		return cfg.APIKey != "" || cfg.APISecret != "" || cfg.BaseURL != ""
	case sourceProbeAdguard:
		return cfg.Adguard.Enabled || cfg.Adguard.BaseURL != "" || cfg.Adguard.Username != "" || cfg.Adguard.Password != ""
	case sourceProbeCloudflare:
		return cfg.Cloudflare.Enabled ||
			cfg.Cloudflare.APIToken != "" ||
			cfg.Cloudflare.AccountID != "" ||
			cfg.Cloudflare.ZoneID != "" ||
			cfg.Cloudflare.TunnelID != "" ||
			cfg.Cloudflare.CaddyServiceURL != ""
	default:
		return false
	}
}

func (s *Server) applyConfigUpdate(request ConfigUpdateRequest) (ConfigResponse, error) {
	configPath, err := s.configPath()
	if err != nil {
		return ConfigResponse{}, err
	}
	cfg, err := s.loadWritableConfig(configPath)
	if err != nil {
		return ConfigResponse{}, err
	}
	if request.Unbound != nil {
		applyUnboundConfigUpdate(&cfg.Config, request.Unbound)
	}
	if request.Adguard != nil {
		applyAdguardConfigUpdate(&cfg.Adguard, request.Adguard)
	}
	if request.Cloudflare != nil {
		applyCloudflareConfigUpdate(&cfg.Cloudflare, request.Cloudflare)
	}
	if request.CaddyEditor != nil {
		applyCaddyEditorConfigUpdate(&cfg.CaddyEditor, request.CaddyEditor)
	}
	if err := config.SaveExtendedConfig(cfg, configPath); err != nil {
		return ConfigResponse{}, err
	}
	if err := s.reloadRuntimeFromConfig(cfg); err != nil {
		return ConfigResponse{}, err
	}
	return s.configResponse()
}

func (s *Server) configPath() (string, error) {
	if s.options.ConfigPath != "" {
		return s.options.ConfigPath, nil
	}
	return config.GetDefaultConfigPath()
}

func (s *Server) loadWritableConfig(configPath string) (config.ExtendedConfig, error) {
	if data, err := os.ReadFile(configPath); err == nil {
		var cfg config.ExtendedConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("error parsing config file: %w", err)
		}
		return cfg, nil
	} else if !os.IsNotExist(err) {
		return config.ExtendedConfig{}, fmt.Errorf("error reading config file: %w", err)
	}
	runtime := s.runtimeSnapshot()
	return config.ExtendedConfig{
		Config:     runtime.UnboundConfig,
		Caddy:      config.CaddyConfig{ServerIP: runtime.CaddyEndpoint.ServerIP, ServerPort: runtime.CaddyEndpoint.ServerPort},
		Adguard:    runtime.AdguardConfig,
		Cloudflare: runtime.CloudflareConfig,
	}, nil
}

func applyUnboundConfigUpdate(cfg *api.Config, update *UnboundConfigUpdate) {
	if update.APIKey != "" {
		cfg.APIKey = update.APIKey
	}
	if update.APISecret != "" {
		cfg.APISecret = update.APISecret
	}
	if update.BaseURL != nil {
		cfg.BaseURL = strings.TrimSpace(*update.BaseURL)
	}
	if update.Insecure != nil {
		cfg.Insecure = *update.Insecure
	}
}

func applyAdguardConfigUpdate(cfg *config.AdguardConfig, update *AdguardConfigUpdate) {
	if update.Enabled != nil {
		cfg.Enabled = *update.Enabled
	}
	if update.Username != "" {
		cfg.Username = update.Username
	}
	if update.Password != "" {
		cfg.Password = update.Password
	}
	if update.BaseURL != nil {
		cfg.BaseURL = strings.TrimSpace(*update.BaseURL)
	}
	if update.Insecure != nil {
		cfg.Insecure = *update.Insecure
	}
	if cfg.Description == "" {
		cfg.Description = "Entry created by caddy-dns-sync adguard-sync"
	}
}

func applyCloudflareConfigUpdate(cfg *config.CloudflareConfig, update *CloudflareConfigUpdate) {
	if update.Enabled != nil {
		cfg.Enabled = *update.Enabled
	}
	if update.APIToken != "" {
		cfg.APIToken = update.APIToken
	}
	if update.AccountID != nil {
		cfg.AccountID = strings.TrimSpace(*update.AccountID)
	}
	if update.ZoneID != nil {
		cfg.ZoneID = strings.TrimSpace(*update.ZoneID)
	}
	if update.TunnelID != nil {
		cfg.TunnelID = strings.TrimSpace(*update.TunnelID)
	}
	if update.Insecure != nil {
		cfg.Insecure = *update.Insecure
	}
	if update.CaddyServiceURL != nil {
		cfg.CaddyServiceURL = strings.TrimSpace(*update.CaddyServiceURL)
	}
}

func applyCaddyEditorConfigUpdate(cfg *caddyeditor.EditorConfig, update *CaddyEditorConfigUpdate) {
	if update.Enabled != nil {
		cfg.Enabled = *update.Enabled
	}
	if update.RepoPath != nil {
		cfg.RepoPath = strings.TrimSpace(*update.RepoPath)
	}
	if update.CaddyfilePath != nil {
		cfg.CaddyfilePath = strings.TrimSpace(*update.CaddyfilePath)
	}
	if update.DeployCommand != nil {
		cfg.DeployCommand = strings.TrimSpace(*update.DeployCommand)
	}
	if update.ValidateCommand != nil {
		cfg.ValidateCommand = strings.TrimSpace(*update.ValidateCommand)
	}
	if update.GitAutoCommit != nil {
		cfg.GitAutoCommit = *update.GitAutoCommit
	}
	if update.GitAutoPush != nil {
		cfg.GitAutoPush = *update.GitAutoPush
	}
	if update.GitRemote != nil {
		cfg.GitRemote = strings.TrimSpace(*update.GitRemote)
	}
	if update.GitBranch != nil {
		cfg.GitBranch = strings.TrimSpace(*update.GitBranch)
	}
	if update.EntryTemplate != nil {
		cfg.EntryTemplate = strings.TrimSpace(*update.EntryTemplate)
	}
}

func (s *Server) reloadRuntimeFromConfig(cfg config.ExtendedConfig) error {
	current := s.runtimeSnapshot()
	nextRuntime, err := app.NewRuntimeFromConfigs(cfg.Config, cfg.Adguard, cfg.Cloudflare, cfg.Authentik, app.RuntimeOptions{
		CaddyServerIP:     current.CaddyEndpoint.ServerIP,
		CaddyServerPort:   current.CaddyEndpoint.ServerPort,
		IncludeUnbound:    true,
		IncludeDNSMasq:    current.Clients.DNSMasq != nil,
		IncludeAdguard:    true,
		IncludeCloudflare: true,
		IncludeAuthentik:  true,
	})
	if err != nil {
		return fmt.Errorf("error refreshing runtime from saved config: %w", err)
	}
	s.runtimeMu.Lock()
	s.runtime = nextRuntime
	s.runtimeMu.Unlock()
	return nil
}

func sanitizeEndpoint(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.User == nil {
		return endpoint
	}
	parsed.User = nil
	return parsed.String()
}

func missingFields(fields map[string]bool) []string {
	missing := make([]string, 0)
	for field, present := range fields {
		if !present {
			missing = append(missing, field)
		}
	}
	return missing
}
