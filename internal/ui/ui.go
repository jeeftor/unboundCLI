package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type Styles struct {
	Header  lipgloss.Style
	Error   lipgloss.Style
	Success lipgloss.Style
	Info    lipgloss.Style
	Warning lipgloss.Style
	Section lipgloss.Style
}

func DefaultStyles() Styles {
	return Styles{
		Header:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		Success: lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		Info:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		Warning: lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		Section: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")),
	}
}

type BaseUI struct {
	Styles Styles
}

func NewBaseUI() *BaseUI {
	return &BaseUI{Styles: DefaultStyles()}
}

func (ui *BaseUI) RenderHeader(title string) string {
	return ui.Styles.Header.Render(fmt.Sprintf(" %s ", title))
}

func (ui *BaseUI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf("❌ %s", err))
}

func (ui *BaseUI) RenderSuccess(msg string) string {
	return ui.Styles.Success.Render(fmt.Sprintf("✅ %s", msg))
}

func (ui *BaseUI) RenderInfo(msg string) string {
	return ui.Styles.Info.Render(fmt.Sprintf("💬 %s", msg))
}

func (ui *BaseUI) RenderWarning(msg string) string {
	return ui.Styles.Warning.Render(fmt.Sprintf("⚠️ %s", msg))
}

func (ui *BaseUI) RenderSection(title string) string {
	return ui.Styles.Section.Render(fmt.Sprintf("%s", title))
}
