package edit

import (
	"fmt"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/tui"
)

// UI represents the UI for the edit command
type UI struct {
	Styles tui.StyleConfig
}

// NewUI creates a new edit UI
func NewUI() *UI {
	return &UI{
		Styles: tui.DefaultStyles(),
	}
}

// RenderHeader renders the header for the edit command
func (ui *UI) RenderHeader() string {
	return ui.Styles.Header.Render(" u270f Edit DNS Override u270f ") + "\n\n"
}

// RenderSuccess renders a success message for editing a DNS override
func (ui *UI) RenderSuccess() string {
	return ui.Styles.Success.Render(" u2705 DNS override updated successfully ")
}

// RenderError renders an error message
func (ui *UI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" u274c Error: %s ", err))
}

// RenderWarning renders a warning message
func (ui *UI) RenderWarning(message string) string {
	return ui.Styles.Warning.Render(fmt.Sprintf(" u26a0ufe0f  %s ", message))
}

// RenderOverrideDetails renders the details of a DNS override
func (ui *UI) RenderOverrideDetails(override api.DNSOverride) string {
	var sb strings.Builder

	sb.WriteString(ui.RenderHeader())
	sb.WriteString(
		ui.Styles.Bold.Render("UUID:") + " " + ui.Styles.Dimmed.Render(override.UUID) + "\n",
	)
	sb.WriteString(
		ui.Styles.Bold.Render("Host:") + " " + ui.Styles.Hostname.Render(override.Host) + "\n",
	)
	sb.WriteString(
		ui.Styles.Bold.Render("Domain:") + " " + ui.Styles.Hostname.Render(override.Domain) + "\n",
	)
	sb.WriteString(
		ui.Styles.Bold.Render("IP Address:") + " " + ui.Styles.IP.Render(override.Server) + "\n",
	)

	if override.Description != "" {
		sb.WriteString(
			ui.Styles.Bold.Render(
				"Description:",
			) + " " + ui.Styles.Description.Render(
				override.Description,
			) + "\n",
		)
	}

	enabled := "No"
	enabledStyle := ui.Styles.Error
	if override.Enabled == "1" {
		enabled = "Yes"
		enabledStyle = ui.Styles.Success
	}
	sb.WriteString(ui.Styles.Bold.Render("Enabled:") + " " + enabledStyle.Render(enabled) + "\n")

	return sb.String()
}

// RenderUpdatingMessage renders a message indicating that a DNS override is being updated
func (ui *UI) RenderUpdatingMessage(host, domain, server string) string {
	return ui.Styles.Info.Render(
		fmt.Sprintf(" u270f Updating DNS override %s.%s -> %s... ", host, domain, server),
	)
}

// RenderApplyingMessage renders a message indicating that changes are being applied
func (ui *UI) RenderApplyingMessage() string {
	return ui.Styles.Info.Render(" ud83dudcbe Applying changes... ")
}

// RenderApplySuccess renders a success message for applying changes
func (ui *UI) RenderApplySuccess() string {
	return ui.Styles.Success.Render(" u2705 Changes applied successfully ")
}

// RenderChangesNotApplied renders a message indicating that changes have not been applied
func (ui *UI) RenderChangesNotApplied() string {
	return ui.Styles.Warning.Render(
		" u26a0ufe0f  Changes have not been applied. Use 'apply' command to apply changes. ",
	)
}
