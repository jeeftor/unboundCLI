// authentik.go in internal/api package
package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	api "goauthentik.io/api/v3"
)

// AuthentikConfig contains configuration for the Authentik API client.
type AuthentikConfig struct {
	APIToken string
	BaseURL  string // e.g. "https://auth.vookie.net"
	Insecure bool   // skip TLS certificate verification
}

// ProxyMode is the Authentik proxy provider mode.
type ProxyMode string

const (
	// ProxyModeForwardSingle — forward_auth for a single application.
	// Caddy uses auth_request / forward_auth to the outpost.
	ProxyModeForwardSingle ProxyMode = "forward_single"
	// ProxyModeForwardDomain — forward_auth for a domain (multiple apps).
	ProxyModeForwardDomain ProxyMode = "forward_domain"
	// ProxyModeProxy — the outpost acts as a full reverse proxy.
	ProxyModeProxy ProxyMode = "proxy"
)

// AuthentikClient wraps the generated Authentik Go API client with
// higher-level methods for managing proxy providers, applications, policy
// bindings, and outposts.
type AuthentikClient struct {
	client  *api.APIClient
	baseURL string
	token   string
}

// DefaultAuthorizationFlow is the standard implicit-consent authorization
// flow slug used by all existing proxy providers in this Authentik instance.
// The actual flow UUID is resolved at runtime via ListFlows.
const DefaultAuthorizationFlow = "default-provider-authorization-implicit-consent"

// DefaultInvalidationFlow is the standard provider invalidation flow slug.
const DefaultInvalidationFlow = "default-provider-invalidation-flow"

// NewAuthentikClient creates a new Authentik API client.
func NewAuthentikClient(config AuthentikConfig) (*AuthentikClient, error) {
	if config.APIToken == "" {
		return nil, fmt.Errorf("authentik API token is required")
	}
	if config.BaseURL == "" {
		return nil, fmt.Errorf("authentik base URL is required")
	}

	baseURL := strings.TrimRight(config.BaseURL, "/")

	apiConfig := api.NewConfiguration()
	apiConfig.Servers = api.ServerConfigurations{
		{URL: baseURL + "/api/v3"},
	}

	if config.Insecure {
		apiConfig.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		}
	}

	// Authentik uses Bearer token auth. Setting it as a default header
	// authenticates every request without needing per-call context injection.
	apiConfig.AddDefaultHeader("Authorization", "Bearer "+config.APIToken)

	client := api.NewAPIClient(apiConfig)

	return &AuthentikClient{
		client:  client,
		baseURL: baseURL,
		token:   config.APIToken,
	}, nil
}

// ctx returns a background context (auth is via default headers).
func (c *AuthentikClient) ctx() context.Context {
	return context.Background()
}

// --- Flows ---

// FlowInfo is a minimal representation of an Authentik flow.
type FlowInfo struct {
	Slug        string
	Title       string
	Designation string
	UUID        string
}

// ListFlows returns all flows.
func (c *AuthentikClient) ListFlows() ([]FlowInfo, error) {
	flows, _, err := c.client.FlowsAPI.FlowsInstancesList(c.ctx()).Execute()
	if err != nil {
		return nil, fmt.Errorf("listing flows: %w", err)
	}
	result := make([]FlowInfo, 0, len(flows.Results))
	for _, f := range flows.Results {
		result = append(result, FlowInfo{
			Slug:        f.Slug,
			Title:       f.Title,
			Designation: string(f.Designation),
			UUID:        f.Pk,
		})
	}
	return result, nil
}

// GetFlowUUID resolves a flow slug to its UUID. ProxyProviderRequest needs
// the UUID, not the slug.
func (c *AuthentikClient) GetFlowUUID(slug string) (string, error) {
	flows, err := c.ListFlows()
	if err != nil {
		return "", err
	}
	for _, f := range flows {
		if f.Slug == slug {
			return f.UUID, nil
		}
	}
	return "", fmt.Errorf("flow %q not found", slug)
}

// --- Proxy Providers ---

// ProxyProviderInfo is a simplified view of an Authentik proxy provider.
type ProxyProviderInfo struct {
	PK                int32
	Name              string
	Mode              ProxyMode
	ExternalHost      string
	InternalHost      string
	AuthorizationFlow string // flow UUID
	AssignedAppSlug   string // slug of the application using this provider
}

