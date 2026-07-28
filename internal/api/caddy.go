package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

// caddyHTTPClient is a shared client with a sane timeout, matching the pattern
// used by the other API clients in this package.
var caddyHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

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

	logging.Debug("Fetching Caddy config", "url", url)
	resp, err := caddyHTTPClient.Get(url)
	if err != nil {
		logging.Error("Failed to connect to Caddy server", "error", err)
		return nil, fmt.Errorf("failed to connect to Caddy server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logging.Error("Caddy returned unexpected status", "code", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var config map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		logging.Error("Failed to parse Caddy config", "error", err)
		return nil, fmt.Errorf("failed to parse Caddy config: %w", err)
	}

	logging.Debug("Loaded Caddy config")
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
					// Sanitize: strip trailing commas/whitespace that result
					// from Caddyfile syntax errors (e.g. `host foo.com, bar.com`
					// — Caddy keeps the comma as part of the hostname).
					original := hostStr
					hostStr = strings.TrimRight(hostStr, ", \t\r\n")
					if hostStr != original {
						logging.Warn("Stripped invalid trailing characters from hostname",
							"original", original, "cleaned", hostStr,
							"hint", "Check Caddyfile host matchers — use spaces, not commas, to separate hostnames")
					}

					// Skip empty after sanitization
					if hostStr == "" {
						continue
					}

					// Skip wildcard entries
					if strings.HasPrefix(hostStr, "*.") {
						continue
					}

					// Skip hosts with port numbers for now
					if strings.Contains(hostStr, ":") {
						hostBase := strings.Split(hostStr, ":")[0]
						if !slices.Contains(*hostnames, hostBase) {
							*hostnames = append(*hostnames, hostBase)
						}
						continue
					}

					// Add the host if not already in list
					if !slices.Contains(*hostnames, hostStr) {
						*hostnames = append(*hostnames, hostStr)
					}
				}
			}
		}
	}
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

// GetHostnameDetails returns a per-hostname CaddyRouteInfo with the full handler chain,
// request/response headers, and TLS-transport flag. Purely read-only.
func (c *CaddyClient) GetHostnameDetails() (map[string]models.CaddyRouteInfo, error) {
	config, err := c.GetConfig()
	if err != nil {
		return nil, err
	}
	result := make(map[string]models.CaddyRouteInfo)

	apps, _ := config["apps"].(map[string]interface{})
	httpApp, _ := apps["http"].(map[string]interface{})
	servers, _ := httpApp["servers"].(map[string]interface{})

	for _, server := range servers {
		serverObj, ok := server.(map[string]interface{})
		if !ok {
			continue
		}
		routes, ok := serverObj["routes"].([]interface{})
		if !ok {
			continue
		}
		c.collectRouteDetails(routes, result)
	}
	return result, nil
}

// collectRouteDetails walks a routes array and populates result for every route that
// has a host-match condition.
func (c *CaddyClient) collectRouteDetails(routes []interface{}, result map[string]models.CaddyRouteInfo) {
	for _, r := range routes {
		rObj, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		var hostnames []string
		c.extractHostsFromMatch(rObj["match"], &hostnames)

		if len(hostnames) > 0 {
			chain, upstream, reqSet, reqAdd, respSet, tlsUpstream := c.extractHandlerChain(rObj)
			hasForwardAuth, conditionalFA := routeContainsForwardAuth(rObj)
			for _, h := range hostnames {
				if _, exists := result[h]; !exists {
					result[h] = models.CaddyRouteInfo{
						Upstream:               upstream,
						HandlerChain:           chain,
						RequestHeadersSet:      reqSet,
						RequestHeadersAdd:      reqAdd,
						ResponseHeadersSet:     respSet,
						TLSToUpstream:          tlsUpstream,
						HasForwardAuth:         hasForwardAuth,
						ConditionalForwardAuth: conditionalFA,
					}
				}
			}
		}

		// Recurse into subroutes that carry their own host matches
		if handle, ok := rObj["handle"].([]interface{}); ok {
			for _, h := range handle {
				hObj, ok := h.(map[string]interface{})
				if !ok {
					continue
				}
				if hType, _ := hObj["handler"].(string); hType == "subroute" {
					if subRoutes, ok := hObj["routes"].([]interface{}); ok {
						c.collectRouteDetails(subRoutes, result)
					}
				}
			}
		}
	}
}

// extractHandlerChain recursively walks the handle list of a single route object and
// collects handler type names, reverse_proxy details (headers, transport), and upstream.
func (c *CaddyClient) extractHandlerChain(routeObj map[string]interface{}) (
	chain []string,
	upstream string,
	reqSet, reqAdd, respSet map[string]string,
	tlsUpstream bool,
) {
	reqSet = make(map[string]string)
	reqAdd = make(map[string]string)
	respSet = make(map[string]string)

	handle, ok := routeObj["handle"].([]interface{})
	if !ok {
		return
	}

	for _, h := range handle {
		hObj, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		handlerType, _ := hObj["handler"].(string)
		if handlerType != "" && handlerType != "subroute" {
			chain = append(chain, handlerType)
		}

		switch handlerType {
		case "reverse_proxy":
			if upstreams, ok := hObj["upstreams"].([]interface{}); ok && len(upstreams) > 0 {
				if u, ok := upstreams[0].(map[string]interface{}); ok {
					if dial, ok := u["dial"].(string); ok {
						upstream = dial
					}
				}
			}
			caddyExtractHeaderMap(hObj, "request", "set", reqSet)
			caddyExtractHeaderMap(hObj, "request", "add", reqAdd)
			caddyExtractHeaderMap(hObj, "response", "set", respSet)
			if transport, ok := hObj["transport"].(map[string]interface{}); ok {
				if _, hasTLS := transport["tls"]; hasTLS {
					tlsUpstream = true
				}
			}

		case "subroute":
			if subRoutes, ok := hObj["routes"].([]interface{}); ok {
				for _, sr := range subRoutes {
					srObj, ok := sr.(map[string]interface{})
					if !ok {
						continue
					}
					subChain, subUpstream, subReqSet, subReqAdd, subRespSet, subTLS := c.extractHandlerChain(srObj)
					chain = append(chain, subChain...)
					if subUpstream != "" {
						upstream = subUpstream
					}
					for k, v := range subReqSet {
						reqSet[k] = v
					}
					for k, v := range subReqAdd {
						reqAdd[k] = v
					}
					for k, v := range subRespSet {
						respSet[k] = v
					}
					if subTLS {
						tlsUpstream = true
					}
				}
			}
		}
	}
	return
}

