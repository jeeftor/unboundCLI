package widgets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

// ColumnConfig defines column sizing behavior
type ColumnConfig struct {
	Title    string
	MinWidth int     // Minimum width to be useful
	Priority int     // 1=critical, 2=important, 3=optional
	FlexGrow float64 // Growth factor when space available
}

// SortMode represents different sorting modes
type SortMode int

const (
	SortByHostname SortMode = iota
	SortByIP
	SortByStatus
)

// String returns the string representation of SortMode
func (s SortMode) String() string {
	switch s {
	case SortByHostname:
		return "Hostname"
	case SortByIP:
		return "IP Address"
	case SortByStatus:
		return "Status"
	default:
		return "Unknown"
	}
}

// TableWidget displays DNS sync status table with responsive column sizing
type TableWidget struct {
	BaseWidget

	// Data
	entries         []*models.Entry
	filteredRows    []table.Row
	filteredEntries []*models.Entry // Entries corresponding to filteredRows
	allRows         []table.Row

	// Table component
	table table.Model

	// Filter/search state
	filterMode    models.FilterMode
	searchQuery   string
	showSearchBar bool
	textInput     textinput.Model

	// Sort state
	sortMode SortMode

	// Selection state
	selectedIndices map[int]bool // Map of selected row indices in filteredEntries

	// Service loading status (for column dimming)
	serviceStatus ServiceLoadStatus

	// Column configuration
	columnConfigs []ColumnConfig
	columnWidths  []int

	// Layout
	theme *Theme
}

// NewTableWidget creates a new table widget
func NewTableWidget() *TableWidget {
	// Define column configurations - let BubbleTea handle the sizing
	configs := []ColumnConfig{
		{Title: "Hostname", MinWidth: 25, Priority: 1, FlexGrow: 3.0},
		{Title: "Source", MinWidth: 8, Priority: 2, FlexGrow: 0.5},
		{Title: "DNS", MinWidth: 15, Priority: 1, FlexGrow: 1.0},
		{Title: "Upstream", MinWidth: 22, Priority: 2, FlexGrow: 1.5},
		{Title: "DHCP", MinWidth: 15, Priority: 2, FlexGrow: 1.0},
		{Title: "Unbound", MinWidth: 8, Priority: 1, FlexGrow: 0.5},
		{Title: "AdGuard", MinWidth: 8, Priority: 1, FlexGrow: 0.5},
		{Title: "CF", MinWidth: 5, Priority: 3, FlexGrow: 0.3},
		{Title: "Status", MinWidth: 15, Priority: 1, FlexGrow: 1.0},
	}

	// Create text input for search
	ti := textinput.New()
	ti.Placeholder = "Search hostnames..."
	ti.CharLimit = 50
	ti.Width = 30

	// Create initial table with styles
	t := table.New(
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Set table styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(CurrentTheme.ColorInfo).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(CurrentTheme.ColorInfo).
		Bold(false)
	t.SetStyles(s)

	w := &TableWidget{
		BaseWidget:      NewBaseWidget(),
		entries:         []*models.Entry{},
		filterMode:      models.FilterNone,
		searchQuery:     "",
		showSearchBar:   false,
		textInput:       ti,
		sortMode:        SortByHostname,
		selectedIndices: make(map[int]bool),
		serviceStatus:   ServiceLoadStatus{},
		columnConfigs:   configs,
		table:           t,
		theme:           CurrentTheme,
	}

	return w
}

// Init initializes the table widget
func (w *TableWidget) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (w *TableWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	var cmd tea.Cmd

	// If search bar is active, route input to text input
	if w.showSearchBar {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				w.searchQuery = w.textInput.Value()
				w.showSearchBar = false
				w.applyFilters()
				return w, nil
			case "esc":
				w.showSearchBar = false
				w.textInput.SetValue("")
				return w, nil
			default:
				w.textInput, cmd = w.textInput.Update(msg)
				return w, cmd
			}
		}
	}

	// Handle selection keys when not in search mode
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			// Toggle selection of current row
			w.toggleSelection()
			w.rebuildTable() // Rebuild to update selection indicators
			return w, nil
		}
	}

	// Otherwise update table
	w.table, cmd = w.table.Update(msg)
	return w, cmd
}

