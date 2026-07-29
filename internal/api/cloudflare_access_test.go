package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestCloudflareClient creates a CloudflareClient pointing at a mock server.
func newTestCloudflareClient(t *testing.T, handler http.HandlerFunc) *CloudflareClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := NewCloudflareClientWithBaseURL(CloudflareConfig{
		APIToken:  "test-token",
		AccountID: "test-account-id",
		ZoneID:    "test-zone-id",
		TunnelID:  "test-tunnel-id",
	}, server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	return client
}

func TestListAccessApps(t *testing.T) {
	client := newTestCloudflareClient(t, func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/accounts/test-account-id/access/apps"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Bearer auth, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{
					"id":               "app-1",
					"name":             "Jellyfin",
					"domain":           "jellyfin.vookie.net",
					"type":             "self_hosted",
					"session_duration": "24h",
				},
				{
					"id":     "app-2",
					"name":   "Wildcard",
					"domain": "*.vookie.net",
					"type":   "self_hosted",
				},
			},
		})
	})

	apps, err := client.ListAccessApps()
	if err != nil {
		t.Fatalf("ListAccessApps failed: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("Expected 2 apps, got %d", len(apps))
	}
	if apps[0].Name != "Jellyfin" || apps[0].Domain != "jellyfin.vookie.net" {
		t.Errorf("Unexpected app[0]: %+v", apps[0])
	}
	if apps[1].Domain != "*.vookie.net" {
		t.Errorf("Expected wildcard domain, got %q", apps[1].Domain)
	}
}

func TestFindAccessAppByDomain_ExactMatch(t *testing.T) {
	client := newTestCloudflareClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{"id": "app-1", "name": "Jellyfin", "domain": "jellyfin.vookie.net", "type": "self_hosted"},
			},
		})
	})

	app, err := client.FindAccessAppByDomain("jellyfin.vookie.net")
	if err != nil {
		t.Fatalf("FindAccessAppByDomain failed: %v", err)
	}
	if app == nil {
		t.Fatal("Expected to find app, got nil")
	}
	if app.ID != "app-1" {
		t.Errorf("Expected app-1, got %s", app.ID)
	}
}

func TestFindAccessAppByDomain_WildcardMatch(t *testing.T) {
	client := newTestCloudflareClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{"id": "wildcard-1", "name": "Wildcard", "domain": "*.vookie.net", "type": "self_hosted"},
			},
		})
	})

	app, err := client.FindAccessAppByDomain("newservice.vookie.net")
	if err != nil {
		t.Fatalf("FindAccessAppByDomain failed: %v", err)
	}
	if app == nil {
		t.Fatal("Expected wildcard match, got nil")
	}
	if app.ID != "wildcard-1" {
		t.Errorf("Expected wildcard-1, got %s", app.ID)
	}
}

func TestFindAccessAppByDomain_NotFound(t *testing.T) {
	client := newTestCloudflareClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{"id": "app-1", "name": "Other", "domain": "other.vookie.net", "type": "self_hosted"},
			},
		})
	})

	app, err := client.FindAccessAppByDomain("nonexistent.vookie.net")
	if err != nil {
		t.Fatalf("FindAccessAppByDomain failed: %v", err)
	}
	if app != nil {
		t.Errorf("Expected nil for nonexistent domain, got %+v", app)
	}
}

func TestCreateAccessApp(t *testing.T) {
	client := newTestCloudflareClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if body["name"] != "test.vookie.net" {
			t.Errorf("Expected name test.vookie.net, got %v", body["name"])
		}
		if body["domain"] != "test.vookie.net" {
			t.Errorf("Expected domain test.vookie.net, got %v", body["domain"])
		}
		if body["type"] != "self_hosted" {
			t.Errorf("Expected type self_hosted, got %v", body["type"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"id":               "new-app-id",
				"name":             "test.vookie.net",
				"domain":           "test.vookie.net",
				"type":             "self_hosted",
				"session_duration": "24h",
			},
		})
	})

	app, err := client.CreateAccessApp(CreateAccessAppRequest{
		Name:   "test.vookie.net",
		Domain: "test.vookie.net",
	})
	if err != nil {
		t.Fatalf("CreateAccessApp failed: %v", err)
	}
	if app.ID != "new-app-id" {
		t.Errorf("Expected new-app-id, got %s", app.ID)
	}
}

func TestDeleteAccessApp(t *testing.T) {
	client := newTestCloudflareClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		expectedPath := "/accounts/test-account-id/access/apps/app-to-delete"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result":  map[string]interface{}{"id": "app-to-delete"},
		})
	})

	if err := client.DeleteAccessApp("app-to-delete"); err != nil {
		t.Fatalf("DeleteAccessApp failed: %v", err)
	}
}

