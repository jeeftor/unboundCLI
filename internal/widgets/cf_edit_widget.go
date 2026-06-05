package widgets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/models"
)

// cfEditField enumerates the focusable fields in the CF edit widget.
type cfEditField int

const (
	cfEditService cfEditField = iota
	cfEditHostHeader
	cfEditNoTLSVerify
	cfEditHttp2Origin
	cfEditSave
	cfEditDelete
	cfEditCancel
	cfEditFieldCount
)

// CFEditWidget is a modal form for editing a Cloudflare tunnel ingress rule.
// The top section shows a read-only overview of all entry state; the bottom
// section has the editable CF tunnel fields.
type CFEditWidget struct {
	BaseWidget

	entry           *models.Entry // kept for overview rendering
	hostname        string
	caddyServiceURL string
	isDefaultTunnel bool // false = entry is in a read-only tunnel, show warning

	serviceInput    textinput.Model
	hostHeaderInput textinput.Model
	noTLSVerify     bool
	http2Origin     bool

	activeField cfEditField

	// Outcome
	done      bool
	cancelled bool
	deleted   bool
	saveErr   string

	theme *Theme
}

// NewCFEditWidget creates a new edit widget pre-filled from the entry's CF status.
func NewCFEditWidget(entry *models.Entry, caddyServiceURL string, theme *Theme) *CFEditWidget {
	si := textinput.New()
	si.Placeholder = "http://192.168.1.15:80"
	si.CharLimit = 120
	si.Width = 52
	si.SetValue(entry.CloudflareStatus.Service)
	si.Focus()

	hi := textinput.New()
	hi.Placeholder = entry.Hostname
	hi.CharLimit = 120
	hi.Width = 52
	hi.SetValue(entry.CloudflareStatus.HTTPHostHeader)

	return &CFEditWidget{
		BaseWidget:      NewBaseWidget(),
		entry:           entry,
		hostname:        entry.Hostname,
		caddyServiceURL: caddyServiceURL,
		isDefaultTunnel: entry.CloudflareStatus.IsDefaultTunnel || !entry.CloudflareStatus.Configured,
		serviceInput:    si,
		hostHeaderInput: hi,
		noTLSVerify:     entry.CloudflareStatus.NoTLSVerify,
		http2Origin:     entry.CloudflareStatus.Http2Origin,
		activeField:     cfEditService,
		theme:           theme,
	}
}

// Init implements Widget.
func (w *CFEditWidget) Init() tea.Cmd {
	return textinput.Blink
}

// IsDone returns true when the user has saved or cancelled.
func (w *CFEditWidget) IsDone() bool { return w.done }

// WasCancelled returns true when the user pressed esc/cancel.
func (w *CFEditWidget) WasCancelled() bool { return w.cancelled }

// WasDeleted returns true when the user confirmed deletion.
func (w *CFEditWidget) WasDeleted() bool { return w.deleted }

// Spec returns the edit result. Only meaningful when IsDone() && !WasCancelled().
func (w *CFEditWidget) Spec() models.CFEditSpec {
	return models.CFEditSpec{
		Hostname:       w.hostname,
		Service:        strings.TrimSpace(w.serviceInput.Value()),
		HTTPHostHeader: strings.TrimSpace(w.hostHeaderInput.Value()),
		NoTLSVerify:    w.noTLSVerify,
		Http2Origin:    w.http2Origin,
	}
}

// SetSaveError stores an error message to display in the widget.
func (w *CFEditWidget) SetSaveError(err string) {
	w.saveErr = err
	w.done = false // allow retrying
}

// Update implements Widget.
func (w *CFEditWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			if !w.isDefaultTunnel {
				w.saveErr = "entry is in a read-only tunnel; cannot edit"
				return w, nil
			}
			w.done = true
			return w, nil

		case "esc":
			w.cancelled = true
			w.done = true
			return w, nil

		case "ctrl+k":
			// Route via Caddy: fill both service and host header
			if w.caddyServiceURL != "" {
				w.serviceInput.SetValue(w.caddyServiceURL)
			}
			w.hostHeaderInput.SetValue(w.hostname)
			return w, nil

		case "ctrl+h":
			// Fill host header from hostname only
			w.hostHeaderInput.SetValue(w.hostname)
			return w, nil

		case "ctrl+d":
			if !w.isDefaultTunnel {
				w.saveErr = "entry is in a read-only tunnel; cannot delete"
				return w, nil
			}
			w.deleted = true
			w.done = true
			return w, nil

		case "tab", "down":
			w.nextField()
			return w, w.focusCurrent()

		case "shift+tab", "up":
			w.prevField()
			return w, w.focusCurrent()

		case " ":
			switch w.activeField {
			case cfEditNoTLSVerify:
				w.noTLSVerify = !w.noTLSVerify
			case cfEditHttp2Origin:
				w.http2Origin = !w.http2Origin
			}
			return w, nil

		case "enter":
			switch w.activeField {
			case cfEditNoTLSVerify:
				w.noTLSVerify = !w.noTLSVerify
			case cfEditHttp2Origin:
				w.http2Origin = !w.http2Origin
			case cfEditSave:
				if !w.isDefaultTunnel {
					w.saveErr = "entry is in a read-only tunnel; cannot edit"
					return w, nil
				}
				w.done = true
			case cfEditDelete:
				if !w.isDefaultTunnel {
					w.saveErr = "entry is in a read-only tunnel; cannot delete"
					return w, nil
				}
				w.deleted = true
				w.done = true
			case cfEditCancel:
				w.cancelled = true
				w.done = true
			default:
				// Move to next field on Enter in text inputs
				w.nextField()
				return w, w.focusCurrent()
			}
			return w, nil
		}
	}

	// Route key events to active text input
	var cmd tea.Cmd
	switch w.activeField {
	case cfEditService:
		w.serviceInput, cmd = w.serviceInput.Update(msg)
	case cfEditHostHeader:
		w.hostHeaderInput, cmd = w.hostHeaderInput.Update(msg)
	}
	return w, cmd
}

