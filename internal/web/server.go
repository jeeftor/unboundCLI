package web

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/auth"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
	"github.com/jeeftor/caddy-dns-sync/internal/status"
	"github.com/jeeftor/caddy-dns-sync/internal/syncplan"
)

//go:embed static/*
var staticFiles embed.FS

// ─── Server Core ────────────────────────────────────────────────────────────

type Options struct {
	ApplyToken      string
	AllowMutations  bool
	AllowedOrigin   string
	AllowUnsafeBind bool
	BoundHost       string
	EnableTestHooks bool
	ConfigPath      string
	Version         string // build version, e.g. "v1.2.3" or "dev"
	Commit          string // git commit hash
	BuildDate       string // build timestamp
}

type Server struct {
	runtime   *app.Runtime
	options   Options
	mux       *http.ServeMux
	runtimeMu sync.RWMutex
	planMu    sync.Mutex
	plans     map[string]storedPlan

	// Auth inventory cache — populated at startup and after mutations.
	authMu    sync.RWMutex
	authCache *AuthInventoryResponse

	// Auth cache refresh dedup — prevents concurrent refresh goroutines.
	refreshMu      sync.Mutex
	refreshRunning bool

	// Entries cache — short-lived cache (30s) to avoid re-fetching from all
	// APIs when multiple endpoints need the same data (entries, diagnostics,
	// plan, auth). Invalidated on mutations.
	entriesMu      sync.Mutex
	entriesCache   []*models.Entry
	entriesReport  status.LoadReport
	entriesCacheAt time.Time

	// Server lifecycle — used for graceful shutdown of background goroutines.
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type storedPlan struct {
	ActionsByID map[string]syncplan.Action
	createdAt   time.Time
}

const planTTL = 10 * time.Minute

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
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{
		runtime: runtime,
		options: options,
		mux:     http.NewServeMux(),
		plans:   make(map[string]storedPlan),
		ctx:     ctx,
		cancel:  cancel,
	}
	server.routes()
	// Pre-populate auth cache at startup so all clients get instant data.
	server.wg.Add(1)
	go func() {
		defer server.wg.Done()
		defer logging.Recover("server: startup auth cache refresh")
		server.refreshAuthCache()
	}()
	// Start periodic background refresh (every 5 minutes).
	server.wg.Add(1)
	go server.periodicAuthRefresh()
	return server
}

// Shutdown cancels background goroutines and waits for them to finish.
func (s *Server) Shutdown() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// periodicAuthRefresh refreshes the auth cache on a periodic interval.
// Runs until the server context is cancelled.
func (s *Server) periodicAuthRefresh() {
	defer s.wg.Done()
	defer logging.Recover("server: periodic auth refresh")
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.refreshAuthCache()
		}
	}
}

// refreshAuthCache queries auth discovery and stores the result in authCache.
// Safe to call concurrently — uses refreshMu to dedup concurrent calls so
// only one refresh runs at a time.
func (s *Server) refreshAuthCache() {
	// Dedup: skip if a refresh is already running.
	s.refreshMu.Lock()
	if s.refreshRunning {
		s.refreshMu.Unlock()
		return
	}
	s.refreshRunning = true
	s.refreshMu.Unlock()
	defer func() {
		s.refreshMu.Lock()
		s.refreshRunning = false
		s.refreshMu.Unlock()
	}()

	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	runtime := s.runtimeSnapshot()

	entries, _, err := s.loadEntries(ctx)
	if err != nil {
		logging.Warn("Auth cache refresh: loading entries failed", "error", err)
		return
	}

	authMap, err := auth.Discover(ctx, entries, runtime.Clients.Cloudflare, runtime.Clients.Authentik)
	if err != nil {
		logging.Warn("Auth cache refresh: discovery failed", "error", err)
		return
	}

	hosts := make([]models.HostAuth, 0, len(authMap))
	for _, ha := range authMap {
		hosts = append(hosts, *ha)
	}
	sortHostAuthByName(hosts)

	resp := &AuthInventoryResponse{
		Hosts: hosts,
		Sources: AuthSources{
			CloudflareAccess: runtime.Clients.Cloudflare != nil,
			Authentik:        runtime.Clients.Authentik != nil,
		},
	}

	s.authMu.Lock()
	s.authCache = resp
	s.authMu.Unlock()
	logging.Info("Auth cache refreshed", "hosts", len(hosts))
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
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/version", s.handleVersion)
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
	s.mux.HandleFunc("/api/diagnostics", s.handleDiagnostics)
	s.mux.HandleFunc("/api/diagnostics/prune", s.handleDiagnosticsPrune)
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

// ─── Basic Handlers ─────────────────────────────────────────────────────────

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Serve index.html for "/" and "/visualize/*" (SPA deep links).
	if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/visualize/") {
		http.NotFound(w, r)
		return
	}
	setSecurityHeaders(w)
	w.Header().Set("Cache-Control", "no-store")
	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}
	body := data
	if s.options.EnableTestHooks {
		body = []byte(strings.Replace(string(body), "</head>", "  <script>window.UNBOUNDCLI_TEST_HOOKS = true;</script>\n</head>", 1))
	}
	body = []byte(strings.Replace(string(body), "</head>", s.clientConfigScript()+"\n</head>", 1))
	_, _ = w.Write(body)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"version":    s.options.Version,
		"commit":     s.options.Commit,
		"build_date": s.options.BuildDate,
	})
}

// GET /api/version — returns build version.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"version":    s.options.Version,
		"commit":     s.options.Commit,
		"build_date": s.options.BuildDate,
	})
}

// ─── Shared Helpers ─────────────────────────────────────────────────────────

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

// validHostname performs a basic sanity check that a hostname looks like a
// DNS name (at least one dot, no spaces, no scheme, reasonable length).
// Wildcard patterns ("*.example.com") are accepted.
func validHostname(hostname string) bool {
	if hostname == "" || len(hostname) > 253 {
		return false
	}
	if strings.ContainsAny(hostname, " /\\") {
		return false
	}
	if strings.Contains(hostname, "://") {
		return false
	}
	// Wildcard is only valid as the leftmost label.
	if strings.HasPrefix(hostname, "*.") {
		hostname = hostname[2:]
	}
	return strings.Count(hostname, ".") >= 1
}
