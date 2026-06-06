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
	"github.com/jeeftor/caddy-dns-sync/internal/config"
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
	if bytes.Contains(rec.Body.Bytes(), []byte("UNBOUNDCLI_TEST_HOOKS = true")) {
		t.Fatalf("index HTML should not enable test hooks by default: %s", rec.Body.String())
	}
}

func TestIndexRouteCanEnableBrowserTestHooks(t *testing.T) {
	server := NewServerWithOptions(&app.Runtime{}, Options{EnableTestHooks: true})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("UNBOUNDCLI_TEST_HOOKS = true")) {
		t.Fatalf("index HTML missing test hook flag: %s", rec.Body.String())
	}
}

func TestConfigRouteReturnsSanitizedConfigurationSummary(t *testing.T) {
	server := NewServer(&app.Runtime{
		UnboundConfig: api.Config{
			APIKey:    "fixture-api-key",
			APISecret: "fixture-api-secret",
			BaseURL:   "https://url-user:url-pass@opnsense.example.test",
			Insecure:  true,
		},
		AdguardConfig: config.AdguardConfig{
			Enabled:  true,
			Username: "fixture-adguard-user",
			Password: "fixture-adguard-password",
			BaseURL:  "https://adguard-user:adguard-url-pass@adguard.example.test",
		},
		CloudflareConfig: config.CloudflareConfig{
			Enabled:         true,
			APIToken:        "fixture-cloudflare-token",
			AccountID:       "fixture-account-id",
			ZoneID:          "fixture-zone-id",
			TunnelID:        "fixture-tunnel-id",
			CaddyServiceURL: "http://caddy-user:caddy-pass@caddy.example.test:2019",
		},
		CaddyEndpoint: app.CaddyEndpoint{ServerIP: "127.0.0.1", ServerPort: 2019},
		Clients: app.ClientSet{
			Caddy:   api.NewCaddyClient("127.0.0.1", 2019),
			Unbound: api.NewClient(api.Config{}),
		},
		CaddyServiceURL: "http://caddy-user:caddy-pass@caddy.example.test:2019",
	})

	resp := getJSON[ConfigResponse](t, server, "/api/config")
	if !resp.Summary.Unbound.Fields["api_key_set"] || !resp.Summary.Unbound.Fields["api_secret_set"] {
		t.Fatalf("expected OPNSense credentials to be summarized as present: %#v", resp.Summary.Unbound.Fields)
	}
	if resp.Summary.Unbound.Endpoint != "https://opnsense.example.test" {
		t.Fatalf("expected non-secret OPNSense endpoint, got %q", resp.Summary.Unbound.Endpoint)
	}
	if resp.Summary.Adguard.Endpoint != "https://adguard.example.test" {
		t.Fatalf("expected sanitized AdGuard endpoint, got %q", resp.Summary.Adguard.Endpoint)
	}
	if !resp.Summary.Cloudflare.Fields["api_token_set"] ||
		resp.Summary.Cloudflare.Details["caddy_service_url"] != "http://caddy.example.test:2019" {
		t.Fatalf("expected Cloudflare config presence and service URL, got %#v", resp.Summary.Cloudflare)
	}

	raw := getRawJSON(t, server, "/api/config")
	for _, secret := range []string{
		"fixture-api-key",
		"fixture-api-secret",
		"fixture-adguard-user",
		"fixture-adguard-password",
		"fixture-cloudflare-token",
		"fixture-account-id",
		"fixture-zone-id",
		"fixture-tunnel-id",
		"url-user",
		"url-pass",
		"adguard-user",
		"adguard-url-pass",
		"caddy-user",
		"caddy-pass",
	} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatalf("config response leaked secret or identifier %q: %s", secret, string(raw))
		}
	}
}

func TestStaticAssetsAreServedWithExpectedContentTypes(t *testing.T) {
	server := NewServer(&app.Runtime{})

	tests := []struct {
		path        string
		contentType string
		contains    []byte
	}{
		{path: "/", contentType: "text/html; charset=utf-8", contains: []byte(`<div id="app"`)},
		{path: "/static/app.js", contentType: "text/javascript; charset=utf-8", contains: []byte("async function refreshEntries")},
		{path: "/static/styles.css", contentType: "text/css; charset=utf-8", contains: []byte(".status-chip")},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tt.path, rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Content-Type"); got != tt.contentType {
			t.Fatalf("%s: expected content type %q, got %q", tt.path, tt.contentType, got)
		}
		if !bytes.Contains(rec.Body.Bytes(), tt.contains) {
			t.Fatalf("%s: response missing %q", tt.path, tt.contains)
		}
	}
}

