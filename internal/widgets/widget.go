package widgets

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Widget represents a reusable UI component following Bubble Tea patterns
type Widget interface {
	// Init initializes the widget (called once at startup)
	Init() tea.Cmd

	// Update handles messages and returns updated widget with optional command
	Update(msg tea.Msg) (Widget, tea.Cmd)

	// View renders the widget to a string
	View() string

	// SetSize updates the widget's dimensions
	SetSize(width, height int)

	// Focus sets keyboard focus to this widget
	Focus()

	// Blur removes keyboard focus from this widget
	Blur()

	// Focused returns true if this widget has keyboard focus
	Focused() bool
}

// BaseWidget provides common functionality for all widgets
type BaseWidget struct {
	width   int
	height  int
	focused bool
}

// NewBaseWidget creates a new BaseWidget
func NewBaseWidget() BaseWidget {
	return BaseWidget{
		width:   0,
		height:  0,
		focused: false,
	}
}

// SetSize updates the widget dimensions
func (b *BaseWidget) SetSize(width, height int) {
	b.width = width
	b.height = height
}

// Width returns the current width
func (b *BaseWidget) Width() int {
	return b.width
}

// Height returns the current height
func (b *BaseWidget) Height() int {
	return b.height
}

// Focus sets keyboard focus
func (b *BaseWidget) Focus() {
	b.focused = true
}

// Blur removes keyboard focus
func (b *BaseWidget) Blur() {
	b.focused = false
}

// Focused returns true if widget has focus
func (b *BaseWidget) Focused() bool {
	return b.focused
}

// ToggleFocus toggles the focus state
func (b *BaseWidget) ToggleFocus() {
	b.focused = !b.focused
}
