package widgets

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// KeyMap defines all keyboard shortcuts for the TUI
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Actions
	Sync       key.Binding
	SyncSingle key.Binding
	Refresh    key.Binding
	Apply      key.Binding
	Delete     key.Binding
	Edit       key.Binding

	// Filtering and Search
	Filter      key.Binding
	Search      key.Binding
	ClearFilter key.Binding
	Sort        key.Binding

	// View Controls
	ToggleIPs  key.Binding
	ToggleHelp key.Binding
	ToggleLogs key.Binding
	Config     key.Binding
	Detail     key.Binding
	CFDetail   key.Binding
	CFEdit     key.Binding

	// Selection
	ToggleSelect key.Binding

	// Global
	Quit   key.Binding
	Enter  key.Binding
	Escape key.Binding
}

// DefaultKeyMap returns the default keyboard shortcuts
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "bottom"),
		),

		// Actions
		Sync: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sync all"),
		),
		SyncSingle: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "sync selected"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Apply: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "apply changes"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),

		// Filtering and Search
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "cycle filter"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clear filter"),
		),
		Sort: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "cycle sort"),
		),

		// View Controls
		ToggleIPs: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "CF detail"),
		),
		ToggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		ToggleLogs: key.NewBinding(
			key.WithKeys("l", "L"),
			key.WithHelp("l", "logs"),
		),
		Config: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "config"),
		),
		Detail: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "entry detail"),
		),
		CFDetail: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "CF detail"),
		),
		CFEdit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit CF"),
		),

		// Selection
		ToggleSelect: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "toggle select"),
		),

		// Global
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// ShortHelp returns a short help view (implements help.KeyMap)
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up,
		k.Down,
		k.Sync,
		k.Refresh,
		k.ToggleLogs,
		k.ToggleHelp,
		k.Quit,
	}
}

// FullHelp returns all help bindings (implements help.KeyMap)
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown, k.Home, k.End},                        // Navigation
		{k.ToggleSelect, k.Sync, k.SyncSingle, k.Refresh},                          // Selection & Actions
		{k.Filter, k.Search, k.ClearFilter, k.Sort, k.ToggleLogs},                  // View
		{k.Detail, k.CFDetail, k.CFEdit, k.Config, k.ToggleHelp, k.Escape, k.Quit}, // Global
	}
}

// HelpWidget displays keyboard shortcuts at the bottom of the screen
type HelpWidget struct {
	BaseWidget

	help     help.Model
	keys     KeyMap
	showFull bool
	theme    *Theme
}

// NewHelpWidget creates a new help widget
func NewHelpWidget() *HelpWidget {
	h := help.New()
	h.ShowAll = false

	return &HelpWidget{
		BaseWidget: NewBaseWidget(),
		help:       h,
		keys:       DefaultKeyMap(),
		showFull:   false,
		theme:      CurrentTheme,
	}
}

// Init initializes the help widget
func (w *HelpWidget) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (w *HelpWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, w.keys.ToggleHelp):
			w.showFull = !w.showFull
			w.help.ShowAll = w.showFull
			return w, nil
		}
	}

	return w, nil
}

// View renders the help widget
func (w *HelpWidget) View() string {
	if w.width == 0 {
		return ""
	}

	// Update help width to match widget width
	w.help.Width = w.width - 4

	// Render help using Bubble Tea help component
	helpView := w.help.View(w.keys)

	// Create title
	titleText := "╭─ Keyboard Shortcuts ─"
	if w.showFull {
		titleText = "╭─ Keyboard Shortcuts (Press ? to hide) ─"
	}
	title := w.theme.Header.Foreground(w.theme.ColorDim).Render(titleText)

	style := lipgloss.NewStyle().
		Border(lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "",
			TopRight:    "╮",
			BottomLeft:  "╰",
			BottomRight: "╯",
		}).
		BorderForeground(w.theme.ColorDim).
		Padding(0, 1).
		Width(w.width - 2).
		Foreground(w.theme.ColorDim)

	bordered := style.Render(helpView)

	// Prepend title line
	lines := strings.Split(bordered, "\n")
	if len(lines) > 0 {
		remainingWidth := w.width - lipgloss.Width(title) - 1
		if remainingWidth < 0 {
			remainingWidth = 0
		}
		titleLine := title + w.theme.Header.Foreground(w.theme.ColorDim).Render(strings.Repeat("─", remainingWidth)+"╮")
		lines[0] = titleLine
	}

	return strings.Join(lines, "\n")
}

// SetSize updates the widget dimensions
func (w *HelpWidget) SetSize(width, height int) {
	w.BaseWidget.SetSize(width, height)
	w.help.Width = width
}

// ToggleShowFull toggles between short and full help display
func (w *HelpWidget) ToggleShowFull() {
	w.showFull = !w.showFull
	w.help.ShowAll = w.showFull
}

// ShowFull returns true if showing full help
func (w *HelpWidget) ShowFull() bool {
	return w.showFull
}

// SetShowFull sets whether to show full help
func (w *HelpWidget) SetShowFull(show bool) {
	w.showFull = show
	w.help.ShowAll = show
}

// Keys returns the current keymap
func (w *HelpWidget) Keys() KeyMap {
	return w.keys
}

// SetKeys updates the keymap
func (w *HelpWidget) SetKeys(keys KeyMap) {
	w.keys = keys
}
