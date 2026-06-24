package sync

import (
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/syncplan"
)

func TestCloudflareSyncEntriesAddsDirectSiblingHost(t *testing.T) {
	entries := cloudflareSyncEntries(
		map[string]string{"ssh.vookie.net": "10.0.0.23:22"},
		nil,
		"-direct",
	)

	plan := syncplan.BuildPlan(entries, syncplan.Options{
		Service:         "cloudflare",
		CaddyServiceURL: "https://10.0.0.15",
	})

	if len(plan.Actions) != 2 {
		t.Fatalf("expected normal and direct Cloudflare actions, got %d: %#v", len(plan.Actions), plan.Actions)
	}

	assertCloudflareAction(t, plan.Actions[0], syncplan.Action{
		Type:                 "add",
		Hostname:             "ssh.vookie.net",
		Service:              "cloudflare",
		NewService:           "https://10.0.0.15",
		NewHTTPHostHeader:    "ssh.vookie.net",
		OriginServerName:     "ssh.vookie.net",
		Details:              "missing in default Cloudflare tunnel",
		Enabled:              true,
		ManagedFields:        "service,http_host_header,origin_server_name",
		OriginRequestSummary: "preserve optional origin request fields",
	})
	assertCloudflareAction(t, plan.Actions[1], syncplan.Action{
		Type:                 "add",
		Hostname:             "ssh-direct.vookie.net",
		Service:              "cloudflare",
		NewService:           "http://10.0.0.23:22",
		NewHTTPHostHeader:    "ssh.vookie.net",
		Details:              "missing in default Cloudflare tunnel",
		Enabled:              true,
		ManagedFields:        "service,http_host_header,origin_server_name",
		OriginRequestSummary: "preserve optional origin request fields",
	})
}

func assertCloudflareAction(t *testing.T, got, want syncplan.Action) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected Cloudflare action\nwant: %#v\n got: %#v", want, got)
	}
}
