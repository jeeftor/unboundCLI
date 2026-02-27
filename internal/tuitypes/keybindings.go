package tuitypes

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the TUI
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding

	// Filters
	FilterNone           key.Binding
	FilterOutOfSync      key.Binding
	FilterMismatches     key.Binding
	FilterUnconfigured   key.Binding
	FilterStale          key.Binding
	FilterUnboundIssues  key.Binding
	FilterAdguardIssues  key.Binding
	FilterManual         key.Binding
	FilterDHCPMismatches key.Binding

	// Search
	Search      key.Binding
	ClearSearch key.Binding

	// Actions
	SyncSelected key.Binding
	SyncAll      key.Binding
	Refresh      key.Binding

	// Utility
	Help key.Binding
	Quit key.Binding
}

// ShortHelp returns keybinding help
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up,
		k.Down,
		k.SyncSelected,
		k.SyncAll,
		k.Search,
		k.Refresh,
		k.Help,
		k.Quit,
	}
}

// FullHelp returns the full set of keybindings
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		// Row 1: Navigation
		{k.Up, k.Down, k.PageUp, k.PageDown},

		// Row 2: Sync Actions
		{k.SyncSelected, k.SyncAll, k.Refresh},

		// Row 3: Filters - Part 1
		{k.FilterNone, k.FilterOutOfSync, k.FilterMismatches, k.FilterUnconfigured},

		// Row 4: Filters - Part 2
		{k.FilterStale, k.FilterUnboundIssues, k.FilterAdguardIssues},

		// Row 5: Search
		{k.Search, k.ClearSearch},

		// Row 6: System
		{k.Help, k.Quit},
	}
}
