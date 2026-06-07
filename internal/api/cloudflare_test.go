package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCloudflareTunnelFixtureIncludesIngressMetadata(t *testing.T) {
	body, err := os.ReadFile("../status/testdata/cloudflare_tunnels.json")
	if err != nil {
		t.Fatalf("failed to read Cloudflare tunnel fixture: %v", err)
	}

	var fixture struct {
		Tunnels []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Ingress []struct {
				Hostname      string `json:"hostname"`
				Service       string `json:"service"`
				OriginRequest struct {
					HTTPHostHeader string `json:"httpHostHeader"`
					NoTLSVerify    bool   `json:"noTLSVerify"`
					Http2Origin    bool   `json:"http2Origin"`
				} `json:"originRequest"`
			} `json:"ingress"`
		} `json:"tunnels"`
	}
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatalf("failed to parse Cloudflare tunnel fixture: %v", err)
	}
	if len(fixture.Tunnels) != 1 || len(fixture.Tunnels[0].Ingress) != 1 {
		t.Fatalf("unexpected fixture shape: %#v", fixture)
	}
	ingress := fixture.Tunnels[0].Ingress[0]
	if ingress.Hostname != "app.example.test" || ingress.OriginRequest.HTTPHostHeader != "app.example.test" {
		t.Fatalf("fixture should include host header ingress metadata: %#v", ingress)
	}
	if !ingress.OriginRequest.NoTLSVerify || !ingress.OriginRequest.Http2Origin {
		t.Fatalf("fixture should include origin request booleans: %#v", ingress.OriginRequest)
	}
}

