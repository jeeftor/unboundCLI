package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
)

// ─── Diagnostics Types ──────────────────────────────────────────────────────

// DiagnosticSeverity classifies how bad a problem is.
type DiagnosticSeverity string

const (
	DiagSevCritical DiagnosticSeverity = "critical" // broken / security risk
	DiagSevWarning  DiagnosticSeverity = "warning"  // works but non-ideal
	DiagSevInfo     DiagnosticSeverity = "info"     // FYI
)

// DiagnosticCategory groups problems by type for UI filtering.
type DiagnosticCategory string

const (
	DiagCatDNS        DiagnosticCategory = "dns"        // DNS resolution / overrides
	DiagCatCloudflare DiagnosticCategory = "cloudflare" // CF tunnel / DNS records
	DiagCatSync       DiagnosticCategory = "sync"       // service mismatches
	DiagCatHostname   DiagnosticCategory = "hostname"   // invalid hostname
	DiagCatAuth       DiagnosticCategory = "auth"       // auth bypass / security
)

// DiagnosticIssue describes a single problem found during the scan.
type DiagnosticIssue struct {
	Severity   DiagnosticSeverity `json:"severity"`
	Category   DiagnosticCategory `json:"category"`
	Hostname   string             `json:"hostname"`
	Title      string             `json:"title"`
	Detail     string             `json:"detail"`
	Suggestion string             `json:"suggestion,omitempty"`
}

// DiagnosticsResponse is the full output of GET /api/diagnostics.
type DiagnosticsResponse struct {
	TotalEntries int               `json:"total_entries"`
	HealthyCount int               `json:"healthy_count"`
	IssueCount   int               `json:"issue_count"`
	Issues       []DiagnosticIssue `json:"issues"`
	Summary      map[string]int    `json:"summary"` // severity → count
}

// PruneAction describes a single cleanup action that would be/was performed.
type PruneAction struct {
	Hostname string `json:"hostname"`
	Service  string `json:"service"` // "unbound", "adguard", "cloudflare_tunnel", "cloudflare_dns"
	Action   string `json:"action"`  // "delete"
	Detail   string `json:"detail"`
	Success  bool   `json:"success,omitempty"`
	Error    string `json:"error,omitempty"`
}

// PruneResponse is the result of POST /api/diagnostics/prune.
type PruneResponse struct {
	DryRun  bool          `json:"dry_run"`
	Total   int           `json:"total"`
	Actions []PruneAction `json:"actions"`
}

// ─── Diagnostics Handlers ───────────────────────────────────────────────────

// GET /api/diagnostics
// Scans all entries and returns a structured list of problems:
//   - DNS resolution failures
//   - Missing Unbound/AdGuard overrides for Caddy hosts
//   - Missing CF tunnel routes for Caddy hosts
//   - Missing CF DNS CNAME records for CF tunnel hosts
//   - Stale entries (in DNS/AdGuard but not in Caddy)
//   - Invalid hostnames (trailing commas, etc.)
//   - Auth bypass risk (forward_auth without CF Access bypass)
func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	entries, _, err := s.loadEntries(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("failed to load entries: %w", err))
		return
	}

	resp := entryResponses(entries)
	issues := runDiagnostics(resp)

	// Build summary
	summary := map[string]int{
		"critical": 0,
		"warning":  0,
		"info":     0,
	}
	healthy := 0
	for _, issue := range issues {
		summary[string(issue.Severity)]++
	}
	for _, e := range resp {
		if e.StatusLabel == "Synced" {
			healthy++
		}
	}

	writeJSON(w, http.StatusOK, DiagnosticsResponse{
		TotalEntries: len(resp),
		HealthyCount: healthy,
		IssueCount:   len(issues),
		Issues:       issues,
		Summary:      summary,
	})
}

// handleDiagnosticsStream streams diagnostics via SSE.
// GET /api/diagnostics/stream — sends progress as entries load, then the full
// diagnostics result.
//
// Events:
//   - event: loading  (entries are being fetched from APIs)
//   - event: done     (diagnostics complete, includes full DiagnosticsResponse)
//   - event: error    (fatal error)
func (s *Server) handleDiagnosticsStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("streaming not supported"))
		return
	}

	sendSSEEvent(w, flusher, "loading", map[string]string{"status": "fetching entries"})

	entries, _, err := s.loadEntries(r.Context())
	if err != nil {
		sendSSEEvent(w, flusher, "error", map[string]string{"error": err.Error()})
		return
	}

	resp := entryResponses(entries)
	issues := runDiagnostics(resp)
	summary := map[string]int{}
	healthy := 0
	for _, issue := range issues {
		summary[string(issue.Severity)]++
	}
	for _, e := range resp {
		if e.StatusLabel == "Synced" {
			healthy++
		}
	}

	sendSSEEvent(w, flusher, "done", DiagnosticsResponse{
		TotalEntries: len(resp),
		HealthyCount: healthy,
		IssueCount:   len(issues),
		Issues:       issues,
		Summary:      summary,
	})
}

