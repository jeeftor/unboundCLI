package app

import (
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
)

const (
	// DefaultCaddyServerIP is the default Caddy host used by existing commands.
	DefaultCaddyServerIP = "10.0.0.15"
	// DefaultCaddyServerPort is the default Caddy admin API port.
	DefaultCaddyServerPort = 2019
)

// CaddyEndpoint identifies the Caddy admin API endpoint.
type CaddyEndpoint struct {
	ServerIP   string // LAN IP for DNS comparison
	ServerPort int
	AdminHost  string // admin API host override (empty → use ServerIP)
}

// ClientSet contains the service clients shared by CLI, TUI, and future web adapters.
type ClientSet struct {
	Caddy      *api.CaddyClient
	Unbound    *api.Client
	DNSMasq    *api.DNSMasqClient
	Adguard    *api.AdguardClient
	Cloudflare *api.CloudflareClient
	Authentik  *api.AuthentikClient
}

// Runtime contains loaded configuration, resolved defaults, and constructed clients.
type Runtime struct {
	UnboundConfig    api.Config
	AdguardConfig    config.AdguardConfig
	CloudflareConfig config.CloudflareConfig
	AuthentikConfig  config.AuthentikConfig
	CaddyEndpoint    CaddyEndpoint
	CaddyServiceURL  string
	Clients          ClientSet
}

// RuntimeOptions controls which optional clients are constructed.
type RuntimeOptions struct {
	// ConfigPath is the selected configuration file. Empty uses the root
	// command's --config selection, then the default path.
	ConfigPath      string
	CaddyServerIP   string
	CaddyServerPort int
	CaddyAdminHost  string // optional override for admin API host (see CaddyEndpoint.AdminHost)

	IncludeUnbound    bool
	IncludeDNSMasq    bool
	IncludeAdguard    bool
	RequireAdguard    bool
	IncludeCloudflare bool
	RequireCloudflare bool
	IncludeAuthentik  bool
}

// LoadRuntime loads repository configuration and builds the requested clients.
func LoadRuntime(options RuntimeOptions) (*Runtime, error) {
	effective, _, err := config.LoadEffectiveConfig(options.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load effective configuration: %w", err)
	}
	var unboundConfig api.Config
	var adguardConfig config.AdguardConfig
	var cloudflareConfig config.CloudflareConfig
	var authentikConfig config.AuthentikConfig
	if options.IncludeUnbound || options.IncludeDNSMasq {
		unboundConfig = effective.Config
	}
	if options.IncludeAdguard {
		adguardConfig = effective.Adguard
	}
	if options.IncludeCloudflare {
		cloudflareConfig = effective.Cloudflare
	}
	if options.IncludeAuthentik {
		authentikConfig = effective.Authentik
	}
	if options.RequireAdguard && !isAdguardComplete(adguardConfig) {
		return nil, fmt.Errorf("AdguardHome configuration missing required fields (BaseURL, Username, Password)")
	}
	if options.RequireCloudflare && (!cloudflareConfig.Enabled || cloudflareConfig.APIToken == "" || cloudflareConfig.AccountID == "") {
		return nil, fmt.Errorf("Cloudflare configuration missing required enabled flag, API token, or account ID")
	}
	if options.CaddyServerIP == "" {
		options.CaddyServerIP = effective.Caddy.ServerIP
	}
	if options.CaddyServerPort == 0 {
		options.CaddyServerPort = effective.Caddy.ServerPort
	}
	if options.CaddyAdminHost == "" {
		options.CaddyAdminHost = effective.Caddy.AdminHost
	}

	return NewRuntimeFromConfigs(unboundConfig, adguardConfig, cloudflareConfig, authentikConfig, options)
}