func TestListTunnels(t *testing.T) {
	accountID := "test-account-id"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		expectedPath := fmt.Sprintf("/client/v4/accounts/%s/cfd_tunnel", accountID)
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [
				{
					"id": "tunnel-1-id",
					"name": "my-tunnel",
					"created_at": "2021-01-01T00:00:00Z",
					"deleted_at": null,
					"connections": []
				},
				{
					"id": "tunnel-2-id",
					"name": "other-tunnel",
					"created_at": "2022-06-01T00:00:00Z",
					"deleted_at": null,
					"connections": []
				}
			],
			"result_info": {
				"page": 1,
				"per_page": 20,
				"total_pages": 1,
				"count": 2,
				"total_count": 2
			}
		}`)
	}))
	defer server.Close()

	cfg := CloudflareConfig{
		APIToken:  "test-token",
		AccountID: accountID,
	}
	client, err := NewCloudflareClientWithBaseURL(cfg, server.URL+"/client/v4")
	if err != nil {
		t.Fatalf("Failed to create CloudflareClient: %v", err)
	}

	tunnels, err := client.ListTunnels()
	if err != nil {
		t.Fatalf("ListTunnels failed: %v", err)
	}

	if len(tunnels) != 2 {
		t.Errorf("Expected 2 tunnels, got %d", len(tunnels))
	}

	if tunnels[0].ID != "tunnel-1-id" {
		t.Errorf("Expected tunnel ID 'tunnel-1-id', got '%s'", tunnels[0].ID)
	}
	if tunnels[0].Name != "my-tunnel" {
		t.Errorf("Expected tunnel name 'my-tunnel', got '%s'", tunnels[0].Name)
	}
	if tunnels[1].ID != "tunnel-2-id" {
		t.Errorf("Expected tunnel ID 'tunnel-2-id', got '%s'", tunnels[1].ID)
	}
}

func TestListZones(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/client/v4/zones" {
			t.Errorf("Expected path /client/v4/zones, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")

		zones := []map[string]interface{}{
			{"id": "zone-1-id", "name": "example.com", "status": "active"},
			{"id": "zone-2-id", "name": "other.com", "status": "active"},
		}
		resp := map[string]interface{}{
			"success":     true,
			"errors":      []interface{}{},
			"messages":    []interface{}{},
			"result":      zones,
			"result_info": map[string]interface{}{"page": 1, "per_page": 20, "total_pages": 1, "count": 2, "total_count": 2},
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := CloudflareConfig{
		APIToken: "test-token",
	}
	client, err := NewCloudflareClientWithBaseURL(cfg, server.URL+"/client/v4")
	if err != nil {
		t.Fatalf("Failed to create CloudflareClient: %v", err)
	}

	zones, err := client.ListZones()
	if err != nil {
		t.Fatalf("ListZones failed: %v", err)
	}

	if len(zones) != 2 {
		t.Errorf("Expected 2 zones, got %d", len(zones))
	}

	if zones[0].ID != "zone-1-id" {
		t.Errorf("Expected zone ID 'zone-1-id', got '%s'", zones[0].ID)
	}
	if zones[0].Name != "example.com" {
		t.Errorf("Expected zone name 'example.com', got '%s'", zones[0].Name)
	}
	if zones[1].ID != "zone-2-id" {
		t.Errorf("Expected zone ID 'zone-2-id', got '%s'", zones[1].ID)
	}
	if zones[1].Name != "other.com" {
		t.Errorf("Expected zone name 'other.com', got '%s'", zones[1].Name)
	}
}

// --- helpers ---

func newTestClient(t *testing.T, handler http.Handler) (*CloudflareClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	cfg := CloudflareConfig{
		APIToken:  "test-token",
		AccountID: "test-account",
		ZoneID:    "test-zone",
		TunnelID:  "test-tunnel-uuid",
	}
	client, err := NewCloudflareClientWithBaseURL(cfg, srv.URL+"/client/v4")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client, srv
}

func tunnelConfigResponse() string {
	return `{
		"success": true,
		"errors": [],
		"messages": [],
		"result": {
			"tunnel_id": "test-tunnel-uuid",
			"version": 1,
			"config": {
				"ingress": [
					{"hostname": "app.example.com", "service": "http://192.168.1.15:80"},
					{"service": "http_status:404"}
				]
			}
		}
	}`
}

func dnsListResponse(records []map[string]interface{}) string {
	resp := map[string]interface{}{
		"success":     true,
		"errors":      []interface{}{},
		"messages":    []interface{}{},
		"result":      records,
		"result_info": map[string]interface{}{"page": 1, "per_page": 100, "total_pages": 1, "count": len(records), "total_count": len(records)},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func dnsRecordResponse(id, name, content string) string {
	resp := map[string]interface{}{
		"success":  true,
		"errors":   []interface{}{},
		"messages": []interface{}{},
		"result":   map[string]interface{}{"id": id, "type": "CNAME", "name": name, "content": content, "proxied": true, "ttl": 1},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

// --- SetTunnelIngress ---

func TestSetTunnelIngress_SendsRulesWithCatchAll(t *testing.T) {
	var capturedBody map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/accounts/test-account/cfd_tunnel/test-tunnel-uuid/configurations",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.Method {
			case http.MethodGet:
				fmt.Fprint(w, tunnelConfigResponse())
			case http.MethodPut:
				json.NewDecoder(r.Body).Decode(&capturedBody)
				fmt.Fprint(w, tunnelConfigResponse())
			default:
				t.Errorf("unexpected method %s", r.Method)
			}
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	err := client.SetTunnelIngress(map[string]string{
		"app.example.com": "http://192.168.1.15:80",
	})
	if err != nil {
		t.Fatalf("SetTunnelIngress failed: %v", err)
	}

	config, _ := capturedBody["config"].(map[string]interface{})
	ingress, _ := config["ingress"].([]interface{})

	if len(ingress) != 2 {
		t.Fatalf("expected 2 ingress rules (1 rule + catch-all), got %d", len(ingress))
	}

	// Last rule must be the catch-all
	last := ingress[len(ingress)-1].(map[string]interface{})
	if last["service"] != "http_status:404" {
		t.Errorf("last ingress rule should be catch-all, got %v", last["service"])
	}

	// First rule should be our hostname
	first := ingress[0].(map[string]interface{})
	if first["hostname"] != "app.example.com" {
		t.Errorf("expected hostname 'app.example.com', got %v", first["hostname"])
	}
	if first["service"] != "http://192.168.1.15:80" {
		t.Errorf("expected service 'http://192.168.1.15:80', got %v", first["service"])
	}
}

func TestSetTunnelIngress_EmptyRulesOnlyCatchAll(t *testing.T) {
	var capturedBody map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/accounts/test-account/cfd_tunnel/test-tunnel-uuid/configurations",
		func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, tunnelConfigResponse())
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := client.SetTunnelIngress(map[string]string{}); err != nil {
		t.Fatalf("SetTunnelIngress failed: %v", err)
	}

	config, _ := capturedBody["config"].(map[string]interface{})
	ingress, _ := config["ingress"].([]interface{})

	if len(ingress) != 1 {
		t.Fatalf("empty rules should produce exactly 1 catch-all rule, got %d", len(ingress))
	}
	only := ingress[0].(map[string]interface{})
	if only["service"] != "http_status:404" {
		t.Errorf("only rule should be catch-all, got %v", only["service"])
	}
}

func TestSetTunnelIngress_MultipleRules(t *testing.T) {
	var capturedBody map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/accounts/test-account/cfd_tunnel/test-tunnel-uuid/configurations",
		func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, tunnelConfigResponse())
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	rules := map[string]string{
		"app.example.com":  "http://192.168.1.15:80",
		"api.example.com":  "http://192.168.1.16:8080",
		"home.example.com": "http://192.168.1.17:443",
	}
	if err := client.SetTunnelIngress(rules); err != nil {
		t.Fatalf("SetTunnelIngress failed: %v", err)
	}

	config, _ := capturedBody["config"].(map[string]interface{})
	ingress, _ := config["ingress"].([]interface{})

	// 3 rules + catch-all
	if len(ingress) != 4 {
		t.Fatalf("expected 4 ingress rules, got %d", len(ingress))
	}
	last := ingress[len(ingress)-1].(map[string]interface{})
	if last["service"] != "http_status:404" {
		t.Errorf("last rule must be catch-all, got %v", last["service"])
	}
}

func TestSetTunnelIngress_PreservesExistingRuleMetadata(t *testing.T) {
	var capturedBody map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/accounts/test-account/cfd_tunnel/test-tunnel-uuid/configurations",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.Method {
			case http.MethodGet:
				fmt.Fprint(w, `{
					"success": true,
					"errors": [],
					"messages": [],
					"result": {
						"tunnel_id": "test-tunnel-uuid",
						"version": 1,
						"config": {
							"ingress": [
								{
									"hostname": "app.example.com",
									"path": "/api/*",
									"service": "http://192.168.1.15:80",
									"originRequest": {
										"httpHostHeader": "app.internal"
									}
								},
								{"service": "http_status:404"}
							]
						}
					}
				}`)
			case http.MethodPut:
				json.NewDecoder(r.Body).Decode(&capturedBody)
				fmt.Fprint(w, tunnelConfigResponse())
			default:
				t.Errorf("unexpected method %s", r.Method)
			}
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := client.SetTunnelIngress(map[string]string{
		"app.example.com": "http://192.168.1.16:80",
	}); err != nil {
		t.Fatalf("SetTunnelIngress failed: %v", err)
	}

	config, _ := capturedBody["config"].(map[string]interface{})
	ingress, _ := config["ingress"].([]interface{})
	if len(ingress) != 2 {
		t.Fatalf("expected preserved rule plus catch-all, got %d rules", len(ingress))
	}

	first := ingress[0].(map[string]interface{})
	if first["hostname"] != "app.example.com" {
		t.Fatalf("expected preserved hostname app.example.com, got %v", first["hostname"])
	}
	if first["service"] != "http://192.168.1.16:80" {
		t.Fatalf("expected service to be updated, got %v", first["service"])
	}
	if first["path"] != "/api/*" {
		t.Fatalf("expected path metadata to be preserved, got %v", first["path"])
	}
	originRequest, ok := first["originRequest"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected originRequest metadata to be preserved, got %#v", first["originRequest"])
	}
	if originRequest["httpHostHeader"] != "app.internal" {
		t.Fatalf("expected httpHostHeader to be preserved, got %v", originRequest["httpHostHeader"])
	}
}

func TestUpdateTunnelRulePreservesOptionalFields(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var capturedBody map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/accounts/test-account/cfd_tunnel/test-tunnel-uuid/configurations",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.Method {
			case http.MethodGet:
				fmt.Fprint(w, `{
					"success": true,
					"errors": [],
					"messages": [],
					"result": {
						"tunnel_id": "test-tunnel-uuid",
						"version": 1,
						"config": {
							"ingress": [
								{
									"hostname": "app.example.com",
									"path": "/api/*",
									"service": "http://old-caddy:80",
									"originRequest": {
										"httpHostHeader": "old.example.com",
										"noTLSVerify": true,
										"http2Origin": true,
										"access": {
											"required": true,
											"teamName": "team"
										}
									}
								},
								{"service": "http_status:404"}
							]
						}
					}
				}`)
			case http.MethodPut:
				json.NewDecoder(r.Body).Decode(&capturedBody)
				fmt.Fprint(w, tunnelConfigResponse())
			default:
				t.Errorf("unexpected method %s", r.Method)
			}
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := client.UpdateTunnelRule(IngressRuleSpec{
		Hostname:       "app.example.com",
		Service:        "http://192.168.1.15:80",
		HTTPHostHeader: "app.example.com",
	}); err != nil {
		t.Fatalf("UpdateTunnelRule failed: %v", err)
	}

	config, _ := capturedBody["config"].(map[string]interface{})
	ingress, _ := config["ingress"].([]interface{})
	if len(ingress) != 2 {
		t.Fatalf("expected patched rule plus catch-all, got %d rules", len(ingress))
	}

	first := ingress[0].(map[string]interface{})
	if first["hostname"] != "app.example.com" {
		t.Fatalf("expected hostname app.example.com, got %v", first["hostname"])
	}
	if first["path"] != "/api/*" {
		t.Fatalf("expected path to be preserved, got %v", first["path"])
	}
	if first["service"] != "http://192.168.1.15:80" {
		t.Fatalf("expected service to be patched, got %v", first["service"])
	}
	originRequest, ok := first["originRequest"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected originRequest to be preserved, got %#v", first["originRequest"])
	}
	if originRequest["httpHostHeader"] != "app.example.com" {
		t.Fatalf("expected host header to be patched, got %v", originRequest["httpHostHeader"])
	}
	if originRequest["noTLSVerify"] != true {
		t.Fatalf("expected noTLSVerify to be preserved, got %v", originRequest["noTLSVerify"])
	}
	if originRequest["http2Origin"] != true {
		t.Fatalf("expected http2Origin to be preserved, got %v", originRequest["http2Origin"])
	}
	access, ok := originRequest["access"].(map[string]interface{})
	if !ok || access["required"] != true {
		t.Fatalf("expected access policy to be preserved, got %#v", originRequest["access"])
	}
	last := ingress[len(ingress)-1].(map[string]interface{})
	if last["service"] != "http_status:404" {
		t.Fatalf("expected catch-all to remain last, got %#v", last)
	}
}

// --- ListManagedDNSRecords ---

func TestListManagedDNSRecords_FiltersToTunnelCNAMEs(t *testing.T) {
	records := []map[string]interface{}{
		{"id": "r1", "type": "CNAME", "name": "app.example.com", "content": "test-tunnel-uuid.cfargotunnel.com"},
		{"id": "r2", "type": "CNAME", "name": "api.example.com", "content": "test-tunnel-uuid.cfargotunnel.com"},
		{"id": "r3", "type": "CNAME", "name": "other.example.com", "content": "some-other-host.com"}, // not a tunnel CNAME
		{"id": "r4", "type": "A", "name": "plain.example.com", "content": "1.2.3.4"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/test-zone/dns_records",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, dnsListResponse(records))
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	managed, err := client.ListManagedDNSRecords()
	if err != nil {
		t.Fatalf("ListManagedDNSRecords failed: %v", err)
	}

	if len(managed) != 2 {
		t.Fatalf("expected 2 managed records, got %d", len(managed))
	}
	if managed["app.example.com"] != "test-tunnel-uuid.cfargotunnel.com" {
		t.Errorf("wrong content for app.example.com: %q", managed["app.example.com"])
	}
	if managed["api.example.com"] != "test-tunnel-uuid.cfargotunnel.com" {
		t.Errorf("wrong content for api.example.com: %q", managed["api.example.com"])
	}
	if _, ok := managed["other.example.com"]; ok {
		t.Error("non-tunnel CNAME should not appear in managed records")
	}
}

func TestListManagedDNSRecords_EmptyZone(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/test-zone/dns_records",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, dnsListResponse([]map[string]interface{}{}))
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	managed, err := client.ListManagedDNSRecords()
	if err != nil {
		t.Fatalf("ListManagedDNSRecords failed: %v", err)
	}
	if len(managed) != 0 {
		t.Errorf("expected 0 records, got %d", len(managed))
	}
}

// --- EnsureDNSRecord ---

func TestEnsureDNSRecord_CreatesWhenAbsent(t *testing.T) {
	var createCalled bool
	var capturedCreate map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/test-zone/dns_records",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.Method {
			case http.MethodGet:
				// No existing records
				fmt.Fprint(w, dnsListResponse([]map[string]interface{}{}))
			case http.MethodPost:
				createCalled = true
				json.NewDecoder(r.Body).Decode(&capturedCreate)
				fmt.Fprint(w, dnsRecordResponse("new-id", "app.example.com", "test-tunnel-uuid.cfargotunnel.com"))
			default:
				t.Errorf("unexpected method %s", r.Method)
			}
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := client.EnsureDNSRecord("app.example.com"); err != nil {
		t.Fatalf("EnsureDNSRecord failed: %v", err)
	}

	if !createCalled {
		t.Error("expected POST to create DNS record")
	}
	if capturedCreate["type"] != "CNAME" {
		t.Errorf("expected type CNAME, got %v", capturedCreate["type"])
	}
	if capturedCreate["name"] != "app.example.com" {
		t.Errorf("expected name 'app.example.com', got %v", capturedCreate["name"])
	}
	wantContent := "test-tunnel-uuid.cfargotunnel.com"
	if capturedCreate["content"] != wantContent {
		t.Errorf("expected content %q, got %v", wantContent, capturedCreate["content"])
	}
}

func TestEnsureDNSRecord_NoOpWhenCorrect(t *testing.T) {
	requestCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/test-zone/dns_records",
		func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodGet {
				// Record already exists with correct target
				records := []map[string]interface{}{
					{"id": "existing-id", "type": "CNAME", "name": "app.example.com", "content": "test-tunnel-uuid.cfargotunnel.com"},
				}
				fmt.Fprint(w, dnsListResponse(records))
			} else {
				t.Errorf("unexpected %s request — should be no-op", r.Method)
			}
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := client.EnsureDNSRecord("app.example.com"); err != nil {
		t.Fatalf("EnsureDNSRecord failed: %v", err)
	}

	if requestCount != 1 {
		t.Errorf("expected exactly 1 request (GET only), got %d", requestCount)
	}
}

func TestEnsureDNSRecord_UpdatesWrongTarget(t *testing.T) {
	var patchCalled bool
	var capturedPatch map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/test-zone/dns_records",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodGet {
				records := []map[string]interface{}{
					{"id": "stale-id", "type": "CNAME", "name": "app.example.com", "content": "old-tunnel.cfargotunnel.com"},
				}
				fmt.Fprint(w, dnsListResponse(records))
			} else {
				t.Errorf("unexpected method on collection endpoint: %s", r.Method)
			}
		})
	mux.HandleFunc("/client/v4/zones/test-zone/dns_records/stale-id",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodPatch {
				patchCalled = true
				json.NewDecoder(r.Body).Decode(&capturedPatch)
				fmt.Fprint(w, dnsRecordResponse("stale-id", "app.example.com", "test-tunnel-uuid.cfargotunnel.com"))
			} else {
				t.Errorf("unexpected method on record endpoint: %s", r.Method)
			}
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := client.EnsureDNSRecord("app.example.com"); err != nil {
		t.Fatalf("EnsureDNSRecord failed: %v", err)
	}

	if !patchCalled {
		t.Error("expected PATCH to update DNS record with wrong target")
	}
	if capturedPatch["content"] != "test-tunnel-uuid.cfargotunnel.com" {
		t.Errorf("expected updated content to be tunnel target, got %v", capturedPatch["content"])
	}
}

// --- DeleteDNSRecord ---

func TestDeleteDNSRecord_DeletesExistingRecord(t *testing.T) {
	var deleteCalled bool

	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/test-zone/dns_records",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			records := []map[string]interface{}{
				{"id": "del-id", "type": "CNAME", "name": "app.example.com", "content": "test-tunnel-uuid.cfargotunnel.com"},
			}
			fmt.Fprint(w, dnsListResponse(records))
		})
	mux.HandleFunc("/client/v4/zones/test-zone/dns_records/del-id",
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			deleteCalled = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":{"id":"del-id"}}`)
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := client.DeleteDNSRecord("app.example.com"); err != nil {
		t.Fatalf("DeleteDNSRecord failed: %v", err)
	}
	if !deleteCalled {
		t.Error("expected DELETE request to be made")
	}
}