// View renders the table widget
func (w *TableWidget) View() string {
	if w.width == 0 || w.height == 0 {
		return ""
	}

	var sections []string

	// Show loading message if no entries yet
	if len(w.entries) == 0 {
		loadingMsg := w.theme.Info.Render("⏳ Loading DNS entries...")
		sections = append(sections, loadingMsg)
	}

	// Filter, sort, and selection indicators
	var indicators []string
	if w.filterMode != models.FilterNone {
		indicators = append(indicators, fmt.Sprintf("Filter: %s", w.filterMode.String()))
	}
	if w.sortMode != SortByHostname {
		indicators = append(indicators, fmt.Sprintf("Sort: %s", w.sortMode.String()))
	}
	if w.HasSelections() {
		indicators = append(indicators, fmt.Sprintf("Selected: %d", w.GetSelectionCount()))
	}
	if len(indicators) > 0 {
		sections = append(sections, w.theme.Info.Render(strings.Join(indicators, " | ")))
	}

	// Search bar
	if w.showSearchBar {
		searchView := w.theme.InputPrompt.Render("Search: ") + w.textInput.View()
		sections = append(sections, searchView)
	} else if w.searchQuery != "" {
		searchLabel := fmt.Sprintf("Search: %s", w.searchQuery)
		sections = append(sections, w.theme.Info.Render(searchLabel))
	}

	// Table (only if we have entries)
	if len(w.entries) > 0 {
		// Update table style based on selected row's status (like Cairo example)
		cursor := w.table.Cursor()
		if cursor >= 0 && cursor < len(w.filteredEntries) {
			selectedEntry := w.filteredEntries[cursor]
			w.applyStyleForStatus(selectedEntry.OverallStatus)
		}
		sections = append(sections, w.table.View())
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Add border with simple title
	filterInfo := ""
	if w.filterMode != models.FilterNone {
		filterInfo = " - " + w.filterMode.String()
	}
	title := w.theme.Header.Foreground(w.theme.ColorCyan).Render("╭─ DNS Sync Status" + filterInfo + " ─")

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
		BorderForeground(w.theme.ColorCyan).
		Padding(0, 1).
		Width(w.width - 4)

	bordered := style.Render(content)

	// Prepend title line
	lines := strings.Split(bordered, "\n")
	if len(lines) > 0 {
		remainingWidth := w.width - lipgloss.Width(title) - 1
		if remainingWidth < 0 {
			remainingWidth = 0
		}
		titleLine := title + w.theme.Header.Foreground(w.theme.ColorCyan).Render(strings.Repeat("─", remainingWidth)+"╮")
		lines[0] = titleLine
	}

	return strings.Join(lines, "\n")
}

// applyStyleForStatus updates table styles based on entry status
func (w *TableWidget) applyStyleForStatus(status models.SyncStatus) {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(w.theme.ColorInfo).
		BorderBottom(true).
		Bold(true)

	// Change selected row color based on status
	switch status {
	case models.FullyInSync:
		// Green background for synced entries
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("22")). // Dark green
			Bold(false)
	case models.OutOfSync:
		// Red background for out of sync
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("88")). // Dark red
			Bold(false)
	case models.PartiallyInSync, models.DHCPMismatch:
		// Yellow background for warnings
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("130")). // Dark yellow/orange
			Bold(false)
	case models.CaddyOnly:
		// Orange background for Caddy only
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("130")). // Dark orange
			Bold(false)
	case models.Stale:
		// Red background for stale entries
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("88")). // Dark red
			Bold(false)
	default:
		// Default blue background
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(w.theme.ColorInfo).
			Bold(false)
	}

	w.table.SetStyles(s)
}

// SetSize updates the widget dimensions and recalculates column widths
func (w *TableWidget) SetSize(width, height int) {
	w.BaseWidget.SetSize(width, height)
	w.recalculateColumns()
}

// SetEntries updates the entries and rebuilds the table
func (w *TableWidget) SetEntries(entries []*models.Entry) {
	w.entries = entries
	w.rebuildTable()
}

// SetFilter sets the filter mode and applies it
func (w *TableWidget) SetFilter(mode models.FilterMode) {
	w.filterMode = mode
	w.ClearSelection() // Clear selections when filter changes
	w.applyFilters()
}

// GetFilterMode returns the current filter mode
func (w *TableWidget) GetFilterMode() models.FilterMode {
	return w.filterMode
}

// ClearFilter clears the current filter
func (w *TableWidget) ClearFilter() {
	w.filterMode = models.FilterNone
	w.searchQuery = ""
	w.ClearSelection() // Clear selections when filter changes
	w.applyFilters()
}

// ShowSearch shows the search input bar
func (w *TableWidget) ShowSearch() {
	w.showSearchBar = true
	w.textInput.SetValue("")
	w.textInput.Focus()
}

// HideSearch hides the search input bar
func (w *TableWidget) HideSearch() {
	w.showSearchBar = false
	w.textInput.Blur()
}