// NewRuntimeFromConfigs builds runtime clients from already-loaded configuration.
func NewRuntimeFromConfigs(
	unboundConfig api.Config,
	adguardConfig config.AdguardConfig,
	cloudflareConfig config.CloudflareConfig,
	authentikConfig config.AuthentikConfig,
	options RuntimeOptions,
) (*Runtime, error) {
	endpoint := ResolveCaddyEndpoint(options.CaddyServerIP, options.CaddyServerPort, options.CaddyAdminHost)

	runtime := &Runtime{
		UnboundConfig:    unboundConfig,
		AdguardConfig:    adguardConfig,
		CloudflareConfig: cloudflareConfig,
		AuthentikConfig:  authentikConfig,
		CaddyEndpoint:    endpoint,
		CaddyServiceURL:  ResolveCaddyServiceURL(cloudflareConfig, endpoint),
		Clients: ClientSet{
			Caddy: &api.CaddyClient{ServerIP: endpoint.ServerIP, ServerPort: endpoint.ServerPort, AdminHost: endpoint.AdminHost},
		},
	}

	if options.IncludeUnbound {
		if isUnboundComplete(unboundConfig) {
			runtime.Clients.Unbound = api.NewClient(unboundConfig)
		}
	}

	if options.IncludeDNSMasq {
		if isUnboundComplete(unboundConfig) {
			runtime.Clients.DNSMasq = api.NewDNSMasqClient(unboundConfig)
		}
	}

	if options.IncludeAdguard {
		if isAdguardComplete(adguardConfig) {
			runtime.Clients.Adguard = api.NewAdguardClient(adguardConfig.GetAdguardAPIConfig())
		} else if options.RequireAdguard {
			return nil, fmt.Errorf("AdguardHome configuration missing required fields (BaseURL, Username, Password)")
		}
	}

	if options.IncludeCloudflare && cloudflareConfig.Enabled && cloudflareConfig.APIToken != "" && cloudflareConfig.AccountID != "" {
		cfClient, err := api.NewCloudflareClient(cloudflareConfig.GetCloudflareAPIConfig())
		if err != nil {
			if options.RequireCloudflare {
				return nil, fmt.Errorf("error creating Cloudflare client: %w", err)
			}
		} else {
			runtime.Clients.Cloudflare = cfClient
		}
	} else if options.IncludeCloudflare && options.RequireCloudflare {
		return nil, fmt.Errorf("Cloudflare configuration missing required enabled flag, API token, or account ID")
	}

	if options.IncludeAuthentik && authentikConfig.Enabled && authentikConfig.APIToken != "" && authentikConfig.BaseURL != "" {
		akClient, err := api.NewAuthentikClient(authentikConfig.GetAuthentikAPIConfig())
		if err != nil {
			logging.Warn("Failed to create Authentik client", "error", err)
		} else {
			runtime.Clients.Authentik = akClient
		}
	}

	return runtime, nil
}

// ResolveCaddyEndpoint applies existing command defaults to an optional endpoint override.
func ResolveCaddyEndpoint(serverIP string, serverPort int, adminHost string) CaddyEndpoint {
	if serverIP == "" {
		serverIP = DefaultCaddyServerIP
	}
	if serverPort == 0 {
		serverPort = DefaultCaddyServerPort
	}
	return CaddyEndpoint{ServerIP: serverIP, ServerPort: serverPort, AdminHost: adminHost}
}

// ResolveCaddyServiceURL returns the service URL used for Cloudflare quick-fill actions.
func ResolveCaddyServiceURL(cloudflareConfig config.CloudflareConfig, endpoint CaddyEndpoint) string {
	if cloudflareConfig.CaddyServiceURL != "" {
		return cloudflareConfig.CaddyServiceURL
	}
	return fmt.Sprintf("http://%s:80", endpoint.ServerIP)
}

func isAdguardComplete(adguardConfig config.AdguardConfig) bool {
	return adguardConfig.Enabled &&
		adguardConfig.BaseURL != "" &&
		adguardConfig.Username != "" &&
		adguardConfig.Password != ""
}

func isUnboundComplete(unboundConfig api.Config) bool {
	return unboundConfig.APIKey != "" && unboundConfig.APISecret != "" && unboundConfig.BaseURL != ""
}
