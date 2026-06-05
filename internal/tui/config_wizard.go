package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/widgets"
)

type configWizardPhase int

const (
	configPhaseEditing configWizardPhase = iota
	configPhaseSaving
	configPhaseTesting
	configPhaseDone
)

type cfPickerMode int

const (
	cfPickerZone   cfPickerMode = iota
	cfPickerTunnel cfPickerMode = iota
)

// --- message types ---

type connTestResult struct {
	name     string
	ok       bool
	disabled bool
	errMsg   string
}

type connTestResultsMsg struct {
	results []connTestResult
}

type configSavedMsg struct{ err error }

type cfZonesFetchedMsg struct {
	zones []api.CloudflareZone
	err   error
}

type cfTunnelsFetchedMsg struct {
	tunnels []api.CloudflareTunnel
	err     error
}

// --- wizard model ---

type ConfigWizard struct {
	editor      *widgets.ConfigEditorWidget
	phase       configWizardPhase
	testResults []connTestResult
	saveErr     error
	statusMsg   string
	width       int
	height      int

	// Cloudflare picker state
	cfPicker     *widgets.CFPickerWidget
	showCFPicker bool
	cfPickerMode cfPickerMode
	cfZoneName   string // display name after zone is resolved
	cfTunnelName string // display name after tunnel is resolved
	cfDebugCurl  string // curl equivalent of the last CF API call, shown for debugging
}

func NewConfigWizard() *ConfigWizard {
	return &ConfigWizard{}
}

func (w *ConfigWizard) Init() tea.Cmd {
	existing, _ := config.LoadExtendedConfig()

	editor := widgets.NewConfigEditor()

	insecureStr := "false"
	if existing.Config.Insecure {
		insecureStr = "true"
	}

	adguardEnabledStr := "false"
	if existing.Adguard.Enabled {
		adguardEnabledStr = "true"
	}
	adguardInsecureStr := "false"
	if existing.Adguard.Insecure {
		adguardInsecureStr = "true"
	}

	cfEnabledStr := "false"
	// Auto-enable when credentials are present, even if the flag was not explicitly set.
	if existing.Cloudflare.Enabled || existing.Cloudflare.APIToken != "" {
		cfEnabledStr = "true"
	}

	cfInsecureStr := "false"
	if existing.Cloudflare.Insecure {
		cfInsecureStr = "true"
	}

	sections := []widgets.ConfigSection{
		{
			Title: "UnboundDNS",
			Fields: []widgets.ConfigField{
				{Key: "base_url", Label: "Base URL", Value: existing.Config.BaseURL, Placeholder: "https://192.168.1.1", IsRequired: true},
				{Key: "api_key", Label: "API Key", Value: existing.Config.APIKey, Placeholder: "API key", IsPassword: true},
				{Key: "api_secret", Label: "API Secret", Value: existing.Config.APISecret, Placeholder: "API secret", IsPassword: true},
				{Key: "insecure", Label: "Skip SSL Verify", Value: insecureStr, IsToggle: true},
			},
		},
		{
			Title: "AdguardHome",
			Fields: []widgets.ConfigField{
				{Key: "adguard_enabled", Label: "Enabled", Value: adguardEnabledStr, IsToggle: true},
				{Key: "adguard_base_url", Label: "Base URL", Value: existing.Adguard.BaseURL, Placeholder: "http://192.168.1.10:3000"},
				{Key: "adguard_username", Label: "Username", Value: existing.Adguard.Username, Placeholder: "admin"},
				{Key: "adguard_password", Label: "Password", Value: existing.Adguard.Password, Placeholder: "password", IsPassword: true},
				{Key: "adguard_insecure", Label: "Skip SSL Verify", Value: adguardInsecureStr, IsToggle: true},
			},
		},
		{
			Title: "Cloudflare",
			Fields: []widgets.ConfigField{
				{Key: "cf_enabled", Label: "Enabled", Value: cfEnabledStr, IsToggle: true},
				{Key: "cf_api_token", Label: "API Token", Value: existing.Cloudflare.APIToken, Placeholder: "CF API token", IsPassword: true},
				{Key: "cf_account_id", Label: "Account ID", Value: existing.Cloudflare.AccountID, Placeholder: "CF account ID"},
				{Key: "cf_zone_id", Label: "Zone ID", Value: existing.Cloudflare.ZoneID, Placeholder: "press z to browse", HelpText: "Press z (outside edit mode) to fetch and pick from your zones"},
				{Key: "cf_tunnel_id", Label: "Default Tunnel", Value: existing.Cloudflare.TunnelID, Placeholder: "press t to browse", HelpText: "Tunnel to add/remove entries in. Other tunnels are scanned read-only."},
				{Key: "cf_insecure", Label: "Skip SSL Verify", Value: cfInsecureStr, IsToggle: true},
			},
		},
		{
			Title: "Caddy",
			Fields: []widgets.ConfigField{
				{Key: "cf_caddy_service_url", Label: "Admin API URL", Value: existing.Cloudflare.CaddyServiceURL, Placeholder: "http://caddy:2019", HelpText: "Caddy admin API used to read reverse-proxy hostnames"},
			},
		},
	}

	editor.SetSections(sections)
	editor.SetSize(80, 40)
	editor.Focus()

	w.editor = editor

	// Pre-populate resolved names if IDs already exist in config
	w.cfZoneName = existing.Cloudflare.ZoneID
	w.cfTunnelName = existing.Cloudflare.TunnelID

	return editor.Init()
}

