package api

import (
	"encoding/json"
	"testing"
)

// TestExtractHostnamesWithUpstreams tests the Caddy config parsing with real-world nested routes
func TestExtractHostnamesWithUpstreams(t *testing.T) {
	// Real Caddy config JSON from production
	configJSON := `{"admin":{"listen":"0.0.0.0:2019"},"apps":{"http":{"servers":{"srv0":{"listen":[":443"],"routes":[{"match":[{"host":["*.donkey-beaver.ts.net"]}],"terminal":true},{"handle":[{"handler":"subroute","routes":[{"group":"group54","handle":[{"handler":"subroute","routes":[{"handle":[{"handler":"reverse_proxy","headers":{"request":{"set":{"Host":["{http.reverse_proxy.upstream.hostport}"],"X-Real-Ip":["{http.request.remote.host}"]}}},"upstreams":[{"dial":"192.168.1.112:9000"}]}]}]}],"match":[{"host":["auth.example.com"]}]},{"group":"group54","handle":[{"handler":"subroute","routes":[{"handle":[{"handler":"reverse_proxy","headers":{"request":{"set":{"Host":["{http.reverse_proxy.upstream.hostport}"],"X-Real-Ip":["{http.request.remote.host}"]}}},"upstreams":[{"dial":"192.168.1.2:5380"}]}]}]}],"match":[{"host":["dns.example.com"]}]},{"group":"group54","handle":[{"handler":"subroute","routes":[{"handle":[{"handler":"reverse_proxy","headers":{"request":{"set":{"Host":["{http.reverse_proxy.upstream.hostport}"],"X-Real-Ip":["{http.request.remote.host}"]}}},"upstreams":[{"dial":"192.168.1.6:8123"}]}]}]}],"match":[{"host":["ha.example.com"]}]},{"group":"group54","handle":[{"handler":"subroute","routes":[{"group":"group51","handle":[{"handler":"subroute","routes":[{"handle":[{"handler":"reverse_proxy","upstreams":[{"dial":"192.168.1.120:5000"}]}]}]}],"match":[{"host":["frigate.example.com"]}]},{"group":"group51","handle":[{"handler":"subroute","routes":[{"handle":[{"handler":"tracing","span":"dvr"},{"handler":"reverse_proxy","upstreams":[{"dial":"192.168.1.120:5000"}]}]}]}],"match":[{"host":["dvr.example.com"]}]},{"group":"group51","handle":[{"handler":"subroute","routes":[{"handle":[{"handler":"reverse_proxy","headers":{"request":{"set":{"Host":["{http.reverse_proxy.upstream.hostport}"],"X-Real-Ip":["{http.request.remote.host}"]}}},"upstreams":[{"dial":"192.168.1.160:8090"}]}]}]}],"match":[{"host":["hdr.example.com"]}]},{"group":"group51","handle":[{"handler":"subroute","routes":[{"handle":[{"handler":"reverse_proxy","headers":{"request":{"set":{"Host":["{http.reverse_proxy.upstream.hostport}"],"X-Real-Ip":["{http.request.remote.host}"]}}},"upstreams":[{"dial":"192.168.1.188:60072"}]}]}]}],"match":[{"host":["scanopy.example.com"]}]},{"group":"group51","handle":[{"handler":"subroute","routes":[{"handle":[{"abort":true,"handler":"static_response"}]}]}]},{"handle":[{"handler":"reverse_proxy","headers":{"request":{"set":{"Host":["{http.reverse_proxy.upstream.hostport}"],"X-Real-Ip":["{http.request.remote.host}"]}}},"upstreams":[{"dial":"192.168.1.6:6052"}]}]}]}],"match":[{"host":["esphome.example.com"]}]}]}],"match":[{"host":["*.example.com"]}],"terminal":true}]}}},"tls":{"automation":{"policies":[{"get_certificate":[{"via":"tailscale"}],"subjects":["*.donkey-beaver.ts.net"]},{"issuers":[{"challenges":{"dns":{"propagation_delay":30000000000,"provider":{"api_token":"{env.CLOUDFLARE_API_TOKEN}","name":"cloudflare"},"resolvers":["1.1.1.1"]}},"email":"jeffstein@gmail.com","module":"acme"}],"subjects":["*.example.com"]}]}}},"logging":{"logs":{"default":{"encoder":{"format":"json"},"level":"DEBUG"}}},"storage":{"module":"file_system","root":"/etc/caddy/storage"}}`

	var config map[string]interface{}
	err := json.Unmarshal([]byte(configJSON), &config)
	if err != nil {
		t.Fatalf("Failed to parse test config JSON: %v", err)
	}

	client := NewCaddyClient("192.168.1.15", 2019)
	result, err := client.ExtractHostnamesWithUpstreams(config)
	if err != nil {
		t.Fatalf("ExtractHostnamesWithUpstreams failed: %v", err)
	}

	// Test cases for critical hostnames with nested routes
	tests := []struct {
		hostname string
		expected string
	}{
		// Simple cases
		{"auth.example.com", "192.168.1.112:9000"},
		{"dns.example.com", "192.168.1.2:5380"},
		{"ha.example.com", "192.168.1.6:8123"},

		// Nested routes (group51 inside esphome.example.com)
		{"frigate.example.com", "192.168.1.120:5000"},
		{"dvr.example.com", "192.168.1.120:5000"},
		{"hdr.example.com", "192.168.1.160:8090"},
		{"scanopy.example.com", "192.168.1.188:60072"},

		// Critical test: esphome.example.com should NOT get upstream from nested routes
		// It should get its own upstream from the handle at the same level as the match
		{"esphome.example.com", "192.168.1.6:6052"},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			actual, exists := result[tt.hostname]
			if !exists {
				t.Errorf("Hostname %s not found in result", tt.hostname)
				return
			}
			if actual != tt.expected {
				t.Errorf("Hostname %s: expected upstream %s, got %s", tt.hostname, tt.expected, actual)
			}
		})
	}

	// Verify esphome.example.com specifically (the bug case)
	t.Run("esphome.example.com_bug_regression", func(t *testing.T) {
		upstream, exists := result["esphome.example.com"]
		if !exists {
			t.Fatal("esphome.example.com not found in result")
		}

		// This was the bug: it was returning 192.168.1.120:5000 (from frigate/dvr)
		// instead of the correct 192.168.1.6:6052
		if upstream == "192.168.1.120:5000" {
			t.Error("BUG DETECTED: esphome.example.com is incorrectly using upstream from nested route (frigate/dvr)")
		}

		if upstream != "192.168.1.6:6052" {
			t.Errorf("esphome.example.com has wrong upstream: expected 192.168.1.6:6052, got %s", upstream)
		}
	})
}

