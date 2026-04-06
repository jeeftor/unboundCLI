package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
)

// unboundSuccessHandler returns a minimal valid UnboundDNS response
func unboundSuccessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"rows":     []interface{}{},
		"rowCount": 0,
	})
}

func runTestConnections(t *testing.T, vals map[string]string) connTestResultsMsg {
	t.Helper()
	cmd := testConnectionsCmd(vals)
	msg, ok := cmd().(connTestResultsMsg)
	if !ok {
		t.Fatal("testConnectionsCmd did not return connTestResultsMsg")
	}
	return msg
}

func resultByName(results []connTestResult, name string) (connTestResult, bool) {
	for _, r := range results {
		if r.name == name {
			return r, true
		}
	}
	return connTestResult{}, false
}

// --- disabled-path tests (no real network) ---

func TestConnTest_AdguardDisabled(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(unboundSuccessHandler))
	defer srv.Close()

	vals := map[string]string{
		"base_url":        srv.URL,
		"api_key":         "k",
		"api_secret":      "s",
		"insecure":        "true",
		"adguard_enabled": "false",
		"cf_enabled":      "false",
	}

	msg := runTestConnections(t, vals)

	ag, ok := resultByName(msg.results, "AdguardHome")
	if !ok {
		t.Fatal("AdguardHome missing from results")
	}
	if !ag.disabled {
		t.Error("AdguardHome should be disabled")
	}
	if ag.ok {
		t.Error("disabled service should not have ok=true")
	}
}

func TestConnTest_CloudflareDisabled(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(unboundSuccessHandler))
	defer srv.Close()

	vals := map[string]string{
		"base_url":        srv.URL,
		"api_key":         "k",
		"api_secret":      "s",
		"insecure":        "true",
		"adguard_enabled": "false",
		"cf_enabled":      "false",
	}

	msg := runTestConnections(t, vals)

	cf, ok := resultByName(msg.results, "Cloudflare")
	if !ok {
		t.Fatal("Cloudflare missing from results")
	}
	if !cf.disabled {
		t.Error("Cloudflare should be disabled when cf_enabled=false")
	}
}

func TestConnTest_CloudflareDisabled_EmptyToken(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(unboundSuccessHandler))
	defer srv.Close()

	// cf_enabled=true but no token — should still be treated as disabled
	vals := map[string]string{
		"base_url":        srv.URL,
		"api_key":         "k",
		"api_secret":      "s",
		"insecure":        "true",
		"adguard_enabled": "false",
		"cf_enabled":      "true",
		"cf_api_token":    "",
	}

	msg := runTestConnections(t, vals)

	cf, ok := resultByName(msg.results, "Cloudflare")
	if !ok {
		t.Fatal("Cloudflare missing from results")
	}
	if !cf.disabled {
		t.Error("Cloudflare should be disabled when token is empty")
	}
}

func TestConnTest_AlwaysReturnsThreeResults(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(unboundSuccessHandler))
	defer srv.Close()

	vals := map[string]string{
		"base_url":        srv.URL,
		"api_key":         "k",
		"api_secret":      "s",
		"insecure":        "true",
		"adguard_enabled": "false",
		"cf_enabled":      "false",
	}

	msg := runTestConnections(t, vals)

	if len(msg.results) != 3 {
		t.Errorf("expected 3 results (UnboundDNS, AdguardHome, Cloudflare), got %d", len(msg.results))
	}

	names := map[string]bool{}
	for _, r := range msg.results {
		names[r.name] = true
	}
	for _, want := range []string{"UnboundDNS", "AdguardHome", "Cloudflare"} {
		if !names[want] {
			t.Errorf("missing result for %q", want)
		}
	}
}

// --- unbound success/failure paths ---

