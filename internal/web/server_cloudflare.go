package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
)

// ─── Cloudflare Types ───────────────────────────────────────────────────────

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

type CloudflareSetRouteRequest struct {
	Hostname         string `json:"hostname"`
	Service          string `json:"service"`            // full URL, e.g. "https://192.168.1.15" or "http://192.168.1.112:8006"
	HTTPHostHeader   string `json:"http_host_header"`   // set when routing via Caddy
	OriginServerName string `json:"origin_server_name"` // TLS SNI hostname for origin
	NoTLSVerify      bool   `json:"no_tls_verify"`
}

// ─── Cloudflare Handlers ────────────────────────────────────────────────────

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
	if !validHostname(req.Hostname) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid hostname %q", req.Hostname))
		return
	}
	// Validate service URL format — must start with http:// or https://
	if !strings.HasPrefix(req.Service, "http://") && !strings.HasPrefix(req.Service, "https://") {
		writeError(w, http.StatusBadRequest, fmt.Errorf("service must be a valid URL starting with http:// or https://"))
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
// If the Accept header includes text/event-stream, streams progress via SSE;
// otherwise returns a JSON response.
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

	// Wrap CF client with request context for cancellation.
	cfClient := runtime.Clients.Cloudflare.WithContext(r.Context())

	hostnames, err := cfClient.GetTunnelHostnames()
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("failed to list tunnel hostnames: %w", err))
		return
	}
	existing, err := cfClient.ListManagedDNSRecords()
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("failed to list DNS records: %w", err))
		return
	}

	// Build the list of missing hostnames (sorted for deterministic output).
	missing := make([]string, 0, len(hostnames))
	for hostname := range hostnames {
		if _, ok := existing[hostname]; !ok {
			missing = append(missing, hostname)
		}
	}
	sort.Strings(missing)

	// Check if client wants SSE streaming.
	wantStream := strings.Contains(r.Header.Get("Accept"), "text/event-stream")

	if !wantStream {
		// Legacy JSON response — process all and return.
		fixed := []string{}
		failed := []string{}
		for _, hostname := range missing {
			if err := cfClient.EnsureDNSRecord(hostname); err != nil {
				logging.Warn("repair-dns: failed to create CNAME", "hostname", hostname, "error", err)
				failed = append(failed, hostname)
			} else {
				logging.Info("repair-dns: created CNAME", "hostname", hostname)
				fixed = append(fixed, hostname)
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"fixed":   fixed,
			"failed":  failed,
			"skipped":  len(hostnames) - len(missing),
		})
		return
	}

	// SSE streaming — send progress as each record is created.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("streaming not supported"))
		return
	}

	// Send initial summary.
	sendSSEEvent(w, flusher, "start", map[string]interface{}{
		"total":   len(hostnames),
		"missing": len(missing),
	})

	fixed := []string{}
	failed := []string{}
	for i, hostname := range missing {
		if r.Context().Err() != nil {
			break
		}
		err := cfClient.EnsureDNSRecord(hostname)
		if err != nil {
			logging.Warn("repair-dns: failed to create CNAME", "hostname", hostname, "error", err)
			failed = append(failed, hostname)
			sendSSEEvent(w, flusher, "progress", map[string]interface{}{
				"hostname": hostname,
				"status":   "failed",
				"error":    err.Error(),
				"current":  i + 1,
				"total":    len(missing),
			})
		} else {
			logging.Info("repair-dns: created CNAME", "hostname", hostname)
			fixed = append(fixed, hostname)
			sendSSEEvent(w, flusher, "progress", map[string]interface{}{
				"hostname": hostname,
				"status":   "fixed",
				"current":  i + 1,
				"total":    len(missing),
			})
		}
	}

	// Send final result.
	sendSSEEvent(w, flusher, "done", map[string]interface{}{
		"fixed":   fixed,
		"failed":  failed,
		"skipped": len(hostnames) - len(missing),
	})
}