func TestPlanRouteSupportsServiceSelection(t *testing.T) {
	caddy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config/" {
			t.Fatalf("unexpected Caddy path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"apps":{"http":{"servers":{"srv0":{"routes":[{"match":[{"host":["plan.example.test"]}],"handle":[{"handler":"reverse_proxy","upstreams":[{"dial":"10.0.0.5:8080"}]}]}]}}}}}`)
	}))
	defer caddy.Close()

	host, port := splitWebTestServerHostPort(t, caddy.URL)
	server := NewServer(&app.Runtime{
		CaddyEndpoint: app.CaddyEndpoint{ServerIP: host, ServerPort: port},
		Clients:       app.ClientSet{Caddy: api.NewCaddyClient(host, port)},
	})

	planResp := getJSON[PlanResponse](t, server, "/api/sync/plan?service=adguard")
	if len(planResp.Actions) != 1 {
		t.Fatalf("expected one action, got %#v", planResp.Actions)
	}
	if planResp.Actions[0].Service != "adguard" {
		t.Fatalf("expected adguard action, got %#v", planResp.Actions[0])
	}
	if planResp.PlanID == "" {
		t.Fatal("expected plan response to include plan_id")
	}
	if len(planResp.ActionIDs) != len(planResp.Actions) || planResp.ActionIDs[0] == "" {
		t.Fatalf("expected plan response to include action IDs, got %#v", planResp.ActionIDs)
	}
}

func TestAllPlanFiltersUnavailableWebServices(t *testing.T) {
	caddy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config/" {
			t.Fatalf("unexpected Caddy path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"apps":{"http":{"servers":{"srv0":{"routes":[{"match":[{"host":["available.example.test"]}],"handle":[{"handler":"reverse_proxy","upstreams":[{"dial":"10.0.0.5:8080"}]}]}]}}}}}`)
	}))
	defer caddy.Close()

	opnsense := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/unbound/settings/searchHostOverride" {
			t.Fatalf("unexpected OPNSense path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"rows":[]}`)
	}))
	defer opnsense.Close()

	host, port := splitWebTestServerHostPort(t, caddy.URL)
	server := NewServer(&app.Runtime{
		CaddyEndpoint: app.CaddyEndpoint{ServerIP: host, ServerPort: port},
		Clients: app.ClientSet{
			Caddy: api.NewCaddyClient(host, port),
			Unbound: api.NewClient(api.Config{
				APIKey:    "fixture-key",
				APISecret: "fixture-secret",
				BaseURL:   opnsense.URL,
				Insecure:  true,
			}),
		},
	})

	planResp := getJSON[PlanResponse](t, server, "/api/sync/plan?service=all")
	if len(planResp.Actions) != 1 {
		t.Fatalf("expected only available Unbound action, got %#v", planResp.Actions)
	}
	if planResp.Actions[0].Service != "unbound" {
		t.Fatalf("expected all plan to filter unavailable AdGuard, got %#v", planResp.Actions[0])
	}
}