func TestConnTest_UnboundSuccess(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(unboundSuccessHandler))
	defer srv.Close()

	vals := map[string]string{
		"base_url":        srv.URL,
		"api_key":         "k",
		"api_secret":      "s",
		"insecure":        "true",
		"adguard_enabled": "false",
		"cf_enabled":      "false",
	}

	msg := runTestConnections(t, vals)

	unbound, ok := resultByName(msg.results, "UnboundDNS")
	if !ok {
		t.Fatal("UnboundDNS missing from results")
	}
	if unbound.disabled {
		t.Error("UnboundDNS should never be disabled")
	}
	if !unbound.ok {
		t.Errorf("UnboundDNS should succeed with valid mock server, got error: %s", unbound.errMsg)
	}
}

func TestConnTest_UnboundFailure(t *testing.T) {
	vals := map[string]string{
		"base_url":        "https://127.0.0.1:19999", // nothing listening here
		"api_key":         "k",
		"api_secret":      "s",
		"insecure":        "true",
		"adguard_enabled": "false",
		"cf_enabled":      "false",
	}

	msg := runTestConnections(t, vals)

	unbound, ok := resultByName(msg.results, "UnboundDNS")
	if !ok {
		t.Fatal("UnboundDNS missing from results")
	}
	if unbound.ok {
		t.Error("UnboundDNS should fail with unreachable address")
	}
	if unbound.errMsg == "" {
		t.Error("UnboundDNS failure should include an error message")
	}
}

// --- buildExtendedConfig tests ---

func TestBuildExtendedConfig_UnboundFields(t *testing.T) {
	vals := map[string]string{
		"api_key":    "mykey",
		"api_secret": "mysecret",
		"base_url":   "https://192.168.1.1",
		"insecure":   "true",
	}

	cfg := buildExtendedConfig(vals, config.ExtendedConfig{})

	if cfg.Config.APIKey != "mykey" {
		t.Errorf("APIKey = %q, want %q", cfg.Config.APIKey, "mykey")
	}
	if cfg.Config.APISecret != "mysecret" {
		t.Errorf("APISecret = %q, want %q", cfg.Config.APISecret, "mysecret")
	}
	if cfg.Config.BaseURL != "https://192.168.1.1" {
		t.Errorf("BaseURL = %q, want %q", cfg.Config.BaseURL, "https://192.168.1.1")
	}
	if !cfg.Config.Insecure {
		t.Error("Insecure should be true")
	}
}

func TestBuildExtendedConfig_AdguardEnabled(t *testing.T) {
	vals := map[string]string{
		"adguard_enabled":  "true",
		"adguard_base_url": "http://192.168.1.10:3000",
		"adguard_username": "admin",
		"adguard_password": "pass",
		"adguard_insecure": "false",
	}

	cfg := buildExtendedConfig(vals, config.ExtendedConfig{})

	if !cfg.Adguard.Enabled {
		t.Error("Adguard.Enabled should be true")
	}
	if cfg.Adguard.BaseURL != "http://192.168.1.10:3000" {
		t.Errorf("Adguard.BaseURL = %q", cfg.Adguard.BaseURL)
	}
	if cfg.Adguard.Username != "admin" {
		t.Errorf("Adguard.Username = %q", cfg.Adguard.Username)
	}
}

func TestBuildExtendedConfig_AdguardDisabled(t *testing.T) {
	vals := map[string]string{"adguard_enabled": "false"}
	cfg := buildExtendedConfig(vals, config.ExtendedConfig{})
	if cfg.Adguard.Enabled {
		t.Error("Adguard.Enabled should be false")
	}
}