// routeContainsForwardAuth checks whether a route object contains an Authentik
// forward_auth pattern by serializing to JSON and looking for the outpost marker.
// It also detects whether forward_auth is conditional — i.e., only present in
// some subroute handlers but skipped for CF tunnel traffic (matched by
// Cf-Connecting-Ip header). This avoids deeply walking the nested handler tree.
func routeContainsForwardAuth(routeObj map[string]interface{}) (bool, bool) {
	data, err := json.Marshal(routeObj)
	if err != nil {
		return false, false
	}
	hasFA := strings.Contains(string(data), "outpost.goauthentik.io")
	if !hasFA {
		return false, false
	}

	// Check if forward_auth is conditional by examining subroute handlers.
	// If there are multiple subroute handlers and only some have forward_auth
	// (specifically, if there's a handler matching Cf-Connecting-Ip that
	// does NOT have forward_auth), then it's conditional.
	conditionalFA := detectConditionalForwardAuth(routeObj)
	return hasFA, conditionalFA
}

// detectConditionalForwardAuth returns true if the route has multiple subroute
// handlers where forward_auth only appears in some (not all) of them,
// AND at least one handler without forward_auth matches CF tunnel traffic
// (via Cf-Connecting-Ip header matcher).
func detectConditionalForwardAuth(routeObj map[string]interface{}) bool {
	// Walk the route tree to find subroute handlers
	subroutes := findSubrouteHandlers(routeObj)
	if len(subroutes) < 2 {
		return false
	}

	// Check if any subroute has forward_auth and any doesn't
	hasFAWithCFSkip := false
	hasFAInAny := false

	for _, sr := range subroutes {
		// Check if this route (or any of its nested handlers) contains forward_auth
		srJSON, _ := json.Marshal(sr)
		srHasFA := strings.Contains(string(srJSON), "outpost.goauthentik.io")
		if srHasFA {
			hasFAInAny = true
			continue
		}

		// This subroute doesn't have forward_auth.
		// Check if it matches CF tunnel traffic (Cf-Connecting-Ip header)
		// or external traffic (not client_ip LAN ranges)
		matches, ok := sr["match"].([]interface{})
		if !ok {
			continue // catch-all without FA — still a conditional pattern
		}
		for _, m := range matches {
			mObj, ok := m.(map[string]interface{})
			if !ok {
				continue
			}
			// Check for Cf-Connecting-Ip header matcher
			if header, ok := mObj["header"].(map[string]interface{}); ok {
				if _, ok := header["Cf-Connecting-Ip"]; ok {
					hasFAWithCFSkip = true
				}
			}
			// Check for "not" client_ip matcher (external traffic)
			if notMatch, ok := mObj["not"].([]interface{}); ok {
				for _, nm := range notMatch {
					if nmObj, ok := nm.(map[string]interface{}); ok {
						if _, ok := nmObj["client_ip"]; ok {
							hasFAWithCFSkip = true
						}
					}
				}
			}
		}
	}

	// Conditional if: some subroutes have FA, some don't, and at least one
	// without FA matches CF tunnel or external traffic
	return hasFAInAny && hasFAWithCFSkip
}

// findSubrouteHandlers walks a route object and returns all top-level routes
// within the first subroute handler found. These routes carry the matchers
// (like Cf-Connecting-Ip, User-Agent, etc.) that determine which traffic
// gets forward_auth vs direct proxying.
func findSubrouteHandlers(routeObj map[string]interface{}) []map[string]interface{} {
	var result []map[string]interface{}

	// Check handle list for subroute handlers
	handle, ok := routeObj["handle"].([]interface{})
	if !ok {
		return result
	}

	for _, h := range handle {
		hObj, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		if hObj["handler"] == "subroute" {
			// This is a subroute — get its inner routes
			if routes, ok := hObj["routes"].([]interface{}); ok {
				for _, r := range routes {
					if rObj, ok := r.(map[string]interface{}); ok {
						result = append(result, rObj)
					}
				}
			}
		}
	}

	return result
}

// caddyExtractHeaderMap extracts a header set/add map from a reverse_proxy handler object.
func caddyExtractHeaderMap(hObj map[string]interface{}, direction, op string, out map[string]string) {
	headers, ok := hObj["headers"].(map[string]interface{})
	if !ok {
		return
	}
	dir, ok := headers[direction].(map[string]interface{})
	if !ok {
		return
	}
	opMap, ok := dir[op].(map[string]interface{})
	if !ok {
		return
	}
	for k, v := range opMap {
		if vals, ok := v.([]interface{}); ok && len(vals) > 0 {
			out[k] = fmt.Sprint(vals[0])
		}
	}
}
