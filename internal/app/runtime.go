package app

import (
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
)

const (
	// DefaultCaddyServerIP is the default Caddy host used by existing commands.
	DefaultCaddyServerIP = "192.168.1.15"
	// DefaultCaddyServerPort is the default Caddy admin API port.
	DefaultCaddyServerPort = 2019
)

// CaddyEndpoint identifies the Caddy admin API endpoint.
type CaddyEndpoint struct {
	ServerIP   string
	ServerPort int
}

// ClientSet contains the service clients shared by CLI, TUI, and future web adapters.
type ClientSet struct {
	Caddy      *api.CaddyClient
	Unbound    *api.Client
	DNSMasq    *api.DNSMasqClient
	Adguard    *api.AdguardClient
	Cloudflare *api.CloudflareClient
}

// Runtime contains loaded configuration, resolved defaults, and constructed clients.
type Runtime struct {
	UnboundConfig    api.Config
	AdguardConfig    config.AdguardConfig
	CloudflareConfig config.CloudflareConfig
	CaddyEndpoint    CaddyEndpoint
	CaddyServiceURL  string
	Clients          ClientSet
}

// RuntimeOptions controls which optional clients are constructed.
type RuntimeOptions struct {
	CaddyServerIP   string
	CaddyServerPort int

	IncludeUnbound    bool
	IncludeDNSMasq    bool
	IncludeAdguard    bool
	RequireAdguard    bool
	IncludeCloudflare bool
	RequireCloudflare bool
}

// LoadRuntime loads repository configuration and builds the requested clients.
func LoadRuntime(options RuntimeOptions) (*Runtime, error) {
	var unboundConfig api.Config
	var err error
	if options.IncludeUnbound || options.IncludeDNSMasq {
		unboundConfig, err = config.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("error loading main configuration: %w", err)
		}
	}

	var adguardConfig config.AdguardConfig
	if options.IncludeAdguard {
		adguardConfig, err = config.LoadAdguardConfig()
		if err != nil && options.RequireAdguard {
			return nil, fmt.Errorf("error loading AdguardHome configuration: %w", err)
		}
	}

	var cloudflareConfig config.CloudflareConfig
	if options.IncludeCloudflare {
		cloudflareConfig, err = config.LoadCloudflareConfig()
		if err != nil {
			return nil, fmt.Errorf("error loading Cloudflare configuration: %w", err)
		}
	}

	return NewRuntimeFromConfigs(unboundConfig, adguardConfig, cloudflareConfig, options)
}

// NewRuntimeFromConfigs builds runtime clients from already-loaded configuration.
func NewRuntimeFromConfigs(
	unboundConfig api.Config,
	adguardConfig config.AdguardConfig,
	cloudflareConfig config.CloudflareConfig,
	options RuntimeOptions,
) (*Runtime, error) {
	endpoint := ResolveCaddyEndpoint(options.CaddyServerIP, options.CaddyServerPort)

	runtime := &Runtime{
		UnboundConfig:    unboundConfig,
		AdguardConfig:    adguardConfig,
		CloudflareConfig: cloudflareConfig,
		CaddyEndpoint:    endpoint,
		CaddyServiceURL:  ResolveCaddyServiceURL(cloudflareConfig, endpoint),
		Clients: ClientSet{
			Caddy: api.NewCaddyClient(endpoint.ServerIP, endpoint.ServerPort),
		},
	}

	if options.IncludeUnbound {
		runtime.Clients.Unbound = api.NewClient(unboundConfig)
	}

	if options.IncludeDNSMasq {
		runtime.Clients.DNSMasq = api.NewDNSMasqClient(unboundConfig)
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

	return runtime, nil
}

// ResolveCaddyEndpoint applies existing command defaults to an optional endpoint override.
func ResolveCaddyEndpoint(serverIP string, serverPort int) CaddyEndpoint {
	if serverIP == "" {
		serverIP = DefaultCaddyServerIP
	}
	if serverPort == 0 {
		serverPort = DefaultCaddyServerPort
	}
	return CaddyEndpoint{ServerIP: serverIP, ServerPort: serverPort}
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
