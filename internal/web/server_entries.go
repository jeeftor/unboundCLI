package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
	"github.com/jeeftor/caddy-dns-sync/internal/ownership"
	"github.com/jeeftor/caddy-dns-sync/internal/status"
	"github.com/jeeftor/caddy-dns-sync/internal/syncplan"
)

// ─── Entry Types ────────────────────────────────────────────────────────────

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
	DNSRecordID      string `json:"dns_record_id"`
}

type EntryResponse struct {
	Hostname                   string                   `json:"hostname"`
	CaddyUpstream              string                   `json:"caddy_upstream"`
	CaddyIP                    string                   `json:"caddy_ip"`
	CaddyPort                  string                   `json:"caddy_port"`
	UnboundStatus              ServiceStatusResponse    `json:"unbound_status"`
	AdguardStatus              ServiceStatusResponse    `json:"adguard_status"`
	DHCPStatus                 DHCPStatusResponse       `json:"dhcp_status"`
	DNSResolved                string                   `json:"dns_resolved"`
	CloudflareStatus           CloudflareStatusResponse `json:"cloudflare_status"`
	OverallStatus              models.SyncStatus        `json:"overall_status"`
	StatusLabel                string                   `json:"status_label"`
	DataSource                 string                   `json:"data_source"`
	HasForwardAuth             bool                     `json:"has_forward_auth"`
	HasConditionalForwardAuth  bool                     `json:"has_conditional_forward_auth,omitempty"`
	HasAuthBypass              bool                     `json:"has_auth_bypass_risk"`
	MissingRequiredForwardAuth bool                     `json:"missing_required_forward_auth,omitempty"`
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
	Status string           `json:"status"`
}

// ─── Entry Handlers ─────────────────────────────────────────────────────────

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

	// Populate the entries cache so subsequent non-streaming calls (diagnostics,
	// plan, auth) can reuse this data without re-fetching from all APIs.
	s.entriesMu.Lock()
	s.entriesCache = entries
	s.entriesReport = report
	s.entriesCacheAt = time.Now()
	s.entriesMu.Unlock()

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

// ─── Plan/Apply Handlers ────────────────────────────────────────────────────

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
	entries, report, err := s.loadFreshEntries(r.Context())
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
	planID, err := newPlanID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("issue sync plan: %w", err))
		return
	}
	actionIDs := actionIDs(actions)
	configPath, err := config.SelectedConfigPath(s.options.ConfigPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("resolve plan configuration: %w", err))
		return
	}
	configRevision, err := config.Revision(configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("read plan configuration revision: %w", err))
		return
	}
	s.storePlan(planID, actions, actionIDs, configPath, configRevision)
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
		actions, priorResult, planStatus, err := s.claimPlan(request.PlanID, request.ActionIDs)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if priorResult != nil {
			writeJSON(w, http.StatusOK, ApplyResponse{Result: priorResult, Status: planStatus})
			return
		}
		if planStatus == "applying" && actions == nil {
			writeJSON(w, http.StatusAccepted, ApplyResponse{Status: planStatus})
			return
		}
		if err := validateApplyActions(actions); err != nil {
			s.completePlan(request.PlanID, failedPlanResult(err))
			writeError(w, http.StatusBadRequest, err)
			return
		}
		result := s.applyActions(r.Context(), actions, false)
		s.completePlan(request.PlanID, result)
		writeJSON(w, http.StatusOK, ApplyResponse{Result: result, Status: "completed"})
		// Refresh auth cache — entries may have changed.
		s.invalidateEntriesCache()
		go s.refreshAuthCache()
		return
	}
	if err := validateApplyActions(request.Actions); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result := s.applyActions(r.Context(), request.Actions, request.DryRun)
	writeJSON(w, http.StatusOK, ApplyResponse{Result: result, Status: "completed"})
	if !request.DryRun {
		s.invalidateEntriesCache()
		go s.refreshAuthCache()
	}
}

