package tui

import (
	"fmt"
	"strings"
)

// UI provides common UI rendering functions for all commands
type UI struct {
	Styles StyleConfig
}

// NewUI creates a new UI with default styles
func NewUI() *UI {
	return &UI{
		Styles: DefaultStyles(),
	}
}

// RenderTitle renders a title with optional emoji
func (ui *UI) RenderTitle(title string, emoji string) string {
	if emoji != "" {
		return ui.Styles.Header.Render(fmt.Sprintf(" %s %s %s ", emoji, title, emoji))
	}
	return ui.Styles.Header.Render(fmt.Sprintf(" %s ", title))
}

// RenderSection renders a section title
func (ui *UI) RenderSection(title string) string {
	return ui.Styles.Section.Render(fmt.Sprintf(" %s ", title))
}

// RenderSuccess renders a success message
func (ui *UI) RenderSuccess(message string) string {
	return ui.Styles.Success.Render(fmt.Sprintf(" ✅ %s ", message))
}

// RenderError renders an error message
func (ui *UI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" ❌ Error: %s ", err))
}

// RenderWarning renders a warning message
func (ui *UI) RenderWarning(message string) string {
	return ui.Styles.Warning.Render(fmt.Sprintf(" ⚠️  %s ", message))
}

// RenderInfo renders an info message
func (ui *UI) RenderInfo(message string) string {
	return ui.Styles.Info.Render(fmt.Sprintf(" ℹ️  %s ", message))
}

// RenderDryRun renders a dry run indicator
func (ui *UI) RenderDryRun() string {
	return ui.Styles.DryRun.Render(" 🔍 DRY RUN ") + "\n"
}

// RenderSeparator renders a horizontal separator line
func (ui *UI) RenderSeparator() string {
	return strings.Repeat("─", 80) + "\n"
}

// RenderHostname renders a hostname
func (ui *UI) RenderHostname(hostname string) string {
	return ui.Styles.Hostname.Render(hostname)
}

// RenderIP renders an IP address
func (ui *UI) RenderIP(ip string) string {
	return ui.Styles.IP.Render(ip)
}

// RenderDescription renders a description
func (ui *UI) RenderDescription(description string) string {
	return ui.Styles.Description.Render(description)
}

// RenderKeyValue renders a key-value pair
func (ui *UI) RenderKeyValue(key, value string) string {
	return ui.Styles.Bold.Render(key+":") + " " + ui.Styles.Normal.Render(value)
}
