package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/config"
	"github.com/jeeftor/unboundCLI/internal/tui"
	"github.com/spf13/cobra"
)

var (
	configPath  string
	forceConfig bool
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure API connection settings for UnboundDNS and AdguardHome",
	Long: `Configure the API connection settings for both OPNSense (UnboundDNS) and AdguardHome.

This command will prompt you for the configuration settings for both systems:
- OPNSense API key, API secret, and base URL for UnboundDNS management
- AdguardHome username, password, and base URL for DNS rewrite management

The configuration will be saved to a file for future use.

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
		// Create UI component
		configUI := newConfigUI()

		fmt.Println(configUI.RenderHeader("🔧 UnboundCLI Configuration Setup 🔧"))
		fmt.Println("This will configure both UnboundDNS (OPNSense) and AdguardHome settings.")
		fmt.Println(configUI.RenderInfo("Press Enter to keep current values or type new ones."))
		fmt.Println()

		// Try to load existing configs
		existingConfig, _ := config.LoadConfig()
		existingAdguardConfig, _ := config.LoadAdguardConfig()

		scanner := bufio.NewScanner(os.Stdin)

		// ========== UnboundDNS (OPNSense) Configuration ==========
		fmt.Println(configUI.RenderSectionHeader("🌐 UnboundDNS (OPNSense) Configuration"))

		// Prompt for API URL
		baseURL := existingConfig.BaseURL
		if baseURL == "" {
			baseURL = "https://192.168.1.1"
		}
		fmt.Printf("OPNSense URL [%s]: ", baseURL)
		scanner.Scan()
		input := scanner.Text()
		if input != "" {
			baseURL = input
		}

		// Prompt for API Key
		apiKey := existingConfig.APIKey
		fmt.Printf("API Key [%s]: ", maskString(apiKey))
		scanner.Scan()
		input = scanner.Text()
		if input != "" {
			apiKey = input
		}

		// Prompt for API Secret
		apiSecret := existingConfig.APISecret
		fmt.Printf("API Secret [%s]: ", maskString(apiSecret))
		scanner.Scan()
		input = scanner.Text()
		if input != "" {
			apiSecret = input
		}

		// Prompt for SSL verification
		insecure := existingConfig.Insecure
		insecureStr := "n"
		if insecure {
			insecureStr = "y"
		}
		fmt.Printf("Skip SSL verification (y/n) [%s]: ", insecureStr)
		scanner.Scan()
		input = scanner.Text()
		if input != "" {
			insecure = strings.ToLower(input) == "y"
		}

		// ========== AdguardHome Configuration ==========
		fmt.Println()
		fmt.Println(configUI.RenderSectionHeader("🛡️  AdguardHome Configuration"))

		// Prompt for enabling AdguardHome
		enabled := existingAdguardConfig.Enabled
		enabledStr := "n"
		if enabled {
			enabledStr = "y"
		}
		fmt.Printf("Enable AdguardHome integration (y/n) [%s]: ", enabledStr)
		scanner.Scan()
		input = scanner.Text()
		if input != "" {
			enabled = strings.ToLower(input) == "y"
		}

		var adguardBaseURL, adguardUsername, adguardPassword string
		var adguardInsecure bool

		if enabled {
			// Prompt for AdguardHome URL
			adguardBaseURL = existingAdguardConfig.BaseURL
			if adguardBaseURL == "" {
				adguardBaseURL = "http://192.168.1.10:3000"
			}
			fmt.Printf("AdguardHome URL [%s]: ", adguardBaseURL)
			scanner.Scan()
			input = scanner.Text()
			if input != "" {
				adguardBaseURL = input
			}

			// Prompt for AdguardHome Username
			adguardUsername = existingAdguardConfig.Username
			fmt.Printf("AdguardHome Username [%s]: ", maskString(adguardUsername))
			scanner.Scan()
			input = scanner.Text()
			if input != "" {
				adguardUsername = input
			}

			// Prompt for AdguardHome Password
			adguardPassword = existingAdguardConfig.Password
			fmt.Printf("AdguardHome Password [%s]: ", maskString(adguardPassword))
			scanner.Scan()
			input = scanner.Text()
			if input != "" {
				adguardPassword = input
			}

			// Prompt for SSL verification
			adguardInsecure = existingAdguardConfig.Insecure
			adguardInsecureStr := "n"
			if adguardInsecure {
				adguardInsecureStr = "y"
			}
			fmt.Printf("Skip SSL verification for AdguardHome (y/n) [%s]: ", adguardInsecureStr)
			scanner.Scan()
			input = scanner.Text()
			if input != "" {
				adguardInsecure = strings.ToLower(input) == "y"
			}
		} else {
			fmt.Println(configUI.RenderInfo("AdguardHome integration disabled. You can enable it later by re-running this command."))
		}

		// ========== Save Configuration ==========
		fmt.Println()
		fmt.Println(configUI.RenderSectionHeader("💾 Save Configuration"))

		// Create the extended config
		extendedConfig := config.ExtendedConfig{
			Config: api.Config{
				APIKey:    apiKey,
				APISecret: apiSecret,
				BaseURL:   baseURL,
				Insecure:  insecure,
			},
			Adguard: config.AdguardConfig{
				Enabled:  enabled,
				Username: adguardUsername,
				Password: adguardPassword,
				BaseURL:  adguardBaseURL,
				Insecure: adguardInsecure,
			},
		}

		// Determine config path
		if configPath == "" {
			defaultPath, err := config.GetDefaultConfigPath()
			if err != nil {
				fmt.Println(configUI.RenderError(fmt.Errorf("error getting default config path: %v", err)))
				return
			}
			configPath = defaultPath
		}

		// Check if file exists and prompt for confirmation if not forced
		if !forceConfig {
			if _, err := os.Stat(configPath); err == nil {
				fmt.Printf("Config file already exists at %s. Overwrite? (y/n) [n]: ", configPath)
				scanner.Scan()
				input = scanner.Text()
				if strings.ToLower(input) != "y" {
					fmt.Println("Configuration not saved.")
					return
				}
			}
		}

		// Save the extended config
		if err := config.SaveExtendedConfig(extendedConfig, configPath); err != nil {
			fmt.Println(configUI.RenderError(fmt.Errorf("error saving configuration: %v", err)))
			return
		}

		fmt.Println(configUI.RenderSuccess(fmt.Sprintf("Configuration saved successfully to %s!", configPath)))

		// ========== Test Connections ==========
		fmt.Println()
		fmt.Println(configUI.RenderSectionHeader("🧪 Test Connections"))

		// Test UnboundDNS connection
		fmt.Print("Testing UnboundDNS connection... ")
		unboundClient := api.NewClient(extendedConfig.Config)
		_, err := unboundClient.GetOverrides()
		if err != nil {
			fmt.Println(configUI.RenderConnectionFailure(err))
		} else {
			fmt.Println(configUI.RenderConnectionSuccess())
		}

		// Test AdguardHome connection if enabled
		if enabled {
			fmt.Print("Testing AdguardHome connection... ")
			adguardClient := api.NewAdguardClient(extendedConfig.Adguard.GetAdguardAPIConfig())
			_, err := adguardClient.ListRewrites()
			if err != nil {
				fmt.Println(configUI.RenderConnectionFailure(err))
			} else {
				fmt.Println(configUI.RenderConnectionSuccess())
			}
		}

		// ========== Environment Variables ==========
		fmt.Println()
		fmt.Println(configUI.RenderSectionHeader("🌍 Environment Variables"))
		fmt.Println("You can also use these environment variables instead of the config file:")
		fmt.Println()
		fmt.Println("UnboundDNS (OPNSense):")
		fmt.Printf("  export %s=%s\n", config.EnvAPIKey, apiKey)
		fmt.Printf("  export %s=%s\n", config.EnvAPISecret, apiSecret)
		fmt.Printf("  export %s=%s\n", config.EnvBaseURL, baseURL)
		if insecure {
			fmt.Printf("  export %s=true\n", config.EnvInsecure)
		}

		if enabled {
			fmt.Println()
			fmt.Println("AdguardHome:")
			fmt.Printf("  export %s=true\n", config.EnvAdguardEnabled)
			fmt.Printf("  export %s=%s\n", config.EnvAdguardUsername, adguardUsername)
			fmt.Printf("  export %s=%s\n", config.EnvAdguardPassword, adguardPassword)
			fmt.Printf("  export %s=%s\n", config.EnvAdguardBaseURL, adguardBaseURL)
			if adguardInsecure {
				fmt.Printf("  export %s=true\n", config.EnvAdguardInsecure)
			}
		}

		fmt.Println()
		fmt.Println(configUI.RenderSuccess("🎉 Configuration complete! You can now run 'unboundCLI status' to check sync status."))
	},
}

type configUI struct {
	Styles tui.StyleConfig
}

func newConfigUI() *configUI {
	return &configUI{
		Styles: tui.DefaultStyles(),
	}
}

func (ui *configUI) RenderHeader(title string) string {
	return ui.Styles.Header.Render(fmt.Sprintf(" %s ", title)) + "\n\n"
}

func (ui *configUI) RenderSectionHeader(title string) string {
	return ui.Styles.Section.Render(fmt.Sprintf(" %s ", title)) + "\n"
}

func (ui *configUI) RenderSuccess(message string) string {
	return ui.Styles.Success.Render(fmt.Sprintf(" ✅ %s ", message))
}

func (ui *configUI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" ❌ Error: %s ", err))
}

func (ui *configUI) RenderInfo(message string) string {
	return ui.Styles.Info.Render(fmt.Sprintf(" 💬 %s ", message))
}

func (ui *configUI) RenderEnvVarSection() string {
	return ui.Styles.Section.Render(" 🏠 Environment Variables ") + "\n"
}

func (ui *configUI) RenderTestingConnection() string {
	return ui.Styles.Info.Render(" 💾 Testing connection... ")
}

func (ui *configUI) RenderConnectionSuccess() string {
	return ui.Styles.Success.Render(" ✅ Connection successful! ")
}

func (ui *configUI) RenderConnectionFailure(err error) string {
	var sb strings.Builder
	sb.WriteString(ui.Styles.Error.Render(" ❌ Connection failed "))
	sb.WriteString("\n")
	sb.WriteString(ui.Styles.Error.Render(fmt.Sprintf("   Error: %s", err)))
	return sb.String()
}

// maskString replaces all but the first and last character with asterisks
func maskString(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 2 {
		return s
	}
	return s[:1] + strings.Repeat("*", len(s)-2) + s[len(s)-1:]
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add flags
	configCmd.Flags().
		StringVar(&configPath, "path", "", "Path to save the configuration file (default: ~/.unboundCLI.json)")
	configCmd.Flags().
		BoolVar(&forceConfig, "force", false, "Force overwrite of existing config file without prompting")
}
