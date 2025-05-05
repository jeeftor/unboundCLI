package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/config"
	"github.com/jeeftor/unboundCLI/internal/logging"
	"github.com/jeeftor/unboundCLI/internal/ui"
	"github.com/spf13/cobra"
)

var (
	host        string
	domain      string
	server      string
	description string
	disabled    bool
	forceAdd    bool
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a DNS override",
	Long: `Add a DNS override to Unbound DNS.

This command adds a new DNS override to Unbound DNS. You must specify
the host, domain, and server (IP address) for the override.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create UI component
		addUI := newAddUI()

		// Validate required flags
		if host == "" || domain == "" || server == "" {
			logging.Error("Missing required flags",
				"host", host,
				"domain", domain,
				"server", server)
			fmt.Println(addUI.RenderError("host, domain, and server are required"))
			cmd.Help()
			os.Exit(1)
		}

		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			logging.Error("Error loading configuration", "error", err)
			fmt.Println(
				addUI.RenderError(fmt.Sprintf("error loading configuration: %v\nPlease run 'config' command to set up API access", err)),
			)
			os.Exit(1)
		}

		// Create client
		client := api.NewClient(cfg)

		// Check if override already exists
		if !forceAdd {
			// Display a user-friendly message when checking for existing overrides
			if logging.GetLogLevel() != logging.LogLevelDebug {
				fmt.Println(addUI.RenderCheckingMessage())
			}
			exists, uuid, err := client.IsOverrideExists(host, domain)
			if err != nil {
				logging.Error("Error checking existing overrides", "error", err)
				fmt.Println(addUI.RenderError(fmt.Sprintf("error checking existing overrides: %v", err)))
				os.Exit(1)
			}

			if exists {
				logging.Warn("DNS override already exists",
					"host", host,
					"domain", domain,
					"uuid", uuid)
				fmt.Println(addUI.RenderDuplicateWarning(host, domain, uuid))
				os.Exit(0)
			}
		}

		// Create override
		enabled := "1"
		if disabled {
			enabled = "0"
		}

		override := api.DNSOverride{
			Enabled:     enabled,
			Host:        host,
			Domain:      domain,
			Server:      server,
			Description: description,
		}

		// Add override
		fmt.Println(addUI.RenderAddingMessage())
		uuid, err := client.AddOverride(override)
		if err != nil {
			logging.Error("Error adding override", "error", err)

			// Check if it's a duplicate error
			if strings.Contains(err.Error(), "already exists") {
				fmt.Println(addUI.RenderWarning(err.Error()))
				fmt.Println(addUI.RenderWarning("Use --force flag to add anyway"))
				os.Exit(0)
			}

			fmt.Println(addUI.RenderError(fmt.Sprintf("error adding override: %v", err)))
			os.Exit(1)
		}

		// Apply changes
		fmt.Println(addUI.RenderApplyingMessage())
		err = client.ApplyChanges()
		if err != nil {
			logging.Error("Error applying changes", "error", err)
			fmt.Println(
				addUI.RenderError(fmt.Sprintf("error applying changes: %v\nThe override was added but changes were not applied", err)),
			)
			os.Exit(1)
		}

		fmt.Println(addUI.RenderSuccess(fmt.Sprintf("DNS override added successfully with UUID: %s", uuid)))
		if logging.GetLogLevel() == logging.LogLevelDebug {
			logging.Info("DNS override added successfully", "uuid", uuid)
		}
	},
}

type addUI struct {
	*ui.BaseUI
}

func newAddUI() *addUI {
	return &addUI{ui.NewBaseUI()}
}

func (ui *addUI) RenderDuplicateWarning(host, domain, uuid string) string {
	return ui.RenderWarning(fmt.Sprintf("DNS override for %s.%s already exists (UUID: %s)", host, domain, uuid))
}

func (ui *addUI) RenderOverrideDetails(override api.DNSOverride) string {
	return fmt.Sprintf(
		"Host: %s\nDomain: %s\nServer: %s\nDescription: %s\nEnabled: %s\nUUID: %s",
		override.Host,
		override.Domain,
		override.Server,
		override.Description,
		override.Enabled,
		override.UUID,
	)
}

func (ui *addUI) RenderAddingMessage() string {
	return ui.RenderInfo("Adding DNS override...")
}

func (ui *addUI) RenderApplyingMessage() string {
	return ui.RenderInfo("Applying configuration...")
}

func (ui *addUI) RenderCheckingMessage() string {
	return ui.RenderInfo("Checking for existing override...")
}

func (ui *addUI) RenderSuccess(message string) string {
	return ui.RenderSuccess(message)
}

func (ui *addUI) RenderError(message string) string {
	return ui.RenderError(message)
}

func (ui *addUI) RenderWarning(message string) string {
	return ui.RenderWarning(message)
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Add flags
	addCmd.Flags().StringVarP(&host, "host", "H", "", "Host name (required)")
	addCmd.Flags().StringVarP(&domain, "domain", "d", "", "Domain name (required)")
	addCmd.Flags().StringVarP(&server, "server", "s", "", "Server IP address (required)")
	addCmd.Flags().StringVarP(&description, "description", "D", "", "Description")
	addCmd.Flags().BoolVar(&disabled, "disabled", false, "Disable this override")
	addCmd.Flags().
		BoolVar(&forceAdd, "force", false, "Force adding the override even if it already exists")
}
