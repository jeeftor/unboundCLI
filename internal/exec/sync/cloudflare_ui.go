package sync

import (
	"fmt"
	"strings"
	"time"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/tui"
)

// CloudflareUI provides Cloudflare-specific UI rendering functions
type CloudflareUI struct {
	Styles tui.StyleConfig
}

// NewCloudflareUI creates a new CloudflareUI instance
func NewCloudflareUI() *CloudflareUI {
	return &CloudflareUI{
		Styles: tui.DefaultStyles(),
	}
}

// RenderHeader renders the Cloudflare header
func (ui *CloudflareUI) RenderHeader() string {
	return ui.Styles.Header.Render("☁️ Cloudflare Tunnel Sync ☁️") + "\n"
}

// RenderTunnelInfo renders information about a Cloudflare tunnel
func (ui *CloudflareUI) RenderTunnelInfo(tunnelID string, tunnelName string) string {
	var sb strings.Builder

	sb.WriteString(ui.Styles.Section.Render(" 🚇 Cloudflare Tunnel Information 🚇 "))
	sb.WriteString("\n\n")

	sb.WriteString("  Tunnel ID: ")
	sb.WriteString(ui.Styles.Dimmed.Render(tunnelID))
	sb.WriteString("\n")

	if tunnelName != "" {
		sb.WriteString("  Tunnel Name: ")
		sb.WriteString(ui.Styles.Hostname.Render(tunnelName))
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderTunnelsList renders a list of available Cloudflare tunnels
func (ui *CloudflareUI) RenderTunnelsList(tunnels []api.CloudflareTunnel) string {
	var sb strings.Builder

	sb.WriteString(ui.Styles.Section.Render(" 🚇 Available Cloudflare Tunnels 🚇 "))
	sb.WriteString("\n\n")

	if len(tunnels) == 0 {
		sb.WriteString(ui.Styles.Warning.Render("  No tunnels found for this account"))
		sb.WriteString("\n")
		return sb.String()
	}

	for i, tunnel := range tunnels {
		sb.WriteString(fmt.Sprintf("  %d. ", i+1))
		sb.WriteString(ui.Styles.Hostname.Render(tunnel.Name))
		sb.WriteString("\n")

		sb.WriteString("     ID: ")
		sb.WriteString(ui.Styles.Dimmed.Render(tunnel.ID))
		sb.WriteString("\n")

		sb.WriteString("     Created: ")
		sb.WriteString(formatTime(tunnel.CreatedAt))
		sb.WriteString("\n")

		// Add information about connections if available
		if len(tunnel.Connections) > 0 {
			sb.WriteString("     Status: ")
			activeCount := 0
			for _, conn := range tunnel.Connections {
				if conn.Status == "active" {
					activeCount++
				}
			}
			if activeCount > 0 {
				sb.WriteString(ui.Styles.Success.Render(fmt.Sprintf("%d active connections", activeCount)))
			} else {
				sb.WriteString(ui.Styles.Warning.Render("No active connections"))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// formatTime formats a time.Time in a user-friendly way
func formatTime(t time.Time) string {
	// If the time is empty, return "N/A"
	if t.IsZero() {
		return "N/A"
	}

	// Format the time
	return t.Format("Jan 02, 2006 15:04:05")
}

// RenderNoTunnelID renders a message indicating that no tunnel ID was provided
func (ui *CloudflareUI) RenderNoTunnelID() string {
	var sb strings.Builder

	sb.WriteString(ui.Styles.Warning.Render(" ⚠️ No tunnel ID provided "))
	sb.WriteString("\n\n")
	sb.WriteString("Please provide a tunnel ID using the --tunnel-id flag, or use one of the tunnels listed above.")
	sb.WriteString("\n\n")
	sb.WriteString("Example: ")
	sb.WriteString(ui.Styles.Dimmed.Render("unboundCLI cloudflare-sync --tunnel-id=<tunnel-id>"))
	sb.WriteString("\n")

	return sb.String()
}

// RenderFetchingMessage renders a message indicating that Cloudflare tunnel config is being fetched
func (ui *CloudflareUI) RenderFetchingMessage() string {
	return ui.Styles.Info.Render(" 📡 Fetching Cloudflare tunnel configuration... ")
}

// RenderSourceSection renders a section title for Cloudflare tunnel source
func (ui *CloudflareUI) RenderSourceSection() string {
	return ui.Styles.Section.Render(" ☁️ Cloudflare Tunnel Routes ☁️ ")
}

// RenderError renders an error message
func (ui *CloudflareUI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" ❌ Error: %s ", err))
}

// RenderInfo renders an informational message
func (ui *CloudflareUI) RenderInfo(message string) string {
	return ui.Styles.Info.Render(fmt.Sprintf(" ℹ️ %s ", message))
}

// RenderWarning renders a warning message
func (ui *CloudflareUI) RenderWarning(message string) string {
	return ui.Styles.Warning.Render(fmt.Sprintf(" ⚠️ %s ", message))
}
