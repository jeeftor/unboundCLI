package delete

import (
	"fmt"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/tui"
)

// UI represents the UI for the delete command
type UI struct {
	Styles tui.StyleConfig
}

// NewUI creates a new delete UI
func NewUI() *UI {
	return &UI{
		Styles: tui.DefaultStyles(),
	}
}

// RenderHeader renders the header for the delete command
func (ui *UI) RenderHeader() string {
	return ui.Styles.Header.Render(" u274c Delete DNS Override u274c ") + "\n\n"
}

// RenderSuccess renders a success message for deleting a DNS override
func (ui *UI) RenderSuccess(uuid string) string {
	var sb strings.Builder

	sb.WriteString(
		ui.Styles.Success.Render(
			fmt.Sprintf(" u2705 DNS override deleted successfully: %s ", uuid),
		),
	)
	return sb.String()
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

// RenderDeletingMessage renders a message indicating that a DNS override is being deleted
func (ui *UI) RenderDeletingMessage(uuid string) string {
	return ui.Styles.Info.Render(fmt.Sprintf(" ud83duddd1 Deleting DNS override %s... ", uuid))
}

// RenderApplyingMessage renders a message indicating that changes are being applied
func (ui *UI) RenderApplyingMessage() string {
	return ui.Styles.Info.Render(" ud83dudcbe Applying changes... ")
}

// RenderConfirmation renders a confirmation message
func (ui *UI) RenderConfirmation(override api.DNSOverride) string {
	var sb strings.Builder

	sb.WriteString(
		ui.Styles.Warning.Render(
			fmt.Sprintf(" u26a0ufe0f  Are you sure you want to delete this DNS override? "),
		),
	)
	sb.WriteString("\n\n")
	sb.WriteString(
		ui.Styles.Bold.Render("UUID:") + " " + ui.Styles.Dimmed.Render(override.UUID) + "\n",
	)
	sb.WriteString(
		ui.Styles.Bold.Render("Host:") + " " + ui.Styles.Hostname.Render(override.Host) + "\n",
	)
	sb.WriteString(
		ui.Styles.Bold.Render("Domain:") + " " + ui.Styles.Hostname.Render(override.Domain) + "\n",
	)

	return sb.String()
}
