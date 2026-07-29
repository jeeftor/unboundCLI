// cloudflare_access.go in internal/api package
//
// This file adds Cloudflare Access management methods to CloudflareClient.
// Access is Cloudflare's Zero Trust authentication layer — it sits at the
// edge and requires users to authenticate (via an IdP like Google/Authentik)
// before traffic reaches the tunnel.
//
// These methods wrap the cloudflare-go SDK's Access APIs:
//   - Access Applications (per-hostname or wildcard)
//   - Access Policies (allow/deny/bypass/service_auth rules)
//   - Access Service Tokens (for API/machine-to-machine auth)
//   - Access Groups (reusable identity groups)
package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
)

// PolicyDecision defines what happens when a policy matches.
type PolicyDecision string

const (
	// DecisionAllow — allow the request through (user must authenticate).
	DecisionAllow PolicyDecision = "allow"
	// DecisionDeny — block the request.
	DecisionDeny PolicyDecision = "deny"
	// DecisionBypass — skip Access auth for this request (used for
	// forward_auth hosts where Authentik handles auth instead of CF Access).
	DecisionBypass PolicyDecision = "bypass"
	// DecisionServiceAuth — require a service token instead of IdP login.
	// Used for API/machine-to-machine access.
	DecisionServiceAuth PolicyDecision = "service_auth"
	// DecisionNonIdentity — allow without identity (no user info passed).
	DecisionNonIdentity PolicyDecision = "non_identity"
)

// --- Access Applications ---

// AccessAppInfo is a simplified view of a CF Access application.
type AccessAppInfo struct {
	ID              string
	Name            string
	Domain          string // e.g. "jellyfin.vookie.net" or "*.vookie.net"
	Type            string // "self_hosted", "wildcard", "app_launcher", etc
	SessionDuration string // e.g. "24h"
	AutoRedirect    bool
}

// ListAccessApps returns all CF Access applications for the account.
func (c *CloudflareClient) ListAccessApps() ([]AccessAppInfo, error) {
	ctx := context.Background()
	apps, _, err := c.api.ListAccessApplications(
		ctx,
		cloudflare.AccountIdentifier(c.accountID),
		cloudflare.ListAccessApplicationsParams{},
	)
	if err != nil {
		return nil, fmt.Errorf("listing CF Access apps: %w", err)
	}
	result := make([]AccessAppInfo, 0, len(apps))
	for _, app := range apps {
		info := AccessAppInfo{
			ID:     app.ID,
			Name:   app.Name,
			Domain: app.Domain,
			Type:   string(app.Type),
		}
		if app.SessionDuration != "" {
			info.SessionDuration = app.SessionDuration
		}
		if app.AutoRedirectToIdentity != nil {
			info.AutoRedirect = *app.AutoRedirectToIdentity
		}
		result = append(result, info)
	}
	return result, nil
}

// FindAccessAppByDomain returns the CF Access app whose domain matches,
// or nil if not found. This handles both exact matches (e.g.
// "jellyfin.vookie.net") and wildcard matches (e.g. "*.vookie.net").
func (c *CloudflareClient) FindAccessAppByDomain(domain string) (*AccessAppInfo, error) {
	apps, err := c.ListAccessApps()
	if err != nil {
		return nil, err
	}
	// First try exact match
	for i := range apps {
		if apps[i].Domain == domain {
			return &apps[i], nil
		}
	}
	// Then try wildcard match (e.g. "*.vookie.net" matches "jellyfin.vookie.net")
	for i := range apps {
		if IsWildcardMatch(apps[i].Domain, domain) {
			return &apps[i], nil
		}
	}
	return nil, nil
}

// CreateAccessAppRequest contains the parameters for creating a CF Access app.
type CreateAccessAppRequest struct {
	Name            string
	Domain          string // e.g. "jellyfin.vookie.net"
	SessionDuration string // e.g. "24h" (default "24h" if empty)
	// AutoRedirect skips the interstitial page and goes straight to the IdP.
	AutoRedirect bool
}

// CreateAccessApp creates a new CF Access application for a hostname.
func (c *CloudflareClient) CreateAccessApp(req CreateAccessAppRequest) (*AccessAppInfo, error) {
	ctx := context.Background()

	sessionDuration := req.SessionDuration
	if sessionDuration == "" {
		sessionDuration = "24h"
	}

	autoRedirect := req.AutoRedirect
	skipInterstitial := req.AutoRedirect

	params := cloudflare.CreateAccessApplicationParams{
		Name:                   req.Name,
		Domain:                 req.Domain,
		Type:                   cloudflare.SelfHosted,
		SessionDuration:        sessionDuration,
		AutoRedirectToIdentity: &autoRedirect,
		SkipInterstitial:       &skipInterstitial,
	}

	app, err := c.api.CreateAccessApplication(
		ctx,
		cloudflare.AccountIdentifier(c.accountID),
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("creating CF Access app for %s: %w", req.Domain, err)
	}

	info := AccessAppInfo{
		ID:              app.ID,
		Name:            app.Name,
		Domain:          app.Domain,
		Type:            string(app.Type),
		SessionDuration: app.SessionDuration,
		AutoRedirect:    autoRedirect,
	}
	logging.Info("Created CF Access app", "name", req.Name, "domain", req.Domain, "id", app.ID)
	return &info, nil
}

