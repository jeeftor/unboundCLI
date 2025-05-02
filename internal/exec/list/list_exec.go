package list

import (
	"fmt"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/tui"
)

// UI represents the UI for the list command
type UI struct {
	Styles tui.StyleConfig
}

// NewUI creates a new list UI
func NewUI() *UI {
	return &UI{
		Styles: tui.DefaultStyles(),
	}
}

// RenderHeader renders the header for the list command
func (ui *UI) RenderHeader() string {
	return ui.Styles.Header.Render(" 📋 DNS Override List 📋 ") + "\n\n"
}

// RenderTable renders the DNS overrides as a table
func (ui *UI) RenderTable(overrides []api.DNSOverride) string {
	var sb strings.Builder

	// Add header
	sb.WriteString(ui.RenderHeader())

	// Table header
	sb.WriteString(fmt.Sprintf("%s %s %s %s %s %s\n",
		ui.Styles.Bold.Width(36).Render("UUID"),
		ui.Styles.Bold.Width(20).Render("HOST"),
		ui.Styles.Bold.Width(20).Render("DOMAIN"),
		ui.Styles.Bold.Width(15).Render("IP ADDRESS"),
		ui.Styles.Bold.Width(30).Render("DESCRIPTION"),
		ui.Styles.Bold.Width(10).Render("ENABLED"),
	))

	// Add separator
	sb.WriteString(strings.Repeat("─", 140))
	sb.WriteString("\n")

	// Table rows
	for _, o := range overrides {
		enabled := "No"
		enabledStyle := ui.Styles.Error
		if o.Enabled == "1" {
			enabled = "Yes"
			enabledStyle = ui.Styles.Success
		}

		sb.WriteString(fmt.Sprintf("%s %s %s %s %s %s\n",
			ui.Styles.Dimmed.Width(36).Render(o.UUID),
			ui.Styles.Hostname.Width(20).Render(o.Host),
			ui.Styles.Hostname.Width(20).Render(o.Domain),
			ui.Styles.IP.Width(15).Render(o.Server),
			ui.Styles.Description.Width(30).Render(o.Description),
			enabledStyle.Width(10).Render(enabled),
		))
	}

	// Add summary
	sb.WriteString("\n")
	sb.WriteString(
		ui.Styles.Info.Render(fmt.Sprintf(" 📊 Total DNS Overrides: %d ", len(overrides))),
	)

	return sb.String()
}

// RenderEmpty renders a message when no overrides are found
func (ui *UI) RenderEmpty() string {
	var sb strings.Builder

	sb.WriteString(ui.RenderHeader())
	sb.WriteString(ui.Styles.Warning.Render(" ⚠️  No DNS overrides found "))
	sb.WriteString("\n\n")
	sb.WriteString(ui.Styles.Info.Render(" 💡 Tip: Use 'add' command to create a new DNS override "))

	return sb.String()
}

// RenderError renders an error message
func (ui *UI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" ❌ Error: %s ", err))
}
