package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// StyleConfig contains all the styles used in the application
type StyleConfig struct {
	// General styles
	Header   lipgloss.Style
	Section  lipgloss.Style
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Normal   lipgloss.Style
	Bold     lipgloss.Style
	Dimmed   lipgloss.Style

	// Status styles
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Info    lipgloss.Style

	// Action styles
	Add    lipgloss.Style
	Update lipgloss.Style
	Remove lipgloss.Style
	Count  lipgloss.Style

	// Content styles
	Hostname    lipgloss.Style
	IP          lipgloss.Style
	Description lipgloss.Style
	DryRun      lipgloss.Style

	// Table styles
	TableStyles table.Styles
}

// DefaultStyles returns the default style configuration
func DefaultStyles() StyleConfig {
	// Define table styles
	tableStyles := table.DefaultStyles()
	tableStyles.Header = tableStyles.Header.Bold(true).Foreground(lipgloss.Color("#f07178"))
	tableStyles.Selected = tableStyles.Selected.Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#7aa2f7"))

	return StyleConfig{
		// General styles
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7aa2f7")).
			Background(lipgloss.Color("#3b4261")).
			Padding(0, 1).
			Width(50).
			Align(lipgloss.Center),
		Section: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7aa2f7")).
			Background(lipgloss.Color("#3b4261")).
			Padding(0, 1),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7aa2f7")).
			MarginBottom(1),
		Subtitle: lipgloss.NewStyle().Foreground(lipgloss.Color("#bb9af7")),
		Normal:   lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5")),
		Bold:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#c0caf5")),
		Dimmed:   lipgloss.NewStyle().Faint(false).Bold(true).Foreground(lipgloss.Color("#e0af68")),

		// Status styles
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")),
		Warning: lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e")),
		Info:    lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff")),

		// Action styles
		Add:    lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")),
		Update: lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")),
		Remove: lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e")),
		Count:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7dcfff")),

		// Content styles
		Hostname:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bb9af7")),
		IP:          lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff")),
		Description: lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")),
		DryRun: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#e0af68")).
			Background(lipgloss.Color("#3b4261")).
			Padding(0, 1),

		// Table styles
		TableStyles: tableStyles,
	}
}