func TestPlanRouteRejectsUnknownService(t *testing.T) {
	server := NewServer(&app.Runtime{})
	req := httptest.NewRequest(http.MethodGet, "/api/sync/plan?service=bogus", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
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

func TestApplyRouteAllowsDryRunOnly(t *testing.T) {
	server := NewServer(&app.Runtime{})
	action := syncplan.Action{
		Type: "add", Service: "unbound", Hostname: "dryrun.example.test", NewIP: "192.168.1.15", Enabled: true,
	}
	body, err := json.Marshal(ApplyRequest{DryRun: true, Actions: []syncplan.Action{action}})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body, err = json.Marshal(ApplyRequest{DryRun: false, PlanID: "plan-1", ActionIDs: []string{"action-1"}})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestApplyRouteRejectsUnsupportedDHCPDryRun(t *testing.T) {
	server := NewServer(&app.Runtime{})
	action := syncplan.Action{
		Type: "add", Service: "dhcp", Hostname: "dhcp.example.test", NewIP: "192.168.1.55", Enabled: true,
	}
	body, err := json.Marshal(ApplyRequest{DryRun: true, Actions: []syncplan.Action{action}})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported DHCP apply, got %d: %s", rec.Code, rec.Body.String())
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

func TestMutatingApplyRequiresTokenAndAllowedOrigin(t *testing.T) {
	server := NewServerWithOptions(&app.Runtime{}, Options{
		ApplyToken:     "test-token",
		AllowMutations: true,
		AllowedOrigin:  "http://127.0.0.1:8080",
		BoundHost:      "127.0.0.1",
	})
	body, err := json.Marshal(ApplyRequest{
		DryRun:    false,
		PlanID:    "missing-plan",
		ActionIDs: []string{"action-1"},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected missing token to be forbidden, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-UnboundCLI-Token", "test-token")
	req.Header.Set("Origin", "http://evil.example")
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected disallowed origin to be forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMutatingApplyRejectsPostedActionsEvenWithToken(t *testing.T) {
	server := NewServerWithOptions(&app.Runtime{}, Options{
		ApplyToken:     "test-token",
		AllowMutations: true,
		AllowedOrigin:  "http://127.0.0.1:8080",
		BoundHost:      "127.0.0.1",
	})
	body, err := json.Marshal(map[string]any{
		"dry_run": false,
		"actions": []syncplan.Action{
			{Type: "add", Service: "unbound", Hostname: "forged.example.test", NewIP: "192.168.1.15", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal forged request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-UnboundCLI-Token", "test-token")
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected forged action request to be rejected, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMutatingApplyUsesServerIssuedPlanActions(t *testing.T) {
	caddy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config/" {
			t.Fatalf("unexpected Caddy path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"apps":{"http":{"servers":{"srv0":{"routes":[{"match":[{"host":["sync.example.test"]}],"handle":[{"handler":"reverse_proxy","upstreams":[{"dial":"10.0.0.5:8080"}]}]}]}}}}}`)
	}))
	defer caddy.Close()

	var added bool
	var reconfigured bool
	opnsense := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/unbound/settings/searchHostOverride":
			fmt.Fprint(w, `{"rows":[]}`)
		case "/api/unbound/settings/addHostOverride":
			added = true
			fmt.Fprint(w, `{"result":"saved","uuid":"new-uuid"}`)
		case "/api/unbound/service/reconfigure":
			reconfigured = true
			fmt.Fprint(w, `{"result":"saved"}`)
		default:
			t.Fatalf("unexpected OPNSense path %s", r.URL.Path)
		}
	}))
	defer opnsense.Close()

	host, port := splitWebTestServerHostPort(t, caddy.URL)
	server := NewServerWithOptions(&app.Runtime{
		CaddyEndpoint: app.CaddyEndpoint{ServerIP: host, ServerPort: port},
		Clients: app.ClientSet{
			Caddy: api.NewCaddyClient(host, port),
			Unbound: api.NewClient(api.Config{
				APIKey:    "fixture-key",
				APISecret: "fixture-secret",
				BaseURL:   opnsense.URL,
				Insecure:  true,
			}),
		},
	}, Options{
		ApplyToken:     "test-token",
		AllowMutations: true,
		AllowedOrigin:  "http://127.0.0.1:8080",
		BoundHost:      "127.0.0.1",
	})

	configResp := getJSON[ConfigResponse](t, server, "/api/config")
	if !configResp.MutationEnabled {
		t.Fatal("expected config to report mutation-enabled local session")
	}
	planResp := getJSON[PlanResponse](t, server, "/api/sync/plan?service=unbound")
	if len(planResp.ActionIDs) != 1 {
		t.Fatalf("expected one action ID, got %#v", planResp.ActionIDs)
	}

	body, err := json.Marshal(ApplyRequest{
		DryRun:    false,
		PlanID:    planResp.PlanID,
		ActionIDs: planResp.ActionIDs,
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-UnboundCLI-Token", "test-token")
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var applyResp ApplyResponse
	if err := json.NewDecoder(rec.Body).Decode(&applyResp); err != nil {
		t.Fatalf("failed to decode apply response: %v", err)
	}
	if !applyResp.Result.Success || applyResp.Result.ItemsAdded != 1 {
		t.Fatalf("expected successful mutating apply, got %#v", applyResp.Result)
	}
	if !added || !reconfigured {
		t.Fatalf("expected add and reconfigure calls, added=%t reconfigured=%t", added, reconfigured)
	}
}

func TestMutatingApplyRejectsUnknownActionID(t *testing.T) {
	server := NewServerWithOptions(&app.Runtime{}, Options{
		ApplyToken:     "test-token",
		AllowMutations: true,
		AllowedOrigin:  "http://127.0.0.1:8080",
		BoundHost:      "127.0.0.1",
	})
	body, err := json.Marshal(ApplyRequest{
		DryRun:    false,
		PlanID:    "missing-plan",
		ActionIDs: []string{"action-missing"},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-UnboundCLI-Token", "test-token")
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected unknown plan/action to be rejected, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMutatingApplyRejectsWildcardBindHost(t *testing.T) {
	server := NewServerWithOptions(&app.Runtime{}, Options{
		ApplyToken:     "test-token",
		AllowMutations: true,
		AllowedOrigin:  "http://127.0.0.1:8080",
		BoundHost:      "",
	})
	body, err := json.Marshal(ApplyRequest{
		DryRun:    false,
		PlanID:    "plan-1",
		ActionIDs: []string{"action-1"},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-UnboundCLI-Token", "test-token")
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected wildcard bind host to be forbidden, got %d: %s", rec.Code, rec.Body.String())
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
