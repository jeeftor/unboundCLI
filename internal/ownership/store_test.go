package ownership

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateRequiresExactResourceIdentity(t *testing.T) {
	state := State{}
	if err := state.Record(Resource{Provider: "adguard", Kind: "rewrite", ID: "app.example.test|10.0.0.5"}); err != nil {
		t.Fatalf("record ownership: %v", err)
	}
	if !state.Owns("adguard", "rewrite", "app.example.test|10.0.0.5") {
		t.Fatal("expected exact resource to be owned")
	}
	if state.Owns("adguard", "rewrite", "app.example.test|10.0.0.6") {
		t.Fatal("matching hostname must not establish ownership of another resource")
	}
}

func TestSaveLoadAndForget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	state := State{}
	resource := Resource{Provider: "cloudflare", Kind: "dns", ID: "zone-1/record-1", Expected: Fingerprint("target.example.test")}
	if err := state.Record(resource); err != nil {
		t.Fatalf("record ownership: %v", err)
	}
	if err := Save(path, state); err != nil {
		t.Fatalf("save ownership state: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat ownership state: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("ownership state mode = %o, want 600", info.Mode().Perm())
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load ownership state: %v", err)
	}
	if !loaded.Owns("cloudflare", "dns", "zone-1/record-1") {
		t.Fatal("saved ownership was not restored")
	}
	loaded.Forget("cloudflare", "dns", "zone-1/record-1")
	if loaded.Owns("cloudflare", "dns", "zone-1/record-1") {
		t.Fatal("forget did not remove ownership")
	}
}

func TestLoadRejectsUnknownStateVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"version":99,"resources":{}}`), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected unknown state version to be rejected")
	}
}
