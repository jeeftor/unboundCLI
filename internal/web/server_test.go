package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/syncplan"
)

func TestAPIRoutesUseSharedServicesWithoutLAN(t *testing.T) {
	caddy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config/" {
			t.Fatalf("unexpected Caddy path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"apps": {
				"http": {
					"servers": {
						"srv0": {
							"routes": [
								{
									"match": [{"host": ["web.example.test"]}],
									"handle": [
										{
											"handler": "reverse_proxy",
											"upstreams": [{"dial": "10.0.0.5:8080"}]
										}
									]
								}
							]
						}
					}
				}
			}
		}`)
	}))
	defer caddy.Close()

	host, port := splitWebTestServerHostPort(t, caddy.URL)
	runtime := &app.Runtime{
		CaddyEndpoint: app.CaddyEndpoint{ServerIP: host, ServerPort: port},
		Clients: app.ClientSet{
			Caddy: api.NewCaddyClient(host, port),
		},
	}
	server := NewServer(runtime)

	configResp := getJSON[ConfigResponse](t, server, "/api/config")
	if configResp.Caddy.ServerIP != host {
		t.Fatalf("expected Caddy host %q, got %q", host, configResp.Caddy.ServerIP)
	}

	entriesResp := getJSON[EntriesResponse](t, server, "/api/entries")
	if len(entriesResp.Entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entriesResp.Entries))
	}
	if entriesResp.Entries[0].Hostname != "web.example.test" {
		t.Fatalf("expected web.example.test, got %q", entriesResp.Entries[0].Hostname)
	}

	rawEntries := getRawJSON(t, server, "/api/entries")
	if !bytes.Contains(rawEntries, []byte(`"hostname"`)) {
		t.Fatalf("entries response should expose lowercase hostname key: %s", string(rawEntries))
	}
	if bytes.Contains(rawEntries, []byte(`"Hostname"`)) {
		t.Fatalf("entries response should not expose Go field names: %s", string(rawEntries))
	}
	if !bytes.Contains(rawEntries, []byte(`"status"`)) || bytes.Contains(rawEntries, []byte(`"Status"`)) {
		t.Fatalf("service report should expose lowercase status key: %s", string(rawEntries))
	}

	planResp := getJSON[PlanResponse](t, server, "/api/sync/plan?service=unbound")
	if len(planResp.Actions) != 1 {
		t.Fatalf("expected one planned action, got %d", len(planResp.Actions))
	}
	if planResp.Actions[0].Service != "unbound" || planResp.Actions[0].Type != "add" {
		t.Fatalf("unexpected planned action: %#v", planResp.Actions[0])
	}

	applyBody, err := json.Marshal(ApplyRequest{
		DryRun:  true,
		Actions: planResp.Actions,
	})
	if err != nil {
		t.Fatalf("failed to marshal apply request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(applyBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var applyResp ApplyResponse
	if err := json.NewDecoder(rec.Body).Decode(&applyResp); err != nil {
		t.Fatalf("failed to decode apply response: %v", err)
	}
	if !applyResp.Result.Success {
		t.Fatalf("expected dry-run apply success, got %#v", applyResp.Result)
	}
	if applyResp.Result.ItemsAdded != 1 {
		t.Fatalf("expected dry-run add count 1, got %d", applyResp.Result.ItemsAdded)
	}
}

func TestIndexRouteServesHTML(t *testing.T) {
	server := NewServer(&app.Runtime{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content type %q", contentType)
	}
	if nosniff := rec.Header().Get("X-Content-Type-Options"); nosniff != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", nosniff)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Caddy DNS Sync")) {
		t.Fatalf("index HTML missing app title: %s", rec.Body.String())
	}
}

func TestApplyRejectsOversizedRequestBody(t *testing.T) {
	server := NewServer(&app.Runtime{})

	body := `"` + strings.Repeat("a", 1<<20+1) + `"`
	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("http: request body too large")) {
		t.Fatalf("expected body-too-large error, got %s", rec.Body.String())
	}
	if nosniff := rec.Header().Get("X-Content-Type-Options"); nosniff != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", nosniff)
	}
}

func TestApplyRejectsRealMutationRequests(t *testing.T) {
	server := NewServer(&app.Runtime{})
	body, err := json.Marshal(ApplyRequest{
		DryRun: false,
		Actions: []syncplan.Action{
			{
				Type:     "add",
				Service:  "unbound",
				Hostname: "unsafe.example.test",
				NewIP:    "192.168.1.15",
				Enabled:  true,
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal apply request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("dry-run")) {
		t.Fatalf("expected dry-run-only error, got %s", rec.Body.String())
	}
}

func getJSON[T any](t *testing.T, handler http.Handler, path string) T {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: expected status 200, got %d: %s", path, rec.Code, rec.Body.String())
	}
	var out T
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("GET %s: failed to decode JSON: %v", path, err)
	}
	return out
}

func getRawJSON(t *testing.T, handler http.Handler, path string) []byte {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: expected status 200, got %d: %s", path, rec.Code, rec.Body.String())
	}
	return rec.Body.Bytes()
}

func splitWebTestServerHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}
	host, portString, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("failed to split server host/port: %v", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatalf("failed to parse server port: %v", err)
	}
	return host, port
}

var _ = syncplan.Action{}
