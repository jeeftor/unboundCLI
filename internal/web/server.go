package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/auth"
	"github.com/jeeftor/caddy-dns-sync/internal/caddyeditor"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
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
	Version         string // build version, e.g. "v1.2.3" or "dev"
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
	createdAt   time.Time
}

const planTTL = 10 * time.Minute

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

// AuthInventoryResponse is the response for GET /api/auth/inventory.
type AuthInventoryResponse struct {
	Hosts       []models.HostAuth `json:"hosts"`
	Sources     AuthSources       `json:"sources"`
}

// AuthSources indicates which auth discovery sources were queried.
type AuthSources struct {
	CloudflareAccess bool `json:"cloudflare_access"`
	Authentik        bool `json:"authentik"`
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
	Configured       bool   `json:"configured"`
	TunnelName       string `json:"tunnel_name"`
	TunnelID         string `json:"tunnel_id"`
	Service          string `json:"service"`
	Path             string `json:"path"`
	IsDefaultTunnel  bool   `json:"is_default_tunnel"`
	HTTPHostHeader   string `json:"http_host_header"`
	OriginServerName string `json:"origin_server_name"`
	NoTLSVerify      bool   `json:"no_tls_verify"`
	Http2Origin      bool   `json:"http2_origin"`
	HasAccessPolicy  bool   `json:"has_access_policy"`
	HasDNSRecord     bool   `json:"has_dns_record"`
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
	HasForwardAuth   bool                     `json:"has_forward_auth"`
	HasAuthBypass    bool                     `json:"has_auth_bypass_risk"`
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
	// Capture log lines into the ring buffer so the web UI can stream them.
	logging.EnableBuffer()
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
	s.mux.HandleFunc("/api/config/raw", s.handleConfigRaw)
	s.mux.HandleFunc("/api/config/test", s.handleConfigTest)
	s.mux.HandleFunc("/api/cloudflare/discover", s.handleCloudflareDiscover)
	s.mux.HandleFunc("/api/cloudflare/tunnels", s.handleCloudflareTunnels)
	s.mux.HandleFunc("/api/cloudflare/set-route", s.handleCloudflareSetRoute)
	s.mux.HandleFunc("/api/cloudflare/remove-route", s.handleCloudflareRemoveRoute)
	s.mux.HandleFunc("/api/cloudflare/repair-dns", s.handleCloudflareRepairDNS)
	s.mux.HandleFunc("/api/entries", s.handleEntries)
	s.mux.HandleFunc("/api/entries/stream", s.handleEntriesStream)
	s.mux.HandleFunc("/api/probe", s.handleProbe)
	s.mux.HandleFunc("/api/dns-probe", s.handleDNSProbe)
	s.mux.HandleFunc("/api/logs", s.handleLogs)
	s.mux.HandleFunc("/api/sync/plan", s.handlePlan)
	s.mux.HandleFunc("/api/sync/apply", s.handleApply)
	s.mux.HandleFunc("/api/sync/remove", s.handleSyncRemove)
	// Caddy Editor routes
	s.mux.HandleFunc("/api/caddy/entries", s.handleCaddyEntries)
	s.mux.HandleFunc("/api/caddy/entries/", s.handleCaddyEntry)
	s.mux.HandleFunc("/api/caddy/diff", s.handleCaddyDiff)
	s.mux.HandleFunc("/api/caddy/git/status", s.handleCaddyGitStatus)
	s.mux.HandleFunc("/api/caddy/git/pull", s.handleCaddyGitPull)
	s.mux.HandleFunc("/api/caddy/validate", s.handleCaddyValidate)
	s.mux.HandleFunc("/api/caddy/validate-draft", s.handleCaddyValidateDraft)
	s.mux.HandleFunc("/api/caddy/deploy", s.handleCaddyDeploy)
	s.mux.HandleFunc("/api/caddy/templates", s.handleCaddyTemplates)
	s.mux.HandleFunc("/api/caddy/preview", s.handleCaddyPreview)
	// Auth inventory
	s.mux.HandleFunc("/api/auth/inventory", s.handleAuthInventory)
	s.mux.HandleFunc("/api/auth/inventory/stream", s.handleAuthInventoryStream)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
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
		_ = s.reloadRuntimeFromConfig(cfg)
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

// handleCloudflareTunnels returns the list of active tunnels from the configured CF client.
// This lets the web wizard populate its tunnel selector without requiring re-authentication.
func (s *Server) handleCloudflareTunnels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	runtime := s.runtimeSnapshot()
	if runtime.Clients.Cloudflare == nil {
		writeJSON(w, http.StatusOK, []api.CloudflareTunnel{})
		return
	}
	tunnels, err := runtime.Clients.Cloudflare.ListTunnels()
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	active := make([]api.CloudflareTunnel, 0, len(tunnels))
	for _, t := range tunnels {
		if t.DeletedAt.IsZero() {
			active = append(active, t)
		}
	}
	writeJSON(w, http.StatusOK, active)
}

