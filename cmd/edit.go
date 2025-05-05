package cmd

import (
	"bufio"
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
	// Variables for edit command flags
	editHost        string
	editDomain      string
	editServer      string
	editDescription string
	editEnabled     bool
	editNoPrompt    bool
	editApplyAfter  bool
)

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit [uuid]",
	Short: "Edit a DNS override",
	Long: `Edit an existing DNS override in Unbound DNS.

This command edits an existing DNS override in Unbound DNS. You must provide the UUID
of the override to edit. Use the 'list' command to find the UUID of the override
you want to edit.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get UUID from args
		uuid := args[0]
		logging.Debug("Edit command called", "uuid", uuid)

		// Create UI component
		editUI := newEditUI()

		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			logging.Error("Error loading configuration", "error", err)
			fmt.Println(editUI.RenderError(err))
			fmt.Println("Please run 'config' command to set up API access")
			os.Exit(1)
		}

		// Create client
		client := api.NewClient(cfg)

		// Get override details
		logging.Debug("Fetching overrides to find target")
		overrides, err := client.GetOverrides()
		if err != nil {
			logging.Error("Error fetching overrides", "error", err)
			fmt.Println(editUI.RenderError(err))
			os.Exit(1)
		}

		// Find the override with the given UUID
		var targetOverride *api.DNSOverride
		for _, o := range overrides {
			if o.UUID == uuid {
				targetOverride = &o
				break
			}
		}

		if targetOverride == nil {
			logging.Error("No override found with UUID", "uuid", uuid)
			fmt.Println(editUI.RenderError(fmt.Errorf("No override found with UUID %s", uuid)))
			os.Exit(1)
		}

		logging.Debug("Found override to edit",
			"uuid", uuid,
			"host", targetOverride.Host,
			"domain", targetOverride.Domain,
			"server", targetOverride.Server)

		// Create a copy of the override to edit
		editedOverride := *targetOverride

		// Update with flags if provided
		if cmd.Flags().Changed("host") {
			editedOverride.Host = editHost
		}
		if cmd.Flags().Changed("domain") {
			editedOverride.Domain = editDomain
		}
		if cmd.Flags().Changed("server") {
			editedOverride.Server = editServer
		}
		if cmd.Flags().Changed("description") {
			editedOverride.Description = editDescription
		}
		if cmd.Flags().Changed("enabled") {
			if editEnabled {
				editedOverride.Enabled = "1"
			} else {
				editedOverride.Enabled = "0"
			}
		}

		// Prompt for updates if not in no-prompt mode
		if !editNoPrompt {
			scanner := bufio.NewScanner(os.Stdin)

			// Host
			fmt.Printf("Host [%s]: ", editedOverride.Host)
			scanner.Scan()
			if scanner.Text() != "" {
				editedOverride.Host = scanner.Text()
			}

			// Domain
			fmt.Printf("Domain [%s]: ", editedOverride.Domain)
			scanner.Scan()
			if scanner.Text() != "" {
				editedOverride.Domain = scanner.Text()
			}

			// Server
			fmt.Printf("IP Address [%s]: ", editedOverride.Server)
			scanner.Scan()
			if scanner.Text() != "" {
				editedOverride.Server = scanner.Text()
			}

			// Description
			fmt.Printf("Description [%s]: ", editedOverride.Description)
			scanner.Scan()
			if scanner.Text() != "" {
				editedOverride.Description = scanner.Text()
			}

			// Enabled
			enabled := "Yes"
			if editedOverride.Enabled == "0" {
				enabled = "No"
			}
			fmt.Printf("Enable (y/n) [%s]: ", enabled)
			scanner.Scan()
			if scanner.Text() != "" {
				if strings.ToLower(scanner.Text()) == "y" {
					editedOverride.Enabled = "1"
				} else if strings.ToLower(scanner.Text()) == "n" {
					editedOverride.Enabled = "0"
				}
			}
		}

		// Validate input
		if editedOverride.Host == "" || editedOverride.Domain == "" || editedOverride.Server == "" {
			logging.Error("Missing required fields",
				"host", editedOverride.Host,
				"domain", editedOverride.Domain,
				"server", editedOverride.Server)
			fmt.Println(editUI.RenderError(fmt.Errorf("Host, Domain, and Server are required")))
			os.Exit(1)
		}

		// Update override
		fmt.Println(
			editUI.RenderUpdatingMessage(
				editedOverride.Host,
				editedOverride.Domain,
				editedOverride.Server,
			),
		)
		if err := client.UpdateOverride(editedOverride); err != nil {
			logging.Error("Error updating override", "error", err)
			fmt.Println(editUI.RenderError(err))
			os.Exit(1)
		}

		fmt.Println(editUI.RenderSuccess("DNS override updated successfully"))

		// Apply changes if requested
		if editApplyAfter {
			fmt.Println(editUI.RenderApplyingMessage())
			if err := client.ApplyChanges(); err != nil {
				logging.Error("Error applying changes", "error", err)
				fmt.Println(editUI.RenderError(err))
				os.Exit(1)
			}
			fmt.Println(editUI.RenderApplySuccess())
		} else {
			fmt.Println(editUI.RenderChangesNotApplied())
		}

		logging.Info("DNS override updated successfully", "uuid", uuid)
	},
}

func init() {
	rootCmd.AddCommand(editCmd)

	// Add flags
	editCmd.Flags().StringVar(&editHost, "host", "", "Hostname (without domain)")
	editCmd.Flags().StringVar(&editDomain, "domain", "", "Domain name")
	editCmd.Flags().StringVar(&editServer, "server", "", "IP address of the server")
	editCmd.Flags().StringVar(&editDescription, "description", "", "Description of the override")
	editCmd.Flags().BoolVar(&editEnabled, "enabled", true, "Enable the override")
	editCmd.Flags().BoolVar(&editNoPrompt, "no-prompt", false, "Do not prompt for updates")
	editCmd.Flags().
		BoolVar(&editApplyAfter, "apply", false, "Apply changes after editing the override")
}

type editUI struct {
	*ui.BaseUI
}

func newEditUI() *editUI {
	return &editUI{ui.NewBaseUI()}
}

func (ui *editUI) RenderOverrideDetails(override api.DNSOverride) string {
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

func (ui *editUI) RenderApplyingMessage() string {
	return ui.RenderInfo("Applying changes...")
}

func (ui *editUI) RenderApplySuccess() string {
	return ui.RenderSuccess("Changes applied successfully!")
}

func (ui *editUI) RenderChangesNotApplied() string {
	return ui.RenderWarning("No changes applied.")
}

func (ui *editUI) RenderUpdatingMessage(host, domain, server string) string {
	return ui.RenderInfo(fmt.Sprintf("Updating DNS override: %s.%s -> %s", host, domain, server))
}
