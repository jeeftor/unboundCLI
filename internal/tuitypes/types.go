package tuitypes

import "github.com/charmbracelet/lipgloss"

// FilterMode represents the current filter applied
type FilterMode int

const (
	FilterNone FilterMode = iota
	FilterOutOfSync
	FilterMismatches
	FilterUnconfigured
	FilterStale
	FilterUnboundIssues
	FilterAdguardIssues
	FilterManual
	FilterDHCPMismatches
)

// StatusIndicator represents a status icon with color
type StatusIndicator struct {
	Icon  string
	Color lipgloss.Color
	Text  string
}

// Status indicator mappings with Tokyo Night colors
var (
	IconSynced = StatusIndicator{
		Icon:  "✅",
		Color: lipgloss.Color("#9ece6a"), // green
		Text:  "Synced",
	}
	IconWarning = StatusIndicator{
		Icon:  "⚠️",
		Color: lipgloss.Color("#e0af68"), // yellow
		Text:  "Mismatch",
	}
	IconError = StatusIndicator{
		Icon:  "❌",
		Color: lipgloss.Color("#f7768e"), // red
		Text:  "Missing",
	}
	IconStale = StatusIndicator{
		Icon:  "🗑️",
		Color: lipgloss.Color("#565f89"), // gray
		Text:  "Stale",
	}
	IconNA = StatusIndicator{
		Icon:  "—",
		Color: lipgloss.Color("#565f89"), // gray
		Text:  "N/A",
	}
	IconCaddy = StatusIndicator{
		Icon:  "✅",
		Color: lipgloss.Color("#7aa2f7"), // blue
		Text:  "Source",
	}
)

// LoadingPhase represents the current loading phase
type LoadingPhase int

const (
	PhaseIdle LoadingPhase = iota
	PhaseLoadingData
	PhaseQueryingDNS
)

// ServiceLoadStatus tracks which services have been loaded
type ServiceLoadStatus struct {
	CaddyLoaded   bool
	UnboundLoaded bool
	AdguardLoaded bool
	DHCPLoaded    bool
}
