package syncplan

import (
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

// Action represents a sync operation to be performed.
type Action struct {
	Type     string `json:"type"` // "add", "update", "delete"
	Hostname string `json:"hostname"`
	Service  string `json:"service"` // "unbound", "adguard", "dhcp"
	OldIP    string `json:"old_ip"`
	NewIP    string `json:"new_ip"`
	Details  string `json:"details"`
	Enabled  bool   `json:"enabled"`
}

// Plan contains the actions selected for one sync operation.
type Plan struct {
	Actions []Action `json:"actions"`
}

// Result represents the result of a sync operation.
type Result struct {
	Success       bool           `json:"success"`
	ItemsAdded    int            `json:"items_added"`
	ItemsUpdated  int            `json:"items_updated"`
	ItemsDeleted  int            `json:"items_deleted"`
	Errors        []string       `json:"errors"`
	Message       string         `json:"message"`
	ActionResults []ActionResult `json:"action_results"`
}

// ActionResult records the outcome of one planned action.
type ActionResult struct {
	Action  Action `json:"action"`
	Success bool   `json:"success"`
	Skipped bool   `json:"skipped"`
	Error   string `json:"error"`
}

// Options controls sync action planning.
type Options struct {
	Service       string
	CaddyServerIP string
}

// BuildPlan creates a sync plan from entries for one service or all services.
func BuildPlan(entries []*models.Entry, options Options) Plan {
	services := servicesFor(options.Service)
	uniqueEntries := uniqueEntriesByHostname(entries)
	actions := make([]Action, 0)

	for _, entry := range uniqueEntries {
		for _, svc := range services {
			var status models.ServiceStatus
			var needsSync bool
			var needsRemoval bool
			var dhcpAction bool

			switch svc {
			case "unbound":
				status = entry.UnboundStatus
				needsSync = entry.NeedsSyncToUnbound()
				needsRemoval = entry.NeedsRemovalFromUnbound()
			case "adguard":
				status = entry.AdguardStatus
				needsSync = entry.NeedsSyncToAdguard()
				needsRemoval = entry.NeedsRemovalFromAdguard()
			case "dhcp":
				needsSync = entry.NeedsDHCPStaticEntry()
				dhcpAction = true
			default:
				continue
			}

			if !needsSync && !needsRemoval {
				continue
			}

			action := buildAction(entry, svc, status, needsRemoval, dhcpAction, options.CaddyServerIP)
			if action.Type != "" {
				actions = append(actions, action)
			}
		}
	}

	return Plan{Actions: actions}
}

// PlanFromEntries creates sync actions from entries for one service or all services.
func PlanFromEntries(entries []*models.Entry, options Options) []Action {
	return BuildPlan(entries, options).Actions
}

func servicesFor(service string) []string {
	if service == "" || service == "all" {
		return []string{"unbound", "adguard"}
	}
	return []string{service}
}

func uniqueEntriesByHostname(entries []*models.Entry) []*models.Entry {
	seen := make(map[string]bool)
	unique := make([]*models.Entry, 0, len(entries))
	for _, entry := range entries {
		if entry == nil || seen[entry.Hostname] {
			continue
		}
		seen[entry.Hostname] = true
		unique = append(unique, entry)
	}
	return unique
}

func buildAction(
	entry *models.Entry,
	service string,
	status models.ServiceStatus,
	needsRemoval bool,
	dhcpAction bool,
	caddyServerIP string,
) Action {
	action := Action{
		Hostname: entry.Hostname,
		Service:  service,
		Enabled:  true,
	}

	switch {
	case needsRemoval:
		action.Type = "delete"
		action.OldIP = status.IP
		action.Details = "no longer in Caddy"
	case dhcpAction:
		action.Type = "add"
		action.NewIP = entry.DHCPStatus.IP
		action.Details = fmt.Sprintf("static lease (MAC: %s)", entry.DHCPStatus.MAC)
	case !status.Configured:
		action.Type = "add"
		action.NewIP = caddyServerIP
	case !status.InSync:
		action.Type = "update"
		action.OldIP = status.IP
		action.NewIP = caddyServerIP
	}

	return action
}
