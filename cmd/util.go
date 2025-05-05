package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/jeeftor/unboundCLI/internal/api"
)

// Override is an alias for api.DNSOverride to simplify usage in the cmd package
type Override = api.DNSOverride

// NewClient creates a new API client
func NewClient(cfg api.Config) *api.Client {
	return api.NewClient(cfg)
}

// formatJSON formats the given data as indented JSON
func formatJSON(data interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling to JSON: %w", err)
	}
	return string(jsonBytes), nil
}
