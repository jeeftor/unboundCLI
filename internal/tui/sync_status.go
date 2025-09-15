package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/api"
)

// SyncStatus represents the sync status of a hostname across all systems
type SyncStatus struct {
	Hostname      string
	CaddyIP       string            // IP from Caddy config (source of truth)
	UnboundStatus ServiceStatus     // Status in UnboundDNS
	AdguardStatus ServiceStatus     // Status in AdguardHome
	Overall       OverallSyncStatus // Overall sync status
}

// ServiceStatus represents the status of a service in a particular DNS system
type ServiceStatus struct {
	Present     bool   // Is the entry present in this system?
	IP          string // What IP does it point to?
	Description string // Entry description
	UUID        string // UUID (for UnboundDNS)
	InSync      bool   // Does the IP match Caddy's IP?
}

// OverallSyncStatus represents the overall sync state
type OverallSyncStatus int

const (
	FullyInSync OverallSyncStatus = iota
	PartiallyInSync
	OutOfSync
	CaddyOnly
)

// SyncStatusDashboard manages the 3-way sync status display
type SyncStatusDashboard struct {
	statuses []SyncStatus
	filters  StatusFilters
}

// StatusFilters defines filtering options for the dashboard
type StatusFilters struct {
	ShowOnlyOutOfSync bool
	ShowOnlyUnbound   bool
	ShowOnlyAdguard   bool
	HostnameFilter    string
}

// NewSyncStatusDashboard creates a new sync status dashboard
func NewSyncStatusDashboard() *SyncStatusDashboard {
	return &SyncStatusDashboard{
		statuses: make([]SyncStatus, 0),
		filters:  StatusFilters{},
	}
}

// LoadSyncData fetches data from all three systems and builds the sync status
func (d *SyncStatusDashboard) LoadSyncData(
	caddyClient *api.CaddyClient,
	unboundClient *api.Client,
	adguardClient *api.AdguardClient,
) error {
	// Clear existing data
	d.statuses = make([]SyncStatus, 0)

	// Fetch data from all systems
	caddyHostnames, err := d.fetchCaddyData(caddyClient)
	if err != nil {
		return fmt.Errorf("failed to fetch Caddy data: %w", err)
	}

	unboundEntries, err := d.fetchUnboundData(unboundClient)
	if err != nil {
		return fmt.Errorf("failed to fetch UnboundDNS data: %w", err)
	}

	adguardEntries, err := d.fetchAdguardData(adguardClient)
	if err != nil {
		return fmt.Errorf("failed to fetch AdguardHome data: %w", err)
	}

	// Build sync status for each hostname
	d.buildSyncStatuses(caddyHostnames, unboundEntries, adguardEntries)

	return nil
}

// fetchCaddyData retrieves hostname mappings from Caddy
func (d *SyncStatusDashboard) fetchCaddyData(client *api.CaddyClient) (map[string]string, error) {
	if client == nil {
		return make(map[string]string), nil
	}
	return client.GetHostnameMap()
}

// fetchUnboundData retrieves DNS overrides from UnboundDNS
func (d *SyncStatusDashboard) fetchUnboundData(client *api.Client) (map[string]api.DNSOverride, error) {
	if client == nil {
		return make(map[string]api.DNSOverride), nil
	}

	overrides, err := client.GetOverrides()
	if err != nil {
		return nil, err
	}

	// Convert to map keyed by hostname
	result := make(map[string]api.DNSOverride)
	for _, override := range overrides {
		hostname := fmt.Sprintf("%s.%s", override.Host, override.Domain)
		result[hostname] = override
	}

	return result, nil
}

// fetchAdguardData retrieves DNS rewrites from AdguardHome
func (d *SyncStatusDashboard) fetchAdguardData(client *api.AdguardClient) (map[string]api.Rewrite, error) {
	if client == nil {
		return make(map[string]api.Rewrite), nil
	}

	rewrites, err := client.ListRewrites()
	if err != nil {
		return nil, err
	}

	// Convert to map keyed by hostname
	result := make(map[string]api.Rewrite)
	for _, rewrite := range rewrites {
		result[rewrite.Domain] = rewrite
	}

	return result, nil
}