// IsSearching returns true if the search bar is currently active
func (w *TableWidget) IsSearching() bool {
	return w.showSearchBar
}

// SetServiceStatus updates which services have been loaded
func (w *TableWidget) SetServiceStatus(status ServiceLoadStatus) {
	w.serviceStatus = status
	w.rebuildTable()
}

// recalculateColumns computes column widths based on terminal width
func (w *TableWidget) recalculateColumns() {
	if w.width == 0 {
		return
	}

	w.columnWidths = calculateColumnWidths(w.width, w.columnConfigs)

	// Build table columns
	columns := make([]table.Column, 0)
	for i, config := range w.columnConfigs {
		if w.columnWidths[i] > 0 {
			columns = append(columns, table.Column{
				Title: config.Title,
				Width: w.columnWidths[i],
			})
		}
	}

	w.table.SetColumns(columns)

	// Set table dimensions
	tableHeight := w.height
	if w.filterMode != models.FilterNone {
		tableHeight-- // Reserve line for filter indicator
	}
	if w.showSearchBar || w.searchQuery != "" {
		tableHeight-- // Reserve line for search bar
	}
	if tableHeight < 5 {
		tableHeight = 5 // Minimum height
	}

	w.table.SetHeight(tableHeight)
	w.table.SetWidth(w.width)

	// Rebuild rows with new widths
	w.rebuildTable()
}

// rebuildTable rebuilds all table rows
func (w *TableWidget) rebuildTable() {
	w.allRows = make([]table.Row, 0, len(w.entries))

	for _, entry := range w.entries {
		row := w.buildRow(entry)
		w.allRows = append(w.allRows, row)
	}

	w.applyFilters()
}

// getRowStyle returns the style for a row based on entry status
func (w *TableWidget) getRowStyle(status models.SyncStatus) lipgloss.Style {
	baseStyle := lipgloss.NewStyle()

	switch status {
	case models.FullyInSync:
		// Subtle green background
		return baseStyle.Background(lipgloss.Color("#1a2e1a"))
	case models.PartiallyInSync:
		// Subtle yellow background
		return baseStyle.Background(lipgloss.Color("#2e2a1a"))
	case models.OutOfSync:
		// Subtle red background
		return baseStyle.Background(lipgloss.Color("#2e1a1a"))
	case models.CaddyOnly:
		// Subtle orange background
		return baseStyle.Background(lipgloss.Color("#2e231a"))
	case models.Stale:
		// Subtle red background
		return baseStyle.Background(lipgloss.Color("#2e1a1a"))
	case models.DHCPMismatch:
		// Subtle yellow background
		return baseStyle.Background(lipgloss.Color("#2e2a1a"))
	default:
		// No background
		return baseStyle
	}
}

// buildRow creates a table row from an entry
func (w *TableWidget) buildRow(entry *models.Entry) table.Row {
	row := make(table.Row, 0, len(w.columnConfigs))

	// Find if this entry is selected
	entryIdx := -1
	for i, e := range w.filteredEntries {
		if e == entry {
			entryIdx = i
			break
		}
	}
	isSelected := entryIdx >= 0 && w.selectedIndices[entryIdx]

	for i, config := range w.columnConfigs {
		// Skip hidden columns
		if w.columnWidths[i] == 0 {
			continue
		}

		var cell string

		switch config.Title {
		case "Hostname":
			// Add selection indicator
			selectionMark := " "
			if isSelected {
				selectionMark = "✓"
			}
			hostname := selectionMark + " " + entry.Hostname
			cell = w.truncate(hostname, w.columnWidths[i])

		case "Source":
			cell = w.truncate(entry.DataSource, w.columnWidths[i])

		case "DNS":
			if entry.DNSResolved == "" || entry.DNSResolved == "NONE" {
				cell = "-"
			} else if entry.DNSResolved == "FAIL" {
				cell = "FAIL"
			} else {
				cell = w.truncate(entry.DNSResolved, w.columnWidths[i])
			}

		case "Upstream":
			if entry.CaddyUpstream == "" {
				cell = "-"
			} else {
				cell = w.truncate(entry.CaddyUpstream, w.columnWidths[i])
			}

		case "DHCP":
			if !w.serviceStatus.DHCP {
				cell = "Loading..."
			} else if !entry.DHCPStatus.Configured {
				cell = "No lease"
			} else {
				leaseType := entry.DHCPStatus.Type
				if entry.DHCPStatus.InSync {
					cell = fmt.Sprintf("OK %s", leaseType)
				} else {
					cell = fmt.Sprintf("!! %s", leaseType)
				}
			}

		case "Unbound":
			if !w.serviceStatus.Unbound {
				cell = " .."
			} else {
				cell = w.formatServiceStatus(entry.UnboundStatus, entry.IsConfiguredInCaddy())
			}

		case "AdGuard":
			if !w.serviceStatus.AdGuard {
				cell = " .."
			} else {
				cell = w.formatServiceStatus(entry.AdguardStatus, entry.IsConfiguredInCaddy())
			}

		case "CF":
			cell = "-"

		case "Status":
			cell = w.formatOverallStatus(entry.OverallStatus)
		}

		row = append(row, cell)
	}

	return row
}

