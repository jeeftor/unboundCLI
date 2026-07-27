package caddyeditor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sampleCaddyfile is a representative wildcard+matcher Caddyfile used across
// the parser tests. It contains two existing entries and a fallback handle.
const sampleCaddyfile = `*.vookie.net {
	@sonarr host sonarr.vookie.net
	handle @sonarr {
		reverse_proxy http://10.0.0.112:8989
	}

	@radarr host radarr.vookie.net
	handle @radarr {
		reverse_proxy http://10.0.0.112:7878
	}

	handle {
		respond "not found" 404
	}
}
`

// emptyCaddyfile has a wildcard block with only a fallback handle — no entries.
const emptyCaddyfile = `*.vookie.net {
	handle {
		respond "not found" 404
	}
}
`

// writeCaddyfile writes content into a Caddyfile inside a temp repo directory
// and returns an EditorConfig pointing at it.
func writeCaddyfile(t *testing.T, content string) EditorConfig {
	t.Helper()
	dir := t.TempDir()
	caddyDir := filepath.Join(dir, "caddy")
	if err := os.MkdirAll(caddyDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(caddyDir, "Caddyfile")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write caddyfile: %v", err)
	}
	return EditorConfig{
		RepoPath:      dir,
		CaddyfilePath: "caddy/Caddyfile",
	}
}

