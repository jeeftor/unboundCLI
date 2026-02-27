package tables

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// TableConfig defines table appearance and behavior
type TableConfig struct {
	Title       string
	Headers     []string
	Rows        [][]string
	Summary     string
	ColorScheme ColorScheme
}

// ColorScheme defines colors for table rendering
type ColorScheme struct {
	Header    lipgloss.Color
	Border    lipgloss.Color
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Dim       lipgloss.Color
	Warning   lipgloss.Color
}

// DefaultColorScheme returns Tokyo Night colors
func DefaultColorScheme() ColorScheme {
	return ColorScheme{
		Header:    lipgloss.Color("#bb9af7"), // Purple
		Border:    lipgloss.Color("#414868"), // Dark gray
		Primary:   lipgloss.Color("#9ece6a"), // Green
		Secondary: lipgloss.Color("#7aa2f7"), // Blue
		Dim:       lipgloss.Color("#565f89"), // Dim gray
		Warning:   lipgloss.Color("#e0af68"), // Yellow
	}
}

// TableRenderer creates beautiful tables
type TableRenderer struct {
	config ColorScheme
}

// NewTableRenderer creates a new table renderer with default color scheme
func NewTableRenderer() *TableRenderer {
	return &TableRenderer{config: DefaultColorScheme()}
}

// NewTableRendererWithColors creates a new table renderer with custom colors
func NewTableRendererWithColors(colors ColorScheme) *TableRenderer {
	return &TableRenderer{config: colors}
}

// Render creates a formatted table from TableConfig
func (r *TableRenderer) Render(cfg TableConfig) string {
	// Use default color scheme if none provided
	scheme := cfg.ColorScheme
	if scheme.Header == "" {
		scheme = r.config
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(scheme.Border)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				// Header style
				return lipgloss.NewStyle().
					Foreground(scheme.Header).
					Bold(true).
					Align(lipgloss.Left)
			}
			return lipgloss.NewStyle().Align(lipgloss.Left)
		}).
		Headers(cfg.Headers...).
		Rows(cfg.Rows...)

	// Build output with title and summary
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(scheme.Header).
		MarginBottom(1)

	output := ""
	if cfg.Title != "" {
		output += titleStyle.Render(cfg.Title) + "\n"
	}
	if cfg.Summary != "" {
		output += cfg.Summary + "\n\n"
	}
	output += t.String() + "\n"

	return output
}
