package widgets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfigField represents a single configuration field
type ConfigField struct {
	Key         string
	Label       string
	Value       string
	Placeholder string
	IsPassword  bool
	IsRequired  bool
	Validator   func(string) error
	HelpText    string
}

// ConfigSection represents a group of related configuration fields
type ConfigSection struct {
	Title  string
	Fields []ConfigField
}

// ConfigEditorWidget allows editing configuration settings
type ConfigEditorWidget struct {
	BaseWidget

	// Data
	sections       []ConfigSection
	currentSection int
	currentField   int
	inputs         []textinput.Model

	// State
	editing          bool
	showHelp         bool
	showPasswords    bool
	validationErrors map[string]string

	// Navigation
	cursorPos int // Overall cursor position across all fields

	// Layout
	theme *Theme
}

// NewConfigEditor creates a new configuration editor widget
func NewConfigEditor() *ConfigEditorWidget {
	return &ConfigEditorWidget{
		BaseWidget:       NewBaseWidget(),
		sections:         []ConfigSection{},
		currentSection:   0,
		currentField:     0,
		inputs:           []textinput.Model{},
		editing:          false,
		showHelp:         false,
		validationErrors: make(map[string]string),
		theme:            CurrentTheme,
	}
}

// Init initializes the configuration editor
func (w *ConfigEditorWidget) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (w *ConfigEditorWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	var cmd tea.Cmd

	// If editing a field, route input to the text input
	if w.editing && w.cursorPos < len(w.inputs) {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// Save current field and move to next
				w.saveCurrentField()
				w.editing = false
				return w, nil
			case "esc":
				// Cancel editing
				w.editing = false
				return w, nil
			default:
				w.inputs[w.cursorPos], cmd = w.inputs[w.cursorPos].Update(msg)
				return w, cmd
			}
		}
		w.inputs[w.cursorPos], cmd = w.inputs[w.cursorPos].Update(msg)
		return w, cmd
	}

	// Handle navigation when not editing
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			w.moveCursorUp()
		case "down", "j":
			w.moveCursorDown()
		case "left", "h":
			w.previousSection()
		case "right", "l":
			w.nextSection()
		case "enter", "e":
			w.startEditing()
			return w, textinput.Blink
		case "?":
			w.showHelp = !w.showHelp
		}
	}

	return w, cmd
}

// View renders the configuration editor
func (w *ConfigEditorWidget) View() string {
	if w.width == 0 || w.height == 0 {
		return ""
	}

	var sections []string

	// Title
	title := w.theme.Header.Render("Configuration Editor")
	sections = append(sections, title)
	sections = append(sections, "")

	// Section tabs
	sections = append(sections, w.renderSectionTabs())
	sections = append(sections, "")

	// Current section fields
	if w.currentSection < len(w.sections) {
		sections = append(sections, w.renderSection(w.sections[w.currentSection]))
	}

	sections = append(sections, "")

	// Help text
	if w.showHelp {
		sections = append(sections, w.renderHelp())
	} else {
		sections = append(sections, w.theme.Dimmed.Render("Press ? for help"))
	}

	// Wrap in a border
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	style := lipgloss.NewStyle().
		Border(w.theme.Border).
		BorderForeground(w.theme.ColorInfo).
		Padding(1, 2).
		Width(w.width - 4)

	return style.Render(content)
}

