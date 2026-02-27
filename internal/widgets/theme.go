package widgets

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme holds all styling definitions for the application
// Uses Tokyo Night color palette for consistency
type Theme struct {
	// Tokyo Night Colors
	ColorSuccess lipgloss.Color // #9ece6a - green
	ColorWarning lipgloss.Color // #e0af68 - yellow
	ColorError   lipgloss.Color // #f7768e - red
	ColorInfo    lipgloss.Color // #7aa2f7 - blue
	ColorDim     lipgloss.Color // #565f89 - gray
	ColorCyan    lipgloss.Color // #7dcfff - cyan
	ColorPurple  lipgloss.Color // #bb9af7 - purple
	ColorBg      lipgloss.Color // #1a1b26 - background

	// Text Styles
	Bold   lipgloss.Style
	Italic lipgloss.Style
	Dimmed lipgloss.Style

	// Semantic Styles
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Info    lipgloss.Style

	// Component Styles
	Header      lipgloss.Style
	Section     lipgloss.Style
	Hostname    lipgloss.Style
	IP          lipgloss.Style
	Count       lipgloss.Style
	Description lipgloss.Style

	// Action Styles
	Add    lipgloss.Style
	Update lipgloss.Style
	Remove lipgloss.Style
	Sync   lipgloss.Style
	DryRun lipgloss.Style

	// Widget-specific Styles
	Border  lipgloss.Border
	Focused lipgloss.Style
	Blurred lipgloss.Style

	// Table Styles
	TableHeader     lipgloss.Style
	TableRowEven    lipgloss.Style
	TableRowOdd     lipgloss.Style
	TableSelected   lipgloss.Style
	TableCellDimmed lipgloss.Style

	// Status Bar Styles
	StatusBar        lipgloss.Style
	StatusBarSuccess lipgloss.Style
	StatusBarWarning lipgloss.Style
	StatusBarError   lipgloss.Style
	StatusBarInfo    lipgloss.Style

	// Progress Styles
	ProgressBar      lipgloss.Style
	ProgressComplete lipgloss.Style
	ProgressEmpty    lipgloss.Style

	// Input Styles
	InputFocused lipgloss.Style
	InputBlurred lipgloss.Style
	InputPrompt  lipgloss.Style
	InputCursor  lipgloss.Style

	// Button Styles
	ButtonFocused lipgloss.Style
	ButtonBlurred lipgloss.Style
	ButtonActive  lipgloss.Style

	// Help Styles
	HelpKey   lipgloss.Style
	HelpValue lipgloss.Style
	HelpSep   lipgloss.Style
}

