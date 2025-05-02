package config

import (
	"fmt"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/tui"
)

// UI represents the UI for the config command
type UI struct {
	Styles tui.StyleConfig
}

// NewUI creates a new config UI
func NewUI() *UI {
	return &UI{
		Styles: tui.DefaultStyles(),
	}
}

// RenderHeader renders the header for the config command
func (ui *UI) RenderHeader() string {
	return ui.Styles.Header.Render(" u2699ufe0f Configuration u2699ufe0f ") + "\n\n"
}

// RenderSuccess renders a success message
func (ui *UI) RenderSuccess(message string) string {
	return ui.Styles.Success.Render(fmt.Sprintf(" u2705 %s ", message))
}

// RenderError renders an error message
func (ui *UI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" u274c Error: %s ", err))
}

// RenderInfo renders an informational message
func (ui *UI) RenderInfo(message string) string {
	return ui.Styles.Info.Render(fmt.Sprintf(" ud83dudcac %s ", message))
}

// RenderEnvVarSection renders the environment variable section
func (ui *UI) RenderEnvVarSection() string {
	return ui.Styles.Section.Render(" ud83cudfe0 Environment Variables ") + "\n"
}

// RenderTestingConnection renders a message indicating that the connection is being tested
func (ui *UI) RenderTestingConnection() string {
	return ui.Styles.Info.Render(" ud83dudcbe Testing connection... ")
}

// RenderConnectionSuccess renders a message indicating that the connection test was successful
func (ui *UI) RenderConnectionSuccess() string {
	return ui.Styles.Success.Render(" u2705 Connection successful! ")
}

// RenderConnectionFailure renders a message indicating that the connection test failed
func (ui *UI) RenderConnectionFailure(err error) string {
	var sb strings.Builder

	sb.WriteString(ui.Styles.Error.Render(" u274c Connection failed "))
	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Error.Render(fmt.Sprintf("   Error: %s", err)))

	return sb.String()
}
