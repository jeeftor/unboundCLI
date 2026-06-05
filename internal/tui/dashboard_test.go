package tui

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
)

func TestSyncStatusDashboardLoadSyncDataLoadsRealEntries(t *testing.T) {
	caddy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
									"match": [{"host": ["dashboard.example.test"]}],
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
	defer caddy.Close()

	host, port := splitDashboardServerHostPort(t, caddy.URL)
	dashboard := NewSyncStatusDashboard("192.168.1.15")

	if err := dashboard.LoadSyncData(api.NewCaddyClient(host, port), nil, nil, nil); err != nil {
		t.Fatalf("LoadSyncData failed: %v", err)
	}

	statuses := dashboard.GetFilteredStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected one status, got %d", len(statuses))
	}
	if statuses[0].Hostname != "dashboard.example.test" {
		t.Fatalf("expected dashboard.example.test, got %q", statuses[0].Hostname)
	}
	if statuses[0].DataSource != "Caddy" {
		t.Fatalf("expected Caddy data source, got %q", statuses[0].DataSource)
	}
	if statuses[0].Overall != CaddyOnly {
		t.Fatalf("expected CaddyOnly status, got %v", statuses[0].Overall)
	}
	if dashboard.GetSummary().Total != 1 {
		t.Fatalf("expected summary total 1, got %d", dashboard.GetSummary().Total)
	}
}

func splitDashboardServerHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}
	host, portString, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("failed to split server host/port: %v", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatalf("failed to parse server port: %v", err)
	}
	return host, port
}