func (s *Server) applyActions(ctx context.Context, actions []syncplan.Action, dryRun bool) *syncplan.Result {
	runtime := s.runtimeSnapshot()
	configPath, err := s.configPath()
	if err != nil {
		return &syncplan.Result{Errors: []string{fmt.Sprintf("resolve ownership state path: %v", err)}, Message: "Unable to apply plan"}
	}
	return syncplan.Apply(ctx, syncplan.Clients{
		Unbound:    runtime.Clients.Unbound,
		Adguard:    runtime.Clients.Adguard,
		Cloudflare: runtime.Clients.Cloudflare,
	}, syncplan.Plan{Actions: actions}, syncplan.ApplyOptions{
		DryRun:        dryRun,
		OwnershipPath: ownership.PathForConfig(configPath),
	})
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
	if !validHostname(req.Hostname) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid hostname %q", req.Hostname))
		return
	}
	if req.Service == "" {
		req.Service = "all"
	}
	if req.Service != "all" && req.Service != "unbound" && req.Service != "adguard" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid removal service %q", req.Service))
		return
	}

	state, statePath, err := s.loadOwnershipState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	runtime := s.runtimeSnapshot()
	removed := 0
	var msgs []string
	var failures []string

	// Remove from Unbound
	if (req.Service == "all" || req.Service == "unbound") && runtime.Clients.Unbound != nil {
		parts := strings.SplitN(req.Hostname, ".", 2)
		if len(parts) == 2 {
			overrides, err := runtime.Clients.Unbound.GetOverrides()
			if err == nil {
				unboundRemoved := 0
				for _, o := range overrides {
					if !strings.EqualFold(o.Host, parts[0]) || !strings.EqualFold(o.Domain, parts[1]) {
						continue
					}
					if o.Description != app.CurrentUnboundDescription {
						msgs = append(msgs, fmt.Sprintf("protected unmanaged Unbound override %s", req.Hostname))
						continue
					}
					if err := beginOwnershipIntent(&state, statePath, "delete", "unbound", "override", o.UUID, o.Server); err != nil {
						failures = append(failures, err.Error())
						continue
					}
					if delErr := runtime.Clients.Unbound.DeleteOverride(o.UUID); delErr != nil {
						failures = append(failures, fmt.Sprintf("delete Unbound override %s: %v", req.Hostname, delErr))
						continue
					}
					current, readErr := runtime.Clients.Unbound.GetOverrides()
					if readErr != nil || unboundOverridePresent(current, o.UUID) {
						if readErr != nil {
							failures = append(failures, fmt.Sprintf("verify Unbound override deletion: %v", readErr))
						} else {
							failures = append(failures, fmt.Sprintf("verify Unbound override deletion: override %s remains", req.Hostname))
						}
						continue
					}
					state.Forget("unbound", "override", o.UUID)
					state.Resolve("unbound", "override", o.UUID)
					if saveErr := ownership.Save(statePath, state); saveErr != nil {
						failures = append(failures, fmt.Sprintf("record Unbound deletion: %v", saveErr))
						continue
					}
					unboundRemoved++
					removed++
				}
				if unboundRemoved > 0 {
					if err := runtime.Clients.Unbound.ApplyChanges(); err != nil {
						failures = append(failures, fmt.Sprintf("apply Unbound changes: %v", err))
					}
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
				resource, known := state.Resources[ownership.Key("adguard", "rewrite", rw.Domain)]
				if !known || !state.Owns("adguard", "rewrite", rw.Domain) || resource.Expected != ownership.Fingerprint(rw.Answer) {
					msgs = append(msgs, fmt.Sprintf("protected unmanaged AdGuard rewrite %s", req.Hostname))
					continue
				}
				if err := beginOwnershipIntent(&state, statePath, "delete", "adguard", "rewrite", rw.Domain, rw.Answer); err != nil {
					failures = append(failures, err.Error())
					continue
				}
				if delErr := runtime.Clients.Adguard.DeleteRewrite(rw.Domain, rw.Answer); delErr != nil {
					failures = append(failures, fmt.Sprintf("delete AdGuard rewrite %s: %v", req.Hostname, delErr))
					continue
				}
				current, readErr := runtime.Clients.Adguard.GetRewritesForDomain(rw.Domain)
				if readErr != nil || adguardRewritePresent(current, rw) {
					if readErr != nil {
						failures = append(failures, fmt.Sprintf("verify AdGuard rewrite deletion: %v", readErr))
					} else {
						failures = append(failures, fmt.Sprintf("verify AdGuard rewrite deletion: rewrite %s remains", req.Hostname))
					}
					continue
				}
				state.Forget("adguard", "rewrite", rw.Domain)
				state.Resolve("adguard", "rewrite", rw.Domain)
				if saveErr := ownership.Save(statePath, state); saveErr != nil {
					failures = append(failures, fmt.Sprintf("record AdGuard deletion: %v", saveErr))
					continue
				}
				n++
				removed++
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
		"errors":  failures,
	})
	// Refresh auth cache — entries may have changed.
	s.invalidateEntriesCache()
	go s.refreshAuthCache()
}

func (s *Server) loadOwnershipState() (ownership.State, string, error) {
	configPath, err := config.SelectedConfigPath(s.options.ConfigPath)
	if err != nil {
		return ownership.State{}, "", fmt.Errorf("resolve ownership state path: %w", err)
	}
	statePath := ownership.PathForConfig(configPath)
	state, err := ownership.Load(statePath)
	if err != nil {
		return ownership.State{}, "", fmt.Errorf("load ownership state: %w", err)
	}
	return state, statePath, nil
}

func beginOwnershipIntent(state *ownership.State, path, operation, provider, kind, id, expected string) error {
	if err := state.Begin(ownership.Intent{Operation: operation, Provider: provider, Kind: kind, ID: id, Expected: ownership.Fingerprint(expected)}); err != nil {
		return fmt.Errorf("record %s intent: %w", operation, err)
	}
	if err := ownership.Save(path, *state); err != nil {
		return fmt.Errorf("persist %s intent: %w", operation, err)
	}
	return nil
}

func unboundOverridePresent(overrides []api.DNSOverride, uuid string) bool {
	for _, override := range overrides {
		if override.UUID == uuid {
			return true
		}
	}
	return false
}

func adguardRewritePresent(rewrites []api.Rewrite, want api.Rewrite) bool {
	for _, rewrite := range rewrites {
		if rewrite.Domain == want.Domain && rewrite.Answer == want.Answer {
			return true
		}
	}
	return false
}

// ─── Entry/Plan Helpers ─────────────────────────────────────────────────────

const entriesCacheTTL = 5 * time.Minute

// entriesStaleGrace is how long a stale cache is still served while a
// background refresh runs. This prevents the plan/apply endpoints from
// blocking on a full API refetch (which can exceed reverse-proxy timeouts).
const entriesStaleGrace = 30 * time.Minute

func (s *Server) loadEntries(ctx context.Context) ([]*models.Entry, status.LoadReport, error) {
	// Check cache first. If fresh, return immediately.
	// If stale but within grace period, return stale data and trigger a
	// background refresh so the next call gets fresh data without blocking.
	s.entriesMu.Lock()
	if s.entriesCache != nil {
		age := time.Since(s.entriesCacheAt)
		if age < entriesCacheTTL {
			entries := s.entriesCache
			report := s.entriesReport
			s.entriesMu.Unlock()
			return entries, report, nil
		}
		if age < entriesStaleGrace {
			entries := s.entriesCache
			report := s.entriesReport
			s.entriesMu.Unlock()
			// Refresh in the background so the next request gets fresh data.
			s.refreshEntriesCacheInBackground(ctx)
			return entries, report, nil
		}
	}
	s.entriesMu.Unlock()

	// No usable cache — do a blocking fetch with a timeout to avoid
	// hitting reverse-proxy (e.g. Cloudflare) 502 timeouts.
	fetchCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	runtime := s.runtimeSnapshot()
	entries, report, err := status.LoadEntries(fetchCtx, runtime.Clients, status.Options{
		CaddyServerIP: runtime.CaddyEndpoint.ServerIP,
	})
	if err != nil {
		// If the fetch failed but we have stale cache, return it rather
		// than erroring — a stale plan is better than a 502.
		s.entriesMu.Lock()
		if s.entriesCache != nil {
			entries := s.entriesCache
			cachedReport := s.entriesReport
			s.entriesMu.Unlock()
			logging.Warn("Entries fetch failed, serving stale cache", "error", err)
			return entries, cachedReport, nil
		}
		s.entriesMu.Unlock()
		return nil, report, err
	}

	// Populate cache.
	s.entriesMu.Lock()
	s.entriesCache = entries
	s.entriesReport = report
	s.entriesCacheAt = time.Now()
	s.entriesMu.Unlock()

	return entries, report, nil
}

// loadFreshEntries loads required plan sources without falling back to cached
// inventory. A stale source must never be treated as evidence for a mutation.
func (s *Server) loadFreshEntries(ctx context.Context) ([]*models.Entry, status.LoadReport, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	runtime := s.runtimeSnapshot()
	entries, report, err := status.LoadEntries(fetchCtx, runtime.Clients, status.Options{
		CaddyServerIP: runtime.CaddyEndpoint.ServerIP,
	})
	if err != nil {
		return nil, report, err
	}

	s.entriesMu.Lock()
	s.entriesCache = entries
	s.entriesReport = report
	s.entriesCacheAt = time.Now()
	s.entriesMu.Unlock()
	return entries, report, nil
}

// refreshEntriesCacheInBackground fetches fresh entries in a goroutine and
// updates the cache. Uses a background context so it isn't cancelled when the
// triggering request finishes.
func (s *Server) refreshEntriesCacheInBackground(_ context.Context) {
	s.refreshMu.Lock()
	if s.refreshRunning {
		s.refreshMu.Unlock()
		return
	}
	s.refreshRunning = true
	s.refreshMu.Unlock()

	go func() {
		defer func() {
			s.refreshMu.Lock()
			s.refreshRunning = false
			s.refreshMu.Unlock()
		}()
		runtime := s.runtimeSnapshot()
		entries, report, err := status.LoadEntries(context.Background(), runtime.Clients, status.Options{
			CaddyServerIP: runtime.CaddyEndpoint.ServerIP,
		})
		if err != nil {
			logging.Warn("Background entries refresh failed", "error", err)
			return
		}
		s.entriesMu.Lock()
		s.entriesCache = entries
		s.entriesReport = report
		s.entriesCacheAt = time.Now()
		s.entriesMu.Unlock()
	}()
}

// invalidateEntriesCache clears the entries cache so the next loadEntries
// call fetches fresh data. Called after mutations (sync apply, remove, etc.).
func (s *Server) invalidateEntriesCache() {
	s.entriesMu.Lock()
	s.entriesCache = nil
	s.entriesReport = status.LoadReport{}
	s.entriesCacheAt = time.Time{}
	s.entriesMu.Unlock()
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
				DNSRecordID:      entry.CloudflareStatus.DNSRecordID,
			},
			OverallStatus:              entry.OverallStatus,
			StatusLabel:                entry.OverallStatus.Label(),
			DataSource:                 entry.DataSource,
			HasForwardAuth:             entry.CaddyRoute.HasForwardAuth,
			HasConditionalForwardAuth:  entry.CaddyRoute.ConditionalForwardAuth,
			HasAuthBypass:              entry.HasAuthBypassRisk(),
			MissingRequiredForwardAuth: entry.Auth != nil && entry.Auth.IsMissingRequiredForwardAuth(),
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

func (s *Server) storePlan(planID string, actions []syncplan.Action, actionIDs []string, configPath, configRevision string) {
	s.cleanExpiredPlans()
	actionsByID := make(map[string]syncplan.Action, len(actions))
	for i, action := range actions {
		if i >= len(actionIDs) {
			break
		}
		actionsByID[actionIDs[i]] = action
	}
	s.planMu.Lock()
	s.plans[planID] = storedPlan{
		ActionsByID:    actionsByID,
		createdAt:      time.Now(),
		configPath:     configPath,
		configRevision: configRevision,
		status:         "pending",
	}
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

func (s *Server) claimPlan(planID string, actionIDs []string) ([]syncplan.Action, *syncplan.Result, string, error) {
	s.planMu.Lock()
	defer s.planMu.Unlock()
	plan, ok := s.plans[planID]
	if !ok {
		return nil, nil, "", fmt.Errorf("unknown or expired sync plan")
	}
	if plan.configPath != "" {
		currentRevision, err := config.Revision(plan.configPath)
		if err != nil {
			return nil, nil, "", fmt.Errorf("read plan configuration revision: %w", err)
		}
		if currentRevision != plan.configRevision {
			return nil, nil, "", fmt.Errorf("sync plan configuration changed; preview again before applying")
		}
	}
	if plan.status == "completed" {
		return nil, plan.result, plan.status, nil
	}
	if plan.status == "applying" {
		return nil, nil, plan.status, nil
	}
	actions := make([]syncplan.Action, 0, len(actionIDs))
	seen := make(map[string]struct{}, len(actionIDs))
	for _, actionID := range actionIDs {
		if _, duplicate := seen[actionID]; duplicate {
			return nil, nil, "", fmt.Errorf("duplicate sync action %q", actionID)
		}
		seen[actionID] = struct{}{}
		action, ok := plan.ActionsByID[actionID]
		if !ok {
			return nil, nil, "", fmt.Errorf("unknown sync action %q", actionID)
		}
		actions = append(actions, action)
	}
	plan.status = "applying"
	s.plans[planID] = plan
	return actions, nil, plan.status, nil
}

func (s *Server) completePlan(planID string, result *syncplan.Result) {
	s.planMu.Lock()
	defer s.planMu.Unlock()
	plan, ok := s.plans[planID]
	if !ok {
		return
	}
	plan.status = "completed"
	plan.result = result
	s.plans[planID] = plan
}

func failedPlanResult(err error) *syncplan.Result {
	return &syncplan.Result{Success: false, Errors: []string{err.Error()}, Message: "Plan validation failed"}
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

func newPlanID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "plan-" + hex.EncodeToString(bytes[:]), nil
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

// escapeJSONString escapes a string for safe embedding in JSON.
func escapeJSONString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