// runDiagnostics contains the diagnostic logic extracted from handleDiagnostics
// so it can be shared between the blocking and streaming handlers.
func runDiagnostics(resp []EntryResponse) []DiagnosticIssue {
	issues := []DiagnosticIssue{}

	for _, e := range resp {
		hostname := e.Hostname

		// ── Skip wildcard / root domain entries (e.g. ".vookie.net", "*.vookie.net").
		if api.IsWildcardOrRootHostname(hostname) {
			continue
		}

		// ── Invalid hostname (trailing comma, whitespace, etc.)
		if strings.ContainsAny(hostname, ", \t") || hostname == "" {
			issues = append(issues, DiagnosticIssue{
				Severity:   DiagSevCritical,
				Category:   DiagCatHostname,
				Hostname:   hostname,
				Title:      "Invalid hostname",
				Detail:     fmt.Sprintf("Hostname %q contains invalid characters (comma, whitespace). This is usually caused by a Caddyfile syntax error — use spaces, not commas, to separate hostnames in `host` matchers.", hostname),
				Suggestion: "Fix the Caddyfile: change `host foo.com, bar.com` to `host foo.com bar.com`",
			})
		}

		// ── DNS resolution failure
		if e.DNSResolved == "FAIL" {
			issues = append(issues, DiagnosticIssue{
				Severity:   DiagSevCritical,
				Category:   DiagCatDNS,
				Hostname:   hostname,
				Title:      "DNS resolution failed",
				Detail:     fmt.Sprintf("Hostname %s does not resolve via local DNS. This means LAN clients cannot reach it.", hostname),
				Suggestion: "Add a DNS override in Unbound and/or AdGuard Home pointing this hostname to the Caddy server IP.",
			})
		}

		// ── Caddy host missing Unbound override
		if e.CaddyUpstream != "" && !e.UnboundStatus.Configured {
			issues = append(issues, DiagnosticIssue{
				Severity:   DiagSevWarning,
				Category:   DiagCatSync,
				Hostname:   hostname,
				Title:      "Missing Unbound DNS override",
				Detail:     fmt.Sprintf("Hostname %s is in Caddy (upstream %s) but has no Unbound DNS override. LAN clients using Unbound won't resolve it.", hostname, e.CaddyUpstream),
				Suggestion: fmt.Sprintf("Add Unbound override: %s → Caddy server IP", hostname),
			})
		}

		// ── Caddy host missing AdGuard override
		if e.CaddyUpstream != "" && !e.AdguardStatus.Configured {
			issues = append(issues, DiagnosticIssue{
				Severity:   DiagSevWarning,
				Category:   DiagCatSync,
				Hostname:   hostname,
				Title:      "Missing AdGuard DNS rewrite",
				Detail:     fmt.Sprintf("Hostname %s is in Caddy (upstream %s) but has no AdGuard DNS rewrite. LAN clients using AdGuard won't resolve it.", hostname, e.CaddyUpstream),
				Suggestion: fmt.Sprintf("Add AdGuard rewrite: %s → Caddy server IP", hostname),
			})
		}

		// ── Caddy host missing CF tunnel route
		if e.CaddyUpstream != "" && !e.CloudflareStatus.Configured {
			issues = append(issues, DiagnosticIssue{
				Severity:   DiagSevWarning,
				Category:   DiagCatCloudflare,
				Hostname:   hostname,
				Title:      "Missing Cloudflare tunnel route",
				Detail:     fmt.Sprintf("Hostname %s is in Caddy but has no Cloudflare tunnel ingress rule. WAN clients cannot reach it.", hostname),
				Suggestion: fmt.Sprintf("Add CF tunnel route: %s → %s", hostname, e.CaddyUpstream),
			})
		}

		// ── CF tunnel route exists but no DNS CNAME
		if e.CloudflareStatus.Configured && !e.CloudflareStatus.HasDNSRecord {
			issues = append(issues, DiagnosticIssue{
				Severity:   DiagSevCritical,
				Category:   DiagCatCloudflare,
				Hostname:   hostname,
				Title:      "Missing Cloudflare DNS CNAME record",
				Detail:     fmt.Sprintf("Hostname %s has a CF tunnel ingress rule but no public DNS CNAME record. WAN clients cannot resolve it.", hostname),
				Suggestion: "Call POST /api/cloudflare/repair-dns to create missing CNAME records, or add manually in the CF dashboard.",
			})
		}

		// ── Stale entry (in DNS/AdGuard but not in Caddy)
		if e.CaddyUpstream == "" && e.StatusLabel == "Stale" {
			source := e.DataSource
			issues = append(issues, DiagnosticIssue{
				Severity:   DiagSevWarning,
				Category:   DiagCatSync,
				Hostname:   hostname,
				Title:      "Stale entry — in DNS but not in Caddy",
				Detail:     fmt.Sprintf("Hostname %s exists in %s but has no Caddy reverse proxy route. It may have been removed from the Caddyfile but not cleaned up from DNS.", hostname, source),
				Suggestion: fmt.Sprintf("Remove the stale DNS override for %s, or re-add it to the Caddyfile.", hostname),
			})
		}

		// ── Caddy-only entry (in Caddy but not in any DNS)
		if e.CaddyUpstream != "" && e.StatusLabel == "Caddy Only" {
			issues = append(issues, DiagnosticIssue{
				Severity:   DiagSevWarning,
				Category:   DiagCatSync,
				Hostname:   hostname,
				Title:      "Caddy-only — not in any DNS",
				Detail:     fmt.Sprintf("Hostname %s is in Caddy but not in Unbound, AdGuard, or DHCP DNS. Only Caddy knows about it.", hostname),
				Suggestion: "Run a sync to propagate this hostname to all DNS services.",
			})
		}

		// ── Auth bypass risk
		if e.HasAuthBypass {
			issues = append(issues, DiagnosticIssue{
				Severity:   DiagSevCritical,
				Category:   DiagCatAuth,
				Hostname:   hostname,
				Title:      "Auth bypass risk — double login",
				Detail:     fmt.Sprintf("Hostname %s has both CF Access and forward_auth without a bypass policy. Users will hit double login (CF Access + Authentik).", hostname),
				Suggestion: "Add a CF Access bypass policy for this hostname, or remove forward_auth.",
			})
		}
	}

	return issues
}