// formatStatusIndicator returns a colored status indicator
func (w *TableWidget) formatStatusIndicator(status models.SyncStatus) string {
	indicator := "●"

	switch status {
	case models.FullyInSync:
		return w.theme.Success.Render(indicator)
	case models.PartiallyInSync:
		return w.theme.Warning.Render(indicator)
	case models.OutOfSync:
		return w.theme.Error.Render(indicator)
	case models.CaddyOnly:
		return w.theme.Warning.Render(indicator)
	case models.Stale:
		return w.theme.Error.Render(indicator)
	case models.DHCPMismatch:
		return w.theme.Warning.Render(indicator)
	default:
		return w.theme.Dimmed.Render(indicator)
	}
}

// formatServiceStatus formats a service status cell (text-based, no emojis)
func (w *TableWidget) formatServiceStatus(status models.ServiceStatus, inCaddy bool) string {
	if !inCaddy {
		// Entry not in Caddy
		if !status.Configured {
			return " OK" // Good - not configured
		}
		return " RM" // Bad - should be removed
	}

	// Entry is in Caddy
	if !status.Configured {
		return " NO" // Missing
	}
	if status.InSync {
		return " OK" // Synced
	}
	return " !!" // Out of sync
}

// formatOverallStatus formats the overall status cell
func (w *TableWidget) formatOverallStatus(status models.SyncStatus) string {
	icon := status.Icon()
	label := status.Label()
	return icon + " " + label
}

