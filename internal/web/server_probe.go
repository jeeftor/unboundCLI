package web

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jeeftor/caddy-dns-sync/internal/logging"
)

// ─── Probe Types ────────────────────────────────────────────────────────────

// ProbeResponse is the result of an HTTP reachability probe.
type ProbeResponse struct {
	Reachable  bool   `json:"reachable"`
	StatusCode int    `json:"status_code,omitempty"`
	LatencyMS  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
	ProbeURL   string `json:"probe_url"`
}

// DNSProbeResponse is the result of a public DNS lookup via Cloudflare's 1.1.1.1 resolver.
type DNSProbeResponse struct {
	Resolved  bool     `json:"resolved"`
	CNAME     string   `json:"cname,omitempty"`
	Addresses []string `json:"addresses,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// ─── Probe Handlers ─────────────────────────────────────────────────────────

// handleProbe does a quick HTTP/HTTPS HEAD probe to an upstream address.
// GET /api/probe?upstream=192.168.1.15:6868&hostname=foo.example.com
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