// DeleteAccessApp removes a CF Access application by its ID.
func (c *CloudflareClient) DeleteAccessApp(appID string) error {
	ctx := context.Background()
	err := c.api.DeleteAccessApplication(
		ctx,
		cloudflare.AccountIdentifier(c.accountID),
		appID,
	)
	if err != nil {
		return fmt.Errorf("deleting CF Access app %s: %w", appID, err)
	}
	logging.Info("Deleted CF Access app", "id", appID)
	return nil
}

// --- Access Policies ---

// AccessPolicyInfo is a simplified view of a CF Access policy.
type AccessPolicyInfo struct {
	ID         string
	Name       string
	Decision   PolicyDecision
	Precedence int
	AppID      string // the application this policy is attached to
}

// ListAccessPolicies returns all policies for a given CF Access application.
func (c *CloudflareClient) ListAccessPolicies(appID string) ([]AccessPolicyInfo, error) {
	ctx := context.Background()
	policies, _, err := c.api.ListAccessPolicies(
		ctx,
		cloudflare.AccountIdentifier(c.accountID),
		cloudflare.ListAccessPoliciesParams{ApplicationID: appID},
	)
	if err != nil {
		return nil, fmt.Errorf("listing CF Access policies for app %s: %w", appID, err)
	}
	result := make([]AccessPolicyInfo, 0, len(policies))
	for _, p := range policies {
		result = append(result, AccessPolicyInfo{
			ID:         p.ID,
			Name:       p.Name,
			Decision:   PolicyDecision(p.Decision),
			Precedence: p.Precedence,
			AppID:      appID,
		})
	}
	return result, nil
}

// CreateAccessPolicyRequest contains the parameters for creating a CF Access policy.
type CreateAccessPolicyRequest struct {
	AppID      string
	Name       string
	Decision   PolicyDecision
	Precedence int
	// Include defines who is allowed (OR logic). Use AccessGroup* types
	// from the cloudflare-go SDK, e.g. cloudflare.AccessGroupEveryone{},
	// cloudflare.AccessGroupEmail{...}, cloudflare.AccessGroupServiceToken{...}
	Include []interface{}
	// Exclude defines who is blocked (NOT logic).
	Exclude []interface{}
	// Require defines additional requirements (AND logic).
	Require []interface{}
}

// CreateAccessPolicy creates a new CF Access policy on the given application.
func (c *CloudflareClient) CreateAccessPolicy(req CreateAccessPolicyRequest) (*AccessPolicyInfo, error) {
	ctx := context.Background()

	params := cloudflare.CreateAccessPolicyParams{
		ApplicationID: req.AppID,
		Name:          req.Name,
		Decision:      string(req.Decision),
		Precedence:    req.Precedence,
		Include:       req.Include,
		Exclude:       req.Exclude,
		Require:       req.Require,
	}

	policy, err := c.api.CreateAccessPolicy(
		ctx,
		cloudflare.AccountIdentifier(c.accountID),
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("creating CF Access policy %q: %w", req.Name, err)
	}

	info := AccessPolicyInfo{
		ID:         policy.ID,
		Name:       policy.Name,
		Decision:   PolicyDecision(policy.Decision),
		Precedence: policy.Precedence,
		AppID:      req.AppID,
	}
	logging.Info("Created CF Access policy", "name", req.Name, "decision", req.Decision, "appID", req.AppID)
	return &info, nil
}

// DeleteAccessPolicy removes a CF Access policy by its ID.
func (c *CloudflareClient) DeleteAccessPolicy(appID, policyID string) error {
	ctx := context.Background()
	err := c.api.DeleteAccessPolicy(
		ctx,
		cloudflare.AccountIdentifier(c.accountID),
		cloudflare.DeleteAccessPolicyParams{
			ApplicationID: appID,
			PolicyID:      policyID,
		},
	)
	if err != nil {
		return fmt.Errorf("deleting CF Access policy %s: %w", policyID, err)
	}
	logging.Info("Deleted CF Access policy", "id", policyID, "appID", appID)
	return nil
}

// --- Access Service Tokens ---