// truncate truncates a string to fit within width, adding "..." if needed
func (w *TableWidget) truncate(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

// applyFilters applies the current filter, search query, and sorting
func (w *TableWidget) applyFilters() {
	w.filteredRows = make([]table.Row, 0, len(w.allRows))
	w.filteredEntries = make([]*models.Entry, 0, len(w.entries))

	for i, entry := range w.entries {
		// Apply filter mode
		if !models.ApplyFilter(entry, w.filterMode) {
			continue
		}

		// Apply search query
		if w.searchQuery != "" {
			if !strings.Contains(strings.ToLower(entry.Hostname), strings.ToLower(w.searchQuery)) {
				continue
			}
		}

		w.filteredRows = append(w.filteredRows, w.allRows[i])
		w.filteredEntries = append(w.filteredEntries, entry)
	}

	// Apply sorting
	w.sortEntries()

	w.table.SetRows(w.filteredRows)
}

// GetSelectedEntry returns the currently selected entry (cursor position)
func (w *TableWidget) GetSelectedEntry() *models.Entry {
	cursor := w.table.Cursor()
	if cursor < 0 || cursor >= len(w.filteredEntries) {
		return nil
	}

	return w.filteredEntries[cursor]
}

// toggleSelection toggles the selection state of the current row
func (w *TableWidget) toggleSelection() {
	cursor := w.table.Cursor()
	if cursor < 0 || cursor >= len(w.filteredEntries) {
		return
	}

	if w.selectedIndices[cursor] {
		delete(w.selectedIndices, cursor)
	} else {
		w.selectedIndices[cursor] = true
	}
}

// GetSelectedEntries returns all selected entries in order
func (w *TableWidget) GetSelectedEntries() []*models.Entry {
	// Collect indices and sort them to ensure consistent order
	indices := make([]int, 0, len(w.selectedIndices))
	for idx := range w.selectedIndices {
		if idx >= 0 && idx < len(w.filteredEntries) {
			indices = append(indices, idx)
		}
	}

	// Sort indices
	for i := 0; i < len(indices)-1; i++ {
		for j := i + 1; j < len(indices); j++ {
			if indices[i] > indices[j] {
				indices[i], indices[j] = indices[j], indices[i]
			}
		}
	}

	// Build selected entries in order
	selected := make([]*models.Entry, 0, len(indices))
	for _, idx := range indices {
		selected = append(selected, w.filteredEntries[idx])
	}
	return selected
}

// ClearSelection clears all selections
func (w *TableWidget) ClearSelection() {
	w.selectedIndices = make(map[int]bool)
}

// SelectAll selects all visible (filtered) entries
func (w *TableWidget) SelectAll() {
	w.selectedIndices = make(map[int]bool)
	for i := range w.filteredEntries {
		w.selectedIndices[i] = true
	}
}

// HasSelections returns true if any items are selected
func (w *TableWidget) HasSelections() bool {
	return len(w.selectedIndices) > 0
}

// GetSelectionCount returns the number of selected items
func (w *TableWidget) GetSelectionCount() int {
	return len(w.selectedIndices)
}

// CycleSort cycles through the available sort modes
func (w *TableWidget) CycleSort() {
	switch w.sortMode {
	case SortByHostname:
		w.sortMode = SortByIP
	case SortByIP:
		w.sortMode = SortByStatus
	case SortByStatus:
		w.sortMode = SortByHostname
	}
	w.applyFilters()
}

// sortEntries sorts the filtered entries based on the current sort mode
func (w *TableWidget) sortEntries() {
	if len(w.filteredEntries) == 0 {
		return
	}

	// Create a slice of indices to sort
	indices := make([]int, len(w.filteredEntries))
	for i := range indices {
		indices[i] = i
	}

	// Sort based on mode
	switch w.sortMode {
	case SortByHostname:
		// Sort by hostname (case-insensitive)
		for i := 0; i < len(indices)-1; i++ {
			for j := i + 1; j < len(indices); j++ {
				if strings.ToLower(w.filteredEntries[indices[i]].Hostname) > strings.ToLower(w.filteredEntries[indices[j]].Hostname) {
					indices[i], indices[j] = indices[j], indices[i]
				}
			}
		}

	case SortByIP:
		// Sort by IP address (CaddyUpstream)
		for i := 0; i < len(indices)-1; i++ {
			for j := i + 1; j < len(indices); j++ {
				ip1 := w.filteredEntries[indices[i]].CaddyUpstream
				ip2 := w.filteredEntries[indices[j]].CaddyUpstream
				if ip1 > ip2 {
					indices[i], indices[j] = indices[j], indices[i]
				}
			}
		}

	case SortByStatus:
		// Sort by status (priority: OutOfSync, Stale, CaddyOnly, PartiallyInSync, DHCPMismatch, FullyInSync)
		statusOrder := map[models.SyncStatus]int{
			models.OutOfSync:       0,
			models.Stale:           1,
			models.CaddyOnly:       2,
			models.PartiallyInSync: 3,
			models.DHCPMismatch:    4,
			models.FullyInSync:     5,
		}
		for i := 0; i < len(indices)-1; i++ {
			for j := i + 1; j < len(indices); j++ {
				status1 := statusOrder[w.filteredEntries[indices[i]].OverallStatus]
				status2 := statusOrder[w.filteredEntries[indices[j]].OverallStatus]
				if status1 > status2 {
					indices[i], indices[j] = indices[j], indices[i]
				}
			}
		}
	}

	// Reorder filteredEntries and filteredRows based on sorted indices
	sortedEntries := make([]*models.Entry, len(w.filteredEntries))
	sortedRows := make([]table.Row, len(w.filteredRows))
	for i, idx := range indices {
		sortedEntries[i] = w.filteredEntries[idx]
		sortedRows[i] = w.filteredRows[idx]
	}

	w.filteredEntries = sortedEntries
	w.filteredRows = sortedRows
}

// calculateColumnWidths distributes terminal width across columns
func calculateColumnWidths(termWidth int, configs []ColumnConfig) []int {
	numCols := len(configs)
	if numCols == 0 {
		return []int{}
	}

	// Calculate space needed for borders/padding (3 chars per column: " | ")
	overhead := numCols * 3

	// Start with minimum widths
	available := termWidth - overhead
	widths := make([]int, numCols)
	totalMin := 0

	for i, cfg := range configs {
		widths[i] = cfg.MinWidth
		totalMin += cfg.MinWidth
	}

	available -= totalMin

	// If we have extra space, distribute by FlexGrow
	if available > 0 {
		totalFlex := 0.0
		for _, cfg := range configs {
			totalFlex += cfg.FlexGrow
		}

		if totalFlex > 0 {
			for i, cfg := range configs {
				extra := int(float64(available) * (cfg.FlexGrow / totalFlex))
				widths[i] += extra
			}
		}
	}

	// If we're short on space, hide low-priority columns
	if available < 0 {
		// Hide Priority 3 columns first, then 2, but never 1
		for priority := 3; priority >= 2 && available < 0; priority-- {
			for i, cfg := range configs {
				if cfg.Priority == priority && widths[i] > 0 {
					available += widths[i] + 3 // Reclaim space
					widths[i] = 0              // Hide this column
				}
			}
		}
	}

	return widths
}