// ListProxyProviders returns all proxy providers.
func (c *AuthentikClient) ListProxyProviders() ([]ProxyProviderInfo, error) {
	providers, _, err := c.client.ProvidersAPI.ProvidersProxyList(c.ctx()).Execute()
	if err != nil {
		return nil, fmt.Errorf("listing proxy providers: %w", err)
	}
	result := make([]ProxyProviderInfo, 0, len(providers.Results))
	for _, p := range providers.Results {
		result = append(result, proxyProviderToInfo(p))
	}
	return result, nil
}

// FindProxyProviderByExternalHost returns the proxy provider whose
// external_host matches the given hostname, or nil if not found.
func (c *AuthentikClient) FindProxyProviderByExternalHost(hostname string) (*ProxyProviderInfo, error) {
	providers, err := c.ListProxyProviders()
	if err != nil {
		return nil, err
	}
	target := "https://" + hostname
	for i := range providers {
		if providers[i].ExternalHost == target {
			return &providers[i], nil
		}
	}
	return nil, nil
}

// CreateProxyProviderRequest contains the parameters for creating a proxy provider.
type CreateProxyProviderRequest struct {
	Name              string
	ExternalHost      string // e.g. "https://users.vookie.net"
	InternalHost      string // optional, e.g. "http://10.0.0.112:9004"
	Mode              ProxyMode
	AuthorizationFlow string // flow UUID (resolve via GetFlowUUID)
	// InterceptHeaderAuth enables Bearer token auth for API access.
	InterceptHeaderAuth bool
}

// CreateProxyProvider creates a new proxy provider in Authentik.
func (c *AuthentikClient) CreateProxyProvider(req CreateProxyProviderRequest) (*ProxyProviderInfo, error) {
	if req.AuthorizationFlow == "" {
		return nil, fmt.Errorf("authorization flow UUID is required")
	}

	mode := api.ProxyMode(req.Mode)
	if mode == "" {
		mode = api.PROXYMODE_FORWARD_SINGLE
	}

	providerReq := api.NewProxyProviderRequest(
		req.Name,
		req.AuthorizationFlow,
		req.AuthorizationFlow, // invalidation flow — same as authz for simplicity
		req.ExternalHost,
	)
	providerReq.SetMode(mode)

	if req.InternalHost != "" {
		ih := req.InternalHost
		providerReq.SetInternalHost(ih)
	}

	if req.InterceptHeaderAuth {
		providerReq.SetInterceptHeaderAuth(true)
	}

	provider, _, err := c.client.ProvidersAPI.
		ProvidersProxyCreate(c.ctx()).
		ProxyProviderRequest(*providerReq).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("creating proxy provider %q: %w", req.Name, err)
	}

	info := proxyProviderToInfo(*provider)
	logging.Info("Created Authentik proxy provider", "name", req.Name, "pk", provider.Pk, "host", req.ExternalHost)
	return &info, nil
}

// DeleteProxyProvider removes a proxy provider by its primary key.
func (c *AuthentikClient) DeleteProxyProvider(pk int32) error {
	_, err := c.client.ProvidersAPI.
		ProvidersProxyDestroy(c.ctx(), pk).
		Execute()
	if err != nil {
		return fmt.Errorf("deleting proxy provider pk=%d: %w", pk, err)
	}
	logging.Info("Deleted Authentik proxy provider", "pk", pk)
	return nil
}

// --- Applications ---

// ApplicationInfo is a simplified view of an Authentik application.
type ApplicationInfo struct {
	Slug     string // unique identifier
	Name     string // display name
	Provider int32  // provider PK (0 if none)
}

// ListApplications returns all applications.
func (c *AuthentikClient) ListApplications() ([]ApplicationInfo, error) {
	apps, _, err := c.client.CoreAPI.CoreApplicationsList(c.ctx()).Execute()
	if err != nil {
		return nil, fmt.Errorf("listing applications: %w", err)
	}
	result := make([]ApplicationInfo, 0, len(apps.Results))
	for _, a := range apps.Results {
		info := ApplicationInfo{
			Slug: a.Slug,
			Name: a.Name,
		}
		if a.Provider.IsSet() {
			if pk := a.Provider.Get(); pk != nil {
				info.Provider = *pk
			}
		}
		result = append(result, info)
	}
	return result, nil
}

// FindApplicationBySlug returns the application with the given slug, or nil.
func (c *AuthentikClient) FindApplicationBySlug(slug string) (*ApplicationInfo, error) {
	apps, err := c.ListApplications()
	if err != nil {
		return nil, err
	}
	for i := range apps {
		if apps[i].Slug == slug {
			return &apps[i], nil
		}
	}
	return nil, nil
}