// CloudflareSetRouteRequest sets the routing mode for a single hostname in the CF tunnel.
type CloudflareSetRouteRequest struct {
	Hostname         string `json:"hostname"`
	Service          string `json:"service"`            // full URL, e.g. "https://10.0.0.15" or "http://10.0.0.112:8006"
	HTTPHostHeader   string `json:"http_host_header"`   // set when routing via Caddy
	OriginServerName string `json:"origin_server_name"` // TLS SNI hostname for origin
	NoTLSVerify      bool   `json:"no_tls_verify"`
}

// handleCloudflareSetRoute updates a single CF tunnel ingress rule to point to the given service.
// POST /api/cloudflare/set-route
func (s *Server) handleCloudflareSetRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	runtime := s.runtimeSnapshot()
	if runtime.Clients.Cloudflare == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("Cloudflare not configured"))
		return
	}
	var req CloudflareSetRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Hostname == "" || req.Service == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("hostname and service are required"))
		return
	}
	spec := api.IngressRuleSpec{
		Hostname:            req.Hostname,
		Service:             req.Service,
		HTTPHostHeader:      req.HTTPHostHeader,
		OriginServerName:    req.OriginServerName,
		SetOriginServerName: req.OriginServerName != "",
		NoTLSVerify:         req.NoTLSVerify,
	}
	if err := runtime.Clients.Cloudflare.UpdateTunnelRule(spec); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if err := runtime.Clients.Cloudflare.EnsureDNSRecord(req.Hostname); err != nil {
		logging.Warn("set-route: failed to ensure DNS CNAME record", "hostname", req.Hostname, "error", err)
		// Tunnel rule was set but DNS CNAME failed — surface the warning to the caller
		// so the UI can prompt the user to repair rather than silently succeeding.
		writeJSON(w, http.StatusOK, map[string]string{
			"status":      "ok",
			"dns_warning": fmt.Sprintf("Tunnel rule saved but CNAME creation failed: %v — use Repair DNS to fix", err),
		})
		return
	}
	logging.Info("set-route: DNS CNAME record ensured", "hostname", req.Hostname)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleCloudflareRemoveRoute removes a hostname from the CF tunnel ingress.