// DefaultTheme creates the default Tokyo Night themed styles
func DefaultTheme() *Theme {
	// Tokyo Night color palette
	successColor := lipgloss.Color("#9ece6a")
	warningColor := lipgloss.Color("#e0af68")
	errorColor := lipgloss.Color("#f7768e")
	infoColor := lipgloss.Color("#7aa2f7")
	dimColor := lipgloss.Color("#565f89")
	cyanColor := lipgloss.Color("#7dcfff")
	purpleColor := lipgloss.Color("#bb9af7")
	bgColor := lipgloss.Color("#1a1b26")

	theme := &Theme{
		// Colors
		ColorSuccess: successColor,
		ColorWarning: warningColor,
		ColorError:   errorColor,
		ColorInfo:    infoColor,
		ColorDim:     dimColor,
		ColorCyan:    cyanColor,
		ColorPurple:  purpleColor,
		ColorBg:      bgColor,

		// Text Styles
		Bold:   lipgloss.NewStyle().Bold(true),
		Italic: lipgloss.NewStyle().Italic(true),
		Dimmed: lipgloss.NewStyle().Foreground(dimColor),

		// Semantic Styles
		Success: lipgloss.NewStyle().Foreground(successColor),
		Warning: lipgloss.NewStyle().Foreground(warningColor),
		Error:   lipgloss.NewStyle().Foreground(errorColor),
		Info:    lipgloss.NewStyle().Foreground(infoColor),

		// Component Styles
		Header:      lipgloss.NewStyle().Bold(true).Foreground(infoColor),
		Section:     lipgloss.NewStyle().Foreground(infoColor),
		Hostname:    lipgloss.NewStyle().Bold(true).Foreground(successColor),
		IP:          lipgloss.NewStyle().Foreground(cyanColor),
		Count:       lipgloss.NewStyle().Bold(true).Foreground(cyanColor),
		Description: lipgloss.NewStyle().Foreground(dimColor).Italic(true),

		// Action Styles
		Add:    lipgloss.NewStyle().Foreground(successColor),
		Update: lipgloss.NewStyle().Foreground(warningColor),
		Remove: lipgloss.NewStyle().Foreground(errorColor),
		Sync:   lipgloss.NewStyle().Foreground(infoColor),
		DryRun: lipgloss.NewStyle().Bold(true).Foreground(purpleColor).Background(bgColor),

		// Border
		Border: lipgloss.RoundedBorder(),
	}

	// Widget-specific Styles
	theme.Focused = lipgloss.NewStyle().
		Border(theme.Border).
		BorderForeground(infoColor).
		Padding(0, 1)

	theme.Blurred = lipgloss.NewStyle().
		Border(theme.Border).
		BorderForeground(dimColor).
		Padding(0, 1)

	// Table Styles
	theme.TableHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(infoColor).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(dimColor).
		BorderBottom(true).
		Padding(0, 1)

	theme.TableRowEven = lipgloss.NewStyle().
		Padding(0, 1)

	theme.TableRowOdd = lipgloss.NewStyle().
		Padding(0, 1)

	theme.TableSelected = lipgloss.NewStyle().
		Background(lipgloss.Color("#283457")).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(0, 1)

	theme.TableCellDimmed = lipgloss.NewStyle().
		Foreground(dimColor).
		Padding(0, 1)

	// Status Bar Styles
	theme.StatusBar = lipgloss.NewStyle().
		Padding(0, 1).
		Background(bgColor)

	theme.StatusBarSuccess = theme.StatusBar.Copy().
		Foreground(successColor)

	theme.StatusBarWarning = theme.StatusBar.Copy().
		Foreground(warningColor)

	theme.StatusBarError = theme.StatusBar.Copy().
		Foreground(errorColor)

	theme.StatusBarInfo = theme.StatusBar.Copy().
		Foreground(infoColor)

	// Progress Styles
	theme.ProgressBar = lipgloss.NewStyle().
		Foreground(infoColor)

	theme.ProgressComplete = lipgloss.NewStyle().
		Foreground(successColor)

	theme.ProgressEmpty = lipgloss.NewStyle().
		Foreground(dimColor)

	// Input Styles
	theme.InputFocused = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(infoColor).
		Padding(0, 1)

	theme.InputBlurred = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dimColor).
		Padding(0, 1)

	theme.InputPrompt = lipgloss.NewStyle().
		Foreground(infoColor).
		Bold(true)

	theme.InputCursor = lipgloss.NewStyle().
		Foreground(successColor)

	// Button Styles
	theme.ButtonFocused = lipgloss.NewStyle().
		Background(infoColor).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(0, 2).
		Bold(true)

	theme.ButtonBlurred = lipgloss.NewStyle().
		Background(dimColor).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(0, 2)

	theme.ButtonActive = lipgloss.NewStyle().
		Background(successColor).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(0, 2).
		Bold(true)

	// Help Styles
	theme.HelpKey = lipgloss.NewStyle().
		Foreground(cyanColor)

	theme.HelpValue = lipgloss.NewStyle().
		Foreground(dimColor)

	theme.HelpSep = lipgloss.NewStyle().
		Foreground(dimColor)

	return theme
}

// CurrentTheme is the global theme instance
// All widgets should use this for consistent styling
var CurrentTheme = DefaultTheme()

// SetTheme allows changing the global theme
func SetTheme(theme *Theme) {
	CurrentTheme = theme
}

// Helper functions for common styling patterns

// RenderIcon returns a styled icon based on sync status
func (t *Theme) RenderIcon(icon string, statusType string) string {
	switch statusType {
	case "success", "synced":
		return t.Success.Render(icon)
	case "warning", "partial":
		return t.Warning.Render(icon)
	case "error", "out-of-sync":
		return t.Error.Render(icon)
	case "info":
		return t.Info.Render(icon)
	case "dim":
		return t.Dimmed.Render(icon)
	default:
		return icon
	}
}

// RenderLabel returns a styled label
func (t *Theme) RenderLabel(label string, styleType string) string {
	switch styleType {
	case "header":
		return t.Header.Render(label)
	case "section":
		return t.Section.Render(label)
	case "hostname":
		return t.Hostname.Render(label)
	case "ip":
		return t.IP.Render(label)
	case "count":
		return t.Count.Render(label)
	case "description":
		return t.Description.Render(label)
	case "success":
		return t.Success.Render(label)
	case "warning":
		return t.Warning.Render(label)
	case "error":
		return t.Error.Render(label)
	case "info":
		return t.Info.Render(label)
	case "dim":
		return t.Dimmed.Render(label)
	default:
		return label
	}
}

// RenderAction returns a styled action indicator
func (t *Theme) RenderAction(action string, actionType string) string {
	switch actionType {
	case "add":
		return t.Add.Render(action)
	case "update":
		return t.Update.Render(action)
	case "remove":
		return t.Remove.Render(action)
	case "sync":
		return t.Sync.Render(action)
	case "dry-run":
		return t.DryRun.Render(action)
	default:
		return action
	}
}
