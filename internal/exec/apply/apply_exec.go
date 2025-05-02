package apply

import (
	"fmt"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/tui"
)

// UI represents the UI for the apply command
type UI struct {
	Styles tui.StyleConfig
}

// NewUI creates a new apply UI
func NewUI() *UI {
	return &UI{
		Styles: tui.DefaultStyles(),
	}
}

// RenderHeader renders the header for the apply command
func (ui *UI) RenderHeader() string {
	return ui.Styles.Header.Render(" u2699ufe0f Apply DNS Changes u2699ufe0f ") + "\n\n"
}

// RenderSuccess renders a success message for applying DNS changes
func (ui *UI) RenderSuccess() string {
	var sb strings.Builder

	sb.WriteString(ui.Styles.Success.Render(" u2705 DNS changes applied successfully "))
	return sb.String()
}

// RenderError renders an error message
func (ui *UI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" u274c Error: %s ", err))
}

// RenderApplyingMessage renders a message indicating that changes are being applied
func (ui *UI) RenderApplyingMessage() string {
	return ui.Styles.Info.Render(" ud83dudcbe Applying DNS changes... ")
}
