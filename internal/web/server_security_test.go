package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

func TestRequestPolicyRejectsUnknownHostAndUnauthenticatedProxyRequests(t *testing.T) {
	server := NewServerWithOptions(&app.Runtime{}, Options{
		ApplyToken:        "token",
		AllowMutations:    true,
		BoundHost:         "127.0.0.1",
		EnforceHostPolicy: true,
		AllowedOrigin:     "https://sync.example.test",
		ProxyAuthHeader:   "X-Proxy-User",
	})

	unknownHost := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	unknownHost.Host = "attacker.example.test"
	assertStatus(t, server, unknownHost, http.StatusForbidden)

	unauthenticated := httptest.NewRequest(http.MethodGet, "/", nil)
	unauthenticated.Host = "sync.example.test"
	assertStatus(t, server, unauthenticated, http.StatusForbidden)

	authenticated := httptest.NewRequest(http.MethodGet, "/", nil)
	authenticated.Host = "sync.example.test"
	authenticated.Header.Set("X-Proxy-User", "operator")
	assertStatus(t, server, authenticated, http.StatusOK)
}

func TestMutationPolicyRejectsMissingTokenAndCrossOrigin(t *testing.T) {
	server := NewServerWithOptions(&app.Runtime{}, Options{
		ApplyToken:        "token",
		AllowMutations:    true,
		BoundHost:         "127.0.0.1",
		EnforceHostPolicy: true,
	})

	missingToken := httptest.NewRequest(http.MethodPost, "/api/cloudflare/set-route", strings.NewReader(`{}`))
	missingToken.Host = "127.0.0.1"
	assertStatus(t, server, missingToken, http.StatusForbidden)

	crossOrigin := httptest.NewRequest(http.MethodPost, "/api/config/test", strings.NewReader(`{"service":"caddy"}`))
	crossOrigin.Host = "127.0.0.1"
	crossOrigin.Header.Set("X-UnboundCLI-Token", "token")
	crossOrigin.Header.Set("Origin", "https://attacker.example.test")
	assertStatus(t, server, crossOrigin, http.StatusForbidden)
}

func TestProbeOnlyPermitsInventoriedDestinationAndRejectsRedirects(t *testing.T) {
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://attacker.example.test", http.StatusFound)
	}))
	defer redirect.Close()
	upstream := strings.TrimPrefix(redirect.URL, "http://")

	server := NewServer(&app.Runtime{})
	server.entriesMu.Lock()
	server.entriesCache = []*models.Entry{{Hostname: "app.example.test", CaddyUpstream: upstream}}
	server.entriesCacheAt = time.Now()
	server.entriesMu.Unlock()

	wrongDestination := httptest.NewRequest(http.MethodGet, "/api/probe?hostname=app.example.test&upstream=127.0.0.1:1", nil)
	assertStatus(t, server, wrongDestination, http.StatusForbidden)

	redirectProbe := httptest.NewRequest(http.MethodGet, "/api/probe?hostname=app.example.test&upstream="+upstream, nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, redirectProbe)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "redirects are not allowed") {
		t.Fatalf("expected blocked redirect probe, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestDNSProbeOnlyPermitsInventoriedHostname(t *testing.T) {
	server := NewServer(&app.Runtime{})
	server.entriesMu.Lock()
	server.entriesCache = []*models.Entry{{Hostname: "app.example.test"}}
	server.entriesCacheAt = time.Now()
	server.entriesMu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/dns-probe?hostname=attacker.example.test", nil)
	assertStatus(t, server, req, http.StatusForbidden)
}

func assertStatus(t *testing.T, server *Server, request *http.Request, want int) {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	if recorder.Code != want {
		t.Fatalf("%s %s: got %d, want %d: %s", request.Method, request.URL, recorder.Code, want, recorder.Body.String())
	}
}
