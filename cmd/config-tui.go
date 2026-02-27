package cmd

import (
	"fmt"
	"os"

	"github.com/jeeftor/caddy-dns-sync/internal/tui"
	"github.com/spf13/cobra"
)

// configTuiCmd represents the config-tui command
var configTuiCmd = &cobra.Command{
	Use:   "config-tui",
	Short: "Interactive TUI-based configuration setup for UnboundDNS and AdguardHome",
	Long: `Launch an interactive Bubble Tea TUI for configuring API connection settings.

This command provides a modern, visual alternative to the CLI-based 'config' command.
It will guide you through setting up:
- OPNSense API key, API secret, and base URL for UnboundDNS management
- AdguardHome username, password, and base URL for DNS rewrite management

The configuration will be saved to ~/.caddy-dns-sync.json and connections will be tested.

You can also use environment variables instead of a config file:

UnboundDNS (OPNSense):
  UNBOUND_CLI_API_KEY    - API key for OPNSense
  UNBOUND_CLI_API_SECRET - API secret for OPNSense
  UNBOUND_CLI_BASE_URL   - Base URL for OPNSense (e.g., https://192.168.1.1)
  UNBOUND_CLI_INSECURE   - Set to "true" or "1" to skip SSL verification

AdguardHome:
  ADGUARD_ENABLED        - Set to "true" to enable AdguardHome integration
  ADGUARD_USERNAME       - Username for AdguardHome
  ADGUARD_PASSWORD       - Password for AdguardHome
  ADGUARD_BASE_URL       - Base URL for AdguardHome (e.g., http://192.168.1.10:3000)
  ADGUARD_INSECURE       - Set to "true" or "1" to skip SSL verification`,
	Run: func(cmd *cobra.Command, args []string) {
		// Launch the configuration wizard
		wizard := tui.NewConfigWizard()
		if err := wizard.Start(); err != nil {
			fmt.Println(UI.RenderError(fmt.Errorf("error in configuration wizard: %v", err)))
			os.Exit(1)
		}

		// Wizard exits when complete or user quits
		fmt.Println(UI.RenderSuccess("Configuration wizard complete!"))
	},
}

func init() {
	rootCmd.AddCommand(configTuiCmd)
}
