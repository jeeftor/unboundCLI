package auth

import (
	"strings"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

func TestClassifyAuth(t *testing.T) {
	tests := []struct {
		name             string
		ha               *models.HostAuth
		authentikHost    string
		wantWANAuth      models.WANAuthMode
		wantLANAuth      models.LANAuthMode
		wantAPIAuth      models.APIAuthMode
		wantStatus       models.AuthStatus
		wantNoteContains string // substring to find in notes (empty = don't check)
	}{
		{
			name: "Authentik IdP itself with bypass — OK",
			ha: &models.HostAuth{
				Hostname:          "auth.vookie.net",
				WANExposed:        true,
				CFAccessAppID:     "app-1",
				CFAccessDecisions: []string{"bypass"},
				HasForwardAuth:    false,
			},
			authentikHost:    "auth.vookie.net",
			wantWANAuth:      models.WANAuthAppNative,
			wantLANAuth:      models.LANAuthNone,
			wantAPIAuth:      models.APIAuthNone,
			wantStatus:       models.AuthStatusOK,
			wantNoteContains: "Authentik identity provider",
		},
		{
			name: "CF Access bypass-only no forward_auth — CRITICAL open",
			ha: &models.HostAuth{
				Hostname:          "open.vookie.net",
				WANExposed:        true,
				CFAccessAppID:     "app-2",
				CFAccessDecisions: []string{"bypass"},
				HasForwardAuth:    false,
			},
			wantWANAuth:      models.WANAuthNone,
			wantLANAuth:      models.LANAuthNone,
			wantAPIAuth:      models.APIAuthNone,
			wantStatus:       models.AuthStatusError,
			wantNoteContains: "CRITICAL",
		},
		{
			name: "CF Access + forward_auth no bypass — double login error",
			ha: &models.HostAuth{
				Hostname:          "jellyfin.vookie.net",
				WANExposed:        true,
				CFAccessAppID:     "app-3",
				CFAccessDecisions: []string{"allow"},
				HasForwardAuth:    true,
			},
			wantWANAuth:      models.WANAuthCFAccess,
			wantLANAuth:      models.LANAuthForwardAuth,
			wantAPIAuth:      models.APIAuthNone,
			wantStatus:       models.AuthStatusError,
			wantNoteContains: "Double-login risk",
		},
		{
			name: "CF Access + conditional forward_auth — deprecated warning",
			ha: &models.HostAuth{
				Hostname:               "sonarr.vookie.net",
				WANExposed:             true,
				CFAccessAppID:          "app-4",
				CFAccessDecisions:      []string{"allow"},
				HasForwardAuth:         true,
				ConditionalForwardAuth: true,
			},
			wantWANAuth:      models.WANAuthCFAccess,
			wantLANAuth:      models.LANAuthForwardAuth,
			wantAPIAuth:      models.APIAuthNone,
			wantStatus:       models.AuthStatusWarning,
			wantNoteContains: "DEPRECATED",
		},
		{
			name: "CF Access bypass + forward_auth — Pattern D OK",
			ha: &models.HostAuth{
				Hostname:          "radarr.vookie.net",
				WANExposed:        true,
				CFAccessAppID:     "app-5",
				CFAccessDecisions: []string{"bypass", "allow"},
				HasForwardAuth:    true,
			},
			wantWANAuth:      models.WANAuthForwardAuth,
			wantLANAuth:      models.LANAuthForwardAuth,
			wantAPIAuth:      models.APIAuthNone,
			wantStatus:       models.AuthStatusOK,
			wantNoteContains: "CF Access bypass",
		},
		{
			name: "CF Access only no forward_auth — Pattern A/B OK",
			ha: &models.HostAuth{
				Hostname:          "prowlarr.vookie.net",
				WANExposed:        true,
				CFAccessAppID:     "app-6",
				CFAccessDecisions: []string{"allow"},
				HasForwardAuth:    false,
			},
			wantWANAuth: models.WANAuthCFAccess,
			wantLANAuth: models.LANAuthNone,
			wantAPIAuth: models.APIAuthNone,
			wantStatus:  models.AuthStatusOK,
		},
		{
			name: "forward_auth only no CF Access — Pattern C",
			ha: &models.HostAuth{
				Hostname:       "internal.vookie.net",
				WANExposed:     true,
				HasForwardAuth: true,
			},
			wantWANAuth: models.WANAuthForwardAuth,
			wantLANAuth: models.LANAuthForwardAuth,
			wantAPIAuth: models.APIAuthNone,
			wantStatus:  models.AuthStatusOK,
		},
		{
			name: "No auth at all WAN-exposed — app_native warning",
			ha: &models.HostAuth{
				Hostname:       "unprotected.vookie.net",
				WANExposed:     true,
				HasForwardAuth: false,
			},
			wantWANAuth:      models.WANAuthAppNative,
			wantLANAuth:      models.LANAuthNone,
			wantAPIAuth:      models.APIAuthNone,
			wantStatus:       models.AuthStatusWarning,
			wantNoteContains: "app-native",
		},
		{
			name: "Not WAN-exposed with forward_auth — LAN only",
			ha: &models.HostAuth{
				Hostname:       "lan-only.vookie.net",
				WANExposed:     false,
				HasForwardAuth: true,
			},
			wantWANAuth: models.WANAuthNone,
			wantLANAuth: models.LANAuthForwardAuth,
			wantAPIAuth: models.APIAuthMode(""), // API auth not classified for LAN-only hosts
			wantStatus:  models.AuthStatusOK,
		},
		{
			name: "Not WAN-exposed no forward_auth — OK",
			ha: &models.HostAuth{
				Hostname:       "lan-open.vookie.net",
				WANExposed:     false,
				HasForwardAuth: false,
			},
			wantWANAuth: models.WANAuthNone,
			wantLANAuth: models.LANAuthNone,
			wantAPIAuth: models.APIAuthMode(""), // API auth not classified for LAN-only hosts
			wantStatus:  models.AuthStatusOK,
		},
		{
			name: "service_auth policy — API auth via CF service token",
			ha: &models.HostAuth{
				Hostname:          "api.vookie.net",
				WANExposed:        true,
				CFAccessAppID:     "app-7",
				CFAccessDecisions: []string{"service_auth"},
				HasForwardAuth:    false,
			},
			wantWANAuth: models.WANAuthCFAccess,
			wantLANAuth: models.LANAuthNone,
			wantAPIAuth: models.APIAuthCFServiceToken,
			wantStatus:  models.AuthStatusOK,
		},
		{
			name: "Authentik provider — API auth via Authentik bearer",
			ha: &models.HostAuth{
				Hostname:            "authentik-protected.vookie.net",
				WANExposed:          true,
				CFAccessAppID:       "app-8",
				CFAccessDecisions:   []string{"allow"},
				HasForwardAuth:      true,
				AuthentikProviderPK: 42,
			},
			wantWANAuth: models.WANAuthCFAccess, // double login since no bypass
			wantLANAuth: models.LANAuthForwardAuth,
			wantAPIAuth: models.APIAuthAuthentikBearer,
			wantStatus:  models.AuthStatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clone to avoid mutation across tests
			ha := *tt.ha
			classifyAuth(&ha, tt.authentikHost)

			if ha.WANAuth != tt.wantWANAuth {
				t.Errorf("WANAuth = %q, want %q", ha.WANAuth, tt.wantWANAuth)
			}
			if ha.LANAuth != tt.wantLANAuth {
				t.Errorf("LANAuth = %q, want %q", ha.LANAuth, tt.wantLANAuth)
			}
			if ha.APIAuth != tt.wantAPIAuth {
				t.Errorf("APIAuth = %q, want %q", ha.APIAuth, tt.wantAPIAuth)
			}
			if ha.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", ha.Status, tt.wantStatus)
			}
			if tt.wantNoteContains != "" {
				found := false
				for _, n := range ha.Notes {
					if strings.Contains(n, tt.wantNoteContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected notes to contain %q, got %v", tt.wantNoteContains, ha.Notes)
				}
			}
		})
	}
}

func TestHasPolicyDecision(t *testing.T) {
	ha := &models.HostAuth{
		CFAccessDecisions: []string{"bypass", "allow"},
	}
	if !hasPolicyDecision(ha, "bypass") {
		t.Error("expected bypass to be found")
	}
	if !hasPolicyDecision(ha, "allow") {
		t.Error("expected allow to be found")
	}
	if hasPolicyDecision(ha, "deny") {
		t.Error("expected deny to not be found")
	}
	if hasPolicyDecision(ha, "service_auth") {
		t.Error("expected service_auth to not be found")
	}
}
