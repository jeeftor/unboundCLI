package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
)

type wizardStep int

const (
	stepToken wizardStep = iota
	stepAccountID
	stepFetchTunnels
	stepPickTunnel
	stepFetchZones
	stepPickZone
	stepCaddyURL
	stepConfirm
	stepDone
)

// tunnelItem implements list.Item for bubbles/list
type tunnelItem struct {
	id   string
	name string
}

func (t tunnelItem) Title() string       { return t.name }
func (t tunnelItem) Description() string { return t.id }
func (t tunnelItem) FilterValue() string { return t.name }

// zoneItem implements list.Item for bubbles/list
type zoneItem struct {
	id   string
	name string
}

func (z zoneItem) Title() string       { return z.name }
func (z zoneItem) Description() string { return z.id }
func (z zoneItem) FilterValue() string { return z.name }

// tunnelsFetchedMsg is sent when tunnel list fetch completes
type tunnelsFetchedMsg struct {
	tunnels []api.CloudflareTunnel
	err     error
}

// zonesFetchedMsg is sent when zone list fetch completes
type zonesFetchedMsg struct {
	zones []api.CloudflareZone
	err   error
}

// CloudflareSetupWizard is the interactive Cloudflare tunnel configuration wizard
type CloudflareSetupWizard struct {
	step       wizardStep
	token      string
	accountID  string
	tunnelID   string
	tunnelName string
	zoneID     string
	zoneName   string
	caddyURL   string

	tokenInput    textinput.Model
	accountInput  textinput.Model
	caddyURLInput textinput.Model
	tunnelList    list.Model
	zoneList      list.Model

	tunnels  []api.CloudflareTunnel
	zones    []api.CloudflareZone
	cfClient *api.CloudflareClient

	err     error
	loading bool
	done    bool
	width   int
	height  int
}

// NewCloudflareSetupWizard creates a new Cloudflare setup wizard
func NewCloudflareSetupWizard() *CloudflareSetupWizard {
	tokenInput := textinput.New()
	tokenInput.Placeholder = "cf_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	tokenInput.Focus()
	tokenInput.EchoMode = textinput.EchoPassword
	tokenInput.EchoCharacter = '*'
	tokenInput.Width = 60

	accountInput := textinput.New()
	accountInput.Placeholder = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	accountInput.Width = 60

	caddyURLInput := textinput.New()
	caddyURLInput.Placeholder = "http://192.168.1.15:80"
	caddyURLInput.Width = 60

	delegate := list.NewDefaultDelegate()
	tunnelList := list.New([]list.Item{}, delegate, 60, 10)
	tunnelList.Title = "Select a Cloudflare Tunnel"
	tunnelList.SetShowStatusBar(false)
	tunnelList.SetFilteringEnabled(false)

	zoneList := list.New([]list.Item{}, delegate, 60, 10)
	zoneList.Title = "Select a DNS Zone"
	zoneList.SetShowStatusBar(false)
	zoneList.SetFilteringEnabled(false)

	return &CloudflareSetupWizard{
		step:          stepToken,
		tokenInput:    tokenInput,
		accountInput:  accountInput,
		caddyURLInput: caddyURLInput,
		tunnelList:    tunnelList,
		zoneList:      zoneList,
		width:         80,
		height:        24,
	}
}

func fetchTunnelsCmd(client *api.CloudflareClient) tea.Cmd {
	return func() tea.Msg {
		tunnels, err := client.ListTunnels()
		return tunnelsFetchedMsg{tunnels: tunnels, err: err}
	}
}

func fetchZonesCmd(client *api.CloudflareClient) tea.Cmd {
	return func() tea.Msg {
		zones, err := client.ListZones()
		return zonesFetchedMsg{zones: zones, err: err}
	}
}

