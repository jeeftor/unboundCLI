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

type ConfigWizard struct {
	editor      *widgets.ConfigEditorWidget
	phase       configWizardPhase
	testResults []connTestResult
	saveErr     error
	statusMsg   string
	width       int
	height      int
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
	if existing.Cloudflare.Enabled {
		cfEnabledStr = "true"
	}

	sections := []widgets.ConfigSection{
		{
			Title: "UnboundDNS",
			Fields: []widgets.ConfigField{
				{Key: "base_url", Label: "Base URL", Value: existing.Config.BaseURL, Placeholder: "https://192.168.1.1", IsRequired: true},
				{Key: "api_key", Label: "API Key", Value: existing.Config.APIKey, Placeholder: "API key", IsPassword: true},
				{Key: "api_secret", Label: "API Secret", Value: existing.Config.APISecret, Placeholder: "API secret", IsPassword: true},
				{Key: "insecure", Label: "Skip SSL Verify", Value: insecureStr, Placeholder: "true/false", HelpText: "Set to true to skip TLS certificate verification"},
			},
		},
		{
			Title: "AdguardHome",
			Fields: []widgets.ConfigField{
				{Key: "adguard_enabled", Label: "Enabled", Value: adguardEnabledStr, Placeholder: "true/false"},
				{Key: "adguard_base_url", Label: "Base URL", Value: existing.Adguard.BaseURL, Placeholder: "http://192.168.1.10:3000"},
				{Key: "adguard_username", Label: "Username", Value: existing.Adguard.Username, Placeholder: "admin"},
				{Key: "adguard_password", Label: "Password", Value: existing.Adguard.Password, Placeholder: "password", IsPassword: true},
				{Key: "adguard_insecure", Label: "Skip SSL Verify", Value: adguardInsecureStr, Placeholder: "true/false"},
			},
		},
		{
			Title: "Cloudflare",
			Fields: []widgets.ConfigField{
				{Key: "cf_enabled", Label: "Enabled", Value: cfEnabledStr, Placeholder: "true/false"},
				{Key: "cf_api_token", Label: "API Token", Value: existing.Cloudflare.APIToken, Placeholder: "CF API token", IsPassword: true},
				{Key: "cf_account_id", Label: "Account ID", Value: existing.Cloudflare.AccountID, Placeholder: "CF account ID"},
				{Key: "cf_zone_id", Label: "Zone ID", Value: existing.Cloudflare.ZoneID, Placeholder: "CF zone ID"},
				{Key: "cf_tunnel_id", Label: "Tunnel ID", Value: existing.Cloudflare.TunnelID, Placeholder: "CF tunnel ID"},
				{Key: "cf_caddy_service_url", Label: "Caddy Service URL", Value: existing.Cloudflare.CaddyServiceURL, Placeholder: "http://caddy:2019"},
			},
		},
	}

	editor.SetSections(sections)
	editor.SetSize(80, 40)
	editor.Focus()

	w.editor = editor
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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return w, tea.Quit

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
	var parts []string

	if w.editor != nil {
		parts = append(parts, w.editor.View())
	}

	parts = append(parts, "")

	// Status message
	if w.statusMsg != "" {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))
		if w.saveErr != nil {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
		}
		parts = append(parts, style.Render(w.statusMsg))
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
		"ctrl+c/q quit",
	}
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
