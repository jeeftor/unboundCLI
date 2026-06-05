package widgets

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CFPickerItem represents a selectable Cloudflare resource (zone or tunnel).
type CFPickerItem struct {
	ID   string
	Name string
}

// CFPickerSelectedMsg is emitted when the user confirms a selection.
type CFPickerSelectedMsg struct {
	Item CFPickerItem
}

// CFPickerCancelledMsg is emitted when the user dismisses the picker.
type CFPickerCancelledMsg struct{}

// CFPickerWidget is a modal list picker for Cloudflare zones or tunnels.
type CFPickerWidget struct {
	Title  string
	Items  []CFPickerItem
	cursor int
	width  int
	height int
	theme  *Theme
}

// NewCFPicker creates a picker with the given title and items.
func NewCFPicker(title string, items []CFPickerItem) *CFPickerWidget {
	return &CFPickerWidget{
		Title: title,
		Items: items,
		theme: CurrentTheme,
	}
}

// SetSize updates the rendering dimensions.
func (p *CFPickerWidget) SetSize(w, h int) {
	p.width = w
	p.height = h
}

// Update handles keyboard input for the picker.
func (p *CFPickerWidget) Update(msg tea.Msg) (*CFPickerWidget, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
		case "down", "j":
			if p.cursor < len(p.Items)-1 {
				p.cursor++
			}
		case "enter":
			if len(p.Items) > 0 && p.cursor < len(p.Items) {
				selected := p.Items[p.cursor]
				return p, func() tea.Msg { return CFPickerSelectedMsg{Item: selected} }
			}
		case "esc", "q":
			return p, func() tea.Msg { return CFPickerCancelledMsg{} }
		}
	}
	return p, nil
}

// View renders the picker as a bordered modal panel.
func (p *CFPickerWidget) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(p.theme.ColorInfo)
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#283457")).
		Foreground(lipgloss.Color("#c0caf5"))
	dimStyle := p.theme.Dimmed
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5"))

	var lines []string
	lines = append(lines, titleStyle.Render(p.Title))
	lines = append(lines, "")

	for i, item := range p.Items {
		name := fmt.Sprintf("%-28s", item.Name)
		id := dimStyle.Render(item.ID)
		if i == p.cursor {
			row := "> " + name + "  " + id
			lines = append(lines, selectedStyle.Render(row))
		} else {
			row := "  " + normalStyle.Render(name) + "  " + id
			lines = append(lines, row)
		}
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("↑/k ↓/j  navigate    enter  select    esc  cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	w := p.width
	if w < 64 {
		w = 64
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.theme.ColorInfo).
		Padding(1, 2).
		Width(w - 8).
		Render(content)
}
