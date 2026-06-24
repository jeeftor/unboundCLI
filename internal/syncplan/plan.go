package syncplan

import (
	"fmt"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

// Action represents a sync operation to be performed.
type Action struct {
	Type                   string `json:"type"` // "add", "update", "delete"
	Hostname               string `json:"hostname"`
	Service                string `json:"service"` // "unbound", "adguard", "dhcp", "cloudflare"
	OldIP                  string `json:"old_ip"`
	NewIP                  string `json:"new_ip"`
	OldService             string `json:"old_service,omitempty"`
	NewService             string `json:"new_service,omitempty"`
	OldHTTPHostHeader      string `json:"old_http_host_header,omitempty"`
	NewHTTPHostHeader      string `json:"new_http_host_header,omitempty"`
	TunnelID               string `json:"tunnel_id,omitempty"`
	TunnelName             string `json:"tunnel_name,omitempty"`
	Path                   string `json:"path,omitempty"`
	NoTLSVerify            bool   `json:"no_tls_verify,omitempty"`
	Http2Origin            bool   `json:"http2_origin,omitempty"`
	OriginServerName       string `json:"origin_server_name,omitempty"`
	DisableChunkedEncoding bool   `json:"disable_chunked_encoding,omitempty"`
	HasAccessPolicy        bool   `json:"has_access_policy,omitempty"`
	ManagedFields          string `json:"managed_fields,omitempty"`
	OriginRequestSummary   string `json:"origin_request_summary,omitempty"`
	Details                string `json:"details"`
	Enabled                bool   `json:"enabled"`
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
	Service           string
	CaddyServerIP     string
	CaddyServiceURL   string
	IncludeCloudflare bool
	// Unsync, when true, generates delete actions for entries that are currently
	// configured in the target service regardless of whether they appear in Caddy.
	// This lets the user forcibly remove a hostname from one DNS service while
	// leaving it in Caddy (partial un-sync).
	Unsync bool

	// Cloudflare-specific overrides (used by the CF wizard in the web UI)
	// OriginMode controls the origin service URL used for new/updated CF rules:
	//   "" or "via-caddy"  → use CaddyServiceURL (route through Caddy; default)
	//   "direct"           → use "http://" + entry.CaddyUpstream (bypass Caddy)
	OriginMode string
	// NoTLSVerify adds NoTLSVerify=true to the CF ingress OriginRequest.
	// For via-caddy mode this should typically be true (Caddy uses a local cert).
	NoTLSVerify bool
	// DisableChunkedEncoding disables chunked transfer encoding.
	// Useful for WSGI backends (Gunicorn, uWSGI) in direct-to-service mode.
	DisableChunkedEncoding bool
	// OverrideTunnelID writes the rule to a specific tunnel instead of the
	// configured default. Empty means use the configured default.
	OverrideTunnelID string
}

// BuildPlan creates a sync plan from entries for one service or all services.
func BuildPlan(entries []*models.Entry, options Options) Plan {
	services := servicesFor(options.Service, options.IncludeCloudflare)
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
				if options.Unsync {
					needsRemoval = status.Configured
				} else {
					needsSync = entry.NeedsSyncToUnbound()
					needsRemoval = entry.NeedsRemovalFromUnbound()
				}
			case "adguard":
				status = entry.AdguardStatus
				if options.Unsync {
					needsRemoval = status.Configured
				} else {
					needsSync = entry.NeedsSyncToAdguard()
					needsRemoval = entry.NeedsRemovalFromAdguard()
				}
			case "dhcp":
				needsSync = entry.NeedsDHCPStaticEntry()
				dhcpAction = true
			case "cloudflare":
				action := buildCloudflareAction(entry, options)
				if action.Type != "" {
					actions = append(actions, action)
				}
				continue
			default:
				continue
			}

			if !needsSync && !needsRemoval {
				continue
			}

			action := buildAction(entry, svc, status, needsRemoval, dhcpAction, options.CaddyServerIP, options.Unsync)
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

func servicesFor(service string, includeCloudflare bool) []string {
	if service == "" || service == "all" {
		services := []string{"unbound", "adguard"}
		if includeCloudflare {
			services = append(services, "cloudflare")
		}
		return services
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
	unsync bool,
) Action {
	action := Action{
		Hostname: entry.Hostname,
		Service:  service,
		Enabled:  true,
	}

	switch {
	case needsRemoval && unsync:
		action.Type = "delete"
		action.OldIP = status.IP
		action.Details = "forced unsync (manual removal)"
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

func buildCloudflareAction(entry *models.Entry, options Options) Action {
	// Determine the desired origin service URL based on OriginMode.
	originMode := options.OriginMode
	if entry.CloudflareOriginMode != "" {
		originMode = entry.CloudflareOriginMode
	}
	viaCaddy := originMode != "direct"
	var desiredService string
	if !viaCaddy && entry.CaddyUpstream != "" {
		// Direct-to-service: connect straight to the backend (bypass Caddy).
		desiredService = withDefaultScheme(entry.CaddyUpstream)
	} else {
		// Via Caddy: connect to Caddy. Caddy speaks HTTPS, so ensure https://.
		desiredService = options.CaddyServiceURL
		if desiredService == "" && options.CaddyServerIP != "" {
			desiredService = fmt.Sprintf("https://%s", options.CaddyServerIP)
		}
		// Upgrade http:// → https:// when the user hasn't specified a scheme,
		// and strip port :80 (default HTTP port is meaningless for HTTPS).
		if strings.HasPrefix(desiredService, "http://") {
			desiredService = "https://" + strings.TrimPrefix(desiredService, "http://")
			desiredService = strings.TrimSuffix(desiredService, ":80")
		}
	}

	base := Action{
		Hostname:               entry.Hostname,
		Service:                "cloudflare",
		Enabled:                true,
		ManagedFields:          "service,http_host_header,origin_server_name",
		OriginRequestSummary:   "preserve optional origin request fields",
		NoTLSVerify:            options.NoTLSVerify,
		DisableChunkedEncoding: options.DisableChunkedEncoding,
	}
	desiredHostHeader := entry.Hostname
	if entry.CloudflareHTTPHostHeader != "" {
		desiredHostHeader = entry.CloudflareHTTPHostHeader
	}

	// Apply tunnel override: if OverrideTunnelID is set, this action targets that
	// specific tunnel (the apply layer will route accordingly).
	if options.OverrideTunnelID != "" {
		base.TunnelID = options.OverrideTunnelID
	}

	cf := entry.CloudflareStatus
	if entry.IsConfiguredInCaddy() {
		// For via-caddy mode, set OriginServerName so cloudflared presents the
		// correct SNI during its TLS handshake with Caddy.
		if viaCaddy {
			base.OriginServerName = entry.Hostname
		}
		// If in a non-default tunnel and no override is set, skip (read-only tunnel).
		if cf.Configured && !cf.IsDefaultTunnel && options.OverrideTunnelID == "" {
			return Action{}
		}
		if !cf.Configured {
			base.Type = "add"
			base.NewService = desiredService
			base.NewHTTPHostHeader = desiredHostHeader
			base.Details = "missing in default Cloudflare tunnel"
			return base
		}
		serviceWrong := desiredService != "" && cf.Service != desiredService
		headerWrong := cf.HTTPHostHeader != desiredHostHeader
		tlsWrong := options.NoTLSVerify != cf.NoTLSVerify
		if serviceWrong || headerWrong || tlsWrong {
			base.Type = "update"
			base.OldService = cf.Service
			base.NewService = desiredService
			base.OldHTTPHostHeader = cf.HTTPHostHeader
			base.NewHTTPHostHeader = desiredHostHeader
			if base.TunnelID == "" {
				base.TunnelID = cf.TunnelID
			}
			base.TunnelName = cf.TunnelName
			base.Path = cf.Path
			base.Http2Origin = cf.Http2Origin
			base.HasAccessPolicy = cf.HasAccessPolicy
			switch {
			case serviceWrong && headerWrong:
				base.Details = "service and host header differ from Caddy"
			case serviceWrong:
				base.Details = "service differs from Caddy"
			case tlsWrong:
				base.Details = "TLS verify setting changed"
			default:
				base.Details = "host header differs from Caddy"
			}
			return base
		}
		return Action{}
	}

	if cf.Configured && cf.IsDefaultTunnel {
		base.Type = "delete"
		base.OldService = cf.Service
		base.OldHTTPHostHeader = cf.HTTPHostHeader
		if base.TunnelID == "" {
			base.TunnelID = cf.TunnelID
		}
		base.TunnelName = cf.TunnelName
		base.Path = cf.Path
		base.Http2Origin = cf.Http2Origin
		base.HasAccessPolicy = cf.HasAccessPolicy
		base.Details = "no longer in Caddy"
		return base
	}

	return Action{}
}

func withDefaultScheme(service string) string {
	if strings.Contains(service, "://") {
		return service
	}
	return "http://" + service
}
