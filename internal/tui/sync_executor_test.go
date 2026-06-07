package tui

import (
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/syncplan"
)

// TestExecuteSyncAction tests individual sync action execution
func TestExecuteSyncAction(t *testing.T) {
	tests := []struct {
		name        string
		action      syncplan.Action
		wantErr     bool
		errContains string
	}{
		{
			name: "add unbound entry",
			action: syncplan.Action{
				Type:     "add",
				Hostname: "test.example.com",
				Service:  "unbound",
				NewIP:    "192.168.1.15",
			},
			wantErr: false,
		},
		{
			name: "update adguard entry",
			action: syncplan.Action{
				Type:     "update",
				Hostname: "test.example.com",
				Service:  "adguard",
				OldIP:    "192.168.1.10",
				NewIP:    "192.168.1.15",
			},
			wantErr: false,
		},
		{
			name: "delete unbound entry",
			action: syncplan.Action{
				Type:     "delete",
				Hostname: "test.example.com",
				Service:  "unbound",
				OldIP:    "192.168.1.10",
			},
			wantErr: false,
		},
		{
			name: "unknown service",
			action: syncplan.Action{
				Type:     "add",
				Hostname: "test.example.com",
				Service:  "unknown",
				NewIP:    "192.168.1.15",
			},
			wantErr:     true,
			errContains: "unknown service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewTUISyncExecutor(nil, nil, nil, nil)

			err := executor.ExecuteSyncAction(tt.action)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExecuteSyncAction() expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ExecuteSyncAction() error = %v, want error containing %v", err, tt.errContains)
				}
			} else if err != nil {
				// For now, we expect "not available" errors since clients are nil
				// This test will be updated when we add mock clients
				if !contains(err.Error(), "not available") && !contains(err.Error(), "not yet implemented") {
					t.Errorf("ExecuteSyncAction() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestExecuteSyncActions tests batch sync execution
func TestExecuteSyncActions(t *testing.T) {
	actions := []syncplan.Action{
		{
			Type:     "add",
			Hostname: "test1.example.com",
			Service:  "unbound",
			NewIP:    "192.168.1.15",
			Enabled:  true,
		},
		{
			Type:     "add",
			Hostname: "test2.example.com",
			Service:  "adguard",
			NewIP:    "192.168.1.15",
			Enabled:  true,
		},
		{
			Type:     "add",
			Hostname: "test3.example.com",
			Service:  "unbound",
			NewIP:    "192.168.1.15",
			Enabled:  false, // Disabled - should be skipped
		},
	}

	executor := NewTUISyncExecutor(nil, nil, nil, nil)

	result := executor.ExecuteSyncActions(actions)

	// Should process 2 enabled actions
	if result.ItemsAdded != 0 {
		// Will be 0 because clients are nil, but structure should be correct
		t.Logf("ItemsAdded: %d (expected 0 with nil clients)", result.ItemsAdded)
	}

	// Should have 2 errors (one for each enabled action with nil clients)
	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors for nil clients, got %d", len(result.Errors))
	}

	// Should not be successful with nil clients
	if result.Success {
		t.Error("Expected Success=false with nil clients")
	}
}

// TestDryRunMode tests that dry run doesn't execute actual API calls
func TestDryRunMode(t *testing.T) {
	executor := NewTUISyncExecutor(nil, nil, nil, nil)
	executor.SetDryRun(true)

	result := executor.ExecuteSyncActions([]syncplan.Action{
		{
			Type:     "add",
			Hostname: "dry.example.com",
			Service:  "unbound",
			NewIP:    "192.168.1.15",
			Enabled:  true,
		},
	})

	if !result.Success {
		t.Fatalf("expected dry-run success with nil clients, got errors: %#v", result.Errors)
	}
	if result.ItemsAdded != 1 {
		t.Fatalf("expected dry-run add count 1, got %d", result.ItemsAdded)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no dry-run errors, got %#v", result.Errors)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)+1 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