func TestDeleteDNSRecord_NoOpWhenAbsent(t *testing.T) {
	deleteCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/test-zone/dns_records",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, dnsListResponse([]map[string]interface{}{}))
		})
	mux.HandleFunc("/client/v4/zones/test-zone/dns_records/",
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleteCount++
			}
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := client.DeleteDNSRecord("gone.example.com"); err != nil {
		t.Fatalf("DeleteDNSRecord should not error when record absent: %v", err)
	}
	if deleteCount != 0 {
		t.Error("should not issue DELETE when record does not exist")
	}
}

func TestDeleteDNSRecord_IgnoresNonTunnelCNAMEs(t *testing.T) {
	deleteCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/client/v4/zones/test-zone/dns_records",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// Record exists but points somewhere other than cfargotunnel.com
			records := []map[string]interface{}{
				{"id": "other-id", "type": "CNAME", "name": "app.example.com", "content": "some-other-cdn.com"},
			}
			fmt.Fprint(w, dnsListResponse(records))
		})
	mux.HandleFunc("/client/v4/zones/test-zone/dns_records/other-id",
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleteCount++
			}
		})

	client, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := client.DeleteDNSRecord("app.example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleteCount != 0 {
		t.Error("should not delete a CNAME that doesn't point to cfargotunnel.com")
	}
}