func (w *CFEditWidget) nextField() {
	w.activeField = (w.activeField + 1) % cfEditFieldCount
}

func (w *CFEditWidget) prevField() {
	if w.activeField == 0 {
		w.activeField = cfEditFieldCount - 1
	} else {
		w.activeField--
	}
}

func (w *CFEditWidget) focusCurrent() tea.Cmd {
	w.serviceInput.Blur()
	w.hostHeaderInput.Blur()
	switch w.activeField {
	case cfEditService:
		return w.serviceInput.Focus()
	case cfEditHostHeader:
		return w.hostHeaderInput.Focus()
	}
	return nil
}

// View implements Widget.
func (w *CFEditWidget) View() string {
	t := w.theme
	// Shared styles — identical to entry_detail.go for visual consistency
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(t.ColorCyan)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5"))
	goodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
	dimStyle := t.Dimmed
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(t.ColorInfo)
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(t.ColorCyan)

	// lbl matches entry_detail.go — 22-char padded label
	lbl := func(s string) string {
		return labelStyle.Render(fmt.Sprintf("  %-22s", s))
	}
	activeLbl := func(s string) string {
		return activeStyle.Render(fmt.Sprintf("  %-22s", s))
	}
	val := func(s string) string { return valueStyle.Render(s) }
	section := func(s string) string { return sectionStyle.Render("── " + s) }
	yesno := func(b bool) string {
		if b {
			return goodStyle.Render("✓ Yes")
		}
		return errStyle.Render("✗ No")
	}
	checkbox := func(checked, active bool) string {
		mark := "[ ]"
		if checked {
			mark = "[x]"
		}
		if active {
			return activeStyle.Render(mark)
		}
		if checked {
			return goodStyle.Render(mark)
		}
		return dimStyle.Render(mark)
	}
	makeBtn := func(label string, fg lipgloss.Color, active bool) string {
		s := lipgloss.NewStyle().Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(fg).
			Foreground(fg)
		if active {
			s = s.Bold(true)
		}
		return s.Render(label)
	}

	var lines []string
	e := w.entry

	// ── Title ──────────────────────────────────────────────────────────────
	statusIcon := e.OverallStatus.Icon()
	statusLabel := e.OverallStatus.Label()
	var statusStyled string
	switch e.OverallStatus {
	case models.FullyInSync:
		statusStyled = goodStyle.Render(statusIcon + " " + statusLabel)
	case models.OutOfSync, models.Stale:
		statusStyled = errStyle.Render(statusIcon + " " + statusLabel)
	default:
		statusStyled = warnStyle.Render(statusIcon + " " + statusLabel)
	}
	title := titleStyle.Render("Edit: "+w.hostname) + "  " + statusStyled
	if !w.isDefaultTunnel {
		title += "  " + warnStyle.Render("⚠ read-only")
	}
	lines = append(lines, title)
	lines = append(lines, "")

	// ── Caddy ──────────────────────────────────────────────────────────────
	lines = append(lines, section("Caddy (Source of Truth)"))
	if e.CaddyUpstream != "" {
		lines = append(lines, lbl("Upstream:")+val(e.CaddyUpstream))
		if e.CaddyIP != "" {
			lines = append(lines, lbl("IP:")+val(e.CaddyIP))
		}
	} else {
		lines = append(lines, lbl("Upstream:")+dimStyle.Render("(not in Caddy)"))
	}
	lines = append(lines, "")

	// ── DNS + Services (compact two-column) ────────────────────────────────
	lines = append(lines, section("DNS & Services"))
	// DNS resolved
	switch e.DNSResolved {
	case "", "NONE":
		lines = append(lines, lbl("DNS Resolved:")+dimStyle.Render("(not resolved)"))
	case "FAIL":
		lines = append(lines, lbl("DNS Resolved:")+errStyle.Render("FAIL"))
	default:
		lines = append(lines, lbl("DNS Resolved:")+val(e.DNSResolved))
	}
	// Unbound
	if e.UnboundStatus.Configured {
		lines = append(lines, lbl("UnboundDNS:")+yesno(e.UnboundStatus.InSync)+dimStyle.Render("  "+e.UnboundStatus.IP))
	} else {
		lines = append(lines, lbl("UnboundDNS:")+dimStyle.Render("not configured"))
	}
	// AdGuard
	if e.AdguardStatus.Configured {
		lines = append(lines, lbl("AdGuard:")+yesno(e.AdguardStatus.InSync)+dimStyle.Render("  "+e.AdguardStatus.IP))
	} else {
		lines = append(lines, lbl("AdGuard:")+dimStyle.Render("not configured"))
	}
	// DHCP
	if !e.DHCPStatus.Configured {
		lines = append(lines, lbl("DHCP:")+dimStyle.Render("no lease"))
	} else {
		leaseType := warnStyle.Render(e.DHCPStatus.Type)
		if e.DHCPStatus.IsStatic() {
			leaseType = goodStyle.Render(e.DHCPStatus.Type)
		}
		dhcpLine := leaseType + " " + val(e.DHCPStatus.IP)
		if e.DHCPStatus.MAC != "" {
			dhcpLine += dimStyle.Render("  " + e.DHCPStatus.MAC)
		}
		lines = append(lines, lbl("DHCP:")+dhcpLine)
	}
	lines = append(lines, "")

	// ── Cloudflare (editable) ──────────────────────────────────────────────
	lines = append(lines, section("Cloudflare Tunnel  (editable)"))
	lines = append(lines, "")

	// Service URL
	svcLbl := lbl("Service URL:")
	if w.activeField == cfEditService {
		svcLbl = activeLbl("Service URL:")
	}
	lines = append(lines, svcLbl)
	lines = append(lines, "    "+w.serviceInput.View())
	lines = append(lines, "")

	// HTTPHostHeader
	hhLbl := lbl("HTTPHostHeader:")
	if w.activeField == cfEditHostHeader {
		hhLbl = activeLbl("HTTPHostHeader:")
	}
	lines = append(lines, hhLbl)
	lines = append(lines, "    "+w.hostHeaderInput.View())
	lines = append(lines, "")

	// Toggles — same label format as view widget but with checkbox prefix
	noTLSActive := w.activeField == cfEditNoTLSVerify
	noTLSLine := checkbox(w.noTLSVerify, noTLSActive) + " "
	if noTLSActive {
		noTLSLine += activeLbl("No TLS Verify")
	} else {
		noTLSLine += lbl("No TLS Verify")
	}
	if w.noTLSVerify {
		noTLSLine += warnStyle.Render("(insecure)")
	}
	lines = append(lines, noTLSLine)

	h2Active := w.activeField == cfEditHttp2Origin
	h2Line := checkbox(w.http2Origin, h2Active) + " "
	if h2Active {
		h2Line += activeLbl("HTTP/2 to origin")
	} else {
		h2Line += lbl("HTTP/2 to origin")
	}
	lines = append(lines, h2Line)
	lines = append(lines, "")

	// Quick-fill shortcuts
	lines = append(lines, dimStyle.Render("  ctrl+k  ⚡ Route via Caddy  (service URL + host header)"))
	lines = append(lines, dimStyle.Render("  ctrl+h  fill host header from hostname only"))
	lines = append(lines, dimStyle.Render("  ctrl+d  delete this tunnel entry"))
	lines = append(lines, "")

	// Buttons — JoinHorizontal places bordered elements side by side.
	// No string prefix here: the outer container's Padding(1,2) provides indentation.
	// Prepending "  " to a multi-line bordered string only indents the first line.
	saveBtnLabel := "ctrl+s  Save"
	if w.done && !w.cancelled && !w.deleted {
		saveBtnLabel = "⏳ Saving..."
	}
	deleteBtnLabel := "ctrl+d  Delete"
	if w.done && w.deleted {
		deleteBtnLabel = "⏳ Deleting..."
	}
	saveBtn := makeBtn(saveBtnLabel, t.ColorCyan, w.activeField == cfEditSave)
	deleteBtn := makeBtn(deleteBtnLabel, lipgloss.Color("#e0af68"), w.activeField == cfEditDelete)
	cancelBtn := makeBtn("esc  Cancel", lipgloss.Color("#f7768e"), w.activeField == cfEditCancel)
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, saveBtn, "   ", deleteBtn, "   ", cancelBtn))

	if w.saveErr != "" {
		lines = append(lines, "")
		lines = append(lines, errStyle.Render("  ✗ "+w.saveErr))
	}
	if !w.isDefaultTunnel {
		lines = append(lines, "")
		lines = append(lines, warnStyle.Render("  ⚠ Entry is in a read-only tunnel — save is disabled."))
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("  tab/↓ next  •  shift+tab/↑ prev  •  space/enter toggle"))

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.ColorCyan).
		Padding(1, 2).
		Width(72).
		Render(content)
}

// Implement remaining Widget interface methods (unused but required):
func (w *CFEditWidget) Focus()        { w.BaseWidget.Focus() }
func (w *CFEditWidget) Blur()         { w.BaseWidget.Blur() }
func (w *CFEditWidget) Focused() bool { return w.BaseWidget.Focused() }
