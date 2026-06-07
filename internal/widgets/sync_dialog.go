package widgets

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jeeftor/caddy-dns-sync/internal/models"
	"github.com/jeeftor/caddy-dns-sync/internal/syncplan"
)

// SyncDialogState represents the current state of the sync dialog
type SyncDialogState int

const (
	SyncStatePreview   SyncDialogState = iota // Showing preview, awaiting confirmation
	SyncStateRunning                          // Sync in progress
	SyncStateComplete                         // Sync completed successfully
	SyncStateError                            // Sync failed with errors
	SyncStateCancelled                        // User cancelled the sync
)

// SyncProgressMsg is sent during sync execution to update progress
type SyncProgressMsg struct {
	ActionIndex int
	Message     string
	Error       error
}

// SyncCompleteMsg is sent when sync completes
type SyncCompleteMsg struct {
	Result *syncplan.Result
}

// SyncDialog displays sync operations and handles user confirmation
type SyncDialog struct {
	BaseWidget

	// State
	state            SyncDialogState
	actions          []syncplan.Action
	allActions       []syncplan.Action // All possible actions before filtering
	result           *syncplan.Result
	currentAction    int
	totalActions     int
	cursor           int  // Current cursor position in action list
	userAcknowledged bool // True when user has pressed a key to acknowledge completion

	// Progress logging
	progressLog []string // Log of sync operations
	errorLog    []string // Log of errors

	// Configuration
	dryRun        bool
	serviceName   string // "Unbound", "AdGuard", "Cloudflare", etc.
	confirmPrompt string
	caddyServerIP string // IP of the Caddy server (what DNS should point to)

	// Service toggles
	enableUnbound bool
	enableAdGuard bool
	enableDHCP    bool

	// Sync executor callback (injected from TUI layer)
	syncExecutor func([]syncplan.Action) *syncplan.Result

	// Components
	spinner  spinner.Model
	progress progress.Model

	// Layout
	theme *Theme
}

// NewSyncDialog creates a new sync dialog widget
func NewSyncDialog(serviceName string) *SyncDialog {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(CurrentTheme.ColorInfo)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	return &SyncDialog{
		BaseWidget:    NewBaseWidget(),
		state:         SyncStatePreview,
		actions:       []syncplan.Action{},
		allActions:    []syncplan.Action{},
		serviceName:   serviceName,
		confirmPrompt: "Press 'y' to confirm, 'n' to cancel",
		enableUnbound: true, // DNS services enabled by default
		enableAdGuard: true,
		enableDHCP:    false, // DHCP disabled by default (more invasive)
		spinner:       s,
		progress:      p,
		theme:         CurrentTheme,
	}
}

// Init initializes the sync dialog
func (w *SyncDialog) Init() tea.Cmd {
	return w.spinner.Tick
}

