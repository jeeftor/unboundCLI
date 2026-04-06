// cloudflare.go in internal/api package
package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
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

// CloudflareZone represents a Cloudflare DNS zone
type CloudflareZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

// NewCloudflareClientWithBaseURL creates a new Cloudflare API client with a custom base URL (useful for testing)
func NewCloudflareClientWithBaseURL(config CloudflareConfig, baseURL string) (*CloudflareClient, error) {
	api, err := cloudflare.NewWithAPIToken(config.APIToken, cloudflare.BaseURL(baseURL))
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

// ListZones returns all zones accessible with the current API token
func (c *CloudflareClient) ListZones() ([]CloudflareZone, error) {
	ctx := context.Background()

	zones, err := c.api.ListZones(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing zones: %w", err)
	}

	result := make([]CloudflareZone, 0, len(zones))
	for _, z := range zones {
		result = append(result, CloudflareZone{
			ID:   z.ID,
			Name: z.Name,
		})
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

// SetTunnelIngress replaces the entire ingress rule list atomically.
// rules maps hostname → internal service URL (e.g. "http://192.168.1.15:80").
// The catch-all rule (http_status:404) is always appended as the last entry.
func (c *CloudflareClient) SetTunnelIngress(rules map[string]string) error {
	ctx := context.Background()

	ingress := make([]cloudflare.UnvalidatedIngressRule, 0, len(rules)+1)
	for hostname, service := range rules {
		ingress = append(ingress, cloudflare.UnvalidatedIngressRule{
			Hostname: hostname,
			Service:  service,
		})
	}
	ingress = append(ingress, cloudflare.UnvalidatedIngressRule{Service: "http_status:404"})

	_, err := c.api.UpdateTunnelConfiguration(ctx,
		cloudflare.ResourceIdentifier(c.accountID),
		cloudflare.TunnelConfigurationParams{
			TunnelID: c.tunnelID,
			Config:   cloudflare.TunnelConfiguration{Ingress: ingress},
		},
	)
	if err != nil {
		return fmt.Errorf("error updating tunnel configuration: %w", err)
	}

	logging.Info("Updated tunnel ingress", "tunnelID", c.tunnelID, "rules", len(rules))
	return nil
}

// ListManagedDNSRecords returns all CNAME records in the zone that point to
// cfargotunnel.com, keyed by hostname. Used to diff current vs desired state.
func (c *CloudflareClient) ListManagedDNSRecords() (map[string]string, error) {
	ctx := context.Background()

	records, _, err := c.api.ListDNSRecords(ctx,
		cloudflare.ResourceIdentifier(c.zoneID),
		cloudflare.ListDNSRecordsParams{Type: "CNAME"},
	)
	if err != nil {
		return nil, fmt.Errorf("error listing DNS records: %w", err)
	}

	result := make(map[string]string)
	for _, r := range records {
		if strings.Contains(r.Content, "cfargotunnel.com") {
			result[r.Name] = r.Content
		}
	}
	return result, nil
}

// EnsureDNSRecord creates a proxied CNAME record pointing hostname to
// <tunnelID>.cfargotunnel.com. If a cfargotunnel.com record already exists for
// the hostname it is updated in-place; if it is already correct it is left alone.
func (c *CloudflareClient) EnsureDNSRecord(hostname string) error {
	ctx := context.Background()
	target := c.tunnelID + ".cfargotunnel.com"
	proxied := true

	existing, _, err := c.api.ListDNSRecords(ctx,
		cloudflare.ResourceIdentifier(c.zoneID),
		cloudflare.ListDNSRecordsParams{Type: "CNAME", Name: hostname},
	)
	if err != nil {
		return fmt.Errorf("error looking up DNS record for %s: %w", hostname, err)
	}

	for _, r := range existing {
		if strings.Contains(r.Content, "cfargotunnel.com") {
			if r.Content == target {
				logging.Debug("DNS record already correct", "hostname", hostname)
				return nil
			}
			_, err := c.api.UpdateDNSRecord(ctx,
				cloudflare.ResourceIdentifier(c.zoneID),
				cloudflare.UpdateDNSRecordParams{
					ID:      r.ID,
					Type:    "CNAME",
					Name:    hostname,
					Content: target,
					Proxied: &proxied,
					TTL:     1,
				},
			)
			if err != nil {
				return fmt.Errorf("error updating DNS record for %s: %w", hostname, err)
			}
			logging.Info("Updated DNS record", "hostname", hostname, "target", target)
			return nil
		}
	}

	_, err = c.api.CreateDNSRecord(ctx,
		cloudflare.ResourceIdentifier(c.zoneID),
		cloudflare.CreateDNSRecordParams{
			Type:    "CNAME",
			Name:    hostname,
			Content: target,
			Proxied: &proxied,
			TTL:     1,
		},
	)
	if err != nil {
		return fmt.Errorf("error creating DNS record for %s: %w", hostname, err)
	}
	logging.Info("Created DNS record", "hostname", hostname, "target", target)
	return nil
}

// DeleteDNSRecord removes the cfargotunnel.com CNAME for hostname, if present.
// It is a no-op if the record does not exist.
func (c *CloudflareClient) DeleteDNSRecord(hostname string) error {
	ctx := context.Background()

	records, _, err := c.api.ListDNSRecords(ctx,
		cloudflare.ResourceIdentifier(c.zoneID),
		cloudflare.ListDNSRecordsParams{Type: "CNAME", Name: hostname},
	)
	if err != nil {
		return fmt.Errorf("error looking up DNS record for %s: %w", hostname, err)
	}

	for _, r := range records {
		if strings.Contains(r.Content, "cfargotunnel.com") {
			if err := c.api.DeleteDNSRecord(ctx, cloudflare.ResourceIdentifier(c.zoneID), r.ID); err != nil {
				return fmt.Errorf("error deleting DNS record for %s: %w", hostname, err)
			}
			logging.Info("Deleted DNS record", "hostname", hostname, "recordID", r.ID)
			return nil
		}
	}

	logging.Warn("DNS record not found, nothing to delete", "hostname", hostname)
	return nil
}
