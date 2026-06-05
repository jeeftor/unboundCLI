package widgets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

// RenderCFDetail renders a modal read-only panel showing Cloudflare ingress details
// for the given entry. Returns a string ready to be placed with lipgloss.Place.
func RenderCFDetail(entry *models.Entry, theme *Theme) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorInfo)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5"))
	goodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68"))
	dimStyle := theme.Dimmed

	label := func(s string) string {
		return labelStyle.Render(fmt.Sprintf("%-20s", s))
	}

	var lines []string
	lines = append(lines, titleStyle.Render("Cloudflare: "+entry.Hostname))
	lines = append(lines, "")

	cf := entry.CloudflareStatus
	if !cf.Configured {
		lines = append(lines, dimStyle.Render("Not configured in any Cloudflare tunnel."))
	} else {
		// Tunnel name with default/non-default indicator
		tunnelLabel := cf.TunnelName
		if cf.IsDefaultTunnel {
			tunnelLabel += "  " + goodStyle.Render("★ default")
		} else {
			tunnelLabel += "  " + dimStyle.Render("(read-only)")
		}
		lines = append(lines, label("Tunnel:")+" "+valueStyle.Render(tunnelLabel))
		lines = append(lines, label("Service:")+" "+valueStyle.Render(cf.Service))

		path := cf.Path
		if path == "" {
			path = dimStyle.Render("(none)")
		} else {
			path = valueStyle.Render(path)
		}
		lines = append(lines, label("Path:")+" "+path)

		// HTTPHostHeader — critical for CF→Caddy routing
		hostHeader := cf.HTTPHostHeader
		if hostHeader == "" {
			hostHeader = warnStyle.Render("NOT SET  ⚠  CF→Caddy routing may break")
		} else {
			hostHeader = valueStyle.Render(hostHeader) + "  " + goodStyle.Render("✓")
		}
		lines = append(lines, label("HTTPHostHeader:")+" "+hostHeader)

		// TLS verify
		tlsVerify := goodStyle.Render("enabled")
		if cf.NoTLSVerify {
			tlsVerify = warnStyle.Render("disabled (insecure)")
		}
		lines = append(lines, label("TLS Verify:")+" "+tlsVerify)

		// HTTP/2 to origin
		h2 := dimStyle.Render("disabled")
		if cf.Http2Origin {
			h2 = valueStyle.Render("enabled")
		}
		lines = append(lines, label("HTTP/2 to origin:")+" "+h2)

		// Access policy
		access := dimStyle.Render("none")
		if cf.HasAccessPolicy {
			access = goodStyle.Render("required")
		}
		lines = append(lines, label("Access Policy:")+" "+access)
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("i / esc  close"))

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorInfo).
		Padding(1, 3).
		Width(66).
		Render(content)
}
