package cmd

import (
	"strings"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
)

func TestManagedOverrideForDelete(t *testing.T) {
	overrides := []api.DNSOverride{
		{UUID: "managed", Description: app.CurrentUnboundDescription},
		{UUID: "legacy", Description: app.LegacyUnboundDescriptions[0]},
		{UUID: "manual", Description: "Created manually"},
	}

	for _, uuid := range []string{"managed", "legacy"} {
		override, err := managedOverrideForDelete(overrides, uuid)
		if err != nil {
			t.Fatalf("managedOverrideForDelete(%q): %v", uuid, err)
		}
		if override.UUID != uuid {
			t.Fatalf("managedOverrideForDelete(%q) returned %q", uuid, override.UUID)
		}
	}

	for _, uuid := range []string{"manual", "missing"} {
		if _, err := managedOverrideForDelete(overrides, uuid); err == nil {
			t.Fatalf("managedOverrideForDelete(%q) unexpectedly succeeded", uuid)
		} else if uuid == "manual" && !strings.Contains(err.Error(), "unowned") {
			t.Fatalf("manual override error %q does not explain ownership", err)
		}
	}
}

func TestManagedOverrideForEdit(t *testing.T) {
	overrides := []api.DNSOverride{
		{UUID: "managed", Description: app.CurrentUnboundDescription},
		{UUID: "manual", Description: "Created manually"},
	}

	if override, err := managedOverrideForEdit(overrides, "managed"); err != nil || override.UUID != "managed" {
		t.Fatalf("managedOverrideForEdit(managed) = %#v, %v", override, err)
	}
	if _, err := managedOverrideForEdit(overrides, "manual"); err == nil || !strings.Contains(err.Error(), "unowned") {
		t.Fatalf("manual override must be rejected, got %v", err)
	}
}
