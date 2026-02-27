package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
)

// Override is an alias for api.DNSOverride to simplify usage in the cmd package
type Override = api.DNSOverride

// NewClient creates a new API client
func NewClient(cfg api.Config) *api.Client {
	return api.NewClient(cfg)
}

// NewCaddyClient creates a new Caddy API client
func NewCaddyClient(serverIP string, serverPort int) *api.CaddyClient {
	return api.NewCaddyClient(serverIP, serverPort)
}

// NewAdguardClient creates a new AdguardHome API client
func NewAdguardClient(cfg api.AdguardConfig) *api.AdguardClient {
	return api.NewAdguardClient(cfg)
}

// formatJSON formats the given data as indented JSON
func formatJSON(data interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling to JSON: %w", err)
	}
	return string(jsonBytes), nil
}
