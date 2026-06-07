package web

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
	"github.com/jeeftor/caddy-dns-sync/internal/status"
	"github.com/jeeftor/caddy-dns-sync/internal/syncplan"
	"github.com/spf13/viper"
)

//go:embed static/*
var staticFiles embed.FS

type Options struct {
	ApplyToken      string
	AllowMutations  bool
	AllowedOrigin   string
	AllowUnsafeBind bool
	BoundHost       string
	EnableTestHooks bool
	ConfigPath      string
}

type Server struct {
	runtime   *app.Runtime
	options   Options
	mux       *http.ServeMux
	runtimeMu sync.RWMutex
	planMu    sync.Mutex
	plans     map[string]storedPlan
}

type storedPlan struct {
	ActionsByID map[string]syncplan.Action
}

type CaddyConfigResponse struct {
	ServerIP   string `json:"server_ip"`
	ServerPort int    `json:"server_port"`
}

type ConfigResponse struct {
	Caddy           CaddyConfigResponse `json:"caddy"`
	Enabled         map[string]bool     `json:"enabled"`
	MutationEnabled bool                `json:"mutation_enabled"`
	SaveTarget      string              `json:"save_target"`
	Summary         ConfigSummary       `json:"summary"`
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
	Unbound    *UnboundConfigUpdate    `json:"unbound,omitempty"`
	Adguard    *AdguardConfigUpdate    `json:"adguard,omitempty"`
	Cloudflare *CloudflareConfigUpdate `json:"cloudflare,omitempty"`
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

type CloudflareDiscoverRequest struct {
	Token     string `json:"token"`
	AccountID string `json:"account_id"`
}

type CloudflareDiscoverResponse struct {
	Accounts []api.CloudflareAccount `json:"accounts"`
	Tunnels  []api.CloudflareTunnel  `json:"tunnels"`
	Zones    []api.CloudflareZone    `json:"zones"`
	Error    string                  `json:"error,omitempty"`
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

type EntriesResponse struct {
	Entries []EntryResponse   `json:"entries"`
	Report  status.LoadReport `json:"report"`
}

type ServiceStatusResponse struct {
	Configured bool   `json:"configured"`
	IP         string `json:"ip"`
	InSync     bool   `json:"in_sync"`
}

type DHCPStatusResponse struct {
	Configured bool   `json:"configured"`
	Type       string `json:"type"`
	IP         string `json:"ip"`
	MAC        string `json:"mac"`
	Hostname   string `json:"hostname"`
	InSync     bool   `json:"in_sync"`
}

type CloudflareStatusResponse struct {
	Configured      bool   `json:"configured"`
	TunnelName      string `json:"tunnel_name"`
	TunnelID        string `json:"tunnel_id"`
	Service         string `json:"service"`
	Path            string `json:"path"`
	IsDefaultTunnel bool   `json:"is_default_tunnel"`
	HTTPHostHeader  string `json:"http_host_header"`
	NoTLSVerify     bool   `json:"no_tls_verify"`
	Http2Origin     bool   `json:"http2_origin"`
	HasAccessPolicy bool   `json:"has_access_policy"`
}

type EntryResponse struct {
	Hostname         string                   `json:"hostname"`
	CaddyUpstream    string                   `json:"caddy_upstream"`
	CaddyIP          string                   `json:"caddy_ip"`
	CaddyPort        string                   `json:"caddy_port"`
	UnboundStatus    ServiceStatusResponse    `json:"unbound_status"`
	AdguardStatus    ServiceStatusResponse    `json:"adguard_status"`
	DHCPStatus       DHCPStatusResponse       `json:"dhcp_status"`
	DNSResolved      string                   `json:"dns_resolved"`
	CloudflareStatus CloudflareStatusResponse `json:"cloudflare_status"`
	OverallStatus    models.SyncStatus        `json:"overall_status"`
	StatusLabel      string                   `json:"status_label"`
	DataSource       string                   `json:"data_source"`
}

type PlanResponse struct {
	PlanID    string            `json:"plan_id"`
	ActionIDs []string          `json:"action_ids"`
	Actions   []syncplan.Action `json:"actions"`
	Report    status.LoadReport `json:"report"`
}

type ApplyRequest struct {
	PlanID    string            `json:"plan_id"`
	ActionIDs []string          `json:"action_ids"`
	DryRun    bool              `json:"dry_run"`
	Actions   []syncplan.Action `json:"actions"`
}

type ApplyResponse struct {
	Result *syncplan.Result `json:"result"`
}

// NewServer creates a web GUI/API server over shared app runtime services.
func NewServer(runtime *app.Runtime) *Server {
	return NewServerWithOptions(runtime, Options{})
}

// NewServerWithOptions creates a web GUI/API server with explicit local safety options.
func NewServerWithOptions(runtime *app.Runtime, options Options) *Server {
	if runtime == nil {
		runtime = &app.Runtime{}
	}
	server := &Server{runtime: runtime, options: options, mux: http.NewServeMux(), plans: make(map[string]storedPlan)}
	server.routes()
	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) runtimeSnapshot() app.Runtime {
	s.runtimeMu.RLock()
	defer s.runtimeMu.RUnlock()
	if s.runtime == nil {
		return app.Runtime{}
	}
	return *s.runtime
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleIndex)
	staticRoot, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	s.mux.Handle("/static/", http.StripPrefix("/static/", staticHandler(http.FileServer(http.FS(staticRoot)))))
	s.mux.HandleFunc("/api/config", s.handleConfig)
	s.mux.HandleFunc("/api/config/test", s.handleConfigTest)
	s.mux.HandleFunc("/api/cloudflare/discover", s.handleCloudflareDiscover)
	s.mux.HandleFunc("/api/entries", s.handleEntries)
	s.mux.HandleFunc("/api/sync/plan", s.handlePlan)
	s.mux.HandleFunc("/api/sync/apply", s.handleApply)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	body, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if s.options.EnableTestHooks {
		body = []byte(strings.Replace(string(body), "</head>", "  <script>window.UNBOUNDCLI_TEST_HOOKS = true;</script>\n</head>", 1))
	}
	body = []byte(strings.Replace(string(body), "</head>", s.clientConfigScript()+"\n</head>", 1))
	_, _ = w.Write(body)
}

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
	}, nil
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