// Update handles messages
func (w *SyncDialog) Update(msg tea.Msg) (Widget, tea.Cmd) {
	var cmds []tea.Cmd

	// Update spinner
	var cmd tea.Cmd
	w.spinner, cmd = w.spinner.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case SyncProgressMsg:
		// Update progress from async sync execution
		w.currentAction = msg.ActionIndex
		if msg.Message != "" {
			w.progressLog = append(w.progressLog, msg.Message)
		}
		if msg.Error != nil {
			w.errorLog = append(w.errorLog, msg.Error.Error())
		}
		return w, nil

	case SyncCompleteMsg:
		// Sync completed
		w.result = msg.Result
		if msg.Result.Success {
			w.state = SyncStateComplete
		} else {
			w.state = SyncStateError
		}
		w.userAcknowledged = false // Wait for user to acknowledge
		return w, nil

	case tea.KeyMsg:
		// Handle completion/error/cancelled states - any key closes
		if w.state == SyncStateComplete || w.state == SyncStateError || w.state == SyncStateCancelled {
			// Any key press acknowledges and marks as done
			w.userAcknowledged = true
			return w, nil
		}

		if w.state == SyncStatePreview {
			switch msg.String() {
			case "y", "Y":
				w.state = SyncStateRunning
				w.progressLog = []string{}
				w.errorLog = []string{}
				w.currentAction = 0
				// Start async sync execution
				return w, w.startAsyncSync
			case "n", "N", "esc":
				w.state = SyncStateCancelled
				return w, tea.Batch(cmds...)
			case "up", "k":
				// Move cursor up
				if w.cursor > 0 {
					w.cursor--
				}
				return w, tea.Batch(cmds...)
			case "down", "j":
				// Move cursor down
				if w.cursor < len(w.actions)-1 {
					w.cursor++
				}
				return w, tea.Batch(cmds...)
			case " ", "s", "enter":
				// Toggle selected action
				if w.cursor >= 0 && w.cursor < len(w.actions) {
					w.actions[w.cursor].Enabled = !w.actions[w.cursor].Enabled
				}
				return w, tea.Batch(cmds...)
			case "a":
				// Enable all actions
				for i := range w.actions {
					w.actions[i].Enabled = true
				}
				return w, tea.Batch(cmds...)
			case "d":
				// Disable all actions
				for i := range w.actions {
					w.actions[i].Enabled = false
				}
				return w, tea.Batch(cmds...)
			}
		}
	}

	return w, tea.Batch(cmds...)
}

// View renders the sync dialog
func (w *SyncDialog) View() string {
	if w.width == 0 || w.height == 0 {
		return ""
	}

	switch w.state {
	case SyncStatePreview:
		return w.renderPreview()
	case SyncStateRunning:
		return w.renderRunning()
	case SyncStateComplete:
		return w.renderComplete()
	case SyncStateError:
		return w.renderError()
	case SyncStateCancelled:
		return w.renderCancelled()
	default:
		return ""
	}
}