// TestExtractHostnamesWithUpstreams_EmptyConfig tests handling of empty config
func TestExtractHostnamesWithUpstreams_EmptyConfig(t *testing.T) {
	client := NewCaddyClient("192.168.1.15", 2019)

	config := map[string]interface{}{}
	_, err := client.ExtractHostnamesWithUpstreams(config)
	if err == nil {
		t.Error("Expected error for empty config, got nil")
	}
}

// TestExtractHostnamesWithUpstreams_NoUpstream tests handling of routes without upstreams
func TestExtractHostnamesWithUpstreams_NoUpstream(t *testing.T) {
	configJSON := `{
		"apps": {
			"http": {
				"servers": {
					"srv0": {
						"routes": [
							{
								"match": [{"host": ["test.example.com"]}],
								"handle": [
									{
										"handler": "static_response",
										"body": "Hello"
									}
								]
							}
						]
					}
				}
			}
		}
	}`

	var config map[string]interface{}
	err := json.Unmarshal([]byte(configJSON), &config)
	if err != nil {
		t.Fatalf("Failed to parse test config JSON: %v", err)
	}

	client := NewCaddyClient("192.168.1.15", 2019)
	result, err := client.ExtractHostnamesWithUpstreams(config)
	if err != nil {
		t.Fatalf("ExtractHostnamesWithUpstreams failed: %v", err)
	}

	// Should not include hostname without upstream
	if _, exists := result["test.example.com"]; exists {
		t.Error("Hostname without upstream should not be in result")
	}
}