func TestAbsCaddyfilePath(t *testing.T) {
	tests := []struct {
		name string
		cfg  EditorConfig
		want string
	}{
		{
			name: "relative path joined to repo",
			cfg:  EditorConfig{RepoPath: "/tmp/repo", CaddyfilePath: "caddy/Caddyfile"},
			want: "/tmp/repo/caddy/Caddyfile",
		},
		{
			name: "absolute path preserved",
			cfg:  EditorConfig{RepoPath: "/tmp/repo", CaddyfilePath: "/etc/caddy/Caddyfile"},
			want: "/etc/caddy/Caddyfile",
		},
		{
			name: "empty repo path still joins",
			cfg:  EditorConfig{RepoPath: "", CaddyfilePath: "caddy/Caddyfile"},
			want: "caddy/Caddyfile",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AbsCaddyfilePath(tc.cfg)
			if got != tc.want {
				t.Fatalf("AbsCaddyfilePath = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseCaddyfile(t *testing.T) {
	cfg := writeCaddyfile(t, sampleCaddyfile)
	blocks, err := ParseCaddyfile(AbsCaddyfilePath(cfg))
	if err != nil {
		t.Fatalf("ParseCaddyfile: %v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("expected 2 site blocks, got %d", len(blocks))
	}

	tests := []struct {
		name        string
		hostname    string
		matcher     string
		upstream    string
		wantInRaw   string
		wantInDirect []string
	}{
		{
			name:        "sonarr entry",
			hostname:    "sonarr.vookie.net",
			matcher:     "sonarr",
			upstream:    "http://10.0.0.112:8989",
			wantInRaw:   "reverse_proxy http://10.0.0.112:8989",
			wantInDirect: nil,
		},
		{
			name:        "radarr entry",
			hostname:    "radarr.vookie.net",
			matcher:     "radarr",
			upstream:    "http://10.0.0.112:7878",
			wantInRaw:   "reverse_proxy http://10.0.0.112:7878",
			wantInDirect: nil,
		},
	}

	blockByMatcher := map[string]SiteBlock{}
	for _, b := range blocks {
		blockByMatcher[b.MatcherName] = b
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, ok := blockByMatcher[tc.matcher]
			if !ok {
				t.Fatalf("matcher %q not found", tc.matcher)
			}
			if b.Hostname != tc.hostname {
				t.Errorf("Hostname = %q, want %q", b.Hostname, tc.hostname)
			}
			if b.Upstream != tc.upstream {
				t.Errorf("Upstream = %q, want %q", b.Upstream, tc.upstream)
			}
			if b.SourceFile == "" {
				t.Error("SourceFile should be set")
			}
			if b.LineStart == 0 || b.LineEnd == 0 {
				t.Errorf("LineStart/LineEnd should be set: start=%d end=%d", b.LineStart, b.LineEnd)
			}
			if !strings.Contains(b.Raw, tc.wantInRaw) {
				t.Errorf("Raw does not contain %q\nRaw:\n%s", tc.wantInRaw, b.Raw)
			}
		})
	}
}

func TestParseCaddyfileEmpty(t *testing.T) {
	cfg := writeCaddyfile(t, emptyCaddyfile)
	blocks, err := ParseCaddyfile(AbsCaddyfilePath(cfg))
	if err != nil {
		t.Fatalf("ParseCaddyfile: %v", err)
	}
	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks for empty caddyfile, got %d", len(blocks))
	}
}

func TestParseCaddyfileMissingFile(t *testing.T) {
	_, err := ParseCaddyfile(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseCaddyfileFromConfig(t *testing.T) {
	cfg := writeCaddyfile(t, sampleCaddyfile)
	blocks, err := ParseCaddyfileFromConfig(cfg)
	if err != nil {
		t.Fatalf("ParseCaddyfileFromConfig: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
}

func TestAddEntry(t *testing.T) {
	tests := []struct {
		name         string
		existing     string
		block        SiteBlock
		templateName string
		wantHostname string
		wantUpstream string
		wantErr      bool
	}{
		{
			name:     "add new entry before fallback",
			existing: sampleCaddyfile,
			block: SiteBlock{
				Hostname: "prowarr.vookie.net",
				Upstream: "http://10.0.0.112:9696",
			},
			templateName: "simple",
			wantHostname: "prowarr.vookie.net",
			wantUpstream: "http://10.0.0.112:9696",
		},
		{
			name:     "add entry to empty wildcard block",
			existing: emptyCaddyfile,
			block: SiteBlock{
				Hostname: "sonarr.vookie.net",
				Upstream: "http://10.0.0.112:8989",
			},
			templateName: "simple",
			wantHostname: "sonarr.vookie.net",
			wantUpstream: "http://10.0.0.112:8989",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := writeCaddyfile(t, tc.existing)
			err := AddEntry(cfg, tc.block, tc.templateName)
			if err != nil {
				t.Fatalf("AddEntry: %v", err)
			}

			// Verify the entry was persisted and is parseable.
			blocks, err := ParseCaddyfile(AbsCaddyfilePath(cfg))
			if err != nil {
				t.Fatalf("re-parse: %v", err)
			}

			var found *SiteBlock
			for i := range blocks {
				if blocks[i].Hostname == tc.wantHostname {
					found = &blocks[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("entry %q not found after add", tc.wantHostname)
			}
			if found.Upstream != tc.wantUpstream {
				t.Errorf("Upstream = %q, want %q", found.Upstream, tc.wantUpstream)
			}
			if found.MatcherName == "" {
				t.Error("MatcherName should be derived and non-empty")
			}
		})
	}
}

func TestAddEntryDuplicate(t *testing.T) {
	cfg := writeCaddyfile(t, emptyCaddyfile)
	block := SiteBlock{
		Hostname: "sonarr.vookie.net",
		Upstream: "http://10.0.0.112:8989",
	}
	if err := AddEntry(cfg, block, "simple"); err != nil {
		t.Fatalf("first AddEntry: %v", err)
	}
	// Adding the same hostname again should be detected as a duplicate because
	// the derived matcher name (sonarr_vookie_net) now exists in the file.
	if err := AddEntry(cfg, block, "simple"); err == nil {
		t.Fatal("expected duplicate error on second AddEntry, got nil")
	}
}

func TestRemoveEntry(t *testing.T) {
	tests := []struct {
		name        string
		existing    string
		hostname    string
		wantErr     bool
		wantRemaining int
	}{
		{
			name:         "remove existing sonarr entry",
			existing:     sampleCaddyfile,
			hostname:     "sonarr.vookie.net",
			wantRemaining: 1, // radarr remains
		},
		{
			name:         "remove existing radarr entry",
			existing:     sampleCaddyfile,
			hostname:     "radarr.vookie.net",
			wantRemaining: 1, // sonarr remains
		},
		{
			name:    "remove non-existent entry errors",
			existing: sampleCaddyfile,
			hostname: "nope.vookie.net",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := writeCaddyfile(t, tc.existing)
			err := RemoveEntry(cfg, tc.hostname)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("RemoveEntry: %v", err)
			}

			blocks, err := ParseCaddyfile(AbsCaddyfilePath(cfg))
			if err != nil {
				t.Fatalf("re-parse: %v", err)
			}
			if len(blocks) != tc.wantRemaining {
				t.Fatalf("expected %d remaining blocks, got %d", tc.wantRemaining, len(blocks))
			}
			for _, b := range blocks {
				if b.Hostname == tc.hostname {
					t.Errorf("entry %q still present after removal", tc.hostname)
				}
			}
		})
	}
}

func TestUpdateEntry(t *testing.T) {
	tests := []struct {
		name         string
		existing     string
		hostname     string
		newUpstream  string
		templateName string
		wantErr      bool
	}{
		{
			name:         "update sonarr upstream",
			existing:     sampleCaddyfile,
			hostname:     "sonarr.vookie.net",
			newUpstream:  "http://10.0.0.112:9999",
			templateName: "simple",
		},
		{
			name:         "update radarr upstream",
			existing:     sampleCaddyfile,
			hostname:     "radarr.vookie.net",
			newUpstream:  "http://10.0.0.5:7878",
			templateName: "simple",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := writeCaddyfile(t, tc.existing)

			// Look up the existing block to preserve its hostname.
			before, err := ParseCaddyfile(AbsCaddyfilePath(cfg))
			if err != nil {
				t.Fatalf("parse before: %v", err)
			}
			var orig *SiteBlock
			for i := range before {
				if before[i].Hostname == tc.hostname {
					orig = &before[i]
					break
				}
			}
			if orig == nil {
				t.Fatalf("entry %q not found before update", tc.hostname)
			}

			block := SiteBlock{Hostname: tc.hostname, Upstream: orig.Upstream}
			data := TemplateData{Upstream: tc.newUpstream}
			if err := UpdateEntry(cfg, block, tc.templateName, data); err != nil {
				t.Fatalf("UpdateEntry: %v", err)
			}

			after, err := ParseCaddyfile(AbsCaddyfilePath(cfg))
			if err != nil {
				t.Fatalf("parse after: %v", err)
			}
			var updated *SiteBlock
			for i := range after {
				if after[i].Hostname == tc.hostname {
					updated = &after[i]
					break
				}
			}
			if updated == nil {
				t.Fatalf("entry %q missing after update", tc.hostname)
			}
			if updated.Upstream != tc.newUpstream {
				t.Errorf("Upstream = %q, want %q", updated.Upstream, tc.newUpstream)
			}
			// Count should be unchanged.
			if len(after) != len(before) {
				t.Errorf("block count changed: before=%d after=%d", len(before), len(after))
			}
		})
	}
}
