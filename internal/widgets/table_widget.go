package widgets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
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

// cursorHighlightStyle is applied to the row under the cursor
var cursorHighlightStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#283457")).
	Foreground(lipgloss.Color("#ffffff"))

// TableWidget displays DNS sync status table with responsive column sizing
type TableWidget struct {
	BaseWidget

	// Data
	entries         []*models.Entry
	filteredRows    [][]string
	filteredEntries []*models.Entry // Entries corresponding to filteredRows
	allRows         [][]string

	// Scroll / cursor
	viewport viewport.Model
	cursor   int

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
	// Define column configurations
	configs := []ColumnConfig{
		{Title: "Hostname", MinWidth: 26, Priority: 1, FlexGrow: 3.0},
		{Title: "Source", MinWidth: 6, Priority: 2, FlexGrow: 0.3},
		{Title: "DNS", MinWidth: 13, Priority: 1, FlexGrow: 0.8},
		{Title: "Upstream", MinWidth: 20, Priority: 2, FlexGrow: 1.5},
		{Title: "DHCP", MinWidth: 9, Priority: 2, FlexGrow: 0.3},
		{Title: "Unbound", MinWidth: 7, Priority: 1, FlexGrow: 0.1},
		{Title: "AdGuard", MinWidth: 7, Priority: 1, FlexGrow: 0.1},
		{Title: "CF", MinWidth: 14, Priority: 3, FlexGrow: 0.8},
		{Title: "Status", MinWidth: 14, Priority: 1, FlexGrow: 1.0},
	}

	// Create text input for search
	ti := textinput.New()
	ti.Placeholder = "Search hostnames..."
	ti.CharLimit = 50
	ti.Width = 30

	vp := viewport.New(0, 10)

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
		viewport:        vp,
		cursor:          0,
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

	// Handle navigation and selection keys
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if w.cursor > 0 {
				w.cursor--
				w.syncViewport()
			}
			return w, nil
		case "down", "j":
			if w.cursor < len(w.filteredEntries)-1 {
				w.cursor++
				w.syncViewport()
			}
			return w, nil
		case "pgup":
			step := w.viewport.Height - 1
			if step < 1 {
				step = 1
			}
			w.cursor -= step
			if w.cursor < 0 {
				w.cursor = 0
			}
			w.syncViewport()
			return w, nil
		case "pgdown":
			step := w.viewport.Height - 1
			if step < 1 {
				step = 1
			}
			w.cursor += step
			if w.cursor >= len(w.filteredEntries) {
				w.cursor = len(w.filteredEntries) - 1
			}
			if w.cursor < 0 {
				w.cursor = 0
			}
			w.syncViewport()
			return w, nil
		case "home":
			w.cursor = 0
			w.syncViewport()
			return w, nil
		case "end":
			if len(w.filteredEntries) > 0 {
				w.cursor = len(w.filteredEntries) - 1
			}
			w.syncViewport()
			return w, nil
		case "enter", " ":
			w.toggleSelection()
			w.rebuildTable()
			return w, nil
		}
	}

	// Pass remaining messages to viewport (handles mouse wheel etc.)
	w.viewport, cmd = w.viewport.Update(msg)
	return w, cmd
}

// syncViewport ensures the viewport is scrolled so the cursor row is visible.
// Each data row is 1 line tall.
func (w *TableWidget) syncViewport() {
	if len(w.filteredEntries) == 0 {
		return
	}
	// cursor line offset within the viewport content (0-based)
	cursorLine := w.cursor
	top := w.viewport.YOffset
	bottom := top + w.viewport.Height - 1

	if cursorLine < top {
		w.viewport.YOffset = cursorLine
	} else if cursorLine > bottom {
		w.viewport.YOffset = cursorLine - w.viewport.Height + 1
	}
	if w.viewport.YOffset < 0 {
		w.viewport.YOffset = 0
	}

	// Refresh rendered content so the highlight moves
	w.viewport.SetContent(w.renderTableContent())
}

