package widgets

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

// RenderEntryDetail renders a modal panel showing all fields for a single entry.
// Returns a string ready to be placed with lipgloss.Place.
func RenderEntryDetail(entry *models.Entry, theme *Theme) string {
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorCyan)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5"))
	goodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
	dimStyle := theme.Dimmed
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorInfo)

	lbl := func(s string) string {
		return labelStyle.Render(fmt.Sprintf("  %-22s", s))
	}
	val := func(s string) string {
		return valueStyle.Render(s)
	}
	section := func(s string) string {
		return sectionStyle.Render("── " + s)
	}
	yesno := func(b bool) string {
		if b {
			return goodStyle.Render("✓ Yes")
		}
		return errStyle.Render("✗ No")
	}

	var lines []string

	// Title with overall status
	statusIcon := entry.OverallStatus.Icon()
	statusLabel := entry.OverallStatus.Label()
	var statusStyled string
	switch entry.OverallStatus {
	case models.FullyInSync:
		statusStyled = goodStyle.Render(statusIcon + " " + statusLabel)
	case models.OutOfSync, models.Stale:
		statusStyled = errStyle.Render(statusIcon + " " + statusLabel)
	default:
		statusStyled = warnStyle.Render(statusIcon + " " + statusLabel)
	}
	lines = append(lines, titleStyle.Render("Entry: "+entry.Hostname)+"  "+statusStyled)
	lines = append(lines, "")

	// ── Caddy ──────────────────────────────────────────────────────────────
	lines = append(lines, section("Caddy (Source of Truth)"))
	if entry.CaddyUpstream != "" {
		lines = append(lines, lbl("Upstream:")+val(entry.CaddyUpstream))
		if entry.CaddyIP != "" {
			lines = append(lines, lbl("IP:")+val(entry.CaddyIP))
		}
		if entry.CaddyPort != "" {
			lines = append(lines, lbl("Port:")+val(entry.CaddyPort))
		}

		// Handler chain
		r := entry.CaddyRoute
		if len(r.HandlerChain) > 0 {
			lines = append(lines, lbl("Handler chain:")+dimStyle.Render(strings.Join(r.HandlerChain, " → ")))
		}

		// TLS to upstream
		if r.TLSToUpstream {
			lines = append(lines, lbl("TLS to upstream:")+goodStyle.Render("✓ Yes"))
		} else {
			lines = append(lines, lbl("TLS to upstream:")+dimStyle.Render("✗ No (plain HTTP)"))
		}

		// Request headers being set/added
		if len(r.RequestHeadersSet)+len(r.RequestHeadersAdd) > 0 {
			lines = append(lines, lbl("Request headers:"))
			// collect and sort for stable output
			hdrs := make([]string, 0, len(r.RequestHeadersSet)+len(r.RequestHeadersAdd))
			for k := range r.RequestHeadersSet {
				hdrs = append(hdrs, k)
			}
			sort.Strings(hdrs)
			for _, k := range hdrs {
				lines = append(lines, "    "+labelStyle.Render(fmt.Sprintf("%-24s", k))+dimStyle.Render(r.RequestHeadersSet[k]))
			}
			hdrs2 := make([]string, 0, len(r.RequestHeadersAdd))
			for k := range r.RequestHeadersAdd {
				hdrs2 = append(hdrs2, k)
			}
			sort.Strings(hdrs2)
			for _, k := range hdrs2 {
				lines = append(lines, "    "+labelStyle.Render(fmt.Sprintf("%-24s", k))+dimStyle.Render(r.RequestHeadersAdd[k])+" "+dimStyle.Render("(add)"))
			}
		}

		// Response headers
		if len(r.ResponseHeadersSet) > 0 {
			lines = append(lines, lbl("Response headers:"))
			hdrs := make([]string, 0, len(r.ResponseHeadersSet))
			for k := range r.ResponseHeadersSet {
				hdrs = append(hdrs, k)
			}
			sort.Strings(hdrs)
			for _, k := range hdrs {
				lines = append(lines, "    "+labelStyle.Render(fmt.Sprintf("%-24s", k))+dimStyle.Render(r.ResponseHeadersSet[k]))
			}
		}

		// Note: Caddy v2 sets X-Forwarded-Proto automatically even if absent here.
		// A redirect loop is more likely caused by the backend's own HTTPS-enforce
		// setting (e.g. AdGuard Home "force HTTPS") ignoring proxy headers entirely.
	} else {
		lines = append(lines, lbl("Upstream:")+dimStyle.Render("(not in Caddy)"))
	}
	lines = append(lines, "")

	// ── DNS Resolution ─────────────────────────────────────────────────────
	lines = append(lines, section("DNS Resolution"))
	switch entry.DNSResolved {
	case "", "NONE":
		lines = append(lines, lbl("Resolved:")+dimStyle.Render("(not resolved)"))
	case "FAIL":
		lines = append(lines, lbl("Resolved:")+errStyle.Render("FAIL"))
	default:
		lines = append(lines, lbl("Resolved:")+val(entry.DNSResolved))
	}
	lines = append(lines, "")

	// ── UnboundDNS ─────────────────────────────────────────────────────────
	lines = append(lines, section("UnboundDNS"))
	lines = append(lines, lbl("Configured:")+yesno(entry.UnboundStatus.Configured))
	if entry.UnboundStatus.Configured {
		lines = append(lines, lbl("IP:")+val(entry.UnboundStatus.IP))
		lines = append(lines, lbl("In Sync:")+yesno(entry.UnboundStatus.InSync))
	}
	lines = append(lines, "")

	// ── AdGuard Home ───────────────────────────────────────────────────────
	lines = append(lines, section("AdGuard Home"))
	lines = append(lines, lbl("Configured:")+yesno(entry.AdguardStatus.Configured))
	if entry.AdguardStatus.Configured {
		lines = append(lines, lbl("IP:")+val(entry.AdguardStatus.IP))
		lines = append(lines, lbl("In Sync:")+yesno(entry.AdguardStatus.InSync))
	}
	lines = append(lines, "")

	// ── DHCP ───────────────────────────────────────────────────────────────
	lines = append(lines, section("DHCP"))
	d := entry.DHCPStatus
	if !d.Configured {
		lines = append(lines, lbl("Lease:")+dimStyle.Render("(no lease)"))
	} else {
		var leaseTypeStr string
		if d.IsStatic() {
			leaseTypeStr = goodStyle.Render("static")
		} else {
			leaseTypeStr = warnStyle.Render("dynamic")
		}
		lines = append(lines, lbl("Lease Type:")+leaseTypeStr)
		lines = append(lines, lbl("IP:")+val(d.IP))
		if d.MAC != "" {
			lines = append(lines, lbl("MAC:")+val(d.MAC))
		}
		if d.Hostname != "" {
			lines = append(lines, lbl("DHCP Hostname:")+val(d.Hostname))
		}
		lines = append(lines, lbl("IP Matches Caddy:")+yesno(d.InSync))
	}
	lines = append(lines, "")

	// ── Cloudflare Tunnel ──────────────────────────────────────────────────
	lines = append(lines, section("Cloudflare Tunnel"))
	cf := entry.CloudflareStatus
	if !cf.Configured {
		lines = append(lines, lbl("Tunnel:")+dimStyle.Render("(not configured)"))
	} else {
		var tunnelStr string
		if cf.IsDefaultTunnel {
			tunnelStr = val(cf.TunnelName) + "  " + goodStyle.Render("★ default")
		} else {
			tunnelStr = val(cf.TunnelName) + "  " + dimStyle.Render("(read-only)")
		}
		lines = append(lines, lbl("Tunnel:")+tunnelStr)
		lines = append(lines, lbl("Service:")+val(cf.Service))

		if cf.Path == "" {
			lines = append(lines, lbl("Path:")+dimStyle.Render("(none)"))
		} else {
			lines = append(lines, lbl("Path:")+val(cf.Path))
		}

		if cf.HTTPHostHeader == "" {
			lines = append(lines, lbl("HTTPHostHeader:")+warnStyle.Render("NOT SET  ⚠  may break routing"))
		} else {
			lines = append(lines, lbl("HTTPHostHeader:")+val(cf.HTTPHostHeader)+"  "+goodStyle.Render("✓"))
		}

		tlsVerify := goodStyle.Render("enabled")
		if cf.NoTLSVerify {
			tlsVerify = warnStyle.Render("disabled (insecure)")
		}
		lines = append(lines, lbl("TLS Verify:")+tlsVerify)

		h2 := dimStyle.Render("disabled")
		if cf.Http2Origin {
			h2 = val("enabled")
		}
		lines = append(lines, lbl("HTTP/2 to origin:")+h2)

		access := dimStyle.Render("none")
		if cf.HasAccessPolicy {
			access = goodStyle.Render("required")
		}
		lines = append(lines, lbl("Access Policy:")+access)
	}

	// Source
	if entry.DataSource != "" {
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("  Source: "+entry.DataSource))
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("v / esc  close"))

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorCyan).
		Padding(1, 3).
		Width(72).
		Render(content)
}