// Init implements tea.Model
func (w *CloudflareSetupWizard) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (w *CloudflareSetupWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height
		w.tunnelList.SetWidth(msg.Width - 4)
		w.zoneList.SetWidth(msg.Width - 4)
		return w, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if w.step != stepPickTunnel && w.step != stepPickZone {
				w.done = true
				return w, tea.Quit
			}
		case "esc":
			w.done = true
			return w, tea.Quit
		}

		switch w.step {
		case stepToken:
			switch msg.String() {
			case "enter":
				token := strings.TrimSpace(w.tokenInput.Value())
				if token == "" {
					w.err = fmt.Errorf("API token is required")
					return w, nil
				}
				w.token = token
				w.err = nil
				w.step = stepAccountID
				w.accountInput.Focus()
				return w, textinput.Blink
			}
			w.tokenInput, cmd = w.tokenInput.Update(msg)
			return w, cmd

		case stepAccountID:
			switch msg.String() {
			case "enter":
				accountID := strings.TrimSpace(w.accountInput.Value())
				if accountID == "" {
					w.err = fmt.Errorf("account ID is required")
					return w, nil
				}
				w.accountID = accountID
				w.err = nil

				cfClient, err := api.NewCloudflareClient(api.CloudflareConfig{
					APIToken:  w.token,
					AccountID: w.accountID,
				})
				if err != nil {
					w.err = fmt.Errorf("failed to create client: %w", err)
					return w, nil
				}
				w.cfClient = cfClient
				w.step = stepFetchTunnels
				w.loading = true
				return w, fetchTunnelsCmd(w.cfClient)
			}
			w.accountInput, cmd = w.accountInput.Update(msg)
			return w, cmd

		case stepPickTunnel:
			switch msg.String() {
			case "enter":
				if selected, ok := w.tunnelList.SelectedItem().(tunnelItem); ok {
					w.tunnelID = selected.id
					w.tunnelName = selected.name
					w.err = nil
					w.step = stepFetchZones
					w.loading = true
					return w, fetchZonesCmd(w.cfClient)
				}
			case "q":
				w.done = true
				return w, tea.Quit
			}
			w.tunnelList, cmd = w.tunnelList.Update(msg)
			return w, cmd

		case stepPickZone:
			switch msg.String() {
			case "enter":
				if selected, ok := w.zoneList.SelectedItem().(zoneItem); ok {
					w.zoneID = selected.id
					w.zoneName = selected.name
					w.err = nil
					w.step = stepCaddyURL
					w.caddyURLInput.Focus()
					return w, textinput.Blink
				}
			case "q":
				w.done = true
				return w, tea.Quit
			}
			w.zoneList, cmd = w.zoneList.Update(msg)
			return w, cmd

		case stepCaddyURL:
			switch msg.String() {
			case "enter":
				caddyURL := strings.TrimSpace(w.caddyURLInput.Value())
				if caddyURL == "" {
					w.err = fmt.Errorf("Caddy service URL is required")
					return w, nil
				}
				w.caddyURL = caddyURL
				w.err = nil
				w.step = stepConfirm
				return w, nil
			}
			w.caddyURLInput, cmd = w.caddyURLInput.Update(msg)
			return w, cmd

		case stepConfirm:
			switch msg.String() {
			case "enter", "y":
				if err := w.saveConfig(); err != nil {
					w.err = err
					return w, nil
				}
				w.step = stepDone
				w.done = true
				return w, tea.Quit
			case "n":
				w.done = true
				return w, tea.Quit
			}
		}

	case tunnelsFetchedMsg:
		w.loading = false
		if msg.err != nil {
			w.err = fmt.Errorf("failed to fetch tunnels: %w", msg.err)
			w.step = stepAccountID
			w.accountInput.Focus()
			return w, textinput.Blink
		}
		w.tunnels = msg.tunnels
		items := make([]list.Item, len(msg.tunnels))
		for i, t := range msg.tunnels {
			items[i] = tunnelItem{id: t.ID, name: t.Name}
		}
		w.tunnelList.SetItems(items)
		w.step = stepPickTunnel
		return w, nil

	case zonesFetchedMsg:
		w.loading = false
		if msg.err != nil {
			w.err = fmt.Errorf("failed to fetch zones: %w", msg.err)
			w.step = stepPickTunnel
			return w, nil
		}
		w.zones = msg.zones
		items := make([]list.Item, len(msg.zones))
		for i, z := range msg.zones {
			items[i] = zoneItem{id: z.ID, name: z.Name}
		}
		w.zoneList.SetItems(items)
		w.step = stepPickZone
		return w, nil
	}

	return w, nil
}

