package sync

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
)

func TestSyncCaddyWithCloudflareAppliesPlannedUpdates(t *testing.T) {
	caddyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config/" {
			t.Fatalf("unexpected Caddy path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"apps": {
				"http": {
					"servers": {
						"srv0": {
							"routes": [
								{
									"match": [{"host": ["app.example.com"]}],
									"handle": [
										{
											"handler": "reverse_proxy",
											"upstreams": [{"dial": "10.0.0.5:8080"}]
										}
									]
								}
							]
						}
					}
				}
			}
		}`)
	}))
	defer caddyServer.Close()

	caddyURL, err := url.Parse(caddyServer.URL)
	if err != nil {
		t.Fatalf("failed to parse Caddy server URL: %v", err)
	}
	caddyHost, caddyPortString, err := net.SplitHostPort(caddyURL.Host)
	if err != nil {
		t.Fatalf("failed to split Caddy host/port: %v", err)
	}
	caddyPort, err := strconv.Atoi(caddyPortString)
	if err != nil {
		t.Fatalf("failed to parse Caddy port: %v", err)
	}

	var updateCalled bool
	var activationCalled bool
	var updatedHost api.DNSOverride
	unboundServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/unbound/settings/searchHostOverride":
			server := "10.0.0.99:8080"
			if updateCalled {
				server = updatedHost.Server
			}
			fmt.Fprintf(w, `{"rows":[{"uuid":"override-1","enabled":"1","hostname":"app","domain":"dev.example.com","server":%q,"description":"managed by test"}],"rowCount":1}`, server)
		case "/api/unbound/settings/setHostOverride/override-1":
			updateCalled = true
			var payload struct {
				Host api.DNSOverride `json:"host"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode update payload: %v", err)
			}
			updatedHost = payload.Host
			fmt.Fprint(w, `{"result":"saved"}`)
		case "/api/unbound/service/reconfigure":
			activationCalled = true
			fmt.Fprint(w, `{"result":"saved"}`)
		case "/api/core/firmware/backup":
			fmt.Fprint(w, `{"result":"saved"}`)
		default:
			t.Fatalf("unexpected Unbound path %s", r.URL.Path)
		}
	}))
	defer unboundServer.Close()

	client := api.NewClient(api.Config{
		APIKey:    "key",
		APISecret: "secret",
		BaseURL:   unboundServer.URL,
		Insecure:  true,
	})

	result, err := SyncCaddyWithCloudflare(client, CaddyCloudflareSyncOptions{
		BaseSyncOptions: BaseSyncOptions{
			CaddyServerIP:    caddyHost,
			CaddyServerPort:  caddyPort,
			EntryDescription: "managed by test",
		},
		DirectSubdomain: "dev",
		SyncDirect:      true,
	})
	if err != nil {
		t.Fatalf("SyncCaddyWithCloudflare failed: %v", err)
	}

	if len(result.ToUpdate) != 1 {
		t.Fatalf("expected one planned update, got %d", len(result.ToUpdate))
	}
	if !updateCalled {
		t.Fatal("expected planned update to call Unbound setHostOverride")
	}
	if !activationCalled {
		t.Fatal("expected planned update to activate Unbound configuration")
	}
	if !result.ChangesApplied {
		t.Fatal("expected verified update to be marked applied")
	}
	if updatedHost.UUID != "override-1" {
		t.Fatalf("expected UUID override-1, got %q", updatedHost.UUID)
	}
	if updatedHost.Server != "10.0.0.5:8080" {
		t.Fatalf("expected server to update to Caddy upstream, got %q", updatedHost.Server)
	}
}

func TestSyncCaddyWithCloudflareReportsActivationFailure(t *testing.T) {
	caddyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"apps":{"http":{"servers":{"srv0":{"routes":[{"match":[{"host":["app.example.com"]}],"handle":[{"handler":"reverse_proxy","upstreams":[{"dial":"10.0.0.5:8080"}]}]}]}}}}}`)
	}))
	defer caddyServer.Close()
	caddyURL, err := url.Parse(caddyServer.URL)
	if err != nil {
		t.Fatalf("parse Caddy server URL: %v", err)
	}
	caddyHost, caddyPortString, err := net.SplitHostPort(caddyURL.Host)
	if err != nil {
		t.Fatalf("split Caddy host/port: %v", err)
	}
	caddyPort, err := strconv.Atoi(caddyPortString)
	if err != nil {
		t.Fatalf("parse Caddy port: %v", err)
	}

	unboundServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/unbound/settings/searchHostOverride":
			fmt.Fprint(w, `{"rows":[{"uuid":"override-1","enabled":"1","hostname":"app","domain":"dev.example.com","server":"10.0.0.99:8080","description":"managed by test"}],"rowCount":1}`)
		case "/api/unbound/settings/setHostOverride/override-1":
			fmt.Fprint(w, `{"result":"saved"}`)
		case "/api/unbound/service/reconfigure":
			fmt.Fprint(w, `{"result":"failed","message":"reload failed"}`)
		default:
			t.Fatalf("unexpected Unbound path %s", r.URL.Path)
		}
	}))
	defer unboundServer.Close()

	result, err := SyncCaddyWithCloudflare(api.NewClient(api.Config{
		APIKey: "key", APISecret: "secret", BaseURL: unboundServer.URL, Insecure: true,
	}), CaddyCloudflareSyncOptions{
		BaseSyncOptions: BaseSyncOptions{CaddyServerIP: caddyHost, CaddyServerPort: caddyPort, EntryDescription: "managed by test"},
		DirectSubdomain: "dev",
		SyncDirect:      true,
	})
	if err == nil || !strings.Contains(err.Error(), "activate Unbound changes") {
		t.Fatalf("expected activation failure, got %v", err)
	}
	if result == nil || result.ChangesApplied {
		t.Fatal("activation failure must not be reported as applied")
	}
}
