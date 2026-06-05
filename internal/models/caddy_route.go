package models

// CaddyRouteInfo holds the full handler chain detail for a single hostname route,
// extracted from the Caddy Admin API config.
type CaddyRouteInfo struct {
	Upstream           string            // reverse_proxy dial target, e.g. "192.168.0.1:3001"
	HandlerChain       []string          // ordered handler types, e.g. ["rewrite", "reverse_proxy"]
	RequestHeadersSet  map[string]string // headers.request.set values
	RequestHeadersAdd  map[string]string // headers.request.add values
	ResponseHeadersSet map[string]string // headers.response.set values
	TLSToUpstream      bool              // transport includes a tls block
}

// HasXForwardedProto returns true if X-Forwarded-Proto is set or added in request headers.
func (r CaddyRouteInfo) HasXForwardedProto() bool {
	if _, ok := r.RequestHeadersSet["X-Forwarded-Proto"]; ok {
		return true
	}
	if _, ok := r.RequestHeadersAdd["X-Forwarded-Proto"]; ok {
		return true
	}
	return false
}

// HasHostOverride returns true if the Host request header is explicitly overridden.
func (r CaddyRouteInfo) HasHostOverride() bool {
	_, ok := r.RequestHeadersSet["Host"]
	return ok
}
