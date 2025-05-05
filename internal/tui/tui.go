package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/unboundCLI/internal/api"
)

// KeyMap defines the keybindings for the TUI
type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Add    key.Binding
	Edit   key.Binding
	Delete key.Binding
	Apply  key.Binding
	Quit   key.Binding
	Help   key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Apply: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "apply"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// ShortHelp returns keybinding help
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Add, k.Edit, k.Delete, k.Apply, k.Quit, k.Help}
}

// FullHelp returns the full set of keybindings
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},            // Navigation
		{k.Add, k.Edit, k.Delete}, // Actions
		{k.Apply},                 // Apply changes
		{k.Help, k.Quit},          // Utility
	}
}

// Model represents the state of the TUI application
type Model struct {
	keys       KeyMap
	help       help.Model
	table      table.Model
	overrides  []api.DNSOverride
	client     *api.Client
	status     string
	showHelp   bool
	pendingMsg string
}

// NewModel creates a new TUI model
func NewModel(client *api.Client) Model {
	keys := DefaultKeyMap()
	helpModel := help.New()
	helpModel.ShowAll = false

	// Get default styles
	styles := DefaultStyles()

	// Define table columns
	columns := []table.Column{
		{Title: "Host", Width: 15},
		{Title: "Domain", Width: 20},
		{Title: "Server", Width: 15},
		{Title: "Enabled", Width: 8},
		{Title: "Description", Width: 30},
	}

	// Create table
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Set table styles
	t.SetStyles(styles.TableStyles)

	return Model{
		keys:     keys,
		help:     helpModel,
		table:    t,
		client:   client,
		status:   "Ready",
		showHelp: false,
	}
}

// Start initializes and runs the TUI
func (m Model) Start() error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Init initializes the TUI model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchOverrides,
	)
}

// fetchOverrides fetches DNS overrides from the API
func (m Model) fetchOverrides() tea.Msg {
	overrides, err := m.client.GetOverrides()
	if err != nil {
		return errMsg{err: fmt.Errorf("error fetching overrides: %w", err)}
	}
	return overridesMsg{overrides: overrides}
}

// Update handles user input and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle key presses
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keys.Apply):
			m.status = "Applying changes..."
			return m, m.applyChanges

		case key.Matches(msg, m.keys.Add):
			// In a real implementation, this would open a form for adding a new override
			m.status = "Add functionality not implemented yet"
			return m, nil

		case key.Matches(msg, m.keys.Edit):
			// In a real implementation, this would open a form for editing the selected override
			if len(m.overrides) == 0 {
				m.status = "No overrides to edit"
				return m, nil
			}
			selected := m.table.SelectedRow()
			if selected[0] != "" {
				m.status = fmt.Sprintf("Editing %s.%s", selected[0], selected[1])
			}
			return m, nil

		case key.Matches(msg, m.keys.Delete):
			// In a real implementation, this would delete the selected override
			if len(m.overrides) == 0 {
				m.status = "No overrides to delete"
				return m, nil
			}
			selected := m.table.SelectedRow()
			if selected[0] != "" {
				m.status = fmt.Sprintf("Deleting %s.%s", selected[0], selected[1])
			}
			return m, nil
		}

	case overridesMsg:
		m.overrides = msg.overrides
		m.updateTable()
		m.status = fmt.Sprintf("Loaded %d overrides", len(m.overrides))
		return m, nil

	case errMsg:
		m.status = msg.err.Error()
		return m, nil

	case applyMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error applying changes: %v", msg.err)
		} else {
			m.status = "Changes applied successfully"
		}
		return m, m.fetchOverrides
	}

	// Handle table key presses
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the TUI
func (m Model) View() string {
	s := strings.Builder{}

	// Title
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7aa2f7")).
		Bold(true).
		Margin(1, 0, 1, 2).
		Render("Unbound DNS Overrides Manager")

	s.WriteString(title)
	s.WriteString("\n")

	// Table
	s.WriteString(m.table.View())
	s.WriteString("\n\n")

	// Status
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff"))
	s.WriteString(statusStyle.Render(fmt.Sprintf("Status: %s", m.status)))
	s.WriteString("\n\n")

	// Help
	helpView := m.help.View(m.keys)
	s.WriteString(helpView)

	return s.String()
}

// updateTable updates the table with the current overrides
func (m *Model) updateTable() {
	rows := []table.Row{}

	for _, o := range m.overrides {
		enabled := "No"
		if o.Enabled == "1" {
			enabled = "Yes"
		}

		rows = append(rows, table.Row{
			o.Host,
			o.Domain,
			o.Server,
			enabled,
			o.Description,
		})
	}

	m.table.SetRows(rows)
}

// applyChanges applies pending DNS changes
func (m Model) applyChanges() tea.Msg {
	err := m.client.ApplyChanges()
	return applyMsg{err: err}
}

// Message types
type overridesMsg struct {
	overrides []api.DNSOverride
}

type errMsg struct {
	err error
}

type applyMsg struct {
	err error
}
