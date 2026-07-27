package api

import (
	"testing"
)

func TestNewAuthentikClient_RequiresToken(t *testing.T) {
	_, err := NewAuthentikClient(AuthentikConfig{
		APIToken: "",
		BaseURL:  "https://auth.example.com",
	})
	if err == nil {
		t.Fatal("Expected error for empty token, got nil")
	}
}

func TestNewAuthentikClient_RequiresBaseURL(t *testing.T) {
	_, err := NewAuthentikClient(AuthentikConfig{
		APIToken: "test-token",
		BaseURL:  "",
	})
	if err == nil {
		t.Fatal("Expected error for empty base URL, got nil")
	}
}

func TestNewAuthentikClient_ValidConfig(t *testing.T) {
	client, err := NewAuthentikClient(AuthentikConfig{
		APIToken: "test-token",
		BaseURL:  "https://auth.example.com",
	})
	if err != nil {
		t.Fatalf("NewAuthentikClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.baseURL != "https://auth.example.com" {
		t.Errorf("Expected baseURL https://auth.example.com, got %s", client.baseURL)
	}
}

func TestNewAuthentikClient_StripsTrailingSlash(t *testing.T) {
	client, err := NewAuthentikClient(AuthentikConfig{
		APIToken: "test-token",
		BaseURL:  "https://auth.example.com/",
	})
	if err != nil {
		t.Fatalf("NewAuthentikClient failed: %v", err)
	}
	if client.baseURL != "https://auth.example.com" {
		t.Errorf("Expected trailing slash stripped, got %s", client.baseURL)
	}
}

func TestProxyModeConstants(t *testing.T) {
	if ProxyModeForwardSingle != "forward_single" {
		t.Errorf("Expected forward_single, got %s", ProxyModeForwardSingle)
	}
	if ProxyModeForwardDomain != "forward_domain" {
		t.Errorf("Expected forward_domain, got %s", ProxyModeForwardDomain)
	}
	if ProxyModeProxy != "proxy" {
		t.Errorf("Expected proxy, got %s", ProxyModeProxy)
	}
}