// renderPreview renders the sync preview with actions to be performed
func (w *SyncDialog) renderPreview() string {
	var sections []string

	// Title
	title := fmt.Sprintf("Sync Preview: %s", w.serviceName)
	if w.dryRun {
		title += " " + w.theme.DryRun.Render("[DRY RUN]")
	}
	sections = append(sections, w.theme.Header.Render(title))
	sections = append(sections, "")

	// Summary (count only ENABLED actions)
	addCount := 0
	updateCount := 0
	deleteCount := 0

	for _, action := range w.actions {
		if !action.Enabled {
			continue
		}
		switch action.Type {
		case "add":
			addCount++
		case "update":
			updateCount++
		case "delete":
			deleteCount++
		}
	}

	summary := fmt.Sprintf("Selected: %s to add, %s to update, %s to delete",
		w.theme.Count.Render(fmt.Sprintf("%d", addCount)),
		w.theme.Count.Render(fmt.Sprintf("%d", updateCount)),
		w.theme.Count.Render(fmt.Sprintf("%d", deleteCount)),
	)
	sections = append(sections, summary)
	sections = append(sections, "")

	// Instructions
	sections = append(sections, w.theme.Section.Render("━━━ Actions (↑/↓ navigate, SPACE/s toggle, a enable all, d disable all) ━━━"))
	sections = append(sections, "")

	// Actions list with checkboxes and cursor
	if len(w.actions) > 0 {
		for i, action := range w.actions {
			// Cursor indicator
			cursor := "  "
			if i == w.cursor {
				cursor = w.theme.Info.Render("> ")
			}

			// Checkbox
			checkbox := "[○]"
			if action.Enabled {
				checkbox = w.theme.Success.Render("[✓]")
			} else {
				checkbox = w.theme.Dimmed.Render("[○]")
			}

			// Action details
			actionText := w.formatActionInline(action)

			line := cursor + checkbox + " " + actionText
			sections = append(sections, line)
		}
		sections = append(sections, "")
	} else {
		sections = append(sections, w.theme.Dimmed.Render("  No actions needed"))
		sections = append(sections, "")
	}

	// Confirmation prompt
	if addCount+updateCount+deleteCount > 0 {
		sections = append(sections, w.theme.Warning.Render("Press 'y' to apply selected actions, 'n' to cancel"))
	} else {
		sections = append(sections, w.theme.Dimmed.Render("No actions selected. Press 'n' to close."))
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

// renderRunning renders the sync in progress view
func (w *SyncDialog) renderRunning() string {
	var sections []string

	// Title with spinner
	title := fmt.Sprintf("%s Syncing %s...", w.spinner.View(), w.serviceName)
	sections = append(sections, w.theme.Info.Render(title))
	sections = append(sections, "")

	// Progress bar
	if w.totalActions > 0 {
		percent := float64(w.currentAction) / float64(w.totalActions)
		progressBar := w.progress.ViewAs(percent)
		label := fmt.Sprintf("Processing: %d/%d", w.currentAction, w.totalActions)
		sections = append(sections, label)
		sections = append(sections, progressBar)
		sections = append(sections, "")
	}

	// Current action
	if w.currentAction < len(w.actions) {
		current := w.actions[w.currentAction]
		sections = append(sections, w.theme.Dimmed.Render("Current:"))
		sections = append(sections, w.formatAction(current))
		sections = append(sections, "")
	}

	// Progress log (last 5 entries)
	if len(w.progressLog) > 0 {
		sections = append(sections, w.theme.Dimmed.Render("Progress:"))
		start := 0
		if len(w.progressLog) > 5 {
			start = len(w.progressLog) - 5
		}
		for _, log := range w.progressLog[start:] {
			sections = append(sections, w.theme.Dimmed.Render("  "+log))
		}
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

// renderComplete renders the successful completion view
func (w *SyncDialog) renderComplete() string {
	var sections []string

	// Title
	title := fmt.Sprintf("✅ Sync Complete: %s", w.serviceName)
	sections = append(sections, w.theme.Success.Render(title))
	sections = append(sections, "")

	// Results
	if w.result != nil {
		if w.result.ItemsAdded > 0 {
			sections = append(sections,
				w.theme.Add.Render(fmt.Sprintf("✓ Added: %d entries", w.result.ItemsAdded)))
		}
		if w.result.ItemsUpdated > 0 {
			sections = append(sections,
				w.theme.Update.Render(fmt.Sprintf("✓ Updated: %d entries", w.result.ItemsUpdated)))
		}
		if w.result.ItemsDeleted > 0 {
			sections = append(sections,
				w.theme.Remove.Render(fmt.Sprintf("✓ Deleted: %d entries", w.result.ItemsDeleted)))
		}

		if w.result.Message != "" {
			sections = append(sections, "")
			sections = append(sections, w.result.Message)
		}
	}

	sections = append(sections, "")
	sections = append(sections, w.theme.Dimmed.Render("Press any key to close"))

	// Wrap in a border
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	style := lipgloss.NewStyle().
		Border(w.theme.Border).
		BorderForeground(w.theme.ColorSuccess).
		Padding(1, 2).
		Width(w.width - 4)

	return style.Render(content)
}

// renderError renders the error view
func (w *SyncDialog) renderError() string {
	var sections []string

	// Title
	title := fmt.Sprintf("❌ Sync Error: %s", w.serviceName)
	sections = append(sections, w.theme.Error.Render(title))
	sections = append(sections, "")

	// Show what was attempted
	if w.result != nil {
		if w.result.ItemsAdded > 0 || w.result.ItemsUpdated > 0 || w.result.ItemsDeleted > 0 {
			sections = append(sections, w.theme.Dimmed.Render("Attempted:"))
			if w.result.ItemsAdded > 0 {
				sections = append(sections, fmt.Sprintf("  Add: %d", w.result.ItemsAdded))
			}
			if w.result.ItemsUpdated > 0 {
				sections = append(sections, fmt.Sprintf("  Update: %d", w.result.ItemsUpdated))
			}
			if w.result.ItemsDeleted > 0 {
				sections = append(sections, fmt.Sprintf("  Delete: %d", w.result.ItemsDeleted))
			}
			sections = append(sections, "")
		}
	}

	// Error log
	if len(w.errorLog) > 0 {
		sections = append(sections, w.theme.Error.Render("Errors:"))
		// Show last 10 errors
		start := 0
		if len(w.errorLog) > 10 {
			start = len(w.errorLog) - 10
		}
		for _, err := range w.errorLog[start:] {
			sections = append(sections, w.theme.Dimmed.Render("  • "+err))
		}
		sections = append(sections, "")
	}

	// Result message
	if w.result != nil && w.result.Message != "" {
		sections = append(sections, w.theme.Warning.Render(w.result.Message))
		sections = append(sections, "")
	}

	sections = append(sections, w.theme.Dimmed.Render("Press any key to close"))

	// Wrap in a border
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	style := lipgloss.NewStyle().
		Border(w.theme.Border).
		BorderForeground(w.theme.ColorError).
		Padding(1, 2).
		Width(w.width - 4)

	return style.Render(content)
}

// renderCancelled renders the cancelled view
func (w *SyncDialog) renderCancelled() string {
	var sections []string

	title := "❌ Sync Cancelled"
	sections = append(sections, w.theme.Warning.Render(title))
	sections = append(sections, "")
	sections = append(sections, "No changes were made.")
	sections = append(sections, "")
	sections = append(sections, w.theme.Dimmed.Render("Press any key to close"))

	// Wrap in a border
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	style := lipgloss.NewStyle().
		Border(w.theme.Border).
		BorderForeground(w.theme.ColorWarning).
		Padding(1, 2).
		Width(w.width - 4)

	return style.Render(content)
}

// formatAction formats a single sync action for display
func (w *SyncDialog) formatAction(action syncplan.Action) string {
	var parts []string

	// Action type with icon and color
	switch action.Type {
	case "add":
		parts = append(parts, w.theme.Add.Render("+ ADD"))
	case "update":
		parts = append(parts, w.theme.Update.Render("~ UPDATE"))
	case "delete":
		if action.Service == "cloudflare" {
			parts = append(parts, w.theme.Remove.Render("- UNSYNC"))
		} else {
			parts = append(parts, w.theme.Remove.Render("- DELETE"))
		}
	default:
		parts = append(parts, action.Type)
	}

	// Service
	parts = append(parts, w.theme.Info.Render(action.Service))

	// Hostname
	parts = append(parts, w.theme.Hostname.Render(action.Hostname))

	// Target details
	if action.Service == "cloudflare" {
		if action.NewService != "" {
			parts = append(parts, w.theme.IP.Render(action.NewService))
		} else if action.OldService != "" {
			parts = append(parts, w.theme.Dimmed.Render(action.OldService))
		}
		if action.NewHTTPHostHeader != "" {
			parts = append(parts, w.theme.Info.Render("host="+action.NewHTTPHostHeader))
		} else if action.OldHTTPHostHeader != "" {
			parts = append(parts, w.theme.Dimmed.Render("host="+action.OldHTTPHostHeader))
		}
	} else if action.Type == "update" && action.OldIP != "" && action.NewIP != "" {
		ipChange := fmt.Sprintf("%s → %s", action.OldIP, action.NewIP)
		parts = append(parts, w.theme.IP.Render(ipChange))
	} else if action.Type == "delete" && action.OldIP != "" {
		parts = append(parts, w.theme.Dimmed.Render("remove "+action.OldIP))
	} else if action.NewIP != "" {
		parts = append(parts, w.theme.IP.Render(action.NewIP))
	}

	// Additional details
	if action.Details != "" {
		parts = append(parts, w.theme.Dimmed.Render(fmt.Sprintf("(%s)", action.Details)))
	}

	return "  " + strings.Join(parts, " ")
}

// formatActionInline formats a single sync action for inline display in checklist
func (w *SyncDialog) formatActionInline(action syncplan.Action) string {
	var parts []string

	// Action type with icon (shorter)
	switch action.Type {
	case "add":
		parts = append(parts, w.theme.Add.Render("ADD"))
	case "update":
		parts = append(parts, w.theme.Update.Render("UPD"))
	case "delete":
		if action.Service == "cloudflare" {
			parts = append(parts, w.theme.Remove.Render("UNSYNC"))
		} else {
			parts = append(parts, w.theme.Remove.Render("DEL"))
		}
	default:
		parts = append(parts, action.Type)
	}

	// Service (shortened)
	serviceName := action.Service
	if serviceName == "unbound" {
		serviceName = "unb"
	} else if serviceName == "adguard" {
		serviceName = "adg"
	}
	parts = append(parts, w.theme.Info.Render(serviceName))

	// Hostname
	parts = append(parts, action.Hostname)

	// IP change details
	if action.Service == "cloudflare" {
		if action.NewService != "" {
			parts = append(parts, w.theme.IP.Render(action.NewService))
		} else if action.OldService != "" {
			parts = append(parts, w.theme.Dimmed.Render(action.OldService))
		}
		if action.NewHTTPHostHeader != "" {
			parts = append(parts, w.theme.Info.Render("host="+action.NewHTTPHostHeader))
		} else if action.OldHTTPHostHeader != "" {
			parts = append(parts, w.theme.Dimmed.Render("host="+action.OldHTTPHostHeader))
		}
	} else if action.Type == "update" && action.OldIP != "" && action.NewIP != "" {
		ipChange := fmt.Sprintf("%s→%s", action.OldIP, action.NewIP)
		parts = append(parts, w.theme.IP.Render(ipChange))
	} else if action.Type == "delete" && action.OldIP != "" {
		parts = append(parts, w.theme.Dimmed.Render(action.OldIP))
	} else if action.NewIP != "" {
		parts = append(parts, w.theme.IP.Render(action.NewIP))
	}

	// Additional details (shortened)
	if action.Details != "" {
		parts = append(parts, w.theme.Dimmed.Render(fmt.Sprintf("(%s)", action.Details)))
	}

	return strings.Join(parts, " ")
}

// SetActions sets the list of sync actions to be performed
func (w *SyncDialog) SetActions(actions []syncplan.Action) {
	w.actions = actions
	w.totalActions = len(actions)
	w.currentAction = 0
	w.cursor = 0
	w.state = SyncStatePreview
}

// SetDryRun sets whether this is a dry run
func (w *SyncDialog) SetDryRun(dryRun bool) {
	w.dryRun = dryRun
}

// SetProgress updates the current progress
func (w *SyncDialog) SetProgress(current, total int) {
	w.currentAction = current
	w.totalActions = total
}

// SetState sets the dialog state
func (w *SyncDialog) SetState(state SyncDialogState) {
	w.state = state
}

// SetResult sets the sync result
func (w *SyncDialog) SetResult(result *syncplan.Result) {
	w.result = result
	if result.Success {
		w.state = SyncStateComplete
	} else {
		w.state = SyncStateError
	}
}

// GetState returns the current state
func (w *SyncDialog) GetState() SyncDialogState {
	return w.state
}

// IsConfirmed returns true if the user confirmed the sync
func (w *SyncDialog) IsConfirmed() bool {
	return w.state == SyncStateRunning || w.state == SyncStateComplete
}

// IsCancelled returns true if the user cancelled the sync
func (w *SyncDialog) IsCancelled() bool {
	return w.state == SyncStateCancelled
}

// IsDone returns true if the sync is complete AND user has acknowledged it
func (w *SyncDialog) IsDone() bool {
	// Only done when in terminal state AND user has pressed a key to acknowledge
	return (w.state == SyncStateComplete || w.state == SyncStateError || w.state == SyncStateCancelled) && w.userAcknowledged
}

// Reset resets the sync dialog to its initial state
func (w *SyncDialog) Reset() {
	w.state = SyncStatePreview
	w.actions = []syncplan.Action{}
	w.allActions = []syncplan.Action{}
	w.result = nil
	w.currentAction = 0
	w.totalActions = 0
	w.cursor = 0
	w.progressLog = []string{}
	w.errorLog = []string{}
	w.userAcknowledged = false
	// Note: Don't clear syncExecutor - it should persist across resets
}

// SetSyncExecutor sets the sync executor callback
func (w *SyncDialog) SetSyncExecutor(executor func([]syncplan.Action) *syncplan.Result) {
	w.syncExecutor = executor
}

// logToFile writes a log message to caddysync.log for debugging
func (w *SyncDialog) logToFile(format string, args ...interface{}) {
	// Simple file logging for debugging
	// Open file in append mode
	f, err := os.OpenFile("caddysync.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Silently fail if we can't log
	}
	defer f.Close()

	// Write timestamp and message
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(f, "[%s] %s\n", timestamp, msg)
}

// startAsyncSync starts the sync operation
func (w *SyncDialog) startAsyncSync() tea.Msg {
	// Execute sync immediately and return completion message
	return w.executeSync()
}

// executeSync performs the actual sync operations and returns completion message
func (w *SyncDialog) executeSync() tea.Msg {
	// Log progress for enabled actions
	enabledCount := 0
	for _, action := range w.actions {
		if action.Enabled {
			enabledCount++
		}
	}

	currentEnabled := 0
	for i, action := range w.actions {
		if !action.Enabled {
			continue
		}

		currentEnabled++
		msg := fmt.Sprintf("[%d/%d] %s %s %s %s",
			currentEnabled, enabledCount, action.Type, action.Service, action.Hostname, action.NewIP)
		w.progressLog = append(w.progressLog, msg)
		w.currentAction = i
	}

	// Log sync start
	w.logToFile("Starting sync execution with %d enabled actions", enabledCount)

	// Execute sync using the injected executor
	var result *syncplan.Result
	if w.syncExecutor != nil {
		w.logToFile("Calling sync executor...")
		result = w.syncExecutor(w.actions)
		w.logToFile("Sync completed: Success=%v, Added=%d, Updated=%d, Deleted=%d, Errors=%d",
			result.Success, result.ItemsAdded, result.ItemsUpdated, result.ItemsDeleted, len(result.Errors))
		// Copy errors to error log
		w.errorLog = append(w.errorLog, result.Errors...)
		for _, err := range result.Errors {
			w.logToFile("Error: %s", err)
		}
	} else {
		// No executor provided - return error
		w.logToFile("ERROR: Sync executor not configured")
		result = &syncplan.Result{
			Success: false,
			Errors:  []string{"Sync executor not configured"},
			Message: "Sync executor not configured",
		}
		w.errorLog = append(w.errorLog, "Sync executor not configured")
	}

	// Return completion message
	return SyncCompleteMsg{
		Result: result,
	}
}

// AddActionsFromEntries creates sync actions from a list of entries for one service or all services.
func (w *SyncDialog) AddActionsFromEntries(
	entries []*models.Entry,
	service string,
	caddyServerIP string,
	caddyServiceURL string,
	includeCloudflare bool,
) {
	w.caddyServerIP = caddyServerIP
	// Reset the dialog state first to clear any previous actions
	w.Reset()

	actions := syncplan.PlanFromEntries(entries, syncplan.Options{
		Service:           service,
		CaddyServerIP:     caddyServerIP,
		CaddyServiceURL:   caddyServiceURL,
		IncludeCloudflare: includeCloudflare,
	})

	// Store actions (all enabled by default)
	w.actions = actions
	w.allActions = actions
	w.totalActions = len(actions)
	w.cursor = 0 // Reset cursor to first action
}
