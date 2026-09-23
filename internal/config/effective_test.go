package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEffectiveConfigUsesSelectedFileAndPreferredEnvironment(t *testing.T) {
	clearConfigEnvVars(t)
	path := filepath.Join(t.TempDir(), "selected.json")
	if err := os.WriteFile(path, []byte(`{"api_key":"file-key","api_secret":"file-secret","base_url":"https://file.example.test","unknown_root":{"kept":true}}`), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	t.Setenv(EnvAPIKeyDeprecated, "legacy-key")
	t.Setenv(EnvAPIKey, "preferred-key")

	cfg, resolvedPath, err := LoadEffectiveConfig(path)
	if err != nil {
		t.Fatalf("load effective config: %v", err)
	}
	if resolvedPath != path || cfg.APIKey != "preferred-key" || cfg.APISecret != "file-secret" || cfg.BaseURL != "https://file.example.test" {
		t.Fatalf("unexpected resolved config: path=%q cfg=%#v", resolvedPath, cfg.Config)
	}
}

func TestSaveExtendedConfigPreservesUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	original := `{
  "api_key":"old-key",
  "api_secret":"old-secret",
  "base_url":"https://old.example.test",
  "future_root":{"enabled":true},
  "cloudflare":{"enabled":true,"api_token":"old-token","future_field":"keep-me"}
}`
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	if err := SaveExtendedConfig(ExtendedConfig{Cloudflare: CloudflareConfig{Enabled: true, APIToken: "new-token"}}, path); err != nil {
		t.Fatalf("save config: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	for _, want := range []string{`"future_root"`, `"future_field"`, `"keep-me"`, `"new-token"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("saved config lost %s: %s", want, data)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 config permissions, got %o", info.Mode().Perm())
	}
}
