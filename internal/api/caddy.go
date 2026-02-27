package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// CaddyClient handles communication with the Caddy server
type CaddyClient struct {
	ServerIP   string
	ServerPort int
}

// NewCaddyClient creates a new Caddy client
func NewCaddyClient(serverIP string, serverPort int) *CaddyClient {
	return &CaddyClient{
		ServerIP:   serverIP,
		ServerPort: serverPort,
	}
}

// GetConfig fetches the Caddy server configuration
func (c *CaddyClient) GetConfig() (map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s:%d/config/", c.ServerIP, c.ServerPort)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Caddy server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var config map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse Caddy config: %w", err)
	}

	return config, nil
}

// ExtractHostnames extracts all hostnames from the Caddy configuration
func (c *CaddyClient) ExtractHostnames(config map[string]interface{}) ([]string, error) {
	var hostnames []string

	// Parse the JSON structure to extract hostnames from route matches
	apps, ok := config["apps"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("apps section not found in Caddy config")
	}

	http, ok := apps["http"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("http section not found in Caddy config")
	}

	servers, ok := http["servers"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("servers section not found in Caddy config")
	}

	// Process each server
	for _, server := range servers {
		serverObj, ok := server.(map[string]interface{})
		if !ok {
			continue
		}

		routes, ok := serverObj["routes"].([]interface{})
		if !ok {
			continue
		}

		// Process routes recursively
		c.processRoutes(routes, &hostnames)
	}

	return hostnames, nil
}

// processRoutes recursively processes routes and extracts hostnames
func (c *CaddyClient) processRoutes(routes []interface{}, hostnames *[]string) {
	for _, route := range routes {
		routeObj, ok := route.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract hosts from match conditions
		c.extractHostsFromMatch(routeObj["match"], hostnames)

		// Check if there are nested routes (handle section)
		handle, ok := routeObj["handle"].([]interface{})
		if ok {
			for _, handler := range handle {
				handlerObj, ok := handler.(map[string]interface{})
				if !ok {
					continue
				}

				// Check for subroutes
				if handlerType, ok := handlerObj["handler"].(string); ok &&
					handlerType == "subroute" {
					subroutes, ok := handlerObj["routes"].([]interface{})
					if ok {
						// Recursively process subroutes
						c.processRoutes(subroutes, hostnames)
					}
				}
			}
		}
	}
}

// extractHostsFromMatch extracts host patterns from match conditions
func (c *CaddyClient) extractHostsFromMatch(match interface{}, hostnames *[]string) {
	matchList, ok := match.([]interface{})
	if !ok {
		return
	}

	for _, matchCondition := range matchList {
		matchObj, ok := matchCondition.(map[string]interface{})
		if !ok {
			continue
		}

		hosts, ok := matchObj["host"].([]interface{})
		if ok {
			for _, host := range hosts {
				hostStr, ok := host.(string)
				if ok {
					// Skip wildcard entries
					if strings.HasPrefix(hostStr, "*.") {
						continue
					}

					// Skip hosts with port numbers for now
					if strings.Contains(hostStr, ":") {
						hostBase := strings.Split(hostStr, ":")[0]
						if !c.containsString(*hostnames, hostBase) {
							*hostnames = append(*hostnames, hostBase)
						}
						continue
					}

					// Add the host if not already in list
					if !c.containsString(*hostnames, hostStr) {
						*hostnames = append(*hostnames, hostStr)
					}
				}
			}
		}
	}
}

// containsString checks if a string is present in a slice
func (c *CaddyClient) containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// GetHostnameMap returns a map of all hostnames in the Caddy configuration,
// with each hostname mapped to its reverse_proxy upstream target
func (c *CaddyClient) GetHostnameMap() (map[string]string, error) {
	config, err := c.GetConfig()
	if err != nil {
		return nil, err
	}

	// Extract hostnames with their upstream targets using manual parsing
	hostnameUpstreams, err := c.ExtractHostnamesWithUpstreams(config)
	if err != nil {
		return nil, err
	}

	return hostnameUpstreams, nil
}

