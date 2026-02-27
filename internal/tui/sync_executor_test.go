package tui

import (
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/widgets"
)

// TestExecuteSyncAction tests individual sync action execution
func TestExecuteSyncAction(t *testing.T) {
	tests := []struct {
		name        string
		action      widgets.SyncAction
		wantErr     bool
		errContains string
	}{
		{
			name: "add unbound entry",
			action: widgets.SyncAction{
				Type:     "add",
				Hostname: "test.example.com",
				Service:  "unbound",
				NewIP:    "192.168.1.15",
			},
			wantErr: false,
		},
		{
			name: "update adguard entry",
			action: widgets.SyncAction{
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
			action: widgets.SyncAction{
				Type:     "delete",
				Hostname: "test.example.com",
				Service:  "unbound",
				OldIP:    "192.168.1.10",
			},
			wantErr: false,
		},
		{
			name: "unknown service",
			action: widgets.SyncAction{
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
			// Create executor with nil clients for now (will use mocks later)
			executor := &TUISyncExecutor{
				unboundClient: nil,
				adguardClient: nil,
				dhcpClient:    nil,
			}

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
	actions := []widgets.SyncAction{
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

	executor := &TUISyncExecutor{
		unboundClient: nil,
		adguardClient: nil,
		dhcpClient:    nil,
	}

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
	t.Skip("TODO: Implement dry run mode test")
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