func (w *ConfigWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height
		if w.editor != nil {
			w.editor.SetSize(msg.Width-4, msg.Height-8)
		}
		if w.cfPicker != nil {
			w.cfPicker.SetSize(msg.Width, msg.Height)
		}
		return w, nil

	case configSavedMsg:
		w.phase = configPhaseEditing
		if msg.err != nil {
			w.saveErr = msg.err
			w.statusMsg = "Error saving: " + msg.err.Error()
		} else {
			w.saveErr = nil
			w.statusMsg = "Configuration saved successfully!"
		}
		return w, nil

	case connTestResultsMsg:
		w.phase = configPhaseEditing
		w.testResults = msg.results
		return w, nil

	case cfZonesFetchedMsg:
		if msg.err != nil {
			w.statusMsg = "Failed to fetch zones: " + msg.err.Error()
			return w, nil
		}
		if len(msg.zones) == 0 {
			w.statusMsg = "No zones found for this API token"
			return w, nil
		}
		if len(msg.zones) == 1 {
			z := msg.zones[0]
			if w.editor != nil {
				w.editor.SetValue("cf_zone_id", z.ID)
			}
			w.cfZoneName = z.Name
			w.statusMsg = fmt.Sprintf("Zone auto-selected: %s", z.Name)
			return w, nil
		}
		// Multiple zones — show picker
		items := make([]widgets.CFPickerItem, len(msg.zones))
		for i, z := range msg.zones {
			items[i] = widgets.CFPickerItem{ID: z.ID, Name: z.Name}
		}
		picker := widgets.NewCFPicker("Select Cloudflare Zone", items)
		picker.SetSize(w.width, w.height)
		w.cfPicker = picker
		w.cfPickerMode = cfPickerZone
		w.showCFPicker = true
		return w, nil

	case cfTunnelsFetchedMsg:
		if msg.err != nil {
			w.statusMsg = "Failed to fetch tunnels: " + msg.err.Error()
			return w, nil
		}
		if len(msg.tunnels) == 0 {
			w.statusMsg = "No tunnels found for this account"
			return w, nil
		}
		if len(msg.tunnels) == 1 {
			t := msg.tunnels[0]
			if w.editor != nil {
				w.editor.SetValue("cf_tunnel_id", t.ID)
			}
			w.cfTunnelName = t.Name
			w.statusMsg = fmt.Sprintf("Tunnel auto-selected: %s", t.Name)
			return w, nil
		}
		// Multiple tunnels — show picker
		items := make([]widgets.CFPickerItem, len(msg.tunnels))
		for i, t := range msg.tunnels {
			items[i] = widgets.CFPickerItem{ID: t.ID, Name: t.Name}
		}
		picker := widgets.NewCFPicker("Select Cloudflare Tunnel", items)
		picker.SetSize(w.width, w.height)
		w.cfPicker = picker
		w.cfPickerMode = cfPickerTunnel
		w.showCFPicker = true
		return w, nil

	case widgets.CFPickerSelectedMsg:
		w.showCFPicker = false
		w.cfPicker = nil
		if w.cfPickerMode == cfPickerZone {
			if w.editor != nil {
				w.editor.SetValue("cf_zone_id", msg.Item.ID)
			}
			w.cfZoneName = msg.Item.Name
			w.statusMsg = fmt.Sprintf("Zone selected: %s", msg.Item.Name)
		} else {
			if w.editor != nil {
				w.editor.SetValue("cf_tunnel_id", msg.Item.ID)
			}
			w.cfTunnelName = msg.Item.Name
			w.statusMsg = fmt.Sprintf("Tunnel selected: %s", msg.Item.Name)
		}
		return w, nil

	case widgets.CFPickerCancelledMsg:
		w.showCFPicker = false
		w.cfPicker = nil
		w.statusMsg = ""
		return w, nil

	case tea.KeyMsg:
		// While the CF picker is open, route all input to it.
		if w.showCFPicker && w.cfPicker != nil {
			updated, cmd := w.cfPicker.Update(msg)
			w.cfPicker = updated
			return w, cmd
		}

		switch msg.String() {
		case "ctrl+c":
			return w, tea.Quit
		case "q":
			// Only quit when not actively typing in a field.
			if w.editor == nil || !w.editor.Focused() {
				return w, tea.Quit
			}

		case "ctrl+s":
			if w.editor != nil {
				w.phase = configPhaseSaving
				w.statusMsg = "Saving..."
				vals := w.getAllCurrentValues()
				existing, _ := config.LoadExtendedConfig()
				return w, saveConfigCmd(vals, existing)
			}
			return w, nil

		case "ctrl+t":
			if w.editor != nil {
				w.phase = configPhaseTesting
				w.statusMsg = "Testing connections..."
				vals := w.getAllCurrentValues()
				return w, testConnectionsCmd(vals)
			}
			return w, nil

		case "ctrl+p":
			if w.editor != nil {
				w.editor.TogglePasswordVisibility()
			}
			return w, nil

		case "z":
			// Only intercept when not in text-editing mode.
			if w.editor != nil && !w.editor.Focused() {
				token := w.editor.GetInputValue("cf_api_token")
				if token == "" {
					w.statusMsg = "Enter CF API Token first, then press z to browse zones"
					return w, nil
				}
				w.cfDebugCurl = cfZonesCurl(token)
				w.statusMsg = "Fetching Cloudflare zones..."
				return w, fetchCFZonesCmd(token)
			}

		case "t":
			// Only intercept when not in text-editing mode.
			if w.editor != nil && !w.editor.Focused() {
				token := w.editor.GetInputValue("cf_api_token")
				accountID := w.editor.GetInputValue("cf_account_id")
				if token == "" || accountID == "" {
					w.statusMsg = "Enter CF API Token and Account ID first, then press t to browse tunnels"
					return w, nil
				}
				w.cfDebugCurl = cfTunnelsCurl(accountID, token)
				w.statusMsg = "Fetching Cloudflare tunnels..."
				return w, fetchCFTunnelsCmd(token, accountID)
			}
		}
	}

	if w.editor != nil {
		updated, cmd := w.editor.Update(msg)
		if ce, ok := updated.(*widgets.ConfigEditorWidget); ok {
			w.editor = ce
		}
		return w, cmd
	}

	return w, nil
}

