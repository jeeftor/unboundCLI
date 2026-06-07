package syncplan

import (
	"encoding/json"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

func TestPlanFromEntriesCreatesDNSAddAndUpdateActions(t *testing.T) {
	actions := PlanFromEntries([]*models.Entry{
		{
			Hostname:      "missing.example.com",
			CaddyUpstream: "10.0.0.5:8080",
			UnboundStatus: models.NotConfigured(),
			AdguardStatus: models.NotConfigured(),
			DHCPStatus:    models.NoDHCP(),
		},
		{
			Hostname:      "wrong.example.com",
			CaddyUpstream: "10.0.0.6:8080",
			UnboundStatus: models.NotInSync("10.0.0.99"),
			AdguardStatus: models.Synced("192.168.1.15"),
			DHCPStatus:    models.NoDHCP(),
		},
	}, Options{
		Service:       "unbound",
		CaddyServerIP: "192.168.1.15",
	})

	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d: %#v", len(actions), actions)
	}

	assertAction(t, actions[0], Action{
		Type:     "add",
		Service:  "unbound",
		Hostname: "missing.example.com",
		NewIP:    "192.168.1.15",
		Enabled:  true,
	})
	assertAction(t, actions[1], Action{
		Type:     "update",
		Service:  "unbound",
		Hostname: "wrong.example.com",
		OldIP:    "10.0.0.99",
		NewIP:    "192.168.1.15",
		Enabled:  true,
	})
}

func TestBuildPlanProducesStableDryRunSnapshot(t *testing.T) {
	plan := BuildPlan([]*models.Entry{
		{
			Hostname:      "missing.example.com",
			CaddyUpstream: "10.0.0.5:8080",
			UnboundStatus: models.NotConfigured(),
			AdguardStatus: models.NotConfigured(),
			DHCPStatus:    models.NoDHCP(),
		},
	}, Options{
		Service:       "all",
		CaddyServerIP: "192.168.1.15",
	})

	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal plan snapshot: %v", err)
	}
	want := `{
  "actions": [
    {
      "type": "add",
      "hostname": "missing.example.com",
      "service": "unbound",
      "old_ip": "",
      "new_ip": "192.168.1.15",
      "details": "",
      "enabled": true
    },
    {
      "type": "add",
      "hostname": "missing.example.com",
      "service": "adguard",
      "old_ip": "",
      "new_ip": "192.168.1.15",
      "details": "",
      "enabled": true
    }
  ]
}`
	if string(data) != want {
		t.Fatalf("unexpected plan snapshot\nwant:\n%s\n got:\n%s", want, string(data))
	}
}

func TestPlanFromEntriesCreatesStaleDeleteActions(t *testing.T) {
	actions := PlanFromEntries([]*models.Entry{
		{
			Hostname:      "stale.example.com",
			UnboundStatus: models.Synced("192.168.1.15"),
			AdguardStatus: models.NotConfigured(),
			DHCPStatus:    models.NoDHCP(),
		},
	}, Options{
		Service:       "all",
		CaddyServerIP: "192.168.1.15",
	})

	if len(actions) != 1 {
		t.Fatalf("expected one stale delete action, got %d: %#v", len(actions), actions)
	}
	assertAction(t, actions[0], Action{
		Type:     "delete",
		Service:  "unbound",
		Hostname: "stale.example.com",
		OldIP:    "192.168.1.15",
		Details:  "no longer in Caddy",
		Enabled:  true,
	})
}