// POST /api/cloudflare/remove-route  { "hostname": "foo.example.com" }
func (s *Server) handleCloudflareRemoveRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	runtime := s.runtimeSnapshot()
	if runtime.Clients.Cloudflare == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("Cloudflare not configured"))
		return
	}
	var req struct {
		Hostname string `json:"hostname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Hostname == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("hostname is required"))
		return
	}
	if err := runtime.Clients.Cloudflare.DeleteTunnelRule(req.Hostname); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if err := runtime.Clients.Cloudflare.DeleteDNSRecord(req.Hostname); err != nil {
		logging.Warn("remove-route: failed to delete DNS CNAME record", "hostname", req.Hostname, "error", err)
	} else {
		logging.Info("remove-route: DNS CNAME record deleted", "hostname", req.Hostname)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleCloudflareRepairDNS creates missing CNAME DNS records for all CF tunnel entries.
// POST /api/cloudflare/repair-dns
func (s *Server) handleCloudflareRepairDNS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	runtime := s.runtimeSnapshot()
	if runtime.Clients.Cloudflare == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("Cloudflare not configured"))
		return
	}
	hostnames, err := runtime.Clients.Cloudflare.GetTunnelHostnames()
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("failed to list tunnel hostnames: %w", err))
		return
	}
	existing, err := runtime.Clients.Cloudflare.ListManagedDNSRecords()
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("failed to list DNS records: %w", err))
		return
	}

	fixed := []string{}
	failed := []string{}
	for hostname := range hostnames {
		if _, ok := existing[hostname]; ok {
			continue // already has a CNAME
		}
		if err := runtime.Clients.Cloudflare.EnsureDNSRecord(hostname); err != nil {
			logging.Warn("repair-dns: failed to create CNAME", "hostname", hostname, "error", err)
			failed = append(failed, hostname)
		} else {
			logging.Info("repair-dns: created CNAME", "hostname", hostname)
			fixed = append(fixed, hostname)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"fixed":  fixed,
		"failed": failed,
	})
}

// ProbeResponse is the result of an HTTP reachability probe.
type ProbeResponse struct {
	Reachable  bool   `json:"reachable"`
	StatusCode int    `json:"status_code,omitempty"`
	LatencyMS  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
	ProbeURL   string `json:"probe_url"`
}

// handleLogs returns buffered log lines since a given cursor index.
// GET /api/logs?since=N  →  { lines: [...], cursor: N }
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	since := 0
	if v := r.URL.Query().Get("since"); v != "" {
		fmt.Sscanf(v, "%d", &since)
	}
	lines, cursor := logging.GetLogLinesSince(since)
	if lines == nil {
		lines = []logging.LogLine{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"lines":  lines,
		"cursor": cursor,
	})
}

// handleProbe does a quick HTTP/HTTPS HEAD probe to an upstream address.
// GET /api/probe?upstream=10.0.0.15:6868&hostname=foo.example.com
// The scheme is inferred from the port: 443/8443 → https, everything else → http.
func (s *Server) handleProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	upstream := strings.TrimSpace(r.URL.Query().Get("upstream"))
	hostname := strings.TrimSpace(r.URL.Query().Get("hostname"))
	if upstream == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("upstream parameter required"))
		return
	}

	// Strip any existing scheme so we can determine it ourselves.
	upstream = strings.TrimPrefix(strings.TrimPrefix(upstream, "https://"), "http://")
	upstream = strings.TrimSuffix(upstream, "/")

	// Infer scheme from port suffix.
	scheme := "http"
	if strings.HasSuffix(upstream, ":443") || strings.HasSuffix(upstream, ":8443") {
		scheme = "https"
	}

	probeURL := scheme + "://" + upstream + "/"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodHead, probeURL, nil)
	if err != nil {
		writeJSON(w, http.StatusOK, ProbeResponse{Reachable: false, Error: err.Error(), ProbeURL: probeURL})
		return
	}
	if hostname != "" {
		req.Host = hostname
	}

	client := &http.Client{
		Timeout: 4 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			DisableKeepAlives:   true,
			MaxIdleConnsPerHost: 1,
		},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse // don't follow redirects
		},
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if resp != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
	if err != nil {
		// Try GET as fallback (some servers reject HEAD)
		req2, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, probeURL, nil)
		if req2 != nil {
			if hostname != "" {
				req2.Host = hostname
			}
			start2 := time.Now()
			resp2, err2 := client.Do(req2)
			latency = time.Since(start2).Milliseconds()
			if resp2 != nil {
				_, _ = io.Copy(io.Discard, resp2.Body)
				_ = resp2.Body.Close()
			}
			if err2 == nil {
				writeJSON(w, http.StatusOK, ProbeResponse{
					Reachable:  true,
					StatusCode: resp2.StatusCode,
					LatencyMS:  latency,
					ProbeURL:   probeURL,
				})
				return
			}
		}
		writeJSON(w, http.StatusOK, ProbeResponse{Reachable: false, Error: err.Error(), LatencyMS: latency, ProbeURL: probeURL})
		return
	}

	writeJSON(w, http.StatusOK, ProbeResponse{
		Reachable:  resp.StatusCode < 500,
		StatusCode: resp.StatusCode,
		LatencyMS:  latency,
		ProbeURL:   probeURL,
	})
}

// DNSProbeResponse is the result of a public DNS lookup via Cloudflare's 1.1.1.1 resolver.
type DNSProbeResponse struct {
	Resolved  bool     `json:"resolved"`
	CNAME     string   `json:"cname,omitempty"`
	Addresses []string `json:"addresses,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// handleDNSProbe looks up a hostname via Cloudflare's public resolver (1.1.1.1)
// to check if a CF tunnel hostname is resolving publicly.
// GET /api/dns-probe?hostname=foo.example.com
func (s *Server) handleDNSProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	hostname := strings.TrimSpace(r.URL.Query().Get("hostname"))
	if hostname == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("hostname parameter required"))
		return
	}

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: 4 * time.Second}
			return d.DialContext(ctx, "udp", "1.1.1.1:53")
		},
	}

	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()

	// Walk the CNAME chain.
	cname, err := resolver.LookupCNAME(ctx, hostname)
	if err != nil {
		writeJSON(w, http.StatusOK, DNSProbeResponse{Resolved: false, Error: err.Error()})
		return
	}
	// LookupCNAME returns the hostname itself (with trailing dot) if there's no CNAME.
	canonicalCNAME := strings.TrimSuffix(cname, ".")
	if canonicalCNAME == hostname {
		canonicalCNAME = ""
	}

	addrs, err := resolver.LookupHost(ctx, hostname)
	if err != nil {
		writeJSON(w, http.StatusOK, DNSProbeResponse{Resolved: false, CNAME: canonicalCNAME, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, DNSProbeResponse{
		Resolved:  true,
		CNAME:     canonicalCNAME,
		Addresses: addrs,
	})
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

// handleEntriesStream streams entry loading progress via Server-Sent Events.
// GET /api/entries/stream — sends progress events as each service loads,
// then a final "done" event with the full entries payload.
//
// Events:
//   - event: progress  (a service status changed: pending → loaded/failed/skipped)
//   - event: done      (all services loaded, includes full entries + report)
//   - event: error     (fatal error, loading aborted)
func (s *Server) handleEntriesStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	// SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("streaming not supported"))
		return
	}

	ctx := r.Context()
	runtime := s.runtimeSnapshot()

	// Create a loader with a progress callback that emits SSE events.
	loader := status.NewDataLoader(
		runtime.Clients.Caddy,
		runtime.Clients.Unbound,
		runtime.Clients.Adguard,
		runtime.Clients.DNSMasq,
		runtime.CaddyEndpoint.ServerIP,
	)
	loader.WithCloudflareClient(runtime.Clients.Cloudflare)
	loader.WithContext(ctx)
	loader.WithProgress(func(ev status.ProgressEvent) {
		data, err := json.Marshal(ev)
		if err != nil {
			logging.Warn("Failed to marshal progress event", "error", err)
			return
		}
		fmt.Fprintf(w, "event: progress\ndata: %s\n\n", data)
		flusher.Flush()
	})

	// Load entries (this emits progress events as each service completes).
	entries, report, err := loader.LoadDataWithReport()
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: {\"error\":%s}\n\n", escapeJSONString(err.Error()))
		flusher.Flush()
		return
	}

	// Send the final entries payload.
	response := EntriesResponse{
		Entries: entryResponses(entries),
		Report:  report,
	}
	data, err := json.Marshal(response)
	if err != nil {
		logging.Warn("Failed to marshal entries response", "error", err)
		return
	}
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", data)
	flusher.Flush()
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
				Configured:       entry.CloudflareStatus.Configured,
				TunnelName:       entry.CloudflareStatus.TunnelName,
				TunnelID:         entry.CloudflareStatus.TunnelID,
				Service:          entry.CloudflareStatus.Service,
				Path:             entry.CloudflareStatus.Path,
				IsDefaultTunnel:  entry.CloudflareStatus.IsDefaultTunnel,
				HTTPHostHeader:   entry.CloudflareStatus.HTTPHostHeader,
				OriginServerName: entry.CloudflareStatus.OriginServerName,
				NoTLSVerify:      entry.CloudflareStatus.NoTLSVerify,
				Http2Origin:      entry.CloudflareStatus.Http2Origin,
				HasAccessPolicy:  entry.CloudflareStatus.HasAccessPolicy,
				HasDNSRecord:     entry.CloudflareStatus.HasDNSRecord,
			},
			OverallStatus:  entry.OverallStatus,
			StatusLabel:    entry.OverallStatus.Label(),
			DataSource:     entry.DataSource,
			HasForwardAuth: entry.CaddyRoute.HasForwardAuth,
			HasAuthBypass:  entry.HasAuthBypassRisk(),
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
	// Cloudflare wizard options (only meaningful when service=cloudflare)
	originMode := strings.TrimSpace(r.URL.Query().Get("origin_mode"))
	noTLSVerify := r.URL.Query().Get("no_tls_verify") == "true"
	overrideTunnelID := strings.TrimSpace(r.URL.Query().Get("tunnel_id"))
	disableChunked := r.URL.Query().Get("disable_chunked_encoding") == "true"
	// unsync=true generates delete actions for entries currently in the target service,
	// regardless of whether they appear in Caddy. Used for manual "remove from service" flows.
	unsync := r.URL.Query().Get("unsync") == "true"

	plan := syncplan.BuildPlan(entries, syncplan.Options{
		Service:                service,
		CaddyServerIP:          runtime.CaddyEndpoint.ServerIP,
		CaddyServiceURL:        runtime.CaddyServiceURL,
		IncludeCloudflare:      runtime.Clients.Cloudflare != nil,
		OriginMode:             originMode,
		NoTLSVerify:            noTLSVerify,
		DisableChunkedEncoding: disableChunked,
		OverrideTunnelID:       overrideTunnelID,
		Unsync:                 unsync,
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

// handleSyncRemove deletes DNS entries for a specific hostname.
// Body: {"hostname":"foo.example.com","service":"all"|"unbound"|"adguard"}
// service defaults to "all" when omitted.
func (s *Server) handleSyncRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	if err := s.allowMutation(r); err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Hostname string `json:"hostname"`
		Service  string `json:"service"` // "all", "unbound", "adguard" — defaults to "all"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Hostname == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("hostname required"))
		return
	}
	if req.Service == "" {
		req.Service = "all"
	}

	runtime := s.runtimeSnapshot()
	removed := 0
	var msgs []string

	// Remove from Unbound
	if (req.Service == "all" || req.Service == "unbound") && runtime.Clients.Unbound != nil {
		parts := strings.SplitN(req.Hostname, ".", 2)
		if len(parts) == 2 {
			overrides, err := runtime.Clients.Unbound.GetOverrides()
			if err == nil {
				unboundRemoved := 0
				for _, o := range overrides {
					if strings.EqualFold(o.Host, parts[0]) && strings.EqualFold(o.Domain, parts[1]) {
						if delErr := runtime.Clients.Unbound.DeleteOverride(o.UUID); delErr == nil {
							unboundRemoved++
							removed++
						}
					}
				}
				if unboundRemoved > 0 {
					_ = runtime.Clients.Unbound.ApplyChanges()
					msgs = append(msgs, fmt.Sprintf("removed %d Unbound override(s)", unboundRemoved))
				}
			}
		}
	}

	// Remove from AdGuard
	if (req.Service == "all" || req.Service == "adguard") && runtime.Clients.Adguard != nil {
		rewrites, err := runtime.Clients.Adguard.GetRewritesForDomain(req.Hostname)
		if err == nil {
			n := 0
			for _, rw := range rewrites {
				if delErr := runtime.Clients.Adguard.DeleteRewrite(rw.Domain, rw.Answer); delErr == nil {
					n++
					removed++
				}
			}
			if n > 0 {
				msgs = append(msgs, fmt.Sprintf("removed %d AdGuard rewrite(s)", n))
			}
		}
	}

	msg := fmt.Sprintf("Removed DNS entries for %s", req.Hostname)
	if len(msgs) > 0 {
		msg = strings.Join(msgs, "; ")
	} else if removed == 0 {
		msg = fmt.Sprintf("No DNS entries found for %s", req.Hostname)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"removed": removed,
		"message": msg,
	})
}

