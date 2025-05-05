package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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
	Short: "Configure API connection settings",
	Long: `Configure the API connection settings for OPNSense.

This command will prompt you for the API key, API secret, and base URL
for your OPNSense installation. The configuration will be saved to a file
for future use.

You can also use environment variables instead of a config file:
  UNBOUND_CLI_API_KEY    - API key for OPNSense
  UNBOUND_CLI_API_SECRET - API secret for OPNSense
  UNBOUND_CLI_BASE_URL   - Base URL for OPNSense (e.g., https://192.168.1.1)
  UNBOUND_CLI_INSECURE   - Set to "true" or "1" to skip SSL verification`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create UI component
		configUI := newConfigUI()

		// Try to load existing config
		existingConfig, err := config.LoadConfig()
		if err == nil {
			fmt.Println(
				configUI.RenderInfo(
					"Existing configuration found. Press Enter to keep current values.",
				),
			)
		}

		scanner := bufio.NewScanner(os.Stdin)

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

		// Create the config
		cfg := api.Config{
			APIKey:    apiKey,
			APISecret: apiSecret,
			BaseURL:   baseURL,
			Insecure:  insecure,
		}

		// Determine config path
		if configPath == "" {
			defaultPath, err := config.GetDefaultConfigPath()
			if err != nil {
				fmt.Println(
					configUI.RenderError(fmt.Errorf("error getting default config path: %v", err)),
				)
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
			} else {
				// File doesn't exist, ask if they want to save to this location
				fmt.Printf("Save configuration to %s? (y/n) [y]: ", configPath)
				scanner.Scan()
				input = scanner.Text()
				if input != "" && strings.ToLower(input) != "y" {
					fmt.Print("Enter alternative path or press Enter to cancel: ")
					scanner.Scan()
					input = scanner.Text()
					if input == "" {
						fmt.Println("Configuration not saved.")
						return
					}
					configPath = input

					// Expand ~ to home directory if present
					if strings.HasPrefix(configPath, "~") {
						home, err := os.UserHomeDir()
						if err == nil {
							configPath = filepath.Join(home, configPath[1:])
						}
					}
				}
			}
		}

		// Save the config
		if err := config.SaveConfig(cfg, configPath); err != nil {
			fmt.Println(configUI.RenderError(fmt.Errorf("error saving configuration: %v", err)))
			return
		}

		fmt.Println(
			configUI.RenderSuccess(
				fmt.Sprintf("Configuration saved successfully to %s!", configPath),
			),
		)

		// Test the connection
		fmt.Print(configUI.RenderTestingConnection())
		client := api.NewClient(cfg)
		_, err = client.GetOverrides()
		if err != nil {
			fmt.Println(configUI.RenderConnectionFailure(err))
			return
		}
		fmt.Println(configUI.RenderConnectionSuccess())

		// Print environment variable information
		fmt.Println(configUI.RenderEnvVarSection())
		fmt.Printf("  export %s=%s\n", config.EnvAPIKey, apiKey)
		fmt.Printf("  export %s=%s\n", config.EnvAPISecret, apiSecret)
		fmt.Printf("  export %s=%s\n", config.EnvBaseURL, baseURL)
		if insecure {
			fmt.Printf("  export %s=true\n", config.EnvInsecure)
		}
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

func (ui *configUI) RenderHeader() string {
	return ui.Styles.Header.Render(" ⚙️ Configuration ⚙️ ") + "\n\n"
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