func (w *ConfigWizard) getAllCurrentValues() map[string]string {
	if w.editor == nil {
		return map[string]string{}
	}
	return w.editor.GetAllValues()
}

func (w *ConfigWizard) View() string {
	// When the CF picker is active, show it fullscreen (centered).
	if w.showCFPicker && w.cfPicker != nil {
		pickerView := w.cfPicker.View()
		if w.width > 0 && w.height > 0 {
			return lipgloss.Place(w.width, w.height, lipgloss.Center, lipgloss.Center, pickerView)
		}
		return pickerView
	}

	var parts []string

	if w.editor != nil {
		parts = append(parts, w.editor.View())
	}

	parts = append(parts, "")

	// CF resolution status (zone/tunnel names) — shown when on Cloudflare (2) or Caddy (3) tabs
	onCFSection := w.editor != nil && (w.editor.GetCurrentSection() == 2 || w.editor.GetCurrentSection() == 3)
	if onCFSection && (w.cfZoneName != "" || w.cfTunnelName != "") {
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff"))
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89"))
		var cfParts []string
		if w.cfZoneName != "" {
			cfParts = append(cfParts, infoStyle.Render("Zone: ")+w.cfZoneName)
		}
		if w.cfTunnelName != "" {
			cfParts = append(cfParts, infoStyle.Render("Tunnel: ")+w.cfTunnelName)
		}
		parts = append(parts, dimStyle.Render("CF: ")+strings.Join(cfParts, dimStyle.Render("  |  ")))
	}

	// Status message
	if w.statusMsg != "" {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))
		if w.saveErr != nil {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
		}
		parts = append(parts, style.Render(w.statusMsg))
	}

	// CF debug curl — shown on the CF/Caddy tabs whenever a fetch has been attempted
	if onCFSection && w.cfDebugCurl != "" {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89"))
		curlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff"))
		parts = append(parts, dimStyle.Render("curl equiv: ")+curlStyle.Render(w.cfDebugCurl))
	}

	// Connection test results
	if len(w.testResults) > 0 {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))
		failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
		disabledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89"))
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#c0caf5")).Width(16)

		parts = append(parts, lipgloss.NewStyle().Bold(true).Render("Connection tests:"))
		for _, r := range w.testResults {
			label := labelStyle.Render(r.name)
			var status string
			if r.disabled {
				status = disabledStyle.Render("(disabled)")
			} else if r.ok {
				status = successStyle.Render("✓ success")
			} else {
				status = failStyle.Render("✗ failed — " + r.errMsg)
			}
			parts = append(parts, fmt.Sprintf("  %s %s", label, status))
		}
	}

	// Phase indicator
	if w.phase == configPhaseSaving {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7")).Render("Saving..."))
	} else if w.phase == configPhaseTesting {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7")).Render("Testing connections..."))
	}

	parts = append(parts, "")

	// Key bindings bar
	showHide := "show"
	if w.editor != nil && w.editor.ShowingPasswords() {
		showHide = "hide"
	}
	keyHints := []string{
		"ctrl+s save",
		"ctrl+t test connections",
		fmt.Sprintf("ctrl+p %s passwords", showHide),
	}
	if onCFSection {
		keyHints = append(keyHints, "z browse zones", "t browse tunnels")
	}
	keyHints = append(keyHints, "ctrl+c/q quit")

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89"))
	parts = append(parts, dimStyle.Render(strings.Join(keyHints, "  |  ")))

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (w *ConfigWizard) Run() error {
	p := tea.NewProgram(w, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (w *ConfigWizard) Start() error {
	return w.Run()
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// buildExtendedConfig constructs an ExtendedConfig from editor values, starting
// from an existing config so unedited fields are preserved.
func buildExtendedConfig(vals map[string]string, existing config.ExtendedConfig) config.ExtendedConfig {
	cfg := existing
	cfg.Config = api.Config{
		APIKey:    vals["api_key"],
		APISecret: vals["api_secret"],
		BaseURL:   vals["base_url"],
		Insecure:  vals["insecure"] == "true",
	}
	cfg.Adguard = config.AdguardConfig{
		Enabled:  vals["adguard_enabled"] == "true",
		BaseURL:  vals["adguard_base_url"],
		Username: vals["adguard_username"],
		Password: vals["adguard_password"],
		Insecure: vals["adguard_insecure"] == "true",
	}
	cfg.Cloudflare = config.CloudflareConfig{
		Enabled:         vals["cf_enabled"] == "true",
		APIToken:        vals["cf_api_token"],
		AccountID:       vals["cf_account_id"],
		ZoneID:          vals["cf_zone_id"],
		TunnelID:        vals["cf_tunnel_id"],
		Insecure:        vals["cf_insecure"] == "true",
		CaddyServiceURL: vals["cf_caddy_service_url"],
	}
	return cfg
}

func saveConfigCmd(vals map[string]string, existing config.ExtendedConfig) tea.Cmd {
	return func() tea.Msg {
		configPath, err := config.GetDefaultConfigPath()
		if err != nil {
			return configSavedMsg{err: err}
		}
		return configSavedMsg{err: config.SaveExtendedConfig(buildExtendedConfig(vals, existing), configPath)}
	}
}

func testConnectionsCmd(vals map[string]string) tea.Cmd {
	return func() tea.Msg {
		var results []connTestResult

		cfg := api.Config{
			APIKey:    vals["api_key"],
			APISecret: vals["api_secret"],
			BaseURL:   vals["base_url"],
			Insecure:  vals["insecure"] == "true",
		}
		client := api.NewClient(cfg)
		_, err := client.GetOverrides()
		results = append(results, connTestResult{name: "UnboundDNS", ok: err == nil, errMsg: errStr(err)})

		if vals["adguard_enabled"] == "true" {
			agCfg := api.AdguardConfig{
				BaseURL:  vals["adguard_base_url"],
				Username: vals["adguard_username"],
				Password: vals["adguard_password"],
				Insecure: vals["adguard_insecure"] == "true",
				Enabled:  true,
			}
			agClient := api.NewAdguardClient(agCfg)
			_, err := agClient.ListRewrites()
			results = append(results, connTestResult{name: "AdguardHome", ok: err == nil, errMsg: errStr(err)})
		} else {
			results = append(results, connTestResult{name: "AdguardHome", disabled: true})
		}

		if vals["cf_enabled"] == "true" && vals["cf_api_token"] != "" {
			cfClient, err := api.NewCloudflareClient(api.CloudflareConfig{
				APIToken:  vals["cf_api_token"],
				AccountID: vals["cf_account_id"],
				ZoneID:    vals["cf_zone_id"],
				TunnelID:  vals["cf_tunnel_id"],
			})
			if err == nil {
				_, err = cfClient.ListTunnels()
				results = append(results, connTestResult{name: "Cloudflare", ok: err == nil, errMsg: errStr(err)})
			} else {
				results = append(results, connTestResult{name: "Cloudflare", ok: false, errMsg: err.Error()})
			}
		} else {
			results = append(results, connTestResult{name: "Cloudflare", disabled: true})
		}

		return connTestResultsMsg{results: results}
	}
}

func fetchCFZonesCmd(token string) tea.Cmd {
	return func() tea.Msg {
		client, err := api.NewCloudflareClient(api.CloudflareConfig{APIToken: token})
		if err != nil {
			return cfZonesFetchedMsg{err: err}
		}
		zones, err := client.ListZones()
		return cfZonesFetchedMsg{zones: zones, err: err}
	}
}

func fetchCFTunnelsCmd(token, accountID string) tea.Cmd {
	return func() tea.Msg {
		client, err := api.NewCloudflareClient(api.CloudflareConfig{
			APIToken:  token,
			AccountID: accountID,
		})
		if err != nil {
			return cfTunnelsFetchedMsg{err: err}
		}
		tunnels, err := client.ListTunnels()
		return cfTunnelsFetchedMsg{tunnels: tunnels, err: err}
	}
}

// cfZonesCurl returns the curl equivalent for a ListZones call.
func cfZonesCurl(token string) string {
	return fmt.Sprintf(
		`curl -s "https://api.cloudflare.com/client/v4/zones" -H "Authorization: Bearer %s"`,
		"<redacted>",
	)
}

// cfTunnelsCurl returns the curl equivalent for a ListTunnels call.
func cfTunnelsCurl(accountID, token string) string {
	return fmt.Sprintf(
		`curl -s "https://api.cloudflare.com/client/v4/accounts/%s/cfdtunnel" -H "Authorization: Bearer %s"`,
		accountID, "<redacted>",
	)
}
