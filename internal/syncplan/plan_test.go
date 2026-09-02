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
			AdguardStatus: models.Synced("10.0.0.15"),
			DHCPStatus:    models.NoDHCP(),
		},
	}, Options{
		Service:       "unbound",
		CaddyServerIP: "10.0.0.15",
	})

	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d: %#v", len(actions), actions)
	}

	assertAction(t, actions[0], Action{
		Type:     "add",
		Service:  "unbound",
		Hostname: "missing.example.com",
		NewIP:    "10.0.0.15",
		Enabled:  true,
	})
	assertAction(t, actions[1], Action{
		Type:     "update",
		Service:  "unbound",
		Hostname: "wrong.example.com",
		OldIP:    "10.0.0.99",
		NewIP:    "10.0.0.15",
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
		CaddyServerIP: "10.0.0.15",
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
      "new_ip": "10.0.0.15",
      "details": "",
      "enabled": true
    },
    {
      "type": "add",
      "hostname": "missing.example.com",
      "service": "adguard",
      "old_ip": "",
      "new_ip": "10.0.0.15",
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
			UnboundStatus: models.Synced("10.0.0.15"),
			AdguardStatus: models.NotConfigured(),
			DHCPStatus:    models.NoDHCP(),
		},
	}, Options{
		Service:       "all",
		CaddyServerIP: "10.0.0.15",
	})

	if len(actions) != 1 {
		t.Fatalf("expected one stale delete action, got %d: %#v", len(actions), actions)
	}
	assertAction(t, actions[0], Action{
		Type:     "delete",
		Service:  "unbound",
		Hostname: "stale.example.com",
		OldIP:    "10.0.0.15",
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
				Service:         "http://10.0.0.15:80",
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
				Service:         "http://10.0.0.15:80",
				HTTPHostHeader:  "other-tunnel.example.com",
			},
		},
	}, Options{
		Service:         "cloudflare",
		CaddyServiceURL: "http://10.0.0.15:80",
	})

	if len(actions) != 3 {
		t.Fatalf("expected three Cloudflare actions, got %d: %#v", len(actions), actions)
	}
	assertAction(t, actions[0], Action{
		Type:                 "add",
		Service:              "cloudflare",
		Hostname:             "missing.example.com",
		NewService:           "https://10.0.0.15",
		NewHTTPHostHeader:    "missing.example.com",
		OriginServerName:     "missing.example.com",
		Details:              "missing in default Cloudflare tunnel",
		Enabled:              true,
		ManagedFields:        "service,http_host_header,origin_server_name",
		OriginRequestSummary: "preserve optional origin request fields",
	})
	assertAction(t, actions[1], Action{
		Type:                 "update",
		Service:              "cloudflare",
		Hostname:             "wrong.example.com",
		OldService:           "http://old-caddy:80",
		NewService:           "https://10.0.0.15",
		OldHTTPHostHeader:    "",
		NewHTTPHostHeader:    "wrong.example.com",
		TunnelID:             "tunnel-default",
		TunnelName:           "default",
		Path:                 "/api/*",
		NoTLSVerify:          false,
		OriginServerName:     "wrong.example.com",
		HasAccessPolicy:      true,
		Details:              "service and host header differ from Caddy",
		Enabled:              true,
		ManagedFields:        "service,http_host_header,origin_server_name",
		OriginRequestSummary: "preserve optional origin request fields",
	})
	assertAction(t, actions[2], Action{
		Type:                 "delete",
		Service:              "cloudflare",
		Hostname:             "stale.example.com",
		OldService:           "http://10.0.0.15:80",
		OldHTTPHostHeader:    "stale.example.com",
		TunnelID:             "tunnel-default",
		TunnelName:           "default",
		Details:              "no longer in Caddy",
		Enabled:              true,
		ManagedFields:        "service,http_host_header,origin_server_name",
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
		CaddyServerIP: "10.0.0.15",
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
			UnboundStatus: models.Synced("10.0.0.15"),
			AdguardStatus: models.Synced("10.0.0.15"),
			DHCPStatus:    models.NewDHCPStatus(true, "dynamic", "10.0.0.5", "aa:bb:cc:dd:ee:ff", "device", true),
		},
	}, Options{
		Service:       "dhcp",
		CaddyServerIP: "10.0.0.15",
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
			UnboundStatus: models.Synced("10.0.0.15"),
			AdguardStatus: models.Synced("10.0.0.15"),
			DHCPStatus:    models.NewDHCPStatus(true, "dynamic", "10.0.0.5", "aa:bb:cc:dd:ee:ff", "device", true),
		},
	}, Options{
		Service:       "all",
		CaddyServerIP: "10.0.0.15",
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

func TestBuildCloudflareActionDirectMode(t *testing.T) {
	entry := &models.Entry{
		Hostname:      "direct.example.com",
		CaddyUpstream: "10.0.0.5:8080",
		CloudflareStatus: models.CloudflareStatus{
			Configured: false,
		},
	}
	action := buildCloudflareAction(entry, Options{
		OriginMode:    "direct",
		CaddyServerIP: "10.0.0.15",
	})
	if action.Type != "add" {
		t.Fatalf("expected add, got %q", action.Type)
	}
	if action.NewService != "http://10.0.0.5:8080" {
		t.Fatalf("expected direct service URL http://10.0.0.5:8080, got %q", action.NewService)
	}
	if action.OriginServerName != "" {
		t.Fatalf("expected no OriginServerName in direct mode, got %q", action.OriginServerName)
	}
}

func TestBuildCloudflareActionSkipNonDefaultTunnel(t *testing.T) {
	entry := &models.Entry{
		Hostname:      "other.example.com",
		CaddyUpstream: "10.0.0.5:8080",
		CloudflareStatus: models.CloudflareStatus{
			Configured:      true,
			IsDefaultTunnel: false,
			TunnelID:        "tunnel-other",
			Service:         "https://10.0.0.15",
			HTTPHostHeader:  "other.example.com",
		},
	}
	action := buildCloudflareAction(entry, Options{
		CaddyServerIP: "10.0.0.15",
	})
	if action.Type != "" {
		t.Fatalf("expected empty action for non-default tunnel, got %q", action.Type)
	}
}

func TestBuildCloudflareActionOverrideTunnel(t *testing.T) {
	entry := &models.Entry{
		Hostname:      "override.example.com",
		CaddyUpstream: "10.0.0.5:8080",
		CloudflareStatus: models.CloudflareStatus{
			Configured:      true,
			IsDefaultTunnel: false,
			TunnelID:        "tunnel-other",
			Service:         "https://10.0.0.15",
			HTTPHostHeader:  "override.example.com",
		},
	}
	action := buildCloudflareAction(entry, Options{
		CaddyServerIP:    "10.0.0.15",
		OverrideTunnelID: "tunnel-target",
	})
	if action.Type != "" {
		// Should still skip — override only applies when the action would be generated
		t.Fatalf("expected empty action even with override for non-default tunnel, got %q", action.Type)
	}
}

func TestBuildCloudflareActionTLSVerifyChange(t *testing.T) {
	entry := &models.Entry{
		Hostname:      "tls.example.com",
		CaddyUpstream: "10.0.0.5:8080",
		CloudflareStatus: models.CloudflareStatus{
			Configured:      true,
			IsDefaultTunnel: true,
			TunnelID:        "tunnel-default",
			TunnelName:      "default",
			Service:         "https://10.0.0.15",
			HTTPHostHeader:  "tls.example.com",
			NoTLSVerify:     false,
		},
	}
	action := buildCloudflareAction(entry, Options{
		CaddyServerIP: "10.0.0.15",
		NoTLSVerify:   true,
	})
	if action.Type != "update" {
		t.Fatalf("expected update for TLS verify change, got %q", action.Type)
	}
	if action.Details != "TLS verify setting changed" {
		t.Fatalf("expected TLS verify detail, got %q", action.Details)
	}
}

func TestBuildCloudflareActionNoChangeReturnsEmpty(t *testing.T) {
	entry := &models.Entry{
		Hostname:      "ok.example.com",
		CaddyUpstream: "10.0.0.5:8080",
		CloudflareStatus: models.CloudflareStatus{
			Configured:      true,
			IsDefaultTunnel: true,
			TunnelID:        "tunnel-default",
			TunnelName:      "default",
			Service:         "https://10.0.0.15",
			HTTPHostHeader:  "ok.example.com",
		},
	}
	action := buildCloudflareAction(entry, Options{
		CaddyServerIP: "10.0.0.15",
	})
	if action.Type != "" {
		t.Fatalf("expected empty action when everything matches, got %q", action.Type)
	}
}

func TestBuildCloudflareActionStaleDeleteNonDefaultTunnel(t *testing.T) {
	entry := &models.Entry{
		Hostname: "stale-other.example.com",
		CloudflareStatus: models.CloudflareStatus{
			Configured:      true,
			IsDefaultTunnel: false,
			TunnelID:        "tunnel-other",
			TunnelName:      "other",
			Service:         "https://10.0.0.15",
			HTTPHostHeader:  "stale-other.example.com",
		},
	}
	action := buildCloudflareAction(entry, Options{
		CaddyServerIP: "10.0.0.15",
	})
	if action.Type != "" {
		t.Fatalf("expected empty action for stale entry on non-default tunnel, got %q", action.Type)
	}
}