// CreateApplicationRequest contains the parameters for creating an application.
type CreateApplicationRequest struct {
	Name     string
	Slug     string
	Provider int32 // provider PK to associate with this app
}

// CreateApplication creates a new application in Authentik and associates it
// with the given provider.
func (c *AuthentikClient) CreateApplication(req CreateApplicationRequest) (*ApplicationInfo, error) {
	if req.Name == "" || req.Slug == "" {
		return nil, fmt.Errorf("name and slug are required")
	}

	appReq := api.NewApplicationRequest(req.Name, req.Slug)
	if req.Provider > 0 {
		appReq.SetProvider(req.Provider)
	}

	app, _, err := c.client.CoreAPI.
		CoreApplicationsCreate(c.ctx()).
		ApplicationRequest(*appReq).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("creating application %q: %w", req.Slug, err)
	}

	info := ApplicationInfo{
		Slug: app.Slug,
		Name: app.Name,
	}
	if app.Provider.IsSet() {
		if pk := app.Provider.Get(); pk != nil {
			info.Provider = *pk
		}
	}
	logging.Info("Created Authentik application", "name", req.Name, "slug", req.Slug)
	return &info, nil
}

// DeleteApplication removes an application by its slug.
func (c *AuthentikClient) DeleteApplication(slug string) error {
	_, err := c.client.CoreAPI.
		CoreApplicationsDestroy(c.ctx(), slug).
		Execute()
	if err != nil {
		return fmt.Errorf("deleting application %q: %w", slug, err)
	}
	logging.Info("Deleted Authentik application", "slug", slug)
	return nil
}

// --- Policy Bindings ---

// PolicyBindingInfo is a simplified view of a policy binding.
type PolicyBindingInfo struct {
	UUID    string
	Policy  string // policy UUID
	Target  string // target (application slug or provider PK)
	Order   int32
	Enabled bool
	Negate  bool
}

// ListPolicyBindings returns all policy bindings.
func (c *AuthentikClient) ListPolicyBindings() ([]PolicyBindingInfo, error) {
	bindings, _, err := c.client.PoliciesAPI.PoliciesBindingsList(c.ctx()).Execute()
	if err != nil {
		return nil, fmt.Errorf("listing policy bindings: %w", err)
	}
	result := make([]PolicyBindingInfo, 0, len(bindings.Results))
	for _, b := range bindings.Results {
		info := PolicyBindingInfo{
			UUID:   b.Pk,
			Target: b.Target,
			Order:  b.Order,
		}
		if b.Policy.IsSet() {
			if p := b.Policy.Get(); p != nil {
				info.Policy = *p
			}
		}
		if b.Enabled != nil {
			info.Enabled = *b.Enabled
		}
		if b.Negate != nil {
			info.Negate = *b.Negate
		}
		result = append(result, info)
	}
	return result, nil
}

// --- Outposts ---

// OutpostInfo is a simplified view of an Authentik outpost.
type OutpostInfo struct {
	UUID      string
	Name      string
	Type      string
	Providers []int32 // provider PKs assigned to this outpost
}

// ListOutposts returns all outposts.
func (c *AuthentikClient) ListOutposts() ([]OutpostInfo, error) {
	outposts, _, err := c.client.OutpostsAPI.OutpostsInstancesList(c.ctx()).Execute()
	if err != nil {
		return nil, fmt.Errorf("listing outposts: %w", err)
	}
	result := make([]OutpostInfo, 0, len(outposts.Results))
	for _, o := range outposts.Results {
		result = append(result, OutpostInfo{
			UUID:      o.Pk,
			Name:      o.Name,
			Type:      string(o.Type),
			Providers: o.Providers,
		})
	}
	return result, nil
}

// AddProviderToOutpost adds a provider PK to an outpost's provider list.
// This is needed so the outpost knows to proxy the new application.
func (c *AuthentikClient) AddProviderToOutpost(outpostUUID string, providerPK int32) error {
	// Fetch current outpost to get its existing providers
	outpost, _, err := c.client.OutpostsAPI.
		OutpostsInstancesRetrieve(c.ctx(), outpostUUID).
		Execute()
	if err != nil {
		return fmt.Errorf("retrieving outpost %s: %w", outpostUUID, err)
	}

	// Check if provider is already assigned
	for _, pk := range outpost.Providers {
		if pk == providerPK {
			logging.Debug("Provider already assigned to outpost", "providerPK", providerPK, "outpost", outpost.Name)
			return nil
		}
	}

	// Add the new provider to the list
	providers := append(outpost.Providers, providerPK)

	// Build the update request
	updateReq := api.NewOutpostRequest(outpost.Name, outpost.Type, providers, outpost.Config)
	_, _, err = c.client.OutpostsAPI.
		OutpostsInstancesUpdate(c.ctx(), outpostUUID).
		OutpostRequest(*updateReq).
		Execute()
	if err != nil {
		return fmt.Errorf("updating outpost %s: %w", outpostUUID, err)
	}

	logging.Info("Added provider to outpost", "providerPK", providerPK, "outpost", outpost.Name)
	return nil
}

