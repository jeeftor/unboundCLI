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
// with each hostname mapped to the server that should handle it
func (c *CaddyClient) GetHostnameMap() (map[string]string, error) {
	config, err := c.GetConfig()
	if err != nil {
		return nil, err
	}

	hostnames, err := c.ExtractHostnames(config)
	if err != nil {
		return nil, err
	}

	// Create a map of hostnames to the Caddy server IP
	result := make(map[string]string)
	for _, hostname := range hostnames {
		result[hostname] = c.ServerIP
	}

	return result, nil
}