func (s *Server) loadEntries(ctx context.Context) ([]*models.Entry, status.LoadReport, error) {
	runtime := s.runtimeSnapshot()
	return status.LoadEntries(ctx, runtime.Clients, status.Options{
		CaddyServerIP: runtime.CaddyEndpoint.ServerIP,
	})
}

// handleAuthInventory returns the auth discovery results for all hostnames.
// GET /api/auth/inventory — queries Caddy, CF Access, and Authentik to
// classify each hostname's WAN/LAN/API auth configuration.
func (s *Server) handleAuthInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	ctx := r.Context()
	runtime := s.runtimeSnapshot()

	// Load entries (Caddy + CF tunnel state) as the base for auth discovery.
	entries, _, err := s.loadEntries(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("loading entries for auth discovery: %w", err))
		return
	}

	// Run auth discovery with whatever clients are available.
	authMap, err := auth.Discover(ctx, entries, runtime.Clients.Cloudflare, runtime.Clients.Authentik)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("auth discovery failed: %w", err))
		return
	}

	// Convert map to sorted slice.
	hosts := make([]models.HostAuth, 0, len(authMap))
	for _, ha := range authMap {
		hosts = append(hosts, *ha)
	}
	// Sort by hostname for stable output.
	sortHostAuthByName(hosts)

	response := AuthInventoryResponse{
		Hosts: hosts,
		Sources: AuthSources{
			CloudflareAccess: runtime.Clients.Cloudflare != nil,
			Authentik:        runtime.Clients.Authentik != nil,
		},
	}
	writeJSON(w, http.StatusOK, response)
}

