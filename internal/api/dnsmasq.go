package api

import (
	"encoding/json"
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/logging"
)

// DNSMasqLease represents a DHCP lease entry from DNSMasq
type DNSMasqLease struct {
	Hostname   string `json:"hostname"`
	IPAddress  string `json:"address"`
	MACAddress string `json:"hwaddr"`      // API returns "hwaddr"
	Expires    int64  `json:"expire"`      // API returns "expire" as Unix timestamp
	IsReserved string `json:"is_reserved"` // "1" = static, "0" = dynamic
	MACInfo    string `json:"mac_info"`    // Manufacturer info
	Type       string `json:"-"`           // Computed field (not from API)
}

// DNSMasqClient handles DNSMasq DHCP lease queries via OPNSense API
type DNSMasqClient struct {
	client *Client // Reuse OPNSense client (same API, different endpoint)
}

// NewDNSMasqClient creates a new DNSMasq client
func NewDNSMasqClient(config Config) *DNSMasqClient {
	return &DNSMasqClient{
		client: NewClient(config),
	}
}

// GetLeases retrieves all DHCP leases from DNSMasq
func (c *DNSMasqClient) GetLeases() ([]DNSMasqLease, error) {
	logging.Debug("Fetching DNSMasq DHCP leases")

	resp, err := c.client.makeRequest("GET", "/api/dnsmasq/leases/search", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DNSMasq leases: %w", err)
	}

	// Parse the rows field (similar to Unbound API)
	if len(resp.Rows) > 0 {
		var rows []DNSMasqLease
		if err := json.Unmarshal(resp.Rows, &rows); err != nil {
			logging.Error("Failed to parse DNSMasq lease rows",
				"error", err,
				"data", string(resp.Rows),
			)
			return nil, fmt.Errorf("error parsing lease rows: %w - Data: %s", err, string(resp.Rows))
		}

		// Populate Type field from IsReserved
		for i := range rows {
			if rows[i].IsReserved == "1" {
				rows[i].Type = "static"
			} else {
				rows[i].Type = "dynamic"
			}
		}

		logging.Debug("Successfully fetched DNSMasq leases", "count", len(rows))
		return rows, nil
	}

	logging.Debug("No DNSMasq leases found")
	return []DNSMasqLease{}, nil
}

// GetLeaseMap returns a map of hostname → IP address for quick lookup
func (c *DNSMasqClient) GetLeaseMap() (map[string]string, error) {
	leases, err := c.GetLeases()
	if err != nil {
		return nil, err
	}

	leaseMap := make(map[string]string)
	for _, lease := range leases {
		if lease.Hostname != "" && lease.IPAddress != "" {
			leaseMap[lease.Hostname] = lease.IPAddress
		}
	}

	logging.Debug("Built DNSMasq lease map", "count", len(leaseMap))
	return leaseMap, nil
}

// GetLeasesByIP returns a map of IP → lease for IP-based lookup
func (c *DNSMasqClient) GetLeasesByIP() (map[string]DNSMasqLease, error) {
	leases, err := c.GetLeases()
	if err != nil {
		return nil, err
	}

	leasesByIP := make(map[string]DNSMasqLease)
	for _, lease := range leases {
		if lease.IPAddress != "" {
			leasesByIP[lease.IPAddress] = lease
		}
	}

	logging.Debug("Built DNSMasq IP→lease map", "count", len(leasesByIP))
	return leasesByIP, nil
}