// caddyServerIP returns the configured Caddy server IP.
func (s *Server) caddyServerIP() string {
	rt := s.runtimeSnapshot()
	if rt.CaddyEndpoint.ServerIP != "" {
		return rt.CaddyEndpoint.ServerIP
	}
	return "10.0.0.15" // fallback
}

// POST /api/diagnostics/prune
// Body: {"dry_run": true, "hostname": "optional.example.com"}
// Finds stale entries (in DNS/Cloudflare but NOT in Caddy) and optionally
// removes them from Unbound, AdGuard, Cloudflare tunnel, and Cloudflare DNS.
// If hostname is omitted, all stale entries are pruned.
// If dry_run is true (default), only reports what would be deleted.
func (s *Server) handleDiagnosticsPrune(w http.ResponseWriter, r *http.Request) {
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
		DryRun   bool   `json:"dry_run"`
		Hostname string `json:"hostname"` // optional: prune only this hostname
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
		// dry_run defaults to true if not specified
	} else {
		req.DryRun = true
	}

	entries, _, err := s.loadEntries(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("failed to load entries: %w", err))
		return
	}

	resp := entryResponses(entries)
	runtime := s.runtimeSnapshot()
	actions := []PruneAction{}

	for _, e := range resp {
		hostname := e.Hostname

		// Skip wildcard/root entries
		if api.IsWildcardOrRootHostname(hostname) {
			continue
		}

		// If a specific hostname was requested, skip others
		if req.Hostname != "" && !strings.EqualFold(hostname, req.Hostname) {
			continue
		}

		// Only prune stale entries (no Caddy upstream)
		if e.CaddyUpstream != "" {
			continue
		}

		// ── Unbound override
		if e.UnboundStatus.Configured && runtime.Clients.Unbound != nil {
			action := PruneAction{
				Hostname: hostname,
				Service:  "unbound",
				Action:   "delete",
				Detail:   fmt.Sprintf("Remove Unbound DNS override for %s (→ %s)", hostname, e.UnboundStatus.IP),
			}
			if !req.DryRun {
				parts := strings.SplitN(hostname, ".", 2)
				if len(parts) == 2 {
					overrides, err := runtime.Clients.Unbound.GetOverrides()
					if err == nil {
						deleted := 0
						for _, o := range overrides {
							if strings.EqualFold(o.Host, parts[0]) && strings.EqualFold(o.Domain, parts[1]) {
								if delErr := runtime.Clients.Unbound.DeleteOverride(o.UUID); delErr == nil {
									deleted++
								}
							}
						}
						if deleted > 0 {
							if err := runtime.Clients.Unbound.ApplyChanges(); err != nil {
								logging.Warn("Failed to apply Unbound changes during prune", "error", err)
							}
							action.Success = true
							action.Detail = fmt.Sprintf("Deleted %d Unbound override(s) for %s", deleted, hostname)
						} else {
							action.Error = "no matching override found"
						}
					} else {
						action.Error = err.Error()
					}
				}
			}
			actions = append(actions, action)
		}

		// ── AdGuard rewrite
		if e.AdguardStatus.Configured && runtime.Clients.Adguard != nil {
			action := PruneAction{
				Hostname: hostname,
				Service:  "adguard",
				Action:   "delete",
				Detail:   fmt.Sprintf("Remove AdGuard DNS rewrite for %s (→ %s)", hostname, e.AdguardStatus.IP),
			}
			if !req.DryRun {
				rewrites, err := runtime.Clients.Adguard.GetRewritesForDomain(hostname)
				if err == nil {
					deleted := 0
					for _, rw := range rewrites {
						if delErr := runtime.Clients.Adguard.DeleteRewrite(rw.Domain, rw.Answer); delErr == nil {
							deleted++
						}
					}
					if deleted > 0 {
						action.Success = true
						action.Detail = fmt.Sprintf("Deleted %d AdGuard rewrite(s) for %s", deleted, hostname)
					} else {
						action.Error = "no matching rewrite found"
					}
				} else {
					action.Error = err.Error()
				}
			}
			actions = append(actions, action)
		}

		// ── Cloudflare tunnel route
		if e.CloudflareStatus.Configured && runtime.Clients.Cloudflare != nil {
			tunnelName := e.CloudflareStatus.TunnelName
			action := PruneAction{
				Hostname: hostname,
				Service:  "cloudflare_tunnel",
				Action:   "delete",
				Detail:   fmt.Sprintf("Remove CF tunnel route for %s on tunnel %s (→ %s)", hostname, tunnelName, e.CloudflareStatus.Service),
			}
			if !req.DryRun {
				// Use DeleteTunnelRuleInTunnel with the specific tunnel ID
				tunnelID := e.CloudflareStatus.TunnelID
				if tunnelID != "" {
					if delErr := runtime.Clients.Cloudflare.DeleteTunnelRuleInTunnel(hostname, tunnelID); delErr == nil {
						action.Success = true
						action.Detail = fmt.Sprintf("Deleted CF tunnel route for %s on tunnel %s", hostname, tunnelName)
					} else {
						action.Error = delErr.Error()
					}
				} else {
					// Fallback: try default tunnel
					if delErr := runtime.Clients.Cloudflare.DeleteTunnelRule(hostname); delErr == nil {
						action.Success = true
						action.Detail = fmt.Sprintf("Deleted CF tunnel route for %s", hostname)
					} else {
						action.Error = delErr.Error()
					}
				}
			}
			actions = append(actions, action)
		}

		// ── Cloudflare DNS CNAME
		if e.CloudflareStatus.HasDNSRecord && runtime.Clients.Cloudflare != nil {
			action := PruneAction{
				Hostname: hostname,
				Service:  "cloudflare_dns",
				Action:   "delete",
				Detail:   fmt.Sprintf("Remove CF DNS CNAME record for %s", hostname),
			}
			if !req.DryRun {
				if delErr := runtime.Clients.Cloudflare.DeleteDNSRecord(hostname); delErr == nil {
					action.Success = true
					action.Detail = fmt.Sprintf("Deleted CF DNS CNAME for %s", hostname)
				} else {
					action.Error = delErr.Error()
				}
			}
			actions = append(actions, action)
		}
	}

	writeJSON(w, http.StatusOK, PruneResponse{
		DryRun:  req.DryRun,
		Total:   len(actions),
		Actions: actions,
	})
}