func (s *Server) handleCloudflareDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req CloudflareDiscoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request: %w", err))
		return
	}

	// Fall back to saved config if token not provided
	runtime := s.runtimeSnapshot()
	token := req.Token
	if token == "" {
		token = runtime.CloudflareConfig.APIToken
	}
	if token == "" {
		writeJSON(w, http.StatusOK, CloudflareDiscoverResponse{Error: "No API token available. Enter a token and try again."})
		return
	}
	accountID := req.AccountID
	if accountID == "" {
		accountID = runtime.CloudflareConfig.AccountID
	}

	cfClient, err := api.NewCloudflareClient(api.CloudflareConfig{
		APIToken:  token,
		AccountID: accountID,
	})
	if err != nil {
		writeJSON(w, http.StatusOK, CloudflareDiscoverResponse{Error: fmt.Sprintf("Failed to create Cloudflare client: %v", err)})
		return
	}

	resp := CloudflareDiscoverResponse{}

	// List zones (validates the token works)
	zones, err := cfClient.ListZones()
	if err != nil {
		writeJSON(w, http.StatusOK, CloudflareDiscoverResponse{Error: fmt.Sprintf("Token invalid or no zone access: %v", err)})
		return
	}
	resp.Zones = zones

	// List accounts
	accounts, err := cfClient.ListAccounts()
	if err == nil {
		resp.Accounts = accounts
	}

	// List tunnels if account ID is known
	if accountID != "" {
		tunnels, err := cfClient.ListTunnels()
		if err == nil {
			// Filter out deleted tunnels
			for _, t := range tunnels {
				if t.DeletedAt.IsZero() {
					resp.Tunnels = append(resp.Tunnels, t)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	entries, report, err := s.loadEntries(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, EntriesResponse{Entries: entryResponses(entries), Report: report})
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

type sourceProbe string

const (
	sourceProbeUnbound    sourceProbe = "unbound"
	sourceProbeAdguard    sourceProbe = "adguard"
	sourceProbeCloudflare sourceProbe = "cloudflare"
)

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

func (s *Server) reloadRuntimeFromConfig(cfg config.ExtendedConfig) error {
	current := s.runtimeSnapshot()
	nextRuntime, err := app.NewRuntimeFromConfigs(cfg.Config, cfg.Adguard, cfg.Cloudflare, app.RuntimeOptions{
		CaddyServerIP:     current.CaddyEndpoint.ServerIP,
		CaddyServerPort:   current.CaddyEndpoint.ServerPort,
		IncludeUnbound:    true,
		IncludeDNSMasq:    current.Clients.DNSMasq != nil,
		IncludeAdguard:    true,
		IncludeCloudflare: true,
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

func entryResponses(entries []*models.Entry) []EntryResponse {
	out := make([]EntryResponse, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		out = append(out, EntryResponse{
			Hostname:      entry.Hostname,
			CaddyUpstream: entry.CaddyUpstream,
			CaddyIP:       entry.CaddyIP,
			CaddyPort:     entry.CaddyPort,
			UnboundStatus: serviceStatusResponse(entry.UnboundStatus),
			AdguardStatus: serviceStatusResponse(entry.AdguardStatus),
			DHCPStatus: DHCPStatusResponse{
				Configured: entry.DHCPStatus.Configured,
				Type:       entry.DHCPStatus.Type,
				IP:         entry.DHCPStatus.IP,
				MAC:        entry.DHCPStatus.MAC,
				Hostname:   entry.DHCPStatus.Hostname,
				InSync:     entry.DHCPStatus.InSync,
			},
			DNSResolved: entry.DNSResolved,
			CloudflareStatus: CloudflareStatusResponse{
				Configured:      entry.CloudflareStatus.Configured,
				TunnelName:      entry.CloudflareStatus.TunnelName,
				TunnelID:        entry.CloudflareStatus.TunnelID,
				Service:         entry.CloudflareStatus.Service,
				Path:            entry.CloudflareStatus.Path,
				IsDefaultTunnel: entry.CloudflareStatus.IsDefaultTunnel,
				HTTPHostHeader:  entry.CloudflareStatus.HTTPHostHeader,
				NoTLSVerify:     entry.CloudflareStatus.NoTLSVerify,
				Http2Origin:     entry.CloudflareStatus.Http2Origin,
				HasAccessPolicy: entry.CloudflareStatus.HasAccessPolicy,
			},
			OverallStatus: entry.OverallStatus,
			StatusLabel:   entry.OverallStatus.Label(),
			DataSource:    entry.DataSource,
		})
	}
	return out
}

func serviceStatusResponse(serviceStatus models.ServiceStatus) ServiceStatusResponse {
	return ServiceStatusResponse{
		Configured: serviceStatus.Configured,
		IP:         serviceStatus.IP,
		InSync:     serviceStatus.InSync,
	}
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	service := r.URL.Query().Get("service")
	if !validPlanService(service) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid sync service %q", service))
		return
	}
	hostname := strings.TrimSpace(r.URL.Query().Get("hostname"))
	entries, report, err := s.loadEntries(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	runtime := s.runtimeSnapshot()
	if service != "" && service != "all" && !serviceEnabled(&runtime, service) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("%s is unavailable in this web session", service))
		return
	}
	plan := syncplan.BuildPlan(entries, syncplan.Options{
		Service:           service,
		CaddyServerIP:     runtime.CaddyEndpoint.ServerIP,
		CaddyServiceURL:   runtime.CaddyServiceURL,
		IncludeCloudflare: runtime.Clients.Cloudflare != nil,
	})
	actions := s.webPlanActions(&runtime, service, plan.Actions)
	if hostname != "" {
		actions = filterPlanActionsByHostname(actions, hostname)
	}
	planID := planID(service, actions)
	actionIDs := actionIDs(actions)
	s.storePlan(planID, actions, actionIDs)
	writeJSON(w, http.StatusOK, PlanResponse{
		PlanID:    planID,
		ActionIDs: actionIDs,
		Actions:   actions,
		Report:    report,
	})
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var request ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid apply request: %w", err))
		return
	}
	if !request.DryRun {
		if err := s.allowMutation(r); err != nil {
			writeError(w, http.StatusForbidden, err)
			return
		}
		if len(request.Actions) > 0 {
			writeError(w, http.StatusBadRequest, fmt.Errorf("mutating apply must use server-issued plan/action IDs"))
			return
		}
		if request.PlanID == "" || len(request.ActionIDs) == 0 {
			writeError(w, http.StatusBadRequest, fmt.Errorf("mutating apply requires plan_id and action_ids"))
			return
		}
		actions, err := s.actionsForIDs(request.PlanID, request.ActionIDs)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateApplyActions(actions); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		result := s.applyActions(r.Context(), actions, false)
		writeJSON(w, http.StatusOK, ApplyResponse{Result: result})
		return
	}
	if err := validateApplyActions(request.Actions); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result := s.applyActions(r.Context(), request.Actions, request.DryRun)
	writeJSON(w, http.StatusOK, ApplyResponse{Result: result})
}

func (s *Server) applyActions(ctx context.Context, actions []syncplan.Action, dryRun bool) *syncplan.Result {
	runtime := s.runtimeSnapshot()
	return syncplan.Apply(ctx, syncplan.Clients{
		Unbound:    runtime.Clients.Unbound,
		Adguard:    runtime.Clients.Adguard,
		Cloudflare: runtime.Clients.Cloudflare,
	}, syncplan.Plan{Actions: actions}, syncplan.ApplyOptions{DryRun: dryRun})
}

func (s *Server) loadEntries(ctx context.Context) ([]*models.Entry, status.LoadReport, error) {
	runtime := s.runtimeSnapshot()
	return status.LoadEntries(ctx, runtime.Clients, status.Options{
		CaddyServerIP: runtime.CaddyEndpoint.ServerIP,
	})
}

func validPlanService(service string) bool {
	switch service {
	case "", "all", "unbound", "adguard", "dhcp", "cloudflare":
		return true
	default:
		return false
	}
}

func validateApplyActions(actions []syncplan.Action) error {
	for _, action := range actions {
		switch action.Service {
		case "unbound", "adguard", "cloudflare":
			continue
		case "dhcp":
			return fmt.Errorf("DHCP apply is not implemented")
		default:
			return fmt.Errorf("invalid sync service %q", action.Service)
		}
	}
	return nil
}

func (s *Server) clientConfigScript() string {
	config := struct {
		ApplyToken      string `json:"applyToken"`
		MutationEnabled bool   `json:"mutationEnabled"`
	}{
		MutationEnabled: s.mutationsEnabled(),
	}
	if config.MutationEnabled {
		config.ApplyToken = s.options.ApplyToken
	}
	data, err := json.Marshal(config)
	if err != nil {
		data = []byte(`{"applyToken":"","mutationEnabled":false}`)
	}
	return fmt.Sprintf("  <script>window.UNBOUNDCLI_WEB_CONFIG = %s;</script>", data)
}

func (s *Server) mutationsEnabled() bool {
	if !s.options.AllowMutations || s.options.ApplyToken == "" {
		return false
	}
	return s.options.AllowUnsafeBind || isLoopbackHost(s.options.BoundHost)
}

func (s *Server) storePlan(planID string, actions []syncplan.Action, actionIDs []string) {
	actionsByID := make(map[string]syncplan.Action, len(actions))
	for i, action := range actions {
		if i >= len(actionIDs) {
			break
		}
		actionsByID[actionIDs[i]] = action
	}
	s.planMu.Lock()
	s.plans[planID] = storedPlan{ActionsByID: actionsByID}
	s.planMu.Unlock()
}

func (s *Server) actionsForIDs(planID string, actionIDs []string) ([]syncplan.Action, error) {
	s.planMu.Lock()
	plan, ok := s.plans[planID]
	s.planMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("unknown or expired sync plan")
	}
	actions := make([]syncplan.Action, 0, len(actionIDs))
	for _, actionID := range actionIDs {
		action, ok := plan.ActionsByID[actionID]
		if !ok {
			return nil, fmt.Errorf("unknown sync action %q", actionID)
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func (s *Server) webPlanActions(runtime *app.Runtime, service string, actions []syncplan.Action) []syncplan.Action {
	out := make([]syncplan.Action, 0, len(actions))
	for _, action := range actions {
		if serviceEnabled(runtime, action.Service) {
			out = append(out, action)
		}
	}
	return out
}

func filterPlanActionsByHostname(actions []syncplan.Action, hostname string) []syncplan.Action {
	out := make([]syncplan.Action, 0, len(actions))
	for _, action := range actions {
		if action.Hostname == hostname {
			out = append(out, action)
		}
	}
	return out
}

func serviceEnabled(runtime *app.Runtime, service string) bool {
	switch service {
	case "unbound":
		return runtime.Clients.Unbound != nil
	case "adguard":
		return runtime.Clients.Adguard != nil
	case "cloudflare":
		return runtime.Clients.Cloudflare != nil
	default:
		return true
	}
}

func (s *Server) allowMutation(r *http.Request) error {
	if !s.options.AllowMutations {
		return fmt.Errorf("web apply mutations are disabled; dry-run is still available")
	}
	if !s.options.AllowUnsafeBind && !isLoopbackHost(s.options.BoundHost) {
		return fmt.Errorf("web apply mutations require a loopback bind address")
	}
	if s.options.ApplyToken == "" || r.Header.Get("X-UnboundCLI-Token") != s.options.ApplyToken {
		return fmt.Errorf("web apply requires a valid local session token")
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != s.options.AllowedOrigin {
		return fmt.Errorf("web apply rejected origin %q", origin)
	}
	return nil
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func planID(service string, actions []syncplan.Action) string {
	data, err := json.Marshal(struct {
		Service string            `json:"service"`
		Actions []syncplan.Action `json:"actions"`
	}{Service: service, Actions: actions})
	if err != nil {
		return "plan-error"
	}
	sum := sha256.Sum256(data)
	return "plan-" + hex.EncodeToString(sum[:8])
}

func actionIDs(actions []syncplan.Action) []string {
	ids := make([]string, 0, len(actions))
	for _, action := range actions {
		data, err := json.Marshal(action)
		if err != nil {
			ids = append(ids, "action-error")
			continue
		}
		sum := sha256.Sum256(data)
		ids = append(ids, "action-"+hex.EncodeToString(sum[:8]))
	}
	return ids
}

func staticHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
}

func writeError(w http.ResponseWriter, statusCode int, err error) {
	writeJSON(w, statusCode, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