// View implements tea.Model
func (w *CloudflareSetupWizard) View() string {
	infoColor := lipgloss.Color("#7aa2f7")
	errorColor := lipgloss.Color("#f7768e")
	successColor := lipgloss.Color("#9ece6a")
	dimColor := lipgloss.Color("#565f89")

	header := lipgloss.NewStyle().Bold(true).Foreground(infoColor).Render("Cloudflare Tunnel Setup Wizard")
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")

	if w.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(errorColor)
		sb.WriteString(errStyle.Render("Error: " + w.err.Error()))
		sb.WriteString("\n\n")
	}

	labelStyle := lipgloss.NewStyle().Foreground(infoColor).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)

	switch w.step {
	case stepToken:
		sb.WriteString(labelStyle.Render("Step 1/6: Cloudflare API Token"))
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render("Enter your Cloudflare API token with tunnel and zone read permissions."))
		sb.WriteString("\n\n")
		sb.WriteString(w.tokenInput.View())
		sb.WriteString("\n\n")
		sb.WriteString(dimStyle.Render("Press Enter to continue, Esc to quit"))

	case stepAccountID:
		sb.WriteString(labelStyle.Render("Step 2/6: Cloudflare Account ID"))
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render("Enter your Cloudflare account ID (found in the Cloudflare dashboard URL)."))
		sb.WriteString("\n\n")
		sb.WriteString(w.accountInput.View())
		sb.WriteString("\n\n")
		sb.WriteString(dimStyle.Render("Press Enter to continue, Esc to quit"))

	case stepFetchTunnels:
		sb.WriteString(dimStyle.Render("Fetching tunnels..."))

	case stepPickTunnel:
		sb.WriteString(labelStyle.Render("Step 3/6: Select Tunnel"))
		sb.WriteString("\n\n")
		sb.WriteString(w.tunnelList.View())
		sb.WriteString("\n\n")
		sb.WriteString(dimStyle.Render("Press Enter to select, Esc to quit"))

	case stepFetchZones:
		sb.WriteString(dimStyle.Render("Fetching DNS zones..."))

	case stepPickZone:
		sb.WriteString(labelStyle.Render("Step 4/6: Select DNS Zone"))
		sb.WriteString("\n\n")
		sb.WriteString(w.zoneList.View())
		sb.WriteString("\n\n")
		sb.WriteString(dimStyle.Render("Press Enter to select, Esc to quit"))

	case stepCaddyURL:
		sb.WriteString(labelStyle.Render("Step 5/6: Caddy Service URL"))
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render("Enter the URL where your Caddy reverse proxy admin API is accessible."))
		sb.WriteString("\n\n")
		sb.WriteString(w.caddyURLInput.View())
		sb.WriteString("\n\n")
		sb.WriteString(dimStyle.Render("Press Enter to continue, Esc to quit"))

	case stepConfirm:
		sb.WriteString(labelStyle.Render("Step 6/6: Confirm Configuration"))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("  Tunnel:    %s (%s)\n", w.tunnelName, w.tunnelID))
		sb.WriteString(fmt.Sprintf("  Zone:      %s (%s)\n", w.zoneName, w.zoneID))
		sb.WriteString(fmt.Sprintf("  Caddy URL: %s\n", w.caddyURL))
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render("Press Enter or 'y' to save, 'n' to quit without saving"))

	case stepDone:
		sb.WriteString(lipgloss.NewStyle().Foreground(successColor).Render("Configuration saved successfully!"))
	}

	return sb.String()
}

func (w *CloudflareSetupWizard) saveConfig() error {
	configPath, err := config.GetDefaultConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	var extCfg config.ExtendedConfig
	existing, loadErr := config.LoadExtendedConfig()
	if loadErr == nil {
		extCfg = existing
	}

	extCfg.Cloudflare = config.CloudflareConfig{
		Enabled:         true,
		APIToken:        w.token,
		AccountID:       w.accountID,
		ZoneID:          w.zoneID,
		TunnelID:        w.tunnelID,
		CaddyServiceURL: w.caddyURL,
	}

	if err := config.SaveExtendedConfig(extCfg, configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// Run starts the wizard as a full Bubble Tea program
func (w *CloudflareSetupWizard) Run() error {
	p := tea.NewProgram(w, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard error: %w", err)
	}

	if wizard, ok := finalModel.(*CloudflareSetupWizard); ok {
		if wizard.step == stepDone {
			fmt.Println("Cloudflare configuration saved successfully.")
		}
	}

	return nil
}
