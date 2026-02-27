package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdguardClient_AddRewrite(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and endpoint
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/control/rewrite/add" {
			t.Errorf("Expected /control/rewrite/add endpoint, got %s", r.URL.Path)
		}

		// Verify Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json content type")
		}

		// Verify Authorization header exists
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Errorf("Expected Authorization header")
		}

		// Parse request body
		var rewrite Rewrite
		if err := json.NewDecoder(r.Body).Decode(&rewrite); err != nil {
			t.Errorf("Failed to parse request body: %v", err)
		}

		// Verify request data
		if rewrite.Domain != "test.example.com" {
			t.Errorf("Expected domain 'test.example.com', got '%s'", rewrite.Domain)
		}
		if rewrite.Answer != "192.168.1.15" {
			t.Errorf("Expected answer '192.168.1.15', got '%s'", rewrite.Answer)
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client
	config := AdguardConfig{
		BaseURL:  server.URL,
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
	}
	client := NewAdguardClient(config)

	// Test AddRewrite
	err := client.AddRewrite("test.example.com", "192.168.1.15")
	if err != nil {
		t.Errorf("AddRewrite failed: %v", err)
	}
}

func TestAdguardClient_ListRewrites(t *testing.T) {
	// Create test response data
	testRewrites := []Rewrite{
		{Domain: "test1.example.com", Answer: "192.168.1.15"},
		{Domain: "test2.example.com", Answer: "192.168.1.15"},
	}

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and endpoint
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/control/rewrite/list" {
			t.Errorf("Expected /control/rewrite/list endpoint, got %s", r.URL.Path)
		}

		// Return test data
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testRewrites)
	}))
	defer server.Close()

	// Create client
	config := AdguardConfig{
		BaseURL:  server.URL,
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
	}
	client := NewAdguardClient(config)

	// Test ListRewrites
	rewrites, err := client.ListRewrites()
	if err != nil {
		t.Errorf("ListRewrites failed: %v", err)
	}

	if len(rewrites) != 2 {
		t.Errorf("Expected 2 rewrites, got %d", len(rewrites))
	}

	if rewrites[0].Domain != "test1.example.com" {
		t.Errorf("Expected first domain 'test1.example.com', got '%s'", rewrites[0].Domain)
	}
}

func TestAdguardClient_UpdateRewrite(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and endpoint
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/control/rewrite/update" {
			t.Errorf("Expected /control/rewrite/update endpoint, got %s", r.URL.Path)
		}

		// Parse request body
		var updatePayload RewriteUpdate
		if err := json.NewDecoder(r.Body).Decode(&updatePayload); err != nil {
			t.Errorf("Failed to parse request body: %v", err)
		}

		// Verify request data
		if updatePayload.Target.Domain != "old.example.com" {
			t.Errorf("Expected target domain 'old.example.com', got '%s'", updatePayload.Target.Domain)
		}
		if updatePayload.Update.Domain != "new.example.com" {
			t.Errorf("Expected update domain 'new.example.com', got '%s'", updatePayload.Update.Domain)
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client
	config := AdguardConfig{
		BaseURL:  server.URL,
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
	}
	client := NewAdguardClient(config)

	// Test UpdateRewrite
	target := Rewrite{Domain: "old.example.com", Answer: "192.168.1.15"}
	update := Rewrite{Domain: "new.example.com", Answer: "192.168.1.16"}

	err := client.UpdateRewrite(target, update)
	if err != nil {
		t.Errorf("UpdateRewrite failed: %v", err)
	}
}

func TestAdguardClient_DeleteRewrite(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and endpoint
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/control/rewrite/delete" {
			t.Errorf("Expected /control/rewrite/delete endpoint, got %s", r.URL.Path)
		}

		// Parse request body
		var rewrite Rewrite
		if err := json.NewDecoder(r.Body).Decode(&rewrite); err != nil {
			t.Errorf("Failed to parse request body: %v", err)
		}

		// Verify request data
		if rewrite.Domain != "test.example.com" {
			t.Errorf("Expected domain 'test.example.com', got '%s'", rewrite.Domain)
		}
		if rewrite.Answer != "192.168.1.15" {
			t.Errorf("Expected answer '192.168.1.15', got '%s'", rewrite.Answer)
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client
	config := AdguardConfig{
		BaseURL:  server.URL,
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
	}
	client := NewAdguardClient(config)

	// Test DeleteRewrite
	err := client.DeleteRewrite("test.example.com", "192.168.1.15")
	if err != nil {
		t.Errorf("DeleteRewrite failed: %v", err)
	}
}

func TestAdguardClient_RewriteExists(t *testing.T) {
	// Create test response data
	testRewrites := []Rewrite{
		{Domain: "existing.example.com", Answer: "192.168.1.15"},
		{Domain: "another.example.com", Answer: "192.168.1.16"},
	}

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testRewrites)
	}))
	defer server.Close()

	// Create client
	config := AdguardConfig{
		BaseURL:  server.URL,
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
	}
	client := NewAdguardClient(config)

	// Test existing rewrite
	exists, err := client.RewriteExists("existing.example.com", "192.168.1.15")
	if err != nil {
		t.Errorf("RewriteExists failed: %v", err)
	}
	if !exists {
		t.Errorf("Expected rewrite to exist")
	}

	// Test non-existing rewrite
	exists, err = client.RewriteExists("nonexistent.example.com", "192.168.1.15")
	if err != nil {
		t.Errorf("RewriteExists failed: %v", err)
	}
	if exists {
		t.Errorf("Expected rewrite to not exist")
	}
}

func TestAdguardClient_GetRewritesForDomain(t *testing.T) {
	// Create test response data
	testRewrites := []Rewrite{
		{Domain: "test.example.com", Answer: "192.168.1.15"},
		{Domain: "test.example.com", Answer: "192.168.1.16"},
		{Domain: "other.example.com", Answer: "192.168.1.17"},
	}

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testRewrites)
	}))
	defer server.Close()

	// Create client
	config := AdguardConfig{
		BaseURL:  server.URL,
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
	}
	client := NewAdguardClient(config)

	// Test getting rewrites for specific domain
	rewrites, err := client.GetRewritesForDomain("test.example.com")
	if err != nil {
		t.Errorf("GetRewritesForDomain failed: %v", err)
	}

	if len(rewrites) != 2 {
		t.Errorf("Expected 2 rewrites for test.example.com, got %d", len(rewrites))
	}

	// Verify all returned rewrites are for the requested domain
	for _, rewrite := range rewrites {
		if rewrite.Domain != "test.example.com" {
			t.Errorf("Expected domain 'test.example.com', got '%s'", rewrite.Domain)
		}
	}
}

func TestNewAdguardClientFromConfig(t *testing.T) {
	config := Config{
		APIKey:    "testkey",
		APISecret: "testsecret",
		BaseURL:   "https://opnsense.example.com",
		Insecure:  true,
	}

	client := NewAdguardClientFromConfig(config, "http://192.168.0.1:3000")

	if client.BaseURL != "http://192.168.0.1:3000" {
		t.Errorf("Expected BaseURL 'http://192.168.0.1:3000', got '%s'", client.BaseURL)
	}

	if client.Username != "testkey" {
		t.Errorf("Expected Username 'testkey', got '%s'", client.Username)
	}

	if client.Password != "testsecret" {
		t.Errorf("Expected Password 'testsecret', got '%s'", client.Password)
	}
}