// ServiceTokenInfo is a simplified view of a CF Access service token.
type ServiceTokenInfo struct {
	ID        string
	Name      string
	ClientID  string
	ExpiresAt *time.Time
	// ClientSecret is only populated at creation/rotation time.
	// It is NOT returned by ListAccessServiceTokens.
	ClientSecret string
}

// ListServiceTokens returns all CF Access service tokens.
func (c *CloudflareClient) ListServiceTokens() ([]ServiceTokenInfo, error) {
	ctx := context.Background()
	tokens, _, err := c.api.ListAccessServiceTokens(
		ctx,
		cloudflare.AccountIdentifier(c.accountID),
		cloudflare.ListAccessServiceTokensParams{},
	)
	if err != nil {
		return nil, fmt.Errorf("listing CF service tokens: %w", err)
	}
	result := make([]ServiceTokenInfo, 0, len(tokens))
	for _, t := range tokens {
		result = append(result, ServiceTokenInfo{
			ID:        t.ID,
			Name:      t.Name,
			ClientID:  t.ClientID,
			ExpiresAt: t.ExpiresAt,
		})
	}
	return result, nil
}

// FindServiceTokenByName returns the service token with the given name, or nil.
func (c *CloudflareClient) FindServiceTokenByName(name string) (*ServiceTokenInfo, error) {
	tokens, err := c.ListServiceTokens()
	if err != nil {
		return nil, err
	}
	for i := range tokens {
		if tokens[i].Name == name {
			return &tokens[i], nil
		}
	}
	return nil, nil
}

// CreateServiceTokenRequest contains the parameters for creating a service token.
type CreateServiceTokenRequest struct {
	Name     string
	Duration string // e.g. "8760h" for 1 year, "forever" for no expiry
}

// CreateServiceToken creates a new CF Access service token.
//
// IMPORTANT: The ClientSecret is only returned once at creation time.
// The caller must store it securely — it cannot be retrieved later.
func (c *CloudflareClient) CreateServiceToken(req CreateServiceTokenRequest) (*ServiceTokenInfo, error) {
	ctx := context.Background()

	duration := req.Duration
	if duration == "" {
		duration = "8760h" // default 1 year
	}

	resp, err := c.api.CreateAccessServiceToken(
		ctx,
		cloudflare.AccountIdentifier(c.accountID),
		cloudflare.CreateAccessServiceTokenParams{
			Name:     req.Name,
			Duration: duration,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("creating CF service token %q: %w", req.Name, err)
	}

	info := ServiceTokenInfo{
		ID:           resp.ID,
		Name:         resp.Name,
		ClientID:     resp.ClientID,
		ClientSecret: resp.ClientSecret, // only available at creation
		ExpiresAt:    resp.ExpiresAt,
	}
	logging.Info("Created CF service token", "name", req.Name, "id", resp.ID)
	return &info, nil
}

// RotateServiceToken generates a new client secret for an existing service token.
// The old secret is invalidated. The new secret is only returned once.
func (c *CloudflareClient) RotateServiceToken(tokenID string) (*ServiceTokenInfo, error) {
	ctx := context.Background()

	resp, err := c.api.RotateAccessServiceToken(
		ctx,
		cloudflare.AccountIdentifier(c.accountID),
		tokenID,
	)
	if err != nil {
		return nil, fmt.Errorf("rotating CF service token %s: %w", tokenID, err)
	}

	info := ServiceTokenInfo{
		ID:           resp.ID,
		Name:         resp.Name,
		ClientID:     resp.ClientID,
		ClientSecret: resp.ClientSecret,
		ExpiresAt:    resp.ExpiresAt,
	}
	logging.Info("Rotated CF service token", "id", tokenID)
	return &info, nil
}

// DeleteServiceToken removes a CF Access service token by its ID.
func (c *CloudflareClient) DeleteServiceToken(tokenID string) error {
	ctx := context.Background()
	_, err := c.api.DeleteAccessServiceToken(
		ctx,
		cloudflare.AccountIdentifier(c.accountID),
		tokenID,
	)
	if err != nil {
		return fmt.Errorf("deleting CF service token %s: %w", tokenID, err)
	}
	logging.Info("Deleted CF service token", "id", tokenID)
	return nil
}

// --- Access Groups ---

// AccessGroupInfo is a simplified view of a CF Access group.
type AccessGroupInfo struct {
	ID   string
	Name string
}

// ListAccessGroups returns all CF Access groups.
func (c *CloudflareClient) ListAccessGroups() ([]AccessGroupInfo, error) {
	ctx := context.Background()
	groups, _, err := c.api.ListAccessGroups(
		ctx,
		cloudflare.AccountIdentifier(c.accountID),
		cloudflare.ListAccessGroupsParams{},
	)
	if err != nil {
		return nil, fmt.Errorf("listing CF Access groups: %w", err)
	}
	result := make([]AccessGroupInfo, 0, len(groups))
	for _, g := range groups {
		result = append(result, AccessGroupInfo{
			ID:   g.ID,
			Name: g.Name,
		})
	}
	return result, nil
}

// FindAccessGroupByName returns the access group with the given name, or nil.
func (c *CloudflareClient) FindAccessGroupByName(name string) (*AccessGroupInfo, error) {
	groups, err := c.ListAccessGroups()
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].Name == name {
			return &groups[i], nil
		}
	}
	return nil, nil
}