// --- GetAllTunnelsHostnames ---

func TestGetAllTunnelsHostnames(t *testing.T) {
	accountID := "test-account"

	// tunnel-alpha: active, has two hostnames
	tunnelAlphaConfig := `{
		"success": true,
		"errors": [],
		"messages": [],
		"result": {
			"tunnel_id": "tunnel-alpha-id",
			"version": 1,
			"config": {
				"ingress": [
					{"hostname": "app.example.com", "service": "http://192.168.1.10:80"},
					{"hostname": "api.example.com", "service": "http://192.168.1.11:8080"},
					{"service": "http_status:404"}
				]
			}
		}
	}`

	// tunnel-beta: active, has one hostname
	tunnelBetaConfig := `{
		"success": true,
		"errors": [],
		"messages": [],
		"result": {
			"tunnel_id": "tunnel-beta-id",
			"version": 1,
			"config": {
				"ingress": [
					{"hostname": "blog.example.com", "service": "http://192.168.1.20:80"},
					{"service": "http_status:404"}
				]
			}
		}
	}`

	tunnelListResp := fmt.Sprintf(`{
		"success": true,
		"errors": [],
		"messages": [],
		"result": [
			{
				"id": "tunnel-alpha-id",
				"name": "alpha",
				"created_at": "2021-01-01T00:00:00Z",
				"deleted_at": null,
				"connections": []
			},
			{
				"id": "tunnel-beta-id",
				"name": "beta",
				"created_at": "2022-01-01T00:00:00Z",
				"deleted_at": null,
				"connections": []
			},
			{
				"id": "tunnel-deleted-id",
				"name": "old-tunnel",
				"created_at": "2020-01-01T00:00:00Z",
				"deleted_at": "2023-06-01T00:00:00Z",
				"connections": []
			}
		],
		"result_info": {
			"page": 1,
			"per_page": 20,
			"total_pages": 1,
			"count": 3,
			"total_count": 3
		}
	}`)

	mux := http.NewServeMux()

	// List tunnels endpoint
	mux.HandleFunc(fmt.Sprintf("/client/v4/accounts/%s/cfd_tunnel", accountID),
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, tunnelListResp)
		})

	// tunnel-alpha configuration endpoint
	mux.HandleFunc(fmt.Sprintf("/client/v4/accounts/%s/cfd_tunnel/tunnel-alpha-id/configurations", accountID),
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, tunnelAlphaConfig)
		})

	// tunnel-beta configuration endpoint
	mux.HandleFunc(fmt.Sprintf("/client/v4/accounts/%s/cfd_tunnel/tunnel-beta-id/configurations", accountID),
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, tunnelBetaConfig)
		})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := CloudflareConfig{
		APIToken:  "test-token",
		AccountID: accountID,
	}
	client, err := NewCloudflareClientWithBaseURL(cfg, srv.URL+"/client/v4")
	if err != nil {
		t.Fatalf("Failed to create CloudflareClient: %v", err)
	}

	result, err := client.GetAllTunnelsHostnames()
	if err != nil {
		t.Fatalf("GetAllTunnelsHostnames failed: %v", err)
	}

	// Expect 3 hostnames from the 2 active tunnels (deleted tunnel skipped)
	if len(result) != 3 {
		t.Fatalf("expected 3 hostnames, got %d: %v", len(result), result)
	}

	// Verify app.example.com (from alpha)
	app, ok := result["app.example.com"]
	if !ok {
		t.Fatal("expected app.example.com in result")
	}
	if app.TunnelID != "tunnel-alpha-id" {
		t.Errorf("app.example.com: expected TunnelID 'tunnel-alpha-id', got %q", app.TunnelID)
	}
	if app.TunnelName != "alpha" {
		t.Errorf("app.example.com: expected TunnelName 'alpha', got %q", app.TunnelName)
	}
	if app.Service != "192.168.1.10:80" {
		t.Errorf("app.example.com: expected Service '192.168.1.10:80', got %q", app.Service)
	}

	// Verify api.example.com (from alpha)
	apiEntry, ok := result["api.example.com"]
	if !ok {
		t.Fatal("expected api.example.com in result")
	}
	if apiEntry.TunnelID != "tunnel-alpha-id" {
		t.Errorf("api.example.com: expected TunnelID 'tunnel-alpha-id', got %q", apiEntry.TunnelID)
	}

	// Verify blog.example.com (from beta)
	blog, ok := result["blog.example.com"]
	if !ok {
		t.Fatal("expected blog.example.com in result")
	}
	if blog.TunnelID != "tunnel-beta-id" {
		t.Errorf("blog.example.com: expected TunnelID 'tunnel-beta-id', got %q", blog.TunnelID)
	}
	if blog.TunnelName != "beta" {
		t.Errorf("blog.example.com: expected TunnelName 'beta', got %q", blog.TunnelName)
	}

	// Verify the deleted tunnel's ID did not appear in results
	for hostname, entry := range result {
		if entry.TunnelID == "tunnel-deleted-id" {
			t.Errorf("hostname %q belongs to deleted tunnel — should have been skipped", hostname)
		}
	}
}

