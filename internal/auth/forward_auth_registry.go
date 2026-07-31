package auth

import (
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// ForwardAuthRegistry tracks which hostnames require forward_auth in Caddy.
// Apps that depend on Authentik headers (X-Authentik-Username, etc.) for user
// identity need forward_auth — without it, they can't tell who's logged in and
// will return 403 or similar errors even when CF Access is active.
//
// The registry is seeded with known hostnames and can be extended via a YAML
// config file at ~/.config/caddy-dns-sync/forward_auth.yaml.

// forwardAuthHosts is the built-in set of hostnames known to require
// forward_auth. These are apps that read Authentik headers for identity.
var forwardAuthHosts = map[string]bool{
	"users.vookie.net": true,
	"sb.vookie.net":    true,
}

var (
	registryMu     sync.RWMutex
	registryLoaded  bool
	registryExtras  map[string]bool
)

// forwardAuthConfig is the YAML structure for the override file.
type forwardAuthConfig struct {
	// Require lists hostnames that require forward_auth.
	Require []string `yaml:"require"`
	// Exclude lists hostnames to exclude from the built-in registry.
	Exclude []string `yaml:"exclude"`
}

// loadRegistry loads the optional YAML override file once.
func loadRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	if registryLoaded {
		return
	}
	registryLoaded = true
	registryExtras = make(map[string]bool)

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	path := home + "/.config/caddy-dns-sync/forward_auth.yaml"
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg forwardAuthConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return
	}
	for _, h := range cfg.Require {
		registryExtras[strings.ToLower(strings.TrimSpace(h))] = true
	}
	for _, h := range cfg.Exclude {
		h = strings.ToLower(strings.TrimSpace(h))
		delete(forwardAuthHosts, h)
		delete(registryExtras, h)
	}
}

// RequiresForwardAuth returns true if the given hostname is known to require
// forward_auth in Caddy.
func RequiresForwardAuth(hostname string) bool {
	loadRegistry()
	h := strings.ToLower(strings.TrimSpace(hostname))
	registryMu.RLock()
	defer registryMu.RUnlock()
	return forwardAuthHosts[h] || registryExtras[h]
}