func sortHostAuthByName(hosts []models.HostAuth) {
	for i := 1; i < len(hosts); i++ {
		for j := i; j > 0 && hosts[j-1].Hostname > hosts[j].Hostname; j-- {
			hosts[j-1], hosts[j] = hosts[j], hosts[j-1]
		}
	}
}

// handleAuthInventoryStream streams auth discovery results via Server-Sent Events.
// GET /api/auth/inventory/stream — sends events as each discovery phase completes:
//   - event: base   (all hosts with Caddy-derived auth, instant)
//   - event: enrich (updated hosts after CF Access or Authentik data arrives)
//   - event: error  (if a source API call fails)
//   - event: done   (discovery complete)
func (s *Server) handleAuthInventoryStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	// SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable Nginx/Caddy buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("streaming not supported"))
		return
	}

	ctx := r.Context()
	runtime := s.runtimeSnapshot()

	// Load entries (Caddy + CF tunnel state) as the base for auth discovery.
	entries, _, err := s.loadEntries(ctx)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: {\"error\":\"loading entries: %s\"}\n\n", escapeJSONString(err.Error()))
		flusher.Flush()
		return
	}

	// Stream discovery events.
	auth.DiscoverStream(ctx, entries, runtime.Clients.Cloudflare, runtime.Clients.Authentik, func(ev auth.StreamEvent) {
		data, err := json.Marshal(ev)
		if err != nil {
			logging.Warn("Failed to marshal auth stream event", "error", err)
			return
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, data)
		flusher.Flush()
	})
}