func TestCreateServiceToken(t *testing.T) {
	client := newTestCloudflareClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		expectedPath := "/accounts/test-account-id/access/service_tokens"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"id":            "token-123",
				"name":          "test-token",
				"client_id":     "client-id-abc",
				"client_secret": "secret-xyz",
			},
		})
	})

	token, err := client.CreateServiceToken(CreateServiceTokenRequest{
		Name:     "test-token",
		Duration: "8760h",
	})
	if err != nil {
		t.Fatalf("CreateServiceToken failed: %v", err)
	}
	if token.ID != "token-123" {
		t.Errorf("Expected token-123, got %s", token.ID)
	}
	if token.ClientID != "client-id-abc" {
		t.Errorf("Expected client-id-abc, got %s", token.ClientID)
	}
	if token.ClientSecret != "secret-xyz" {
		t.Errorf("Expected secret-xyz, got %s", token.ClientSecret)
	}
}

func TestListServiceTokens(t *testing.T) {
	client := newTestCloudflareClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{"id": "token-1", "name": "Token One", "client_id": "cid-1"},
				{"id": "token-2", "name": "Token Two", "client_id": "cid-2"},
			},
		})
	})

	tokens, err := client.ListServiceTokens()
	if err != nil {
		t.Fatalf("ListServiceTokens failed: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("Expected 2 tokens, got %d", len(tokens))
	}
	if tokens[0].Name != "Token One" {
		t.Errorf("Expected Token One, got %s", tokens[0].Name)
	}
	// ClientSecret should NOT be populated on list
	if tokens[0].ClientSecret != "" {
		t.Errorf("Expected empty ClientSecret on list, got %q", tokens[0].ClientSecret)
	}
}

func TestFindServiceTokenByName(t *testing.T) {
	client := newTestCloudflareClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{"id": "token-1", "name": "Apprise", "client_id": "cid-1"},
				{"id": "token-2", "name": "Go2RTC", "client_id": "cid-2"},
			},
		})
	})

	token, err := client.FindServiceTokenByName("Go2RTC")
	if err != nil {
		t.Fatalf("FindServiceTokenByName failed: %v", err)
	}
	if token == nil {
		t.Fatal("Expected to find Go2RTC token, got nil")
	}
	if token.ID != "token-2" {
		t.Errorf("Expected token-2, got %s", token.ID)
	}
}

func TestListAccessGroups(t *testing.T) {
	client := newTestCloudflareClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{"id": "group-1", "name": "Admins"},
				{"id": "group-2", "name": "Household"},
			},
		})
	})

	groups, err := client.ListAccessGroups()
	if err != nil {
		t.Fatalf("ListAccessGroups failed: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(groups))
	}
	if groups[0].Name != "Admins" {
		t.Errorf("Expected Admins, got %s", groups[0].Name)
	}
}

func TestIsWildcardMatch(t *testing.T) {
	tests := []struct {
		wildcard string
		hostname string
		want     bool
	}{
		{"*.vookie.net", "jellyfin.vookie.net", true},
		{"*.vookie.net", "sub.app.vookie.net", true},
		{"*.vookie.net", "vookie.net", false}, // no subdomain prefix
		{"*.vookie.net", "other.example.com", false},
		{"jellyfin.vookie.net", "jellyfin.vookie.net", false}, // not a wildcard
		{"*", "anything.com", false},                          // not a valid wildcard pattern
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.wildcard, tt.hostname), func(t *testing.T) {
			got := IsWildcardMatch(tt.wildcard, tt.hostname)
			if got != tt.want {
				t.Errorf("IsWildcardMatch(%q, %q) = %v, want %v", tt.wildcard, tt.hostname, got, tt.want)
			}
		})
	}
}

func TestCreateAccessPolicy(t *testing.T) {
	client := newTestCloudflareClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if body["decision"] != "bypass" {
			t.Errorf("Expected decision bypass, got %v", body["decision"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"id":         "policy-1",
				"name":       "bypass-for-forward-auth",
				"decision":   "bypass",
				"precedence": 1,
			},
		})
	})

	policy, err := client.CreateAccessPolicy(CreateAccessPolicyRequest{
		AppID:    "app-1",
		Name:     "bypass-for-forward-auth",
		Decision: DecisionBypass,
	})
	if err != nil {
		t.Fatalf("CreateAccessPolicy failed: %v", err)
	}
	if policy.ID != "policy-1" {
		t.Errorf("Expected policy-1, got %s", policy.ID)
	}
	if policy.Decision != DecisionBypass {
		t.Errorf("Expected bypass, got %s", policy.Decision)
	}
}
