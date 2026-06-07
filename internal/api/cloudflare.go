// cloudflare.go in internal/api package
package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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
	Insecure  bool // skip TLS certificate verification for Cloudflare API calls
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
	opts := []cloudflare.Option{}
	if config.Insecure {
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		}
		opts = append(opts, cloudflare.HTTPClient(httpClient))
	}

	cfAPI, err := cloudflare.NewWithAPIToken(config.APIToken, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating Cloudflare API client: %w", err)
	}

	return &CloudflareClient{
		api:       cfAPI,
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

// CloudflareAccount represents a Cloudflare account
type CloudflareAccount struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListAccounts returns all accounts accessible with the current API token
func (c *CloudflareClient) ListAccounts() ([]CloudflareAccount, error) {
	ctx := context.Background()
	accounts, _, err := c.api.Accounts(ctx, cloudflare.AccountsListParams{})
	if err != nil {
		return nil, fmt.Errorf("error listing accounts: %w", err)
	}
	result := make([]CloudflareAccount, 0, len(accounts))
	for _, a := range accounts {
		result = append(result, CloudflareAccount{ID: a.ID, Name: a.Name})
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

// TunnelHostEntry identifies a hostname found in a specific tunnel.
type TunnelHostEntry struct {
	TunnelID   string
	TunnelName string
	Service    string
}

// CloudflareIngressEntry represents a fully-populated ingress rule from any tunnel.
// It merges the tunnel-level default OriginRequest with the per-rule OriginRequest
// (per-rule wins). Used for display and status checks.
type CloudflareIngressEntry struct {
	TunnelID         string
	TunnelName       string
	Hostname         string
	Path             string
	Service          string // LAN endpoint, e.g. "http://192.168.1.15:8096"
	HTTPHostHeader   string // empty = not configured (common Caddy routing issue)
	NoTLSVerify      bool
	Http2Origin      bool
	HasAccessPolicy  bool // true if OriginRequest.Access.Required
	IsDefaultTunnel  bool // true if TunnelID == c.tunnelID
	RawOriginRequest *cloudflare.OriginRequestConfig
}

// GetAllTunnelsHostnames scans every active tunnel in the account and returns
// a consolidated hostname map. First tunnel wins on duplicates (logged as warning).
// Does NOT use c.tunnelID — scans the whole account.
func (c *CloudflareClient) GetAllTunnelsHostnames() (map[string]TunnelHostEntry, error) {
	ctx := context.Background()

	if c.accountID == "" {
		return nil, fmt.Errorf("account ID is required to scan all tunnels")
	}

	tunnels, err := c.ListTunnels()
	if err != nil {
		return nil, fmt.Errorf("error listing tunnels: %w", err)
	}

	result := make(map[string]TunnelHostEntry)

	for _, tunnel := range tunnels {
		// Only scan active (non-deleted) tunnels
		if !tunnel.DeletedAt.IsZero() {
			logging.Debug("Skipping deleted tunnel", "tunnelID", tunnel.ID, "tunnelName", tunnel.Name)
			continue
		}

		config, err := c.api.GetTunnelConfiguration(ctx, cloudflare.ResourceIdentifier(c.accountID), tunnel.ID)
		if err != nil {
			logging.Warn("Failed to get configuration for tunnel", "tunnelID", tunnel.ID, "tunnelName", tunnel.Name, "error", err)
			continue
		}

		if config.Config.Ingress == nil {
			continue
		}

		for _, ingress := range config.Config.Ingress {
			if ingress.Hostname == "" {
				continue
			}

			if existing, exists := result[ingress.Hostname]; exists {
				logging.Warn("Hostname found in multiple tunnels, keeping first",
					"hostname", ingress.Hostname,
					"firstTunnel", existing.TunnelName,
					"duplicateTunnel", tunnel.Name)
				continue
			}

			result[ingress.Hostname] = TunnelHostEntry{
				TunnelID:   tunnel.ID,
				TunnelName: tunnel.Name,
				Service:    extractServiceIP(ingress.Service),
			}
			logging.Debug("Found hostname in account scan",
				"hostname", ingress.Hostname,
				"tunnelName", tunnel.Name,
				"service", ingress.Service)
		}
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

// SetTunnelIngress updates the desired hostname rules while preserving existing rule metadata.
// rules maps hostname → internal service URL (e.g. "http://192.168.1.15:80").
// The catch-all rule is preserved when present, or http_status:404 is appended as the last entry.
func (c *CloudflareClient) SetTunnelIngress(rules map[string]string) error {
	ctx := context.Background()

	config, err := c.api.GetTunnelConfiguration(ctx,
		cloudflare.ResourceIdentifier(c.accountID),
		c.tunnelID,
	)
	if err != nil {
		return fmt.Errorf("error getting tunnel configuration before update: %w", err)
	}

	ingress := mergeTunnelIngress(config.Config.Ingress, rules)

	_, err = c.api.UpdateTunnelConfiguration(ctx,
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

func mergeTunnelIngress(existing []cloudflare.UnvalidatedIngressRule, rules map[string]string) []cloudflare.UnvalidatedIngressRule {
	remaining := make(map[string]string, len(rules))
	for hostname, service := range rules {
		remaining[hostname] = service
	}

	ingress := make([]cloudflare.UnvalidatedIngressRule, 0, len(rules)+1)
	var catchAll *cloudflare.UnvalidatedIngressRule

	for _, rule := range existing {
		if rule.Hostname == "" {
			if catchAll == nil {
				r := rule
				catchAll = &r
			}
			continue
		}

		service, keep := remaining[rule.Hostname]
		if !keep {
			continue
		}
		if !servicesEquivalent(rule.Service, service) {
			rule.Service = service
		}
		ingress = append(ingress, rule)
		delete(remaining, rule.Hostname)
	}

	hostnames := make([]string, 0, len(remaining))
	for hostname := range remaining {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)
	for _, hostname := range hostnames {
		ingress = append(ingress, cloudflare.UnvalidatedIngressRule{
			Hostname: hostname,
			Service:  remaining[hostname],
		})
	}

	if catchAll != nil {
		ingress = append(ingress, *catchAll)
	} else {
		ingress = append(ingress, cloudflare.UnvalidatedIngressRule{Service: "http_status:404"})
	}

	return ingress
}

func servicesEquivalent(existing, desired string) bool {
	return existing == desired || extractServiceIP(existing) == desired || existing == extractServiceIP(desired)
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

// IngressRuleSpec describes the desired state for a single tunnel ingress rule.
type IngressRuleSpec struct {
	Hostname       string
	Service        string
	HTTPHostHeader string // empty = not set in OriginRequest
	NoTLSVerify    bool
	Http2Origin    bool
}

// UpdateTunnelRule updates (or adds) a single ingress rule in the default tunnel,
// preserving all other existing rules unchanged. If the hostname does not currently
// have a rule it is added before the catch-all.
// A backup of the pre-edit state is written to ~/.caddy-dns-sync-backups/ automatically.
func (c *CloudflareClient) UpdateTunnelRule(spec IngressRuleSpec) error {
	ctx := context.Background()

	current, err := c.api.GetTunnelConfiguration(ctx, cloudflare.ResourceIdentifier(c.accountID), c.tunnelID)
	if err != nil {
		return fmt.Errorf("error getting tunnel config: %w", err)
	}

	// Auto-backup before mutating
	if backupPath, berr := c.saveTunnelBackup(current.Config.Ingress); berr != nil {
		logging.Warn("Could not write CF tunnel backup", "error", berr)
	} else {
		logging.Info("CF tunnel backup saved", "path", backupPath)
	}

	found := false
	newIngress := make([]cloudflare.UnvalidatedIngressRule, 0, len(current.Config.Ingress)+1)

	for _, rule := range current.Config.Ingress {
		if rule.Hostname == "" {
			continue // skip catch-all; re-added at end
		}
		if rule.Hostname == spec.Hostname {
			newIngress = append(newIngress, buildCFIngressRule(spec))
			found = true
		} else {
			newIngress = append(newIngress, rule)
		}
	}
	if !found {
		newIngress = append(newIngress, buildCFIngressRule(spec))
	}
	// Catch-all must always be last
	newIngress = append(newIngress, cloudflare.UnvalidatedIngressRule{Service: "http_status:404"})

	_, err = c.api.UpdateTunnelConfiguration(ctx,
		cloudflare.ResourceIdentifier(c.accountID),
		cloudflare.TunnelConfigurationParams{
			TunnelID: c.tunnelID,
			Config:   cloudflare.TunnelConfiguration{Ingress: newIngress},
		},
	)
	if err != nil {
		return fmt.Errorf("error updating tunnel configuration: %w", err)
	}

	logging.Info("Updated tunnel rule", "hostname", spec.Hostname, "service", spec.Service)
	return nil
}

// DeleteTunnelRule removes a single ingress rule from the default tunnel by hostname,
// preserving all other rules. A backup is written before the mutation.
// It is a no-op if the hostname is not found.
func (c *CloudflareClient) DeleteTunnelRule(hostname string) error {
	ctx := context.Background()

	current, err := c.api.GetTunnelConfiguration(ctx, cloudflare.ResourceIdentifier(c.accountID), c.tunnelID)
	if err != nil {
		return fmt.Errorf("error getting tunnel config: %w", err)
	}

	// Auto-backup before mutating
	if backupPath, berr := c.saveTunnelBackup(current.Config.Ingress); berr != nil {
		logging.Warn("Could not write CF tunnel backup", "error", berr)
	} else {
		logging.Info("CF tunnel backup saved", "path", backupPath)
	}

	newIngress := make([]cloudflare.UnvalidatedIngressRule, 0, len(current.Config.Ingress))
	found := false
	for _, rule := range current.Config.Ingress {
		if rule.Hostname == "" {
			continue // skip catch-all; re-added at end
		}
		if rule.Hostname == hostname {
			found = true
			continue // drop this rule
		}
		newIngress = append(newIngress, rule)
	}
	if !found {
		logging.Warn("DeleteTunnelRule: hostname not found, nothing to delete", "hostname", hostname)
		return nil
	}
	newIngress = append(newIngress, cloudflare.UnvalidatedIngressRule{Service: "http_status:404"})

	_, err = c.api.UpdateTunnelConfiguration(ctx,
		cloudflare.ResourceIdentifier(c.accountID),
		cloudflare.TunnelConfigurationParams{
			TunnelID: c.tunnelID,
			Config:   cloudflare.TunnelConfiguration{Ingress: newIngress},
		},
	)
	if err != nil {
		return fmt.Errorf("error updating tunnel configuration: %w", err)
	}

	logging.Info("Deleted tunnel rule", "hostname", hostname)
	return nil
}

// buildCFIngressRule constructs a cloudflare ingress rule from a spec.
func buildCFIngressRule(spec IngressRuleSpec) cloudflare.UnvalidatedIngressRule {
	rule := cloudflare.UnvalidatedIngressRule{
		Hostname: spec.Hostname,
		Service:  spec.Service,
	}
	if spec.HTTPHostHeader != "" || spec.NoTLSVerify || spec.Http2Origin {
		or := &cloudflare.OriginRequestConfig{}
		if spec.HTTPHostHeader != "" {
			hh := spec.HTTPHostHeader
			or.HTTPHostHeader = &hh
		}
		if spec.NoTLSVerify {
			v := true
			or.NoTLSVerify = &v
		}
		if spec.Http2Origin {
			v := true
			or.Http2Origin = &v
		}
		rule.OriginRequest = or
	}
	return rule
}

// defaultBackupDir returns the path to the backup directory, creating it if needed.
func defaultBackupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".caddy-dns-sync-backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// saveTunnelBackup serializes the given ingress rules to a timestamped JSON file.
// Returns the path of the written file.
func (c *CloudflareClient) saveTunnelBackup(rules []cloudflare.UnvalidatedIngressRule) (string, error) {
	dir, err := defaultBackupDir()
	if err != nil {
		return "", err
	}
	ts := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("cf-tunnel-%s-%s.json", c.tunnelID[:8], ts)
	path := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal backup: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write backup: %w", err)
	}
	return path, nil
}

// BackupTunnelConfig fetches the current tunnel ingress rules and saves them to
// ~/.caddy-dns-sync-backups/. Returns the backup file path.
func (c *CloudflareClient) BackupTunnelConfig() (string, error) {
	ctx := context.Background()
	current, err := c.api.GetTunnelConfiguration(ctx, cloudflare.ResourceIdentifier(c.accountID), c.tunnelID)
	if err != nil {
		return "", fmt.Errorf("error getting tunnel config: %w", err)
	}
	return c.saveTunnelBackup(current.Config.Ingress)
}

// RestoreTunnelConfig reads a backup JSON file produced by BackupTunnelConfig and
// pushes those ingress rules back to Cloudflare, overwriting the current config.
func (c *CloudflareClient) RestoreTunnelConfig(backupPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read backup file: %w", err)
	}

	var rules []cloudflare.UnvalidatedIngressRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("parse backup file: %w", err)
	}

	// Ensure the catch-all is present
	hasCatchAll := false
	for _, r := range rules {
		if r.Hostname == "" {
			hasCatchAll = true
			break
		}
	}
	if !hasCatchAll {
		rules = append(rules, cloudflare.UnvalidatedIngressRule{Service: "http_status:404"})
	}

	ctx := context.Background()
	_, err = c.api.UpdateTunnelConfiguration(ctx,
		cloudflare.ResourceIdentifier(c.accountID),
		cloudflare.TunnelConfigurationParams{
			TunnelID: c.tunnelID,
			Config:   cloudflare.TunnelConfiguration{Ingress: rules},
		},
	)
	if err != nil {
		return fmt.Errorf("error restoring tunnel configuration: %w", err)
	}

	logging.Info("Tunnel configuration restored", "file", backupPath, "rules", len(rules))
	return nil
}

// ListTunnelBackups returns backup files in ~/.caddy-dns-sync-backups/ for this tunnel,
// sorted newest-first.
func (c *CloudflareClient) ListTunnelBackups() ([]string, error) {
	dir, err := defaultBackupDir()
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("cf-tunnel-%s-", c.tunnelID[:8])
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for i := len(entries) - 1; i >= 0; i-- { // ReadDir is sorted ascending; reverse for newest-first
		e := entries[i]
		if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	return paths, nil
}

// GetAllTunnelsDetails scans every active tunnel in the account and returns a
// consolidated map of hostname → CloudflareIngressEntry with full OriginRequest data.
// Per-rule OriginRequest settings override the tunnel-level defaults.
// First tunnel wins on duplicate hostnames (with warning log).
// Does NOT use c.tunnelID for filtering — scans the whole account.
func (c *CloudflareClient) GetAllTunnelsDetails() (map[string]CloudflareIngressEntry, error) {
	ctx := context.Background()

	if c.accountID == "" {
		return nil, fmt.Errorf("account ID is required to scan all tunnels")
	}

	tunnels, err := c.ListTunnels()
	if err != nil {
		return nil, fmt.Errorf("error listing tunnels: %w", err)
	}

	result := make(map[string]CloudflareIngressEntry)

	for _, tunnel := range tunnels {
		if !tunnel.DeletedAt.IsZero() {
			logging.Debug("Skipping deleted tunnel", "tunnelID", tunnel.ID, "tunnelName", tunnel.Name)
			continue
		}

		config, err := c.api.GetTunnelConfiguration(ctx, cloudflare.ResourceIdentifier(c.accountID), tunnel.ID)
		if err != nil {
			logging.Warn("Failed to get configuration for tunnel", "tunnelID", tunnel.ID, "tunnelName", tunnel.Name, "error", err)
			continue
		}

		if config.Config.Ingress == nil {
			continue
		}

		// Tunnel-level default OriginRequest (applies to all rules unless overridden)
		tunnelDefault := config.Config.OriginRequest

		for _, ingress := range config.Config.Ingress {
			if ingress.Hostname == "" {
				continue // skip the catch-all rule
			}

			if existing, exists := result[ingress.Hostname]; exists {
				logging.Warn("Hostname found in multiple tunnels, keeping first",
					"hostname", ingress.Hostname,
					"firstTunnel", existing.TunnelName,
					"duplicateTunnel", tunnel.Name)
				continue
			}

			// Merge: start from tunnel default, override with per-rule values
			merged := tunnelDefault
			var rawOrigin *cloudflare.OriginRequestConfig
			if ingress.OriginRequest != nil {
				rawOrigin = ingress.OriginRequest
				if ingress.OriginRequest.HTTPHostHeader != nil {
					merged.HTTPHostHeader = ingress.OriginRequest.HTTPHostHeader
				}
				if ingress.OriginRequest.NoTLSVerify != nil {
					merged.NoTLSVerify = ingress.OriginRequest.NoTLSVerify
				}
				if ingress.OriginRequest.Http2Origin != nil {
					merged.Http2Origin = ingress.OriginRequest.Http2Origin
				}
				if ingress.OriginRequest.Access != nil {
					merged.Access = ingress.OriginRequest.Access
				}
			}

			entry := CloudflareIngressEntry{
				TunnelID:         tunnel.ID,
				TunnelName:       tunnel.Name,
				Hostname:         ingress.Hostname,
				Path:             ingress.Path,
				Service:          ingress.Service,
				IsDefaultTunnel:  tunnel.ID == c.tunnelID,
				RawOriginRequest: rawOrigin,
			}
			if merged.HTTPHostHeader != nil {
				entry.HTTPHostHeader = *merged.HTTPHostHeader
			}
			if merged.NoTLSVerify != nil {
				entry.NoTLSVerify = *merged.NoTLSVerify
			}
			if merged.Http2Origin != nil {
				entry.Http2Origin = *merged.Http2Origin
			}
			if merged.Access != nil {
				entry.HasAccessPolicy = merged.Access.Required
			}

			result[ingress.Hostname] = entry
			logging.Debug("Found hostname in tunnel details scan",
				"hostname", ingress.Hostname,
				"tunnelName", tunnel.Name,
				"service", ingress.Service,
				"httpHostHeader", entry.HTTPHostHeader)
		}
	}

	return result, nil
}
