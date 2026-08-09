package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"slices"

	"github.com/jeeftor/caddy-dns-sync/internal/auth"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

// ─── Auth Inventory Types ───────────────────────────────────────────────────

// AuthInventoryResponse is the response for GET /api/auth/inventory.
type AuthInventoryResponse struct {
	Hosts   []models.HostAuth `json:"hosts"`
	Sources AuthSources       `json:"sources"`
}

// AuthSources indicates which auth discovery sources were queried.
type AuthSources struct {
	CloudflareAccess bool `json:"cloudflare_access"`
	Authentik        bool `json:"authentik"`
}

// ─── Auth Inventory Handlers ────────────────────────────────────────────────

// handleAuthInventory returns the auth discovery results for all hostnames.
// GET /api/auth/inventory — returns cached auth inventory (populated at
// startup and after mutations). Falls back to live query if cache is empty.
func (s *Server) handleAuthInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	// Try cache first.
	s.authMu.RLock()
	cached := s.authCache
	s.authMu.RUnlock()

	if cached != nil {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	// Cache miss — do a live query (and populate cache for next time).
	ctx := r.Context()
	runtime := s.runtimeSnapshot()

	entries, _, err := s.loadEntries(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("loading entries for auth discovery: %w", err))
		return
	}

	authMap, err := auth.Discover(ctx, entries, runtime.Clients.Cloudflare, runtime.Clients.Authentik)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("auth discovery failed: %w", err))
		return
	}

	hosts := make([]models.HostAuth, 0, len(authMap))
	for _, ha := range authMap {
		hosts = append(hosts, *ha)
	}
	sortHostAuthByName(hosts)

	response := &AuthInventoryResponse{
		Hosts: hosts,
		Sources: AuthSources{
			CloudflareAccess: runtime.Clients.Cloudflare != nil,
			Authentik:        runtime.Clients.Authentik != nil,
		},
	}

	// Store in cache.
	s.authMu.Lock()
	s.authCache = response
	s.authMu.Unlock()

	writeJSON(w, http.StatusOK, response)
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

// ─── Auth Helpers ───────────────────────────────────────────────────────────

// handleAuthFixDoubleLogin creates a CF Access bypass app + policy for a
// hostname to fix the double-login pattern (Pattern F).
// POST /api/auth/fix-double-login — body: {"hostname": "users.vookie.net"}
// This creates a self-hosted CF Access app with a bypass-all policy so
// CF Access lets traffic through and Caddy's forward_auth becomes the
// sole auth layer.
func (s *Server) handleAuthFixDoubleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req struct {
		Hostname string `json:"hostname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if req.Hostname == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("hostname is required"))
		return
	}

	runtime := s.runtimeSnapshot()
	if runtime.Clients.Cloudflare == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("Cloudflare client not configured"))
		return
	}

	app, policy, err := runtime.Clients.Cloudflare.CreateBypassApp(req.Hostname)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("creating bypass app for %s: %w", req.Hostname, err))
		return
	}

	// Invalidate auth cache so next request re-discovers.
	s.authMu.Lock()
	s.authCache = nil
	s.authMu.Unlock()

	response := map[string]interface{}{
		"hostname":   req.Hostname,
		"app_id":     app.ID,
		"app_name":   app.Name,
		"app_domain": app.Domain,
		"fixed":      true,
	}
	if policy != nil {
		response["policy_id"] = policy.ID
		response["policy_decision"] = policy.Decision
	}

	logging.Info("Fixed double-login via bypass app", "hostname", req.Hostname, "appID", app.ID)
	writeJSON(w, http.StatusOK, response)
}

func sortHostAuthByName(hosts []models.HostAuth) {
	slices.SortFunc(hosts, func(a, b models.HostAuth) int {
		if a.Hostname < b.Hostname {
			return -1
		}
		if a.Hostname > b.Hostname {
			return 1
		}
		return 0
	})
}