// renderSectionTabs renders the section navigation tabs
func (w *ConfigEditorWidget) renderSectionTabs() string {
	var tabs []string

	for i, section := range w.sections {
		tab := section.Title
		if i == w.currentSection {
			// Active tab
			style := lipgloss.NewStyle().
				Background(w.theme.ColorInfo).
				Foreground(lipgloss.Color("#ffffff")).
				Padding(0, 2).
				Bold(true)
			tabs = append(tabs, style.Render(tab))
		} else {
			// Inactive tab
			style := lipgloss.NewStyle().
				Background(w.theme.ColorDim).
				Foreground(lipgloss.Color("#ffffff")).
				Padding(0, 2)
			tabs = append(tabs, style.Render(tab))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

// renderSection renders a configuration section
func (w *ConfigEditorWidget) renderSection(section ConfigSection) string {
	var lines []string

	lines = append(lines, w.theme.Section.Render(section.Title))
	lines = append(lines, "")

	fieldOffset := w.getFieldOffset(w.currentSection)

	for i, field := range section.Fields {
		globalIndex := fieldOffset + i
		lines = append(lines, w.renderField(field, globalIndex))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderField renders a single configuration field
func (w *ConfigEditorWidget) renderField(field ConfigField, globalIndex int) string {
	var parts []string

	// Label
	label := field.Label
	if field.IsRequired {
		label += " *"
	}
	parts = append(parts, w.theme.Info.Render(label+":"))

	// Input field
	if globalIndex < len(w.inputs) {
		input := w.inputs[globalIndex]

		// Style based on focus and validation
		var inputStyle lipgloss.Style
		if w.editing && w.cursorPos == globalIndex {
			inputStyle = w.theme.InputFocused
		} else if w.cursorPos == globalIndex {
			inputStyle = w.theme.InputBlurred.Copy().BorderForeground(w.theme.ColorInfo)
		} else {
			inputStyle = w.theme.InputBlurred
		}

		// Check for validation error
		if errMsg, hasError := w.validationErrors[field.Key]; hasError {
			inputStyle = inputStyle.Copy().BorderForeground(w.theme.ColorError)
			parts = append(parts, inputStyle.Render(input.View()))
			parts = append(parts, w.theme.Error.Render(fmt.Sprintf("  ✗ %s", errMsg)))
		} else {
			parts = append(parts, inputStyle.Render(input.View()))
		}

		// Help text
		if field.HelpText != "" && w.cursorPos == globalIndex {
			parts = append(parts, w.theme.Description.Render(fmt.Sprintf("  ℹ %s", field.HelpText)))
		}
	}

	return strings.Join(parts, "\n")
}

// renderHelp renders the help text
func (w *ConfigEditorWidget) renderHelp() string {
	var lines []string

	lines = append(lines, w.theme.Section.Render("Help:"))
	lines = append(lines, "")
	lines = append(lines, "  ↑/k, ↓/j     Navigate fields")
	lines = append(lines, "  ←/h, →/l     Switch sections")
	lines = append(lines, "  enter/e      Edit field")
	lines = append(lines, "  esc          Cancel editing")
	lines = append(lines, "  ?            Toggle help")
	lines = append(lines, "  q            Quit")

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// AddSection adds a configuration section
func (w *ConfigEditorWidget) AddSection(section ConfigSection) {
	w.sections = append(w.sections, section)
	w.rebuildInputs()
}

// SetSections sets all configuration sections
func (w *ConfigEditorWidget) SetSections(sections []ConfigSection) {
	w.sections = sections
	w.rebuildInputs()
}

// rebuildInputs creates text inputs for all fields
func (w *ConfigEditorWidget) rebuildInputs() {
	w.inputs = []textinput.Model{}

	for _, section := range w.sections {
		for _, field := range section.Fields {
			input := textinput.New()
			input.Placeholder = field.Placeholder
			input.SetValue(field.Value)
			input.Width = 50

			if field.IsPassword && !w.showPasswords {
				input.EchoMode = textinput.EchoPassword
				input.EchoCharacter = '•'
			}

			w.inputs = append(w.inputs, input)
		}
	}

	// Focus first input
	if len(w.inputs) > 0 {
		w.inputs[0].Focus()
	}
}

// moveCursorUp moves the cursor to the previous field
func (w *ConfigEditorWidget) moveCursorUp() {
	if w.cursorPos > 0 {
		w.inputs[w.cursorPos].Blur()
		w.cursorPos--
		w.inputs[w.cursorPos].Focus()
		w.updateCurrentSection()
	}
}

// moveCursorDown moves the cursor to the next field
func (w *ConfigEditorWidget) moveCursorDown() {
	if w.cursorPos < len(w.inputs)-1 {
		w.inputs[w.cursorPos].Blur()
		w.cursorPos++
		w.inputs[w.cursorPos].Focus()
		w.updateCurrentSection()
	}
}

// previousSection switches to the previous section
func (w *ConfigEditorWidget) previousSection() {
	if w.currentSection > 0 {
		w.currentSection--
		w.cursorPos = w.getFieldOffset(w.currentSection)
		w.focusCurrentInput()
	}
}

// nextSection switches to the next section
func (w *ConfigEditorWidget) nextSection() {
	if w.currentSection < len(w.sections)-1 {
		w.currentSection++
		w.cursorPos = w.getFieldOffset(w.currentSection)
		w.focusCurrentInput()
	}
}

// startEditing starts editing the current field
func (w *ConfigEditorWidget) startEditing() {
	w.editing = true
	if w.cursorPos < len(w.inputs) {
		w.inputs[w.cursorPos].Focus()
	}
}

// saveCurrentField saves the current field value
func (w *ConfigEditorWidget) saveCurrentField() {
	if w.cursorPos >= len(w.inputs) {
		return
	}

	// Get the field
	sectionIdx, fieldIdx := w.getFieldIndices(w.cursorPos)
	if sectionIdx < 0 || fieldIdx < 0 {
		return
	}

	field := w.sections[sectionIdx].Fields[fieldIdx]
	value := w.inputs[w.cursorPos].Value()

	// Validate
	if field.Validator != nil {
		if err := field.Validator(value); err != nil {
			w.validationErrors[field.Key] = err.Error()
			return
		}
	}

	// Clear validation error
	delete(w.validationErrors, field.Key)

	// Save value
	w.sections[sectionIdx].Fields[fieldIdx].Value = value
}

// updateCurrentSection updates the current section based on cursor position
func (w *ConfigEditorWidget) updateCurrentSection() {
	for i := range w.sections {
		offset := w.getFieldOffset(i)
		numFields := len(w.sections[i].Fields)
		if w.cursorPos >= offset && w.cursorPos < offset+numFields {
			w.currentSection = i
			return
		}
	}
}

// getFieldOffset returns the global field index offset for a section
func (w *ConfigEditorWidget) getFieldOffset(sectionIdx int) int {
	offset := 0
	for i := 0; i < sectionIdx && i < len(w.sections); i++ {
		offset += len(w.sections[i].Fields)
	}
	return offset
}

// getFieldIndices returns the section and field indices for a global field index
func (w *ConfigEditorWidget) getFieldIndices(globalIdx int) (int, int) {
	offset := 0
	for i, section := range w.sections {
		numFields := len(section.Fields)
		if globalIdx >= offset && globalIdx < offset+numFields {
			return i, globalIdx - offset
		}
		offset += numFields
	}
	return -1, -1
}

// focusCurrentInput focuses the current input
func (w *ConfigEditorWidget) focusCurrentInput() {
	for i := range w.inputs {
		w.inputs[i].Blur()
	}
	if w.cursorPos < len(w.inputs) {
		w.inputs[w.cursorPos].Focus()
	}
}

// GetValue returns the value of a configuration field by key
func (w *ConfigEditorWidget) GetValue(key string) string {
	for _, section := range w.sections {
		for _, field := range section.Fields {
			if field.Key == key {
				return field.Value
			}
		}
	}
	return ""
}

// SetValue sets the value of a configuration field by key
func (w *ConfigEditorWidget) SetValue(key, value string) {
	for i, section := range w.sections {
		for j, field := range section.Fields {
			if field.Key == key {
				w.sections[i].Fields[j].Value = value
				w.rebuildInputs()
				return
			}
		}
	}
}

// GetAllValues returns all configuration values as a map
func (w *ConfigEditorWidget) GetAllValues() map[string]string {
	values := make(map[string]string)
	for _, section := range w.sections {
		for _, field := range section.Fields {
			values[field.Key] = field.Value
		}
	}
	return values
}

// Validate validates all fields
func (w *ConfigEditorWidget) Validate() bool {
	w.validationErrors = make(map[string]string)
	valid := true

	for _, section := range w.sections {
		for _, field := range section.Fields {
			// Check required fields
			if field.IsRequired && field.Value == "" {
				w.validationErrors[field.Key] = "This field is required"
				valid = false
				continue
			}

			// Run custom validator
			if field.Validator != nil {
				if err := field.Validator(field.Value); err != nil {
					w.validationErrors[field.Key] = err.Error()
					valid = false
				}
			}
		}
	}

	return valid
}

// HasErrors returns true if there are validation errors
func (w *ConfigEditorWidget) HasErrors() bool {
	return len(w.validationErrors) > 0
}

// GetErrors returns the current validation errors
func (w *ConfigEditorWidget) GetErrors() map[string]string {
	return w.validationErrors
}

// Focused returns true if the editor is currently editing a field
func (w *ConfigEditorWidget) Focused() bool {
	return w.editing
}

// TogglePasswordVisibility toggles whether password fields show their content
func (w *ConfigEditorWidget) TogglePasswordVisibility() {
	w.showPasswords = !w.showPasswords
	// Save current values before rebuilding
	for i, section := range w.sections {
		for j := range section.Fields {
			offset := w.getFieldOffset(i)
			idx := offset + j
			if idx < len(w.inputs) {
				w.sections[i].Fields[j].Value = w.inputs[idx].Value()
			}
		}
	}
	w.rebuildInputs()
}

// ShowingPasswords returns whether password fields are currently visible
func (w *ConfigEditorWidget) ShowingPasswords() bool {
	return w.showPasswords
}
