package cmd

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	// When invoked over SSH without a TTY (e.g. `ssh host caddy-dns-sync doctor`)
	// termenv detects no color support and lipgloss strips all ANSI codes.
	// Force ANSI256 unless the caller explicitly opts out with NO_COLOR.
	if os.Getenv("NO_COLOR") == "" {
		lipgloss.SetColorProfile(termenv.ANSI256)
	}
}

// ── Shared CLI colour palette ─────────────────────────────────────────────────
// Use these styles consistently across all commands so the output feels cohesive.

var (
	// Status indicators
	StyleOK   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true) // green
	StyleFail = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true) // red
	StyleWarn = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true) // yellow
	StyleInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))           // bright blue

	// Text / decoration
	StyleMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // gray
	StyleCode    = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))            // white/bright
	StyleSection = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true) // bright cyan
	StyleBold    = lipgloss.NewStyle().Bold(true)

	// Pre-rendered symbols
	SymOK   = StyleOK.Render("✓")
	SymFail = StyleFail.Render("✗")
	SymWarn = StyleWarn.Render("!")
	SymInfo = StyleInfo.Render("ℹ")
	SymDot  = StyleMuted.Render("•")
)