// --- GetAllTunnelsDetails ---

func TestGetAllTunnelsDetails(t *testing.T) {
	accountID := "test-account"
	defaultTunnelID := "tunnel-alpha-id" // This is the "default" tunnel for this client

	// tunnel-alpha: active, 2 hostnames.
	// app.example.com has a per-rule HTTPHostHeader (overrides tunnel default).
	// api.example.com uses tunnel-level default (no per-rule OriginRequest).
	// Tunnel-level default has NoTLSVerify = true.
	tunnelAlphaConfig := `{
		"success": true,
		"errors": [],
		"messages": [],
		"result": {
			"tunnel_id": "tunnel-alpha-id",
			"version": 1,
			"config": {
				"originRequest": {
					"noTLSVerify": true
				},
				"ingress": [
					{
						"hostname": "app.example.com",
						"service": "http://192.168.1.10:80",
						"originRequest": {
							"httpHostHeader": "app.example.com"
						}
					},
					{
						"hostname": "api.example.com",
						"service": "http://192.168.1.11:8080"
					},
					{"service": "http_status:404"}
				]
			}
		}
	}`

	// tunnel-beta: active, 1 hostname, no OriginRequest settings.
	tunnelBetaConfig := `{
		"success": true,
		"errors": [],
		"messages": [],
		"result": {
			"tunnel_id": "tunnel-beta-id",
			"version": 1,
			"config": {
				"ingress": [
					{"hostname": "blog.example.com", "service": "http://192.168.1.20:80"},
					{"service": "http_status:404"}
				]
			}
		}
	}`

	tunnelListResp := fmt.Sprintf(`{
		"success": true,
		"errors": [],
		"messages": [],
		"result": [
			{
				"id": "tunnel-alpha-id",
				"name": "alpha",
				"created_at": "2021-01-01T00:00:00Z",
				"deleted_at": null,
				"connections": []
			},
			{
				"id": "tunnel-beta-id",
				"name": "beta",
				"created_at": "2022-01-01T00:00:00Z",
				"deleted_at": null,
				"connections": []
			},
			{
				"id": "tunnel-deleted-id",
				"name": "old-tunnel",
				"created_at": "2020-01-01T00:00:00Z",
				"deleted_at": "2023-06-01T00:00:00Z",
				"connections": []
			}
		],
		"result_info": {
			"page": 1,
			"per_page": 20,
			"total_pages": 1,
			"count": 3,
			"total_count": 3
		}
	}`)

	deletedCalled := false
	mux := http.NewServeMux()

	mux.HandleFunc(fmt.Sprintf("/client/v4/accounts/%s/cfd_tunnel", accountID),
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, tunnelListResp)
		})

	mux.HandleFunc(fmt.Sprintf("/client/v4/accounts/%s/cfd_tunnel/tunnel-alpha-id/configurations", accountID),
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, tunnelAlphaConfig)
		})

	mux.HandleFunc(fmt.Sprintf("/client/v4/accounts/%s/cfd_tunnel/tunnel-beta-id/configurations", accountID),
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, tunnelBetaConfig)
		})

	mux.HandleFunc(fmt.Sprintf("/client/v4/accounts/%s/cfd_tunnel/tunnel-deleted-id/configurations", accountID),
		func(w http.ResponseWriter, r *http.Request) {
			deletedCalled = true
			t.Error("should not fetch configuration for deleted tunnel")
		})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := CloudflareConfig{
		APIToken:  "test-token",
		AccountID: accountID,
		TunnelID:  defaultTunnelID,
	}
	client, err := NewCloudflareClientWithBaseURL(cfg, srv.URL+"/client/v4")
	if err != nil {
		t.Fatalf("Failed to create CloudflareClient: %v", err)
	}

	result, err := client.GetAllTunnelsDetails()
	if err != nil {
		t.Fatalf("GetAllTunnelsDetails failed: %v", err)
	}

	if deletedCalled {
		t.Error("Configuration was fetched for the deleted tunnel")
	}

	// Expect 3 hostnames from 2 active tunnels
	if len(result) != 3 {
		t.Fatalf("expected 3 hostnames, got %d: %v", len(result), result)
	}

	// app.example.com: per-rule HTTPHostHeader should win; tunnel default NoTLSVerify=true should merge
	app, ok := result["app.example.com"]
	if !ok {
		t.Fatal("expected app.example.com in result")
	}
	if app.TunnelID != "tunnel-alpha-id" {
		t.Errorf("app.example.com: wrong TunnelID: %q", app.TunnelID)
	}
	if app.HTTPHostHeader != "app.example.com" {
		t.Errorf("app.example.com: expected HTTPHostHeader 'app.example.com', got %q", app.HTTPHostHeader)
	}
	if !app.NoTLSVerify {
		t.Error("app.example.com: expected NoTLSVerify=true (from tunnel default)")
	}
	if !app.IsDefaultTunnel {
		t.Error("app.example.com: expected IsDefaultTunnel=true")
	}

	// api.example.com: no per-rule OriginRequest, tunnel default NoTLSVerify=true applies; HTTPHostHeader empty
	api, ok := result["api.example.com"]
	if !ok {
		t.Fatal("expected api.example.com in result")
	}
	if api.HTTPHostHeader != "" {
		t.Errorf("api.example.com: expected empty HTTPHostHeader, got %q", api.HTTPHostHeader)
	}
	if !api.NoTLSVerify {
		t.Error("api.example.com: expected NoTLSVerify=true (from tunnel default)")
	}
	if !api.IsDefaultTunnel {
		t.Error("api.example.com: expected IsDefaultTunnel=true")
	}

	// blog.example.com: from beta tunnel, not the default
	blog, ok := result["blog.example.com"]
	if !ok {
		t.Fatal("expected blog.example.com in result")
	}
	if blog.TunnelID != "tunnel-beta-id" {
		t.Errorf("blog.example.com: wrong TunnelID: %q", blog.TunnelID)
	}
	if blog.IsDefaultTunnel {
		t.Error("blog.example.com: expected IsDefaultTunnel=false (it's in beta tunnel)")
	}

	// No entries should come from the deleted tunnel
	for hostname, entry := range result {
		if entry.TunnelID == "tunnel-deleted-id" {
			t.Errorf("hostname %q belongs to deleted tunnel — should have been skipped", hostname)
		}
	}
}

// ensure strings import is used (avoids lint warnings if helpers are inlined later)
var _ = strings.Contains