func TestPlanFromEntriesCreatesCloudflareAddUpdateAndDeleteActions(t *testing.T) {
	actions := PlanFromEntries([]*models.Entry{
		{
			Hostname:      "missing.example.com",
			CaddyUpstream: "10.0.0.5:8080",
			CloudflareStatus: models.CloudflareStatus{
				Configured: false,
			},
		},
		{
			Hostname:      "wrong.example.com",
			CaddyUpstream: "10.0.0.6:8080",
			CloudflareStatus: models.CloudflareStatus{
				Configured:      true,
				IsDefaultTunnel: true,
				TunnelID:        "tunnel-default",
				TunnelName:      "default",
				Service:         "http://old-caddy:80",
				HTTPHostHeader:  "",
				Path:            "/api/*",
				NoTLSVerify:     true,
				HasAccessPolicy: true,
			},
		},
		{
			Hostname: "stale.example.com",
			CloudflareStatus: models.CloudflareStatus{
				Configured:      true,
				IsDefaultTunnel: true,
				TunnelID:        "tunnel-default",
				TunnelName:      "default",
				Service:         "http://192.168.1.15:80",
				HTTPHostHeader:  "stale.example.com",
			},
		},
		{
			Hostname:      "other-tunnel.example.com",
			CaddyUpstream: "10.0.0.7:8080",
			CloudflareStatus: models.CloudflareStatus{
				Configured:      true,
				IsDefaultTunnel: false,
				TunnelID:        "tunnel-other",
				TunnelName:      "other",
				Service:         "http://192.168.1.15:80",
				HTTPHostHeader:  "other-tunnel.example.com",
			},
		},
	}, Options{
		Service:         "cloudflare",
		CaddyServiceURL: "http://192.168.1.15:80",
	})

	if len(actions) != 3 {
		t.Fatalf("expected three Cloudflare actions, got %d: %#v", len(actions), actions)
	}
	assertAction(t, actions[0], Action{
		Type:                 "add",
		Service:              "cloudflare",
		Hostname:             "missing.example.com",
		NewService:           "http://192.168.1.15:80",
		NewHTTPHostHeader:    "missing.example.com",
		Details:              "missing in default Cloudflare tunnel",
		Enabled:              true,
		ManagedFields:        "service,http_host_header",
		OriginRequestSummary: "preserve optional origin request fields",
	})
	assertAction(t, actions[1], Action{
		Type:                 "update",
		Service:              "cloudflare",
		Hostname:             "wrong.example.com",
		OldService:           "http://old-caddy:80",
		NewService:           "http://192.168.1.15:80",
		OldHTTPHostHeader:    "",
		NewHTTPHostHeader:    "wrong.example.com",
		TunnelID:             "tunnel-default",
		TunnelName:           "default",
		Path:                 "/api/*",
		NoTLSVerify:          true,
		HasAccessPolicy:      true,
		Details:              "service and host header differ from Caddy",
		Enabled:              true,
		ManagedFields:        "service,http_host_header",
		OriginRequestSummary: "preserve optional origin request fields",
	})
	assertAction(t, actions[2], Action{
		Type:                 "delete",
		Service:              "cloudflare",
		Hostname:             "stale.example.com",
		OldService:           "http://192.168.1.15:80",
		OldHTTPHostHeader:    "stale.example.com",
		TunnelID:             "tunnel-default",
		TunnelName:           "default",
		Details:              "no longer in Caddy",
		Enabled:              true,
		ManagedFields:        "service,http_host_header",
		OriginRequestSummary: "preserve optional origin request fields",
	})
}

func TestPlanFromEntriesDeduplicatesHostnames(t *testing.T) {
	entry := &models.Entry{
		Hostname:      "duplicate.example.com",
		CaddyUpstream: "10.0.0.5:8080",
		UnboundStatus: models.NotConfigured(),
		AdguardStatus: models.NotConfigured(),
		DHCPStatus:    models.NoDHCP(),
	}

	actions := PlanFromEntries([]*models.Entry{entry, entry}, Options{
		Service:       "unbound",
		CaddyServerIP: "192.168.1.15",
	})

	if len(actions) != 1 {
		t.Fatalf("expected one action for duplicate hostname, got %d: %#v", len(actions), actions)
	}
}

func TestPlanFromEntriesCreatesDHCPStaticLeaseActions(t *testing.T) {
	actions := PlanFromEntries([]*models.Entry{
		{
			Hostname:      "device.example.com",
			CaddyUpstream: "10.0.0.5:8080",
			UnboundStatus: models.Synced("192.168.1.15"),
			AdguardStatus: models.Synced("192.168.1.15"),
			DHCPStatus:    models.NewDHCPStatus(true, "dynamic", "10.0.0.5", "aa:bb:cc:dd:ee:ff", "device", true),
		},
	}, Options{
		Service:       "dhcp",
		CaddyServerIP: "192.168.1.15",
	})

	if len(actions) != 1 {
		t.Fatalf("expected one DHCP action, got %d: %#v", len(actions), actions)
	}
	assertAction(t, actions[0], Action{
		Type:     "add",
		Service:  "dhcp",
		Hostname: "device.example.com",
		NewIP:    "10.0.0.5",
		Details:  "static lease (MAC: aa:bb:cc:dd:ee:ff)",
		Enabled:  true,
	})
}

func TestPlanFromEntriesExcludesDHCPFromDefaultAll(t *testing.T) {
	actions := PlanFromEntries([]*models.Entry{
		{
			Hostname:      "device.example.com",
			CaddyUpstream: "10.0.0.5:8080",
			UnboundStatus: models.Synced("192.168.1.15"),
			AdguardStatus: models.Synced("192.168.1.15"),
			DHCPStatus:    models.NewDHCPStatus(true, "dynamic", "10.0.0.5", "aa:bb:cc:dd:ee:ff", "device", true),
		},
	}, Options{
		Service:       "all",
		CaddyServerIP: "192.168.1.15",
	})

	if len(actions) != 0 {
		t.Fatalf("expected default all plan to exclude unimplemented DHCP actions, got %#v", actions)
	}
}

func assertAction(t *testing.T, got Action, want Action) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected action\nwant: %#v\n got: %#v", want, got)
	}
}
