package find

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/tui"
)

// UI represents the UI for the find command
type UI struct {
	Styles tui.StyleConfig
}

// NewUI creates a new find UI
func NewUI() *UI {
	return &UI{
		Styles: tui.DefaultStyles(),
	}
}

// RenderHeader renders the header for the find command
func (ui *UI) RenderHeader() string {
	return ui.Styles.Header.Render(" 🔍 Find DNS Overrides 🔍 ") + "\n\n"
}

// RenderError renders an error message
func (ui *UI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" ❌ Error: %s ", err))
}

// RenderWarning renders a warning message
func (ui *UI) RenderWarning(message string) string {
	return ui.Styles.Warning.Render(fmt.Sprintf(" ⚠️ %s ", message))
}

// RenderSearchingMessage renders a message indicating that we're searching for DNS overrides
func (ui *UI) RenderSearchingMessage() string {
	return ui.Styles.Info.Render(" 🔍 Searching for DNS overrides... ")
}

// RenderNoMatches renders a message indicating that no matches were found
func (ui *UI) RenderNoMatches() string {
	return ui.Styles.Warning.Render(" ⚠️ No matching DNS overrides found ")
}

// RenderMatches renders the list of matching DNS overrides
func (ui *UI) RenderMatches(overrides []api.DNSOverride) string {
	var sb strings.Builder

	sb.WriteString(ui.RenderHeader())
	sb.WriteString(fmt.Sprintf("Found %d matching DNS overrides:\n\n", len(overrides)))

	// Table header
	sb.WriteString(ui.Styles.Bold.Render("UUID") + "\t" +
		ui.Styles.Bold.Render("Host") + "\t" +
		ui.Styles.Bold.Render("Domain") + "\t" +
		ui.Styles.Bold.Render("Server") + "\t" +
		ui.Styles.Bold.Render("Description") + "\n")

	// Table rows
	for _, override := range overrides {
		enabled := ""
		if override.Enabled == "0" {
			enabled = " (disabled)"
		}

		sb.WriteString(
			ui.Styles.Dimmed.Render(override.UUID) + "\t" +
				ui.Styles.Hostname.Render(override.Host) + "\t" +
				ui.Styles.Hostname.Render(override.Domain) + "\t" +
				ui.Styles.IP.Render(override.Server) + "\t" +
				ui.Styles.Description.Render(override.Description+enabled) + "\n",
		)
	}

	return sb.String()
}

// RenderJSON renders the list of matching DNS overrides as JSON
func (ui *UI) RenderJSON(overrides []api.DNSOverride) string {
	jsonData, err := json.MarshalIndent(overrides, "", "  ")
	if err != nil {
		return ui.RenderError(fmt.Errorf("error generating JSON output: %v", err))
	}

	return string(jsonData)
}