// --- High-level helpers ---

// EnsureProxyApp creates a proxy provider + application pair for the given
// hostname, if one doesn't already exist. This is the main entry point for
// provisioning forward_auth for a new hostname.
//
// Parameters:
//   - hostname: e.g. "users.vookie.net"
//   - internalHost: e.g. "http://10.0.0.112:9004" (optional for forward_single mode)
//   - mode: ProxyModeForwardSingle (most common) or ProxyModeProxy
//   - appSlug: desired application slug (e.g. "user-manager-proxy")
//   - outpostUUID: the outpost to assign the provider to
func (c *AuthentikClient) EnsureProxyApp(
	hostname string,
	internalHost string,
	mode ProxyMode,
	appSlug string,
	outpostUUID string,
) (*ProxyProviderInfo, *ApplicationInfo, error) {
	// Check if a provider already exists for this hostname
	existing, err := c.FindProxyProviderByExternalHost(hostname)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		logging.Debug("Proxy provider already exists", "hostname", hostname, "pk", existing.PK)
		// Find the associated app
		app, _ := c.FindApplicationBySlug(existing.AssignedAppSlug)
		return existing, app, nil
	}

	// Resolve the authorization flow UUID
	authzFlowUUID, err := c.GetFlowUUID(DefaultAuthorizationFlow)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving authorization flow: %w", err)
	}

	// Create the proxy provider
	provider, err := c.CreateProxyProvider(CreateProxyProviderRequest{
		Name:              appSlug,
		ExternalHost:      "https://" + hostname,
		InternalHost:      internalHost,
		Mode:              mode,
		AuthorizationFlow: authzFlowUUID,
	})
	if err != nil {
		return nil, nil, err
	}

	// Create the application associated with this provider
	app, err := c.CreateApplication(CreateApplicationRequest{
		Name:     appSlug,
		Slug:     appSlug,
		Provider: provider.PK,
	})
	if err != nil {
		// Rollback: delete the provider if app creation fails
		_ = c.DeleteProxyProvider(provider.PK)
		return nil, nil, err
	}

	// Add the provider to the outpost
	if outpostUUID != "" {
		if err := c.AddProviderToOutpost(outpostUUID, provider.PK); err != nil {
			logging.Error("Failed to add provider to outpost", "error", err, "providerPK", provider.PK)
		}
	}

	return provider, app, nil
}

// RemoveProxyApp removes a proxy provider + application pair for the given
// hostname. This is the cleanup path when forward_auth is removed from a host.
func (c *AuthentikClient) RemoveProxyApp(hostname string) error {
	provider, err := c.FindProxyProviderByExternalHost(hostname)
	if err != nil {
		return err
	}
	if provider == nil {
		return nil // nothing to remove
	}

	// Delete the application first (if it exists)
	if provider.AssignedAppSlug != "" {
		if err := c.DeleteApplication(provider.AssignedAppSlug); err != nil {
			logging.Error("Failed to delete application", "slug", provider.AssignedAppSlug, "error", err)
		}
	}

	// Delete the provider
	if err := c.DeleteProxyProvider(provider.PK); err != nil {
		return err
	}

	logging.Info("Removed proxy provider + app", "hostname", hostname)
	return nil
}

// --- internal helpers ---

func proxyProviderToInfo(p api.ProxyProvider) ProxyProviderInfo {
	info := ProxyProviderInfo{
		PK:                p.Pk,
		Name:              p.Name,
		ExternalHost:      p.ExternalHost,
		AuthorizationFlow: p.AuthorizationFlow,
	}
	if p.Mode != nil {
		info.Mode = ProxyMode(*p.Mode)
	}
	if p.InternalHost != nil {
		info.InternalHost = *p.InternalHost
	}
	if p.AssignedApplicationSlug.IsSet() {
		if slug := p.AssignedApplicationSlug.Get(); slug != nil {
			info.AssignedAppSlug = *slug
		}
	}
	return info
}
