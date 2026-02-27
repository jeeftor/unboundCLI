package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
	"github.com/jeeftor/caddy-dns-sync/internal/widgets"
)

// ViewMode represents the current view mode
type ViewMode int

const (
	ViewModeTable  ViewMode = iota // Main table view
	ViewModeSync                   // Sync dialog view
	ViewModeConfig                 // Configuration editor view
)

// AppModel is the main TUI application model
type AppModel struct {
	// Widgets
	statusWidget *widgets.StatusWidget
	tableWidget  *widgets.TableWidget
	helpWidget   *widgets.HelpWidget
	logWidget    *widgets.LogWidget
	syncDialog   *widgets.SyncDialog
	configEditor *widgets.ConfigEditorWidget

	// API Clients
	caddyClient   *api.CaddyClient
	unboundClient *api.Client
	adguardClient *api.AdguardClient
	dnsmasqClient *api.DNSMasqClient

	// Data
	entries       []*models.Entry
	caddyServerIP string

	// State
	currentView ViewMode
	loading     bool
	err         error
	quitting    bool

	// Terminal dimensions
	width  int
	height int
}

// NewAppModel creates a new TUI application model
func NewAppModel(
	caddyClient *api.CaddyClient,
	unboundClient *api.Client,
	adguardClient *api.AdguardClient,
	dnsmasqClient *api.DNSMasqClient,
	caddyServerIP string,
) *AppModel {
	return &AppModel{
		statusWidget:  widgets.NewStatusWidget(),
		tableWidget:   widgets.NewTableWidget(),
		helpWidget:    widgets.NewHelpWidget(),
		logWidget:     widgets.NewLogWidget(),
		syncDialog:    widgets.NewSyncDialog("DNS Services"),
		configEditor:  widgets.NewConfigEditor(),
		caddyClient:   caddyClient,
		unboundClient: unboundClient,
		adguardClient: adguardClient,
		dnsmasqClient: dnsmasqClient,
		caddyServerIP: caddyServerIP,
		currentView:   ViewModeTable,
		loading:       false,
		entries:       []*models.Entry{},
	}
}

// Init initializes the application
func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.statusWidget.Init(),
		m.tableWidget.Init(),
		m.helpWidget.Init(),
		m.logWidget.Init(),
		m.syncDialog.Init(),
		m.configEditor.Init(),
		m.loadData(),
	)
}