// --- High-level helpers ---

// EnsureAccessApp creates a CF Access application for the given hostname if
// one doesn't already exist (checking both exact and wildcard matches).
// Returns the app info (existing or newly created).
func (c *CloudflareClient) EnsureAccessApp(hostname string) (*AccessAppInfo, error) {
	// Check if an app already exists (exact or wildcard match)
	existing, err := c.FindAccessAppByDomain(hostname)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		logging.Debug("CF Access app already exists", "hostname", hostname, "domain", existing.Domain, "id", existing.ID)
		return existing, nil
	}

	// Create a new per-hostname app
	return c.CreateAccessApp(CreateAccessAppRequest{
		Name:   hostname,
		Domain: hostname,
	})
}

// EnsureBypassPolicy creates a "bypass" policy on the given CF Access app.
// This is used for Pattern D hosts (forward_auth on WAN) where CF Access
// should skip authentication and let Caddy's forward_auth handle it.
func (c *CloudflareClient) EnsureBypassPolicy(appID string) (*AccessPolicyInfo, error) {
	// Check if a bypass policy already exists
	policies, err := c.ListAccessPolicies(appID)
	if err != nil {
		return nil, err
	}
	for _, p := range policies {
		if p.Decision == DecisionBypass {
			logging.Debug("Bypass policy already exists", "appID", appID, "policyID", p.ID)
			return &p, nil
		}
	}

	// Create a bypass policy that matches everyone
	return c.CreateAccessPolicy(CreateAccessPolicyRequest{
		AppID:    appID,
		Name:     "bypass-for-forward-auth",
		Decision: DecisionBypass,
		Include:  []interface{}{cloudflare.AccessGroupEveryone{Everyone: struct{}{}}},
	})
}

// EnsureServiceAuthPolicy creates a "service_auth" policy on the given CF
// Access app. This is used for API access — callers send CF-Access-Client-Id
// and CF-Access-Client-Secret headers instead of doing a browser login.
func (c *CloudflareClient) EnsureServiceAuthPolicy(appID, serviceTokenID string) (*AccessPolicyInfo, error) {
	policyName := "service-token-auth"

	// Check if a service_auth policy already exists
	policies, err := c.ListAccessPolicies(appID)
	if err != nil {
		return nil, err
	}
	for _, p := range policies {
		if p.Decision == DecisionServiceAuth && p.Name == policyName {
			logging.Debug("Service auth policy already exists", "appID", appID, "policyID", p.ID)
			return &p, nil
		}
	}

	// Create a service_auth policy that requires the specific service token
	return c.CreateAccessPolicy(CreateAccessPolicyRequest{
		AppID:    appID,
		Name:     policyName,
		Decision: DecisionServiceAuth,
		Include: []interface{}{
			cloudflare.AccessGroupServiceToken{
				ServiceToken: struct {
					ID string `json:"token_id"`
				}{ID: serviceTokenID},
			},
		},
	})
}

// --- internal helpers ---

// IsWildcardMatch checks if a wildcard domain (e.g. "*.vookie.net") matches
// a specific hostname (e.g. "jellyfin.vookie.net").
// Exported so other packages (e.g. internal/auth) can reuse the same logic.
func IsWildcardMatch(wildcard, hostname string) bool {
	if len(wildcard) < 2 || wildcard[:2] != "*." {
		return false
	}
	suffix := wildcard[1:] // ".vookie.net"
	return len(hostname) > len(suffix) && hostname[len(hostname)-len(suffix):] == suffix
}

// IsWildcardDomain returns true if domain starts with "*." (e.g. "*.vookie.net").
func IsWildcardDomain(domain string) bool {
	return len(domain) > 2 && domain[:2] == "*."
}

// IsWildcardOrRootHostname returns true for wildcard patterns ("*.example.com"),
// root domain entries (".example.com"), or empty strings. These are catch-all
// DNS overrides, not real hosts, and should be skipped in host-level scans.
func IsWildcardOrRootHostname(hostname string) bool {
	return hostname == "" ||
		strings.HasPrefix(hostname, ".") ||
		strings.HasPrefix(hostname, "*.")
}