func TestBuildExtendedConfig_CloudflareFields(t *testing.T) {
	vals := map[string]string{
		"cf_enabled":           "true",
		"cf_api_token":         "tok123",
		"cf_account_id":        "acc456",
		"cf_zone_id":           "zone789",
		"cf_tunnel_id":         "tun000",
		"cf_caddy_service_url": "http://caddy:80",
	}

	cfg := buildExtendedConfig(vals, config.ExtendedConfig{})

	if !cfg.Cloudflare.Enabled {
		t.Error("Cloudflare.Enabled should be true")
	}
	if cfg.Cloudflare.APIToken != "tok123" {
		t.Errorf("APIToken = %q", cfg.Cloudflare.APIToken)
	}
	if cfg.Cloudflare.AccountID != "acc456" {
		t.Errorf("AccountID = %q", cfg.Cloudflare.AccountID)
	}
	if cfg.Cloudflare.ZoneID != "zone789" {
		t.Errorf("ZoneID = %q", cfg.Cloudflare.ZoneID)
	}
	if cfg.Cloudflare.TunnelID != "tun000" {
		t.Errorf("TunnelID = %q", cfg.Cloudflare.TunnelID)
	}
	if cfg.Cloudflare.CaddyServiceURL != "http://caddy:80" {
		t.Errorf("CaddyServiceURL = %q", cfg.Cloudflare.CaddyServiceURL)
	}
}

func TestBuildExtendedConfig_PreservesExistingCaddyConfig(t *testing.T) {
	existing := config.ExtendedConfig{
		Caddy: config.CaddyConfig{ServerIP: "10.0.0.5", ServerPort: 2019},
	}

	cfg := buildExtendedConfig(map[string]string{}, existing)

	if cfg.Caddy.ServerIP != "10.0.0.5" {
		t.Errorf("Caddy.ServerIP should be preserved, got %q", cfg.Caddy.ServerIP)
	}
	if cfg.Caddy.ServerPort != 2019 {
		t.Errorf("Caddy.ServerPort should be preserved, got %d", cfg.Caddy.ServerPort)
	}
}

func TestBuildExtendedConfig_InsecureFalse(t *testing.T) {
	vals := map[string]string{"insecure": "false"}
	cfg := buildExtendedConfig(vals, config.ExtendedConfig{})
	if cfg.Config.Insecure {
		t.Error("Insecure should be false when val is 'false'")
	}
}

// --- connTestResult struct behaviour ---

func TestConnTestResult_DisabledHasNoError(t *testing.T) {
	r := connTestResult{name: "AdguardHome", disabled: true}
	if r.ok {
		t.Error("disabled result should not be ok")
	}
	if r.errMsg != "" {
		t.Error("disabled result should have empty errMsg")
	}
}

func TestConnTestResult_SuccessFields(t *testing.T) {
	r := connTestResult{name: "UnboundDNS", ok: true}
	if r.disabled {
		t.Error("successful result should not be disabled")
	}
	if r.errMsg != "" {
		t.Error("successful result should have empty errMsg")
	}
}

func TestConnTestResult_FailureFields(t *testing.T) {
	r := connTestResult{name: "UnboundDNS", ok: false, errMsg: "connection refused"}
	if r.disabled {
		t.Error("failure result should not be disabled")
	}
	if r.errMsg == "" {
		t.Error("failure result should have errMsg")
	}
}

// TestConnTest_UnboundNeverDisabled verifies UnboundDNS always appears regardless of other settings
func TestConnTest_UnboundNeverDisabled(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(unboundSuccessHandler))
	defer srv.Close()

	combinations := []map[string]string{
		{"adguard_enabled": "true", "cf_enabled": "true"},
		{"adguard_enabled": "false", "cf_enabled": "false"},
		{"adguard_enabled": "true", "cf_enabled": "false"},
	}

	for _, extra := range combinations {
		vals := map[string]string{
			"base_url":   srv.URL,
			"api_key":    "k",
			"api_secret": "s",
			"insecure":   "true",
		}
		for k, v := range extra {
			vals[k] = v
		}

		msg := runTestConnections(t, vals)
		unbound, ok := resultByName(msg.results, "UnboundDNS")
		if !ok {
			t.Error("UnboundDNS missing")
			continue
		}
		if unbound.disabled {
			t.Errorf("UnboundDNS should never be disabled (vals: %v)", extra)
		}
	}
}

// Ensure api package is used (avoids unused import errors if tests are restructured)
var _ = api.Config{}