// ExtractHostnamesWithUpstreams extracts hostnames and their reverse_proxy upstream targets
func (c *CaddyClient) ExtractHostnamesWithUpstreams(config map[string]interface{}) (map[string]string, error) {
	result := make(map[string]string)

	apps, ok := config["apps"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("apps section not found in Caddy config")
	}

	http, ok := apps["http"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("http section not found in Caddy config")
	}

	servers, ok := http["servers"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("servers section not found in Caddy config")
	}

	// Process each server
	for _, server := range servers {
		serverObj, ok := server.(map[string]interface{})
		if !ok {
			continue
		}

		routes, ok := serverObj["routes"].([]interface{})
		if !ok {
			continue
		}

		// Process routes to extract hostname -> upstream mappings
		c.processRoutesForUpstreams(routes, result)
	}

	return result, nil
}

// processRoutesForUpstreams recursively processes routes and extracts hostname -> upstream mappings
func (c *CaddyClient) processRoutesForUpstreams(routes []interface{}, result map[string]string) {
	for _, route := range routes {
		routeObj, ok := route.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract hosts from match conditions
		var hostnames []string
		c.extractHostsFromMatch(routeObj["match"], &hostnames)

		// Extract upstream from reverse_proxy handler at THIS level only
		upstream := c.extractUpstreamFromHandle(routeObj["handle"])

		// Only map hostnames if we found an upstream at this level
		// This prevents nested route upstreams from overwriting parent route hostnames
		if len(hostnames) > 0 && upstream != "" {
			for _, hostname := range hostnames {
				// Only set if not already set (first match wins)
				if _, exists := result[hostname]; !exists {
					result[hostname] = upstream
				}
			}
		}

		// Always process nested routes to find their own hostname->upstream mappings
		handle, ok := routeObj["handle"].([]interface{})
		if ok {
			for _, h := range handle {
				hObj, ok := h.(map[string]interface{})
				if !ok {
					continue
				}
				if routes, ok := hObj["routes"].([]interface{}); ok {
					c.processRoutesForUpstreams(routes, result)
				}
			}
		}
	}
}

// extractUpstreamFromHandle extracts the first upstream target from a reverse_proxy handler
// Prioritizes direct reverse_proxy handlers over nested subroutes
func (c *CaddyClient) extractUpstreamFromHandle(handle interface{}) string {
	handlers, ok := handle.([]interface{})
	if !ok {
		return ""
	}

	// First pass: look for direct reverse_proxy handlers (not in subroutes)
	for _, h := range handlers {
		hObj, ok := h.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is a reverse_proxy handler
		if handler, ok := hObj["handler"].(string); ok && handler == "reverse_proxy" {
			// Extract upstreams
			if upstreams, ok := hObj["upstreams"].([]interface{}); ok && len(upstreams) > 0 {
				if upstream, ok := upstreams[0].(map[string]interface{}); ok {
					if dial, ok := upstream["dial"].(string); ok {
						return dial
					}
				}
			}
		}
	}

	// Second pass: if no direct reverse_proxy found, check subroutes
	// This ensures we don't use nested route upstreams when a direct upstream exists
	for _, h := range handlers {
		hObj, ok := h.(map[string]interface{})
		if !ok {
			continue
		}

		// Check for subroutes
		if handlerType, ok := hObj["handler"].(string); ok && handlerType == "subroute" {
			if routes, ok := hObj["routes"].([]interface{}); ok {
				upstream := c.extractUpstreamFromRoutes(routes)
				if upstream != "" {
					return upstream
				}
			}
		}
	}

	return ""
}

// extractUpstreamFromRoutes helper to extract upstream from nested routes
// Prioritizes routes without host matches (fallback routes) over routes with matches
func (c *CaddyClient) extractUpstreamFromRoutes(routes []interface{}) string {
	// First pass: look for routes WITHOUT host matches (fallback routes)
	// These represent the upstream for the parent route
	for _, route := range routes {
		routeObj, ok := route.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this route has a host match
		hasHostMatch := false
		if match, ok := routeObj["match"].([]interface{}); ok {
			for _, m := range match {
				if mObj, ok := m.(map[string]interface{}); ok {
					if _, hasHost := mObj["host"]; hasHost {
						hasHostMatch = true
						break
					}
				}
			}
		}

		// If no host match, this is a fallback route - use its upstream
		if !hasHostMatch {
			upstream := c.extractUpstreamFromHandle(routeObj["handle"])
			if upstream != "" {
				return upstream
			}
		}
	}

	// Second pass: if no fallback route found, use first route with upstream
	for _, route := range routes {
		routeObj, ok := route.(map[string]interface{})
		if !ok {
			continue
		}

		upstream := c.extractUpstreamFromHandle(routeObj["handle"])
		if upstream != "" {
			return upstream
		}
	}
	return ""
}
