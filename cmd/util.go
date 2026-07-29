package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
)

// Override is an alias for api.DNSOverride to simplify usage in the cmd package
type Override = api.DNSOverride

// formatJSON formats the given data as indented JSON
func formatJSON(data interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling to JSON: %w", err)
	}
	return string(jsonBytes), nil
}
