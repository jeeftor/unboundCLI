package caddyeditor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		data         TemplateData
		wantContains []string
		wantErr      bool
	}{
		{
			name:         "default template",
			templateName: "default",
			data: TemplateData{
				Hostname: "sonarr.vookie.net",
				Upstream: "http://10.0.0.112:8989",
			},
			wantContains: []string{
				"@sonarr_vookie_net host sonarr.vookie.net",
				"handle @sonarr_vookie_net {",
				"reverse_proxy http://10.0.0.112:8989",
				"import proxy_headers",
			},
		},
		{
			name:         "simple template",
			templateName: "simple",
			data: TemplateData{
				Hostname: "radarr.vookie.net",
				Upstream: "http://10.0.0.112:7878",
			},
			wantContains: []string{
				"@radarr_vookie_net host radarr.vookie.net",
				"handle @radarr_vookie_net {",
				"reverse_proxy http://10.0.0.112:7878",
			},
		},
		{
			name:         "no-tls-verify template",
			templateName: "no-tls-verify",
			data: TemplateData{
				Hostname: "plex.vookie.net",
				Upstream: "https://10.0.0.112:32400",
			},
			wantContains: []string{
				"tls_insecure_skip_verify",
				"reverse_proxy https://10.0.0.112:32400",
			},
		},
		{
			name:         "compression template",
			templateName: "compression",
			data: TemplateData{
				Hostname: "grafana.vookie.net",
				Upstream: "http://10.0.0.112:3000",
			},
			wantContains: []string{
				"encode gzip zstd",
				"reverse_proxy http://10.0.0.112:3000",
			},
		},
		{
			name:         "long-timeout template",
			templateName: "long-timeout",
			data: TemplateData{
				Hostname: "ai.vookie.net",
				Upstream: "http://10.0.0.112:8080",
			},
			wantContains: []string{
				"read_timeout 600s",
				"write_timeout 600s",
				"flush_interval -1",
			},
		},
		{
			name:         "headers-inline template preserves caddy placeholders",
			templateName: "headers-inline",
			data: TemplateData{
				Hostname: "svc.vookie.net",
				Upstream: "http://10.0.0.112:1234",
			},
			wantContains: []string{
				"header_up Host {upstream_hostport}",
				"header_up X-Real-IP {remote_host}",
			},
		},
		{
			name:         "explicit matcher name is respected",
			templateName: "simple",
			data: TemplateData{
				Hostname:    "custom.vookie.net",
				Upstream:    "http://127.0.0.1:8080",
				MatcherName: "myname",
			},
			wantContains: []string{
				"@myname host custom.vookie.net",
				"handle @myname {",
			},
		},
		{
			name:         "unknown template errors",
			templateName: "does-not-exist",
			data: TemplateData{
				Hostname: "x.vookie.net",
				Upstream: "http://127.0.0.1:1",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := RenderTemplate("", tc.templateName, tc.data)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("RenderTemplate: %v", err)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q\noutput:\n%s", want, out)
				}
			}
		})
	}
}

func TestRenderTemplateForwardAuthParams(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]string
		wantContains []string
		wantAbsent   []string
	}{
		{
			name: "uses provided authentik_url",
			params: map[string]string{
				"authentik_url": "10.0.0.112:9000",
			},
			wantContains: []string{
				"reverse_proxy /outpost.goauthentik.io/* 10.0.0.112:9000",
				"forward_auth 10.0.0.112:9000 {",
			},
		},
		{
			name:   "falls back to default authentik_url",
			params: nil,
			wantContains: []string{
				"AUTHENTIK_HOST:PORT",
			},
		},
		{
			name: "no split-horizon matchers",
			params: map[string]string{
				"authentik_url": "10.0.0.112:9000",
			},
			wantContains: []string{
				"forward_auth 10.0.0.112:9000 {",
			},
			wantAbsent: []string{
				"not client_ip",
				"_external",
				"Cf-Connecting-Ip",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := RenderTemplate("", "forward-auth", TemplateData{
				Hostname: "app.vookie.net",
				Upstream: "http://10.0.0.112:8080",
				Params:   tc.params,
			})
			if err != nil {
				t.Fatalf("RenderTemplate: %v", err)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q\noutput:\n%s", want, out)
				}
			}
			for _, absent := range tc.wantAbsent {
				if strings.Contains(out, absent) {
					t.Errorf("output should not contain %q\noutput:\n%s", absent, out)
				}
			}
		})
	}
}

func TestListTemplates(t *testing.T) {
	// Without a repo path, only builtins are returned.
	names := ListTemplates("")
	if len(names) == 0 {
		t.Fatal("expected at least one builtin template")
	}

	wantBuiltins := []string{"default", "simple", "no-tls-verify", "compression", "long-timeout", "forward-auth"}
	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	for _, w := range wantBuiltins {
		if !nameSet[w] {
			t.Errorf("expected builtin template %q in list", w)
		}
	}
}

func TestListTemplatesWithCustom(t *testing.T) {
	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	custom := `@[[ .MatcherName ]] host [[ .Hostname ]]
handle @[[ .MatcherName ]] {
	reverse_proxy [[ .Upstream ]]
}`
	if err := os.WriteFile(filepath.Join(tmplDir, "mycustom.caddytemplate"), []byte(custom), 0o644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}

	names := ListTemplates(dir)
	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["mycustom"] {
		t.Errorf("expected custom template %q in list: %v", "mycustom", names)
	}
	// Builtins should still be present.
	if !nameSet["default"] {
		t.Error("expected default builtin still present alongside custom")
	}
}

func TestRenderTemplateCustomFromRepo(t *testing.T) {
	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	custom := `@[[ .MatcherName ]] host [[ .Hostname ]]
handle @[[ .MatcherName ]] {
	reverse_proxy [[ .Upstream ]]
}`
	if err := os.WriteFile(filepath.Join(tmplDir, "mycustom.caddytemplate"), []byte(custom), 0o644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}

	out, err := RenderTemplate(dir, "mycustom", TemplateData{
		Hostname: "svc.vookie.net",
		Upstream: "http://127.0.0.1:9000",
	})
	if err != nil {
		t.Fatalf("RenderTemplate custom: %v", err)
	}
	if !strings.Contains(out, "handle @svc_vookie_net {") {
		t.Errorf("custom template output unexpected:\n%s", out)
	}
}
