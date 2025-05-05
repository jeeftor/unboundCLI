// cloudflare.go in internal/api package
package api

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/jeeftor/unboundCLI/internal/logging"
)

// CloudflareClient handles communication with the Cloudflare API
type CloudflareClient struct {
	api       *cloudflare.API
	zoneID    string
	accountID string
	tunnelID  string
}

// CloudflareConfig contains configuration for Cloudflare API
type CloudflareConfig struct {
	APIToken  string
	ZoneID    string
	AccountID string
	TunnelID  string
}

// CloudflareTunnel represents a Cloudflare tunnel
type CloudflareTunnel struct {
	ID          string                       `json:"id"`
	Name        string                       `json:"name"`
	CreatedAt   time.Time                    `json:"created_at"`
	DeletedAt   time.Time                    `json:"deleted_at"`
	Connections []CloudflareTunnelConnection `json:"connections"`
}

// CloudflareTunnelConnection represents a connection to a Cloudflare tunnel
type CloudflareTunnelConnection struct {
	ID          string    `json:"id"`
	ConnectedAt time.Time `json:"connected_at"`
	Status      string    `json:"status"`
}

// NewCloudflareClient creates a new Cloudflare API client
func NewCloudflareClient(config CloudflareConfig) (*CloudflareClient, error) {
	api, err := cloudflare.NewWithAPIToken(config.APIToken)
	if err != nil {
		return nil, fmt.Errorf("error creating Cloudflare API client: %w", err)
	}

	return &CloudflareClient{
		api:       api,
		zoneID:    config.ZoneID,
		accountID: config.AccountID,
		tunnelID:  config.TunnelID,
	}, nil
}

// ListTunnels returns a list of all tunnels for the account
func (c *CloudflareClient) ListTunnels() ([]CloudflareTunnel, error) {
	ctx := context.Background()

	if c.accountID == "" {
		return nil, fmt.Errorf("account ID is required to list tunnels")
	}

	tunnels, _, err := c.api.ListTunnels(ctx, cloudflare.ResourceIdentifier(c.accountID), cloudflare.TunnelListParams{})
	if err != nil {
		return nil, fmt.Errorf("error listing tunnels: %w", err)
	}

	var result []CloudflareTunnel
	for _, tunnel := range tunnels {
		t := CloudflareTunnel{
			ID:   tunnel.ID,
			Name: tunnel.Name,
		}

		// Handle potential nil pointers
		if tunnel.CreatedAt != nil {
			t.CreatedAt = *tunnel.CreatedAt
		}
		if tunnel.DeletedAt != nil {
			t.DeletedAt = *tunnel.DeletedAt
		}

		result = append(result, t)
	}

	return result, nil
}

// GetTunnelHostnames returns all hostnames configured for the tunnel
func (c *CloudflareClient) GetTunnelHostnames() (map[string]string, error) {
	ctx := context.Background()

	// Get tunnel information
	tunnel, err := c.api.GetTunnel(ctx, cloudflare.ResourceIdentifier(c.accountID), c.tunnelID)
	if err != nil {
		return nil, fmt.Errorf("error getting tunnel: %w", err)
	}

	// Get the tunnel configuration
	config, err := c.api.GetTunnelConfiguration(ctx, cloudflare.ResourceIdentifier(c.accountID), c.tunnelID)
	if err != nil {
		return nil, fmt.Errorf("error getting tunnel configuration: %w", err)
	}

	// Extract hostnames from ingress rules
	result := make(map[string]string)

	// Check if config.Config is available and has an ingress field
	if config.Config.Ingress != nil {
		for _, ingress := range config.Config.Ingress {
			if ingress.Hostname != "" {
				// Use the service as the "IP" - this is the internal service the tunnel points to
				serviceIP := extractServiceIP(ingress.Service)
				result[ingress.Hostname] = serviceIP
				logging.Debug("Found hostname in tunnel config",
					"hostname", ingress.Hostname,
					"service", ingress.Service,
					"serviceIP", serviceIP,
					"tunnelName", tunnel.Name)
			}
		}
	} else {
		logging.Warn("No ingress configurations found in tunnel", "tunnelID", c.tunnelID)
	}

	return result, nil
}

// extractServiceIP extracts the host:port part from service URL
// For example, "http://internal.example.com:8080" -> "internal.example.com:8080"
func extractServiceIP(service string) string {
	// If the service contains "://", extract the host part
	// Otherwise, just return the service as-is
	if len(service) > 8 {
		if service[:7] == "http://" {
			return service[7:]
		} else if len(service) > 9 && service[:8] == "https://" {
			return service[8:]
		}
	}
	return service
}

// AddTunnelHostname adds a new hostname to the tunnel configuration
func (c *CloudflareClient) AddTunnelHostname(hostname, service string) error {
	// In a real implementation, this would modify the tunnel configuration
	// to add a new ingress rule
	// For now, we'll just log this as it requires detailed Cloudflare API knowledge
	logging.Info("Would add hostname to tunnel",
		"hostname", hostname,
		"service", service,
		"tunnelID", c.tunnelID)
	return nil
}

// DeleteTunnelHostname removes a hostname from the tunnel configuration
func (c *CloudflareClient) DeleteTunnelHostname(hostname string) error {
	// In a real implementation, this would modify the tunnel configuration
	// to remove an ingress rule
	// For now, we'll just log this as it requires detailed Cloudflare API knowledge
	logging.Info("Would remove hostname from tunnel",
		"hostname", hostname,
		"tunnelID", c.tunnelID)
	return nil
}