// escapeJSONString escapes a string for safe embedding in JSON.
func escapeJSONString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
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
	s.cleanExpiredPlans()
	actionsByID := make(map[string]syncplan.Action, len(actions))
	for i, action := range actions {
		if i >= len(actionIDs) {
			break
		}
		actionsByID[actionIDs[i]] = action
	}
	s.planMu.Lock()
	s.plans[planID] = storedPlan{ActionsByID: actionsByID, createdAt: time.Now()}
	s.planMu.Unlock()
}

// cleanExpiredPlans removes plans older than planTTL from the in-memory store.
func (s *Server) cleanExpiredPlans() {
	s.planMu.Lock()
	defer s.planMu.Unlock()
	cutoff := time.Now().Add(-planTTL)
	for id, plan := range s.plans {
		if plan.createdAt.Before(cutoff) {
			delete(s.plans, id)
		}
	}
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
	if s.options.ApplyToken == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-UnboundCLI-Token")), []byte(s.options.ApplyToken)) != 1 {
		return fmt.Errorf("web apply requires a valid local session token")
	}
	if s.options.AllowedOrigin != "" {
		if origin := r.Header.Get("Origin"); origin != "" && origin != s.options.AllowedOrigin {
			return fmt.Errorf("web apply rejected origin %q", origin)
		}
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

const appCSP = "default-src 'self' 'unsafe-inline' https://static.cloudflareinsights.com; script-src 'self' 'unsafe-inline' https://static.cloudflareinsights.com; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' https://static.cloudflareinsights.com; font-src 'self'; frame-ancestors 'none'"

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", appCSP)
}

func staticHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)
		// Hashed assets (app.<hash>.js, styles.<hash>.css) are immutable — cache forever.
		// index.html is served by handleIndex with no-store, so it always fetches fresh.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
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