// Update handles messages and updates the model
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "q" && m.currentView == ViewModeTable {
			m.quitting = true
			return m, tea.Quit
		}

		// Route based on current view
		switch m.currentView {
		case ViewModeTable:
			return m.handleTableViewKeys(msg)
		case ViewModeSync:
			return m.handleSyncViewKeys(msg)
		case ViewModeConfig:
			return m.handleConfigViewKeys(msg)
		}

	case dataLoadedMsg:
		m.loading = false
		m.entries = msg.entries
		m.statusWidget.SetLoading(false)
		m.statusWidget.SetEntries(m.entries)
		m.tableWidget.SetEntries(m.entries)
		m.updateServiceStatus()

	case dataLoadErrorMsg:
		m.loading = false
		m.err = msg.err
		m.statusWidget.SetLoading(false)

	case serviceLoadedMsg:
		m.updateServiceStatus()
		m.statusWidget.SetLoadingPhase(msg.phase)

	case dnsProgressMsg:
		m.statusWidget.SetProgress(msg.completed, msg.total)
	}

	// Update widgets based on current view
	switch m.currentView {
	case ViewModeTable:
		var cmd tea.Cmd
		var widget widgets.Widget

		widget, cmd = m.statusWidget.Update(msg)
		m.statusWidget = widget.(*widgets.StatusWidget)
		cmds = append(cmds, cmd)

		widget, cmd = m.tableWidget.Update(msg)
		m.tableWidget = widget.(*widgets.TableWidget)
		cmds = append(cmds, cmd)

		widget, cmd = m.helpWidget.Update(msg)
		m.helpWidget = widget.(*widgets.HelpWidget)
		cmds = append(cmds, cmd)

		widget, cmd = m.logWidget.Update(msg)
		m.logWidget = widget.(*widgets.LogWidget)
		cmds = append(cmds, cmd)

	case ViewModeSync:
		var cmd tea.Cmd
		var widget widgets.Widget

		widget, cmd = m.syncDialog.Update(msg)
		m.syncDialog = widget.(*widgets.SyncDialog)
		cmds = append(cmds, cmd)

		// Check if sync is done
		if m.syncDialog.IsDone() {
			m.currentView = ViewModeTable
			// Reload data after sync
			cmds = append(cmds, m.loadData())
		}

	case ViewModeConfig:
		var cmd tea.Cmd
		var widget widgets.Widget

		widget, cmd = m.configEditor.Update(msg)
		m.configEditor = widget.(*widgets.ConfigEditorWidget)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the application
func (m *AppModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	switch m.currentView {
	case ViewModeTable:
		return m.renderTableView()
	case ViewModeSync:
		return m.renderSyncView()
	case ViewModeConfig:
		return m.renderConfigView()
	default:
		return "Unknown view"
	}
}

// renderTableView renders the main table view
func (m *AppModel) renderTableView() string {
	sections := []string{}

	// Status widget at top
	if m.statusWidget != nil {
		statusView := m.statusWidget.View()
		if statusView != "" {
			sections = append(sections, statusView)
			sections = append(sections, "")
		}
	}

	// Log widget (if visible) or Table widget
	if m.logWidget != nil && m.logWidget.IsVisible() {
		logView := m.logWidget.View()
		if logView != "" {
			sections = append(sections, logView)
		}
	} else if m.tableWidget != nil {
		tableView := m.tableWidget.View()
		if tableView != "" {
			sections = append(sections, tableView)
		}
	}

	// Help widget at bottom
	if m.helpWidget != nil {
		sections = append(sections, "")
		helpView := m.helpWidget.View()
		if helpView != "" {
			sections = append(sections, helpView)
		}
	}

	// Error display if any
	if m.err != nil {
		sections = append(sections, "")
		errorStyle := lipgloss.NewStyle().Foreground(widgets.CurrentTheme.ColorError)
		sections = append(sections, errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderSyncView renders the sync dialog view
func (m *AppModel) renderSyncView() string {
	if m.syncDialog == nil {
		return "Sync dialog not initialized"
	}

	// Center the sync dialog
	dialogView := m.syncDialog.View()

	// Create a centered view
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	return style.Render(dialogView)
}

// renderConfigView renders the configuration editor view
func (m *AppModel) renderConfigView() string {
	if m.configEditor == nil {
		return "Config editor not initialized"
	}

	return m.configEditor.View()
}

// handleTableViewKeys handles keyboard input in table view
func (m *AppModel) handleTableViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If table widget is in search mode, let it handle ALL keys
	if m.tableWidget.IsSearching() {
		var cmd tea.Cmd
		var widget widgets.Widget
		widget, cmd = m.tableWidget.Update(msg)
		m.tableWidget = widget.(*widgets.TableWidget)
		return m, cmd
	}

	switch {
	case msg.String() == "s":
		// Show sync dialog for selected entries (or all if none selected)
		m.showSyncDialog()
		m.currentView = ViewModeSync
		return m, nil

	case msg.String() == "o", msg.String() == "O":
		// Sync only the currently highlighted entry (ignores selections)
		m.showSingleEntrySync()
		m.currentView = ViewModeSync
		return m, nil

	case msg.String() == "r":
		// Refresh data
		return m, m.loadData()

	case msg.String() == "C":
		// Show config editor
		m.initializeConfigEditor()
		m.currentView = ViewModeConfig
		return m, nil

	case msg.String() == "f":
		// Cycle through filters
		m.cycleFilter()
		return m, nil

	case msg.String() == "c":
		// Clear filter
		m.tableWidget.ClearFilter()
		return m, nil

	case msg.String() == "/":
		// Show search
		m.tableWidget.ShowSearch()
		return m, nil

	case msg.String() == "t":
		// Cycle sort mode
		m.tableWidget.CycleSort()
		return m, nil

	case msg.String() == "i":
		// Toggle IP display (if implemented)
		return m, nil

	case msg.String() == "?":
		// Toggle help
		m.helpWidget.ToggleShowFull()
		return m, nil

	case msg.String() == "l", msg.String() == "L":
		// Toggle logs
		m.logWidget.Toggle()
		m.updateLayout()
		return m, nil

	default:
		// Route to log widget if visible, otherwise table widget
		var cmd tea.Cmd
		var widget widgets.Widget

		if m.logWidget.IsVisible() {
			widget, cmd = m.logWidget.Update(msg)
			m.logWidget = widget.(*widgets.LogWidget)
		} else {
			// Let table widget handle the key (it will handle search mode internally)
			widget, cmd = m.tableWidget.Update(msg)
			m.tableWidget = widget.(*widgets.TableWidget)
		}
		return m, cmd
	}
}

// handleSyncViewKeys handles keyboard input in sync dialog view
func (m *AppModel) handleSyncViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Let sync dialog handle all input
	var cmd tea.Cmd
	var widget widgets.Widget
	widget, cmd = m.syncDialog.Update(msg)
	m.syncDialog = widget.(*widgets.SyncDialog)

	// Check if we should exit sync view
	if m.syncDialog.IsDone() && msg.String() != "y" && msg.String() != "n" {
		m.currentView = ViewModeTable
		return m, m.loadData()
	}

	return m, cmd
}

// handleConfigViewKeys handles keyboard input in config editor view
func (m *AppModel) handleConfigViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// ESC to exit config editor
	if msg.String() == "esc" && !m.configEditor.Focused() {
		m.currentView = ViewModeTable
		return m, nil
	}

	// Let config editor handle input
	var cmd tea.Cmd
	var widget widgets.Widget
	widget, cmd = m.configEditor.Update(msg)
	m.configEditor = widget.(*widgets.ConfigEditorWidget)
	return m, cmd
}

// updateLayout updates widget sizes based on terminal dimensions
func (m *AppModel) updateLayout() {
	statusHeight := 10 // Max height for status widget when loading
	helpHeight := 4    // Height for help widget

	if !m.statusWidget.IsLoading() {
		statusHeight = 5 // Compact mode
	}

	tableHeight := m.height - statusHeight - helpHeight - 4 // 4 for spacing
	if tableHeight < 10 {
		tableHeight = 10 // Minimum table height
	}

	m.statusWidget.SetSize(m.width, statusHeight)
	m.tableWidget.SetSize(m.width, tableHeight)
	m.logWidget.SetSize(m.width, tableHeight) // Same size as table
	m.helpWidget.SetSize(m.width, helpHeight)
	m.syncDialog.SetSize(m.width, m.height)
	m.configEditor.SetSize(m.width, m.height)
}

// showSyncDialog prepares and shows the sync dialog for selected entries or all entries
func (m *AppModel) showSyncDialog() {
	// Create sync executor with API clients
	executor := NewTUISyncExecutor(m.unboundClient, m.adguardClient, m.dnsmasqClient)

	// Inject sync executor into dialog
	m.syncDialog.SetSyncExecutor(executor.ExecuteSyncActions)

	// Check if any entries are selected
	selectedEntries := m.tableWidget.GetSelectedEntries()
	if len(selectedEntries) > 0 {
		// Use selected entries
		m.syncDialog.AddActionsFromEntries(selectedEntries, "all", m.caddyServerIP)
	} else {
		// Use all entries if nothing selected
		m.syncDialog.AddActionsFromEntries(m.entries, "all", m.caddyServerIP)
	}
}

// showSingleEntrySync prepares and shows the sync dialog for the selected entry only
func (m *AppModel) showSingleEntrySync() {
	// Get the selected entry from the table
	entry := m.tableWidget.GetSelectedEntry()
	if entry == nil {
		// No entry selected, return without showing dialog
		m.currentView = ViewModeTable
		return
	}

	// Create sync executor with API clients
	executor := NewTUISyncExecutor(m.unboundClient, m.adguardClient, m.dnsmasqClient)

	// Inject sync executor into dialog
	m.syncDialog.SetSyncExecutor(executor.ExecuteSyncActions)

	// Generate actions for just this entry
	m.syncDialog.AddActionsFromEntries([]*models.Entry{entry}, "all", m.caddyServerIP)
}

// cycleFilter cycles through the available filters
func (m *AppModel) cycleFilter() {
	// Build filter list based on what's actually in the data
	filters := []models.FilterMode{
		models.FilterNone,
	}

	// Count entries by status to determine which filters to show
	var hasOutOfSync, hasCaddyOnly, hasStale, hasSynced bool
	for _, entry := range m.entries {
		switch entry.OverallStatus {
		case models.OutOfSync:
			hasOutOfSync = true
		case models.CaddyOnly:
			hasCaddyOnly = true
		case models.Stale:
			hasStale = true
		case models.FullyInSync:
			hasSynced = true
		}
	}

	// Add filters only if there are matching entries
	if hasSynced {
		// Add a "synced" filter by using FilterMismatches inverted logic
		// We'll need to add this to the filter modes
	}
	if hasOutOfSync {
		filters = append(filters, models.FilterOutOfSync)
	}
	if hasCaddyOnly {
		filters = append(filters, models.FilterCaddyOnly)
	}
	if hasStale {
		filters = append(filters, models.FilterStale)
	}

	// Always add these utility filters
	filters = append(filters, models.FilterMismatches)

	// Find current filter and cycle to next
	currentFilter := m.tableWidget.GetFilterMode()
	currentIdx := 0
	for i, f := range filters {
		if f == currentFilter {
			currentIdx = i
			break
		}
	}

	nextIdx := (currentIdx + 1) % len(filters)
	m.tableWidget.SetFilter(filters[nextIdx])
}

// initializeConfigEditor populates the config editor with configuration sections
func (m *AppModel) initializeConfigEditor() {
	// Create configuration sections
	sections := []widgets.ConfigSection{
		{
			Title: "Caddy Server",
			Fields: []widgets.ConfigField{
				{
					Key:         "caddy_ip",
					Label:       "Caddy Server IP",
					Value:       m.caddyServerIP,
					Placeholder: "192.168.1.15",
					IsRequired:  true,
					HelpText:    "IP address of the Caddy server (source of truth)",
				},
			},
		},
		{
			Title: "DNS Services",
			Fields: []widgets.ConfigField{
				{
					Key:         "unbound_enabled",
					Label:       "Unbound Enabled",
					Value:       fmt.Sprintf("%t", m.unboundClient != nil),
					Placeholder: "true",
					HelpText:    "Enable Unbound DNS service integration",
				},
				{
					Key:         "adguard_enabled",
					Label:       "AdGuard Enabled",
					Value:       fmt.Sprintf("%t", m.adguardClient != nil),
					Placeholder: "true",
					HelpText:    "Enable AdGuard Home integration",
				},
			},
		},
	}

	m.configEditor.SetSections(sections)
}

// updateServiceStatus updates which services have been loaded
func (m *AppModel) updateServiceStatus() {
	status := widgets.ServiceLoadStatus{
		Caddy:    m.caddyClient != nil,
		Unbound:  m.unboundClient != nil,
		AdGuard:  m.adguardClient != nil,
		DHCP:     m.dnsmasqClient != nil,
		Complete: !m.loading,
	}

	m.statusWidget.SetServiceStatus(status)
	m.tableWidget.SetServiceStatus(status)
}

// Messages for async operations

type dataLoadedMsg struct {
	entries []*models.Entry
}

type dataLoadErrorMsg struct {
	err error
}

type serviceLoadedMsg struct {
	service string
	phase   widgets.LoadingPhase
}

type dnsProgressMsg struct {
	completed int
	total     int
}

// loadData loads data from all API clients with progress updates
func (m *AppModel) loadData() tea.Cmd {
	// Mark as loading (this happens in Update() before calling loadData())
	m.loading = true
	m.statusWidget.SetLoading(true)
	m.statusWidget.SetLoadingPhase(widgets.PhaseCaddy)

	return func() tea.Msg {
		// Create data loader
		loader := NewDataLoader(
			m.caddyClient,
			m.unboundClient,
			m.adguardClient,
			m.dnsmasqClient,
			m.caddyServerIP,
		)

		// Load data
		entries, err := loader.LoadData()
		if err != nil {
			return dataLoadErrorMsg{err: err}
		}

		return dataLoadedMsg{entries: entries}
	}
}

// Error returns the current error state
func (m *AppModel) Error() error {
	return m.err
}

// AddLog adds a log entry to the log widget
func (m *AppModel) AddLog(level, message string) {
	if m.logWidget != nil {
		m.logWidget.AddLog(level, message)
	}
}

// LogWidget returns the log widget (for external logging integration)
func (m *AppModel) LogWidget() *widgets.LogWidget {
	return m.logWidget
}