// renderTableContent builds the full table content string (header + rows)
// using lipgloss-aware cell sizing so ANSI-colored cells are handled correctly.
func (w *TableWidget) renderTableContent() string {
	if len(w.columnWidths) == 0 {
		return ""
	}

	dimColor := w.theme.ColorDim

	// Collect visible column widths and titles
	type visibleCol struct {
		title string
		width int
	}
	var cols []visibleCol
	for i, cfg := range w.columnConfigs {
		if i < len(w.columnWidths) && w.columnWidths[i] > 0 {
			cols = append(cols, visibleCol{title: cfg.Title, width: w.columnWidths[i]})
		}
	}
	if len(cols) == 0 {
		return ""
	}

	cellStyle := func(w int) lipgloss.Style {
		return lipgloss.NewStyle().Width(w).MaxWidth(w).Inline(true)
	}

	// Header row
	headerParts := make([]string, len(cols))
	for i, col := range cols {
		headerParts[i] = cellStyle(col.width).Bold(true).Foreground(lipgloss.Color("#7aa2f7")).Render(col.title)
	}
	header := strings.Join(headerParts, " ")

	// Separator line
	sepWidth := 0
	for _, col := range cols {
		sepWidth += col.width + 1 // +1 for the space separator
	}
	if sepWidth > 0 {
		sepWidth-- // no trailing separator
	}
	separator := lipgloss.NewStyle().Foreground(dimColor).Render(strings.Repeat("─", sepWidth))

	var lines []string
	lines = append(lines, header)
	lines = append(lines, separator)

	// Subtle row background colors by status (for non-cursor rows)
	rowBg := map[models.SyncStatus]lipgloss.Color{
		models.OutOfSync:       "#2a1a1a",
		models.Stale:           "#2a1a1a",
		models.CaddyOnly:       "#2a221a",
		models.PartiallyInSync: "#2a251a",
		models.DHCPMismatch:    "#2a251a",
	}

	// Data rows
	for rowIdx, row := range w.filteredRows {
		colIdx := 0
		parts := make([]string, 0, len(cols))
		for _, col := range cols {
			var cellVal string
			if colIdx < len(row) {
				cellVal = row[colIdx]
			}
			rendered := cellStyle(col.width).Render(cellVal)
			parts = append(parts, rendered)
			colIdx++
		}
		line := strings.Join(parts, " ")

		if rowIdx == w.cursor {
			// Cursor row: strong blue highlight
			line = cursorHighlightStyle.Width(sepWidth + 1).Render(line)
		} else if rowIdx < len(w.filteredEntries) {
			// Non-cursor rows: subtle status-based tint
			status := w.filteredEntries[rowIdx].OverallStatus
			if bg, ok := rowBg[status]; ok {
				line = lipgloss.NewStyle().Background(bg).Width(sepWidth + 1).Render(line)
			}
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// View renders the table widget
func (w *TableWidget) View() string {
	if w.width == 0 || w.height == 0 {
		return ""
	}

	var sections []string

	// Show loading message if no entries yet
	if len(w.entries) == 0 {
		loadingMsg := w.theme.Info.Render("Loading DNS entries...")
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

	// Table viewport (only if we have entries)
	if len(w.entries) > 0 {
		sections = append(sections, w.viewport.View())
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
		Width(w.width - 2)

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

// SetServiceStatus updates which services have been loaded.
// The CF column is only included in the layout when Cloudflare data is available.
func (w *TableWidget) SetServiceStatus(status ServiceLoadStatus) {
	w.serviceStatus = status
	w.rebuildColumnConfigs()
	w.rebuildTable()
}

// rebuildColumnConfigs rebuilds the column configuration based on which services are active.
func (w *TableWidget) rebuildColumnConfigs() {
	configs := []ColumnConfig{
		{Title: "Hostname", MinWidth: 26, Priority: 1, FlexGrow: 3.0},
		{Title: "Source", MinWidth: 6, Priority: 2, FlexGrow: 0.3},
		{Title: "DNS", MinWidth: 13, Priority: 1, FlexGrow: 0.8},
		{Title: "Upstream", MinWidth: 20, Priority: 2, FlexGrow: 1.5},
		{Title: "DHCP", MinWidth: 9, Priority: 2, FlexGrow: 0.3},
		{Title: "Unbound", MinWidth: 7, Priority: 1, FlexGrow: 0.1},
		{Title: "AdGuard", MinWidth: 7, Priority: 1, FlexGrow: 0.1},
	}
	if w.serviceStatus.Cloudflare {
		configs = append(configs, ColumnConfig{Title: "CF", MinWidth: 14, Priority: 3, FlexGrow: 0.8})
	}
	configs = append(configs, ColumnConfig{Title: "Status", MinWidth: 14, Priority: 1, FlexGrow: 1.0})
	w.columnConfigs = configs
}

// recalculateColumns computes column widths and updates viewport dimensions
func (w *TableWidget) recalculateColumns() {
	if w.width == 0 {
		return
	}

	w.columnWidths = calculateColumnWidths(w.width-4, w.columnConfigs)

	// Calculate viewport height — reserve lines for status/search indicators
	vpHeight := w.height
	if w.filterMode != models.FilterNone {
		vpHeight-- // Reserve line for filter indicator
	}
	if w.showSearchBar || w.searchQuery != "" {
		vpHeight-- // Reserve line for search bar
	}
	// Reserve 2 lines for the header row + separator inside viewport content
	// (viewport height = visible data rows, header/sep are part of content)
	if vpHeight < 5 {
		vpHeight = 5 // Minimum height
	}

	w.viewport.Width = w.width - 4
	w.viewport.Height = vpHeight

	// Rebuild rows with new widths then refresh viewport
	w.rebuildTable()
}

// rebuildTable rebuilds all table rows
func (w *TableWidget) rebuildTable() {
	w.allRows = make([][]string, 0, len(w.entries))

	for _, entry := range w.entries {
		row := w.buildRow(entry)
		w.allRows = append(w.allRows, row)
	}

	w.applyFilters()
}

// buildRow creates a table row ([]string) from an entry.
// Cell values may contain lipgloss ANSI color sequences; the custom renderer
// handles them correctly via lipgloss width-aware cell sizing.
func (w *TableWidget) buildRow(entry *models.Entry) []string {
	row := make([]string, 0, len(w.columnConfigs))

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
		if i >= len(w.columnWidths) || w.columnWidths[i] == 0 {
			continue
		}

		var cell string

		switch config.Title {
		case "Hostname":
			selectionMark := " "
			if isSelected {
				selectionMark = "✓"
			}
			hostname := selectionMark + " " + entry.Hostname
			cell = w.truncate(hostname, w.columnWidths[i])

		case "Source":
			src := entry.DataSource
			if src == "CloudFlare" {
				src = "CF"
			}
			cell = w.truncate(src, w.columnWidths[i])

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
				cell = w.theme.Dimmed.Render("Loading...")
			} else if !entry.DHCPStatus.Configured {
				cell = w.theme.Dimmed.Render("No lease")
			} else {
				leaseType := entry.DHCPStatus.Type
				if entry.DHCPStatus.InSync {
					cell = w.theme.Success.Render("OK " + leaseType)
				} else {
					cell = w.theme.Warning.Render("!! " + leaseType)
				}
			}

		case "Unbound":
			if !w.serviceStatus.Unbound {
				cell = w.theme.Dimmed.Render("..")
			} else {
				cell = w.formatServiceStatus(entry.UnboundStatus, entry.IsConfiguredInCaddy())
			}

		case "AdGuard":
			if !w.serviceStatus.AdGuard {
				cell = w.theme.Dimmed.Render("..")
			} else {
				cell = w.formatServiceStatus(entry.AdguardStatus, entry.IsConfiguredInCaddy())
			}

		case "CF":
			cell = w.formatCFStatus(entry)

		case "Status":
			cell = w.formatOverallStatus(entry.OverallStatus)
		}

		row = append(row, cell)
	}

	return row
}

// formatServiceStatus formats a service status cell with color
func (w *TableWidget) formatServiceStatus(status models.ServiceStatus, inCaddy bool) string {
	if !inCaddy {
		if !status.Configured {
			return w.theme.Success.Render("OK")
		}
		return w.theme.Error.Render("RM") // stale — should be removed
	}
	if !status.Configured {
		return w.theme.Warning.Render("NO")
	}
	if status.InSync {
		return w.theme.Success.Render("OK")
	}
	return w.theme.Error.Render("!!")
}

// formatCFStatus renders the CF column cell for an entry with color.
// * = default (managed) tunnel, ~ = non-default tunnel, ! = missing HTTPHostHeader.
func (w *TableWidget) formatCFStatus(entry *models.Entry) string {
	cf := entry.CloudflareStatus
	if !cf.Configured {
		return w.theme.Dimmed.Render("-")
	}

	colWidth := 14 // max tunnel name within column budget
	name := cf.TunnelName
	if len(name) > colWidth-2 {
		name = name[:colWidth-4] + ".."
	}

	if entry.NeedsHTTPHostHeader() {
		return w.theme.Warning.Render("! " + name)
	}
	if cf.IsDefaultTunnel {
		return w.theme.Success.Render("* " + name)
	}
	return w.theme.Dimmed.Render("~ " + name)
}

// formatOverallStatus formats the overall status cell with color
func (w *TableWidget) formatOverallStatus(status models.SyncStatus) string {
	text := status.Icon() + " " + status.Label()
	switch status {
	case models.FullyInSync:
		return w.theme.Success.Render(text)
	case models.OutOfSync, models.Stale:
		return w.theme.Error.Render(text)
	case models.CaddyOnly, models.PartiallyInSync, models.DHCPMismatch:
		return w.theme.Warning.Render(text)
	default:
		return w.theme.Dimmed.Render(text)
	}
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

// applyFilters applies the current filter, search query, and sorting,
// then refreshes the viewport content and clamps the cursor.
func (w *TableWidget) applyFilters() {
	w.filteredRows = make([][]string, 0, len(w.allRows))
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

	// Clamp cursor to valid range
	if len(w.filteredEntries) == 0 {
		w.cursor = 0
	} else if w.cursor >= len(w.filteredEntries) {
		w.cursor = len(w.filteredEntries) - 1
	}
	if w.cursor < 0 {
		w.cursor = 0
	}

	// Refresh viewport content
	w.viewport.SetContent(w.renderTableContent())
	w.syncViewport()
}

// GetSelectedEntry returns the currently selected entry (cursor position)
func (w *TableWidget) GetSelectedEntry() *models.Entry {
	if w.cursor < 0 || w.cursor >= len(w.filteredEntries) {
		return nil
	}
	return w.filteredEntries[w.cursor]
}

// toggleSelection toggles the selection state of the current row
func (w *TableWidget) toggleSelection() {
	if w.cursor < 0 || w.cursor >= len(w.filteredEntries) {
		return
	}

	if w.selectedIndices[w.cursor] {
		delete(w.selectedIndices, w.cursor)
	} else {
		w.selectedIndices[w.cursor] = true
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
		for i := 0; i < len(indices)-1; i++ {
			for j := i + 1; j < len(indices); j++ {
				if strings.ToLower(w.filteredEntries[indices[i]].Hostname) > strings.ToLower(w.filteredEntries[indices[j]].Hostname) {
					indices[i], indices[j] = indices[j], indices[i]
				}
			}
		}

	case SortByIP:
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
	sortedRows := make([][]string, len(w.filteredRows))
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

	// Calculate space needed for separators (1 space between each column)
	overhead := numCols - 1

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
		for priority := 3; priority >= 2 && available < 0; priority-- {
			for i, cfg := range configs {
				if cfg.Priority == priority && widths[i] > 0 {
					available += widths[i] + 1 // Reclaim space
					widths[i] = 0              // Hide this column
				}
			}
		}
	}

	return widths
}
