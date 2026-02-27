package widgets

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// LogWidget displays log messages in a scrollable view
type LogWidget struct {
	BaseWidget

	// Data
	entries    []LogEntry
	maxEntries int

	// View state
	visible      bool
	scrollOffset int

	// Layout
	theme *Theme
}

// NewLogWidget creates a new log widget
func NewLogWidget() *LogWidget {
	return &LogWidget{
		BaseWidget:   NewBaseWidget(),
		entries:      []LogEntry{},
		maxEntries:   1000, // Keep last 1000 log entries
		visible:      false,
		scrollOffset: 0,
		theme:        CurrentTheme,
	}
}

// Init initializes the log widget
func (w *LogWidget) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (w *LogWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	if !w.visible {
		return w, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if w.scrollOffset > 0 {
				w.scrollOffset--
			}
		case "down", "j":
			maxScroll := len(w.entries) - w.height + 4 // Account for border and padding
			if maxScroll < 0 {
				maxScroll = 0
			}
			if w.scrollOffset < maxScroll {
				w.scrollOffset++
			}
		case "pgup":
			w.scrollOffset -= 10
			if w.scrollOffset < 0 {
				w.scrollOffset = 0
			}
		case "pgdown":
			maxScroll := len(w.entries) - w.height + 4
			if maxScroll < 0 {
				maxScroll = 0
			}
			w.scrollOffset += 10
			if w.scrollOffset > maxScroll {
				w.scrollOffset = maxScroll
			}
		case "home", "g":
			w.scrollOffset = 0
		case "end", "G":
			maxScroll := len(w.entries) - w.height + 4
			if maxScroll < 0 {
				maxScroll = 0
			}
			w.scrollOffset = maxScroll
		}
	}

	return w, nil
}

// View renders the log widget
func (w *LogWidget) View() string {
	if !w.visible || w.width == 0 || w.height == 0 {
		return ""
	}

	var lines []string

	// Show log entries with scroll offset
	visibleHeight := w.height - 4 // Account for border and padding
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	startIdx := w.scrollOffset
	endIdx := w.scrollOffset + visibleHeight
	if endIdx > len(w.entries) {
		endIdx = len(w.entries)
	}
	if startIdx >= len(w.entries) {
		startIdx = len(w.entries) - 1
		if startIdx < 0 {
			startIdx = 0
		}
	}

	if len(w.entries) == 0 {
		lines = append(lines, w.theme.Dimmed.Render("No logs yet..."))
	} else {
		for i := startIdx; i < endIdx; i++ {
			entry := w.entries[i]
			lines = append(lines, w.formatLogEntry(entry))
		}
	}

	// Add scroll indicator
	if len(w.entries) > visibleHeight {
		scrollInfo := w.theme.Dimmed.Render(
			lipgloss.NewStyle().
				Align(lipgloss.Right).
				Width(w.width - 6).
				Render(formatScrollInfo(w.scrollOffset, len(w.entries), visibleHeight)),
		)
		lines = append(lines, scrollInfo)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// Add border with title
	title := w.theme.Bold.Render(" Logs ")
	if len(w.entries) > 0 {
		title = w.theme.Bold.Render(" Logs (" + formatInt(len(w.entries)) + " entries) ")
	}

	style := lipgloss.NewStyle().
		Border(w.theme.Border).
		BorderForeground(w.theme.ColorPurple).
		Padding(0, 1).
		Width(w.width - 4).
		Height(w.height - 2)

	bordered := style.Render(content)

	// Add title to top border
	titleStyle := lipgloss.NewStyle().
		Foreground(w.theme.ColorPurple).
		Bold(true)

	lines2 := strings.Split(bordered, "\n")
	if len(lines2) > 0 {
		topBorder := lines2[0]
		titleLen := lipgloss.Width(title)
		if len(topBorder) > titleLen+4 {
			lines2[0] = topBorder[:2] + titleStyle.Render(title) + topBorder[2+titleLen:]
		}
		bordered = strings.Join(lines2, "\n")
	}

	return bordered
}

// formatLogEntry formats a single log entry
func (w *LogWidget) formatLogEntry(entry LogEntry) string {
	timestamp := entry.Timestamp.Format("15:04:05")

	var levelStyle lipgloss.Style
	switch entry.Level {
	case "ERROR":
		levelStyle = w.theme.Error
	case "WARN":
		levelStyle = w.theme.Warning
	case "INFO":
		levelStyle = w.theme.Info
	case "DEBUG":
		levelStyle = w.theme.Dimmed
	default:
		levelStyle = lipgloss.NewStyle()
	}

	level := levelStyle.Render(padRight(entry.Level, 5))
	time := w.theme.Dimmed.Render(timestamp)

	return time + " " + level + " " + entry.Message
}

// AddLog adds a log entry
func (w *LogWidget) AddLog(level, message string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	w.entries = append(w.entries, entry)

	// Trim to max entries
	if len(w.entries) > w.maxEntries {
		w.entries = w.entries[len(w.entries)-w.maxEntries:]
	}

	// Auto-scroll to bottom if we're already at the bottom
	maxScroll := len(w.entries) - w.height + 4
	if maxScroll < 0 {
		maxScroll = 0
	}
	if w.scrollOffset >= maxScroll-1 {
		w.scrollOffset = maxScroll
	}
}

// Toggle toggles the visibility of the log widget
func (w *LogWidget) Toggle() {
	w.visible = !w.visible
}

// Show shows the log widget
func (w *LogWidget) Show() {
	w.visible = true
}

// Hide hides the log widget
func (w *LogWidget) Hide() {
	w.visible = false
}

// IsVisible returns true if the log widget is visible
func (w *LogWidget) IsVisible() bool {
	return w.visible
}

// Clear clears all log entries
func (w *LogWidget) Clear() {
	w.entries = []LogEntry{}
	w.scrollOffset = 0
}

// Helper functions

func formatScrollInfo(offset, total, visible int) string {
	if total <= visible {
		return ""
	}
	start := offset + 1
	end := offset + visible
	if end > total {
		end = total
	}
	return formatInt(start) + "-" + formatInt(end) + "/" + formatInt(total)
}

func formatInt(n int) string {
	if n < 0 {
		n = 0
	}
	s := ""
	i := n
	for {
		s = string(rune('0'+(i%10))) + s
		i /= 10
		if i == 0 {
			break
		}
	}
	return s
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}
