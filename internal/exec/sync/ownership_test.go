package sync

import (
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
)

func TestIsLegacyDescription(t *testing.T) {
	legacy := []string{"old desc 1", "old desc 2"}
	tests := []struct {
		desc string
		want bool
	}{
		{"old desc 1", true},
		{"old desc 2", true},
		{"current desc", false},
		{"", false},
	}
	for _, tt := range tests {
		got := IsLegacyDescription(tt.desc, legacy)
		if got != tt.want {
			t.Errorf("IsLegacyDescription(%q) = %v, want %v", tt.desc, got, tt.want)
		}
	}
}

func TestIsLegacyDescription_EmptyList(t *testing.T) {
	if IsLegacyDescription("anything", nil) {
		t.Error("expected false for nil legacy list")
	}
	if IsLegacyDescription("anything", []string{}) {
		t.Error("expected false for empty legacy list")
	}
}

func TestIsSyncOwnedOverride(t *testing.T) {
	entryDesc := "Managed by caddy-dns-sync"
	legacy := []string{"Entry created by unboundCLI"}

	tests := []struct {
		name     string
		override api.DNSOverride
		want     bool
	}{
		{
			name:     "matches current description",
			override: api.DNSOverride{Description: "Managed by caddy-dns-sync"},
			want:     true,
		},
		{
			name:     "matches legacy description",
			override: api.DNSOverride{Description: "Entry created by unboundCLI"},
			want:     true,
		},
		{
			name:     "foreign description",
			override: api.DNSOverride{Description: "Manually configured"},
			want:     false,
		},
		{
			name:     "empty description",
			override: api.DNSOverride{Description: ""},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSyncOwnedOverride(tt.override, entryDesc, legacy)
			if got != tt.want {
				t.Errorf("IsSyncOwnedOverride() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrganizeOverridesByOwnership(t *testing.T) {
	entryDesc := "Managed by caddy-dns-sync"
	legacy := []string{"Entry created by unboundCLI"}

	overrides := []api.DNSOverride{
		{UUID: "1", Host: "app", Domain: "example.com", Description: "Managed by caddy-dns-sync"},
		{UUID: "2", Host: "wiki", Domain: "example.com", Description: "Entry created by unboundCLI"},
		{UUID: "3", Host: "manual", Domain: "example.com", Description: "Manually configured"},
		{UUID: "4", Host: "grafana", Domain: "example.com", Description: ""},
	}

	syncCreated, other := OrganizeOverridesByOwnership(overrides, entryDesc, legacy)

	if len(syncCreated) != 2 {
		t.Fatalf("expected 2 sync-created overrides, got %d", len(syncCreated))
	}
	if len(other) != 2 {
		t.Fatalf("expected 2 other overrides, got %d", len(other))
	}

	if _, ok := syncCreated["app.example.com"]; !ok {
		t.Error("expected app.example.com in syncCreated")
	}
	if _, ok := syncCreated["wiki.example.com"]; !ok {
		t.Error("expected wiki.example.com in syncCreated (legacy desc)")
	}
	if _, ok := other["manual.example.com"]; !ok {
		t.Error("expected manual.example.com in other")
	}
	if _, ok := other["grafana.example.com"]; !ok {
		t.Error("expected grafana.example.com in other (empty desc)")
	}
}

func TestOrganizeOverridesByOwnership_Empty(t *testing.T) {
	syncCreated, other := OrganizeOverridesByOwnership(nil, "desc", nil)
	if len(syncCreated) != 0 || len(other) != 0 {
		t.Fatal("expected empty maps for nil input")
	}
}

func TestSplitHostname(t *testing.T) {
	tests := []struct {
		hostname string
		host     string
		domain   string
	}{
		{"app.example.com", "app", "example.com"},
		{"wiki.sub.example.com", "wiki", "sub.example.com"},
		{"localhost", "localhost", ""},
		{"", "", ""},
		{".example.com", "", "example.com"},
	}

	for _, tt := range tests {
		host, domain := SplitHostname(tt.hostname)
		if host != tt.host || domain != tt.domain {
			t.Errorf("SplitHostname(%q) = (%q, %q), want (%q, %q)",
				tt.hostname, host, domain, tt.host, tt.domain)
		}
	}
}

func TestIsFQDN(t *testing.T) {
	tests := []struct {
		hostname string
		want     bool
	}{
		{"app.example.com", true},
		{"localhost", false},
		{"", false},
		{".", true},
		{"app.", true},
	}

	for _, tt := range tests {
		got := IsFQDN(tt.hostname)
		if got != tt.want {
			t.Errorf("IsFQDN(%q) = %v, want %v", tt.hostname, got, tt.want)
		}
	}
}
