package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
	"github.com/jeeftor/caddy-dns-sync/internal/status"
	"github.com/jeeftor/caddy-dns-sync/internal/syncplan"
)

type Server struct {
	runtime *app.Runtime
	mux     *http.ServeMux
}

type CaddyConfigResponse struct {
	ServerIP   string `json:"server_ip"`
	ServerPort int    `json:"server_port"`
}

type ConfigResponse struct {
	Caddy   CaddyConfigResponse `json:"caddy"`
	Enabled map[string]bool     `json:"enabled"`
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
	Actions []syncplan.Action `json:"actions"`
	Report  status.LoadReport `json:"report"`
}

type ApplyRequest struct {
	Actions []syncplan.Action `json:"actions"`
	DryRun  bool              `json:"dry_run"`
}

type ApplyResponse struct {
	Result *syncplan.Result `json:"result"`
}

// NewServer creates a web GUI/API server over shared app runtime services.
func NewServer(runtime *app.Runtime) *Server {
	if runtime == nil {
		runtime = &app.Runtime{}
	}
	server := &Server{runtime: runtime, mux: http.NewServeMux()}
	server.routes()
	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/api/config", s.handleConfig)
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
	_, _ = w.Write([]byte(indexHTML))
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, ConfigResponse{
		Caddy: CaddyConfigResponse{
			ServerIP:   s.runtime.CaddyEndpoint.ServerIP,
			ServerPort: s.runtime.CaddyEndpoint.ServerPort,
		},
		Enabled: map[string]bool{
			"caddy":      s.runtime.Clients.Caddy != nil,
			"unbound":    s.runtime.Clients.Unbound != nil,
			"adguard":    s.runtime.Clients.Adguard != nil,
			"dhcp":       s.runtime.Clients.DNSMasq != nil,
			"cloudflare": s.runtime.Clients.Cloudflare != nil,
		},
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
	entries, report, err := s.loadEntries(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	service := r.URL.Query().Get("service")
	plan := syncplan.BuildPlan(entries, syncplan.Options{
		Service:       service,
		CaddyServerIP: s.runtime.CaddyEndpoint.ServerIP,
	})
	writeJSON(w, http.StatusOK, PlanResponse{Actions: plan.Actions, Report: report})
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
		writeError(w, http.StatusForbidden, fmt.Errorf("web apply is dry-run only until mutating actions are protected"))
		return
	}
	result := syncplan.Apply(r.Context(), syncplan.Clients{
		Unbound: s.runtime.Clients.Unbound,
		Adguard: s.runtime.Clients.Adguard,
	}, syncplan.Plan{Actions: request.Actions}, syncplan.ApplyOptions{DryRun: request.DryRun})
	writeJSON(w, http.StatusOK, ApplyResponse{Result: result})
}

func (s *Server) loadEntries(ctx context.Context) ([]*models.Entry, status.LoadReport, error) {
	return status.LoadEntries(ctx, s.runtime.Clients, status.Options{
		CaddyServerIP: s.runtime.CaddyEndpoint.ServerIP,
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
