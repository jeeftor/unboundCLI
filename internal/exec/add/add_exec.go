package add

import (
	"fmt"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/tui"
)

// UI represents the UI for the add command
type UI struct {
	Styles tui.StyleConfig
}

// NewUI creates a new add UI
func NewUI() *UI {
	return &UI{
		Styles: tui.DefaultStyles(),
	}
}

// RenderHeader renders the header for the add command
func (ui *UI) RenderHeader() string {
	return ui.Styles.Header.Render(" ➕ Add DNS Override ➕ ") + "\n\n"
}

// RenderSuccess renders a success message for adding a DNS override
func (ui *UI) RenderSuccess(uuid string) string {
	var sb strings.Builder

	sb.WriteString(
		ui.Styles.Success.Render(
			fmt.Sprintf(" ✅ DNS override added successfully with UUID: %s ", uuid),
		),
	)
	return sb.String()
}

// RenderError renders an error message
func (ui *UI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" ❌ Error: %s ", err))
}

// RenderWarning renders a warning message
func (ui *UI) RenderWarning(message string) string {
	return ui.Styles.Warning.Render(fmt.Sprintf(" ⚠️  %s ", message))
}

// RenderDuplicateWarning renders a warning about duplicate DNS override
func (ui *UI) RenderDuplicateWarning(host, domain, uuid string) string {
	var sb strings.Builder

	sb.WriteString(
		ui.Styles.Warning.Render(
			fmt.Sprintf(
				" ⚠️  DNS override for %s.%s already exists with UUID: %s ",
				host,
				domain,
				uuid,
			),
		),
	)
	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Warning.Render(" ⚠️  Use --force flag to add anyway "))

	return sb.String()
}

// RenderOverrideDetails renders the details of a DNS override
func (ui *UI) RenderOverrideDetails(override api.DNSOverride) string {
	var sb strings.Builder

	sb.WriteString(ui.RenderHeader())
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

// RenderAddingMessage renders a message indicating that a DNS override is being added
func (ui *UI) RenderAddingMessage() string {
	return ui.Styles.Info.Render(" 📾 Adding DNS override... ")
}

// RenderApplyingMessage renders a message indicating that changes are being applied
func (ui *UI) RenderApplyingMessage() string {
	return ui.Styles.Info.Render(" 📾 Applying changes... ")
}

// RenderCheckingMessage renders a message indicating that we're checking for existing overrides
func (ui *UI) RenderCheckingMessage() string {
	return ui.Styles.Info.Render(" 🔍 Checking for existing overrides... ")
}
