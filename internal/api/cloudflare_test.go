package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