// buildSyncStatuses creates SyncStatus entries by comparing all three systems
func (d *SyncStatusDashboard) buildSyncStatuses(
	caddyHostnames map[string]string,
	unboundEntries map[string]api.DNSOverride,
	adguardEntries map[string]api.Rewrite,
) {
	// Get all unique hostnames across all systems
	allHostnames := make(map[string]bool)

	for hostname := range caddyHostnames {
		allHostnames[hostname] = true
	}
	for hostname := range unboundEntries {
		allHostnames[hostname] = true
	}
	for hostname := range adguardEntries {
		allHostnames[hostname] = true
	}

	// Build status for each hostname
	for hostname := range allHostnames {
		status := SyncStatus{
			Hostname: hostname,
		}

		// Caddy data (source of truth)
		if caddyIP, exists := caddyHostnames[hostname]; exists {
			status.CaddyIP = caddyIP
		}

		// UnboundDNS data
		if unboundEntry, exists := unboundEntries[hostname]; exists {
			status.UnboundStatus = ServiceStatus{
				Present:     true,
				IP:          unboundEntry.Server,
				Description: unboundEntry.Description,
				UUID:        unboundEntry.UUID,
				InSync:      unboundEntry.Server == status.CaddyIP,
			}
		} else {
			status.UnboundStatus = ServiceStatus{
				Present: false,
				InSync:  status.CaddyIP == "", // In sync if neither system has it
			}
		}

		// AdguardHome data
		if adguardEntry, exists := adguardEntries[hostname]; exists {
			status.AdguardStatus = ServiceStatus{
				Present:     true,
				IP:          adguardEntry.Answer,
				Description: "", // AdguardHome doesn't have descriptions
				InSync:      adguardEntry.Answer == status.CaddyIP,
			}
		} else {
			status.AdguardStatus = ServiceStatus{
				Present: false,
				InSync:  status.CaddyIP == "", // In sync if neither system has it
			}
		}

		// Determine overall sync status
		status.Overall = d.determineOverallStatus(status)

		d.statuses = append(d.statuses, status)
	}

	// Sort by hostname for consistent display
	sort.Slice(d.statuses, func(i, j int) bool {
		return d.statuses[i].Hostname < d.statuses[j].Hostname
	})
}

// determineOverallStatus calculates the overall sync status
func (d *SyncStatusDashboard) determineOverallStatus(status SyncStatus) OverallSyncStatus {
	hasCaddy := status.CaddyIP != ""
	unboundInSync := status.UnboundStatus.InSync
	adguardInSync := status.AdguardStatus.InSync

	if !hasCaddy {
		// If not in Caddy, it should not be in other systems either
		if !status.UnboundStatus.Present && !status.AdguardStatus.Present {
			return FullyInSync
		}
		return OutOfSync
	}

	// If in Caddy, check other systems
	if unboundInSync && adguardInSync {
		return FullyInSync
	} else if unboundInSync || adguardInSync {
		return PartiallyInSync
	} else {
		if !status.UnboundStatus.Present && !status.AdguardStatus.Present {
			return CaddyOnly
		}
		return OutOfSync
	}
}

// GetFilteredStatuses returns statuses that match current filters
func (d *SyncStatusDashboard) GetFilteredStatuses() []SyncStatus {
	filtered := make([]SyncStatus, 0)

	for _, status := range d.statuses {
		// Apply filters
		if d.filters.ShowOnlyOutOfSync && status.Overall == FullyInSync {
			continue
		}

		if d.filters.HostnameFilter != "" {
			if !strings.Contains(strings.ToLower(status.Hostname),
				strings.ToLower(d.filters.HostnameFilter)) {
				continue
			}
		}

		filtered = append(filtered, status)
	}

	return filtered
}

// GetSummary returns a summary of the sync status
func (d *SyncStatusDashboard) GetSummary() SyncSummary {
	summary := SyncSummary{}

	for _, status := range d.statuses {
		summary.Total++

		switch status.Overall {
		case FullyInSync:
			summary.FullyInSync++
		case PartiallyInSync:
			summary.PartiallyInSync++
		case OutOfSync:
			summary.OutOfSync++
		case CaddyOnly:
			summary.CaddyOnly++
		}

		if status.CaddyIP != "" {
			summary.InCaddy++
		}
		if status.UnboundStatus.Present {
			summary.InUnbound++
		}
		if status.AdguardStatus.Present {
			summary.InAdguard++
		}
	}

	return summary
}

// SyncSummary provides statistics about the sync status
type SyncSummary struct {
	Total           int
	FullyInSync     int
	PartiallyInSync int
	OutOfSync       int
	CaddyOnly       int
	InCaddy         int
	InUnbound       int
	InAdguard       int
}

// SetFilters updates the current filters
func (d *SyncStatusDashboard) SetFilters(filters StatusFilters) {
	d.filters = filters
}

// GetStatusIcon returns an icon representing the sync status
func GetStatusIcon(status ServiceStatus, hasInCaddy bool) string {
	if !hasInCaddy {
		// If not in Caddy, should not be present elsewhere
		if !status.Present {
			return "✅" // Correctly absent
		}
		return "❌" // Should not be present
	}

	// If in Caddy, should be present and in sync
	if !status.Present {
		return "❌" // Missing
	}
	if status.InSync {
		return "✅" // In sync
	}
	return "⚠️" // Present but wrong IP
}

// GetOverallStatusIcon returns an icon for the overall status
func GetOverallStatusIcon(status OverallSyncStatus) string {
	switch status {
	case FullyInSync:
		return "✅"
	case PartiallyInSync:
		return "⚠️"
	case OutOfSync:
		return "❌"
	case CaddyOnly:
		return "📋"
	default:
		return "❓"
	}
}
