package cmd

import (
	"fmt"
	"os"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/config"
	"github.com/jeeftor/unboundCLI/internal/logging"
	"github.com/jeeftor/unboundCLI/internal/ui"
	"github.com/spf13/cobra"
)

var force bool

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:     "delete [uuid]",
	Short:   "Delete a DNS override",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"del", "remove", "rm"},
	Long: `Delete a DNS override from Unbound DNS.

This command deletes a DNS override from Unbound DNS. You must specify
the UUID of the override to delete. Use the 'list' command to find UUIDs.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create UI component
		deleteUI := newDeleteUI()

		// Get UUID from args
		uuid := args[0]
		if logging.GetLogLevel() == logging.LogLevelDebug {
			logging.Debug("Delete command called", "uuid", uuid, "force", force)
		}

		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			if logging.GetLogLevel() == logging.LogLevelDebug {
				logging.Error("Error loading configuration", "error", err)
			}
			fmt.Println(
				deleteUI.RenderError(
					fmt.Errorf(
						"error loading configuration: %v\nPlease run 'config' command to set up API access",
						err,
					),
				),
			)
			os.Exit(1)
		}

		// Create client
		client := api.NewClient(cfg)

		// Get override details to confirm
		if !force {
			overrides, err := client.GetOverrides()
			if err != nil {
				if logging.GetLogLevel() == logging.LogLevelDebug {
					logging.Error("Error fetching overrides", "error", err)
				}
				fmt.Println(
					deleteUI.RenderError(
						fmt.Errorf("error fetching overrides: %v", err),
					),
				)
				os.Exit(1)
			}

			// Find the override with the matching UUID
			var targetOverride *api.DNSOverride
			for _, o := range overrides {
				if o.UUID == uuid {
					targetOverride = &o
					break
				}
			}

			if targetOverride == nil {
				if logging.GetLogLevel() == logging.LogLevelDebug {
					logging.Error("No override found with UUID", "uuid", uuid)
				}
				fmt.Println(
					deleteUI.RenderError(
						fmt.Errorf("no override found with UUID %s", uuid),
					),
				)
				os.Exit(1)
			}

			// Confirm deletion
			fmt.Println(deleteUI.RenderConfirmation(*targetOverride))
			fmt.Print("Confirm deletion (y/N): ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				if logging.GetLogLevel() == logging.LogLevelDebug {
					logging.Info("Delete operation cancelled by user", "uuid", uuid)
				}
				fmt.Println(deleteUI.RenderWarning("Deletion cancelled"))
				return
			}
		}

		// Delete override
		fmt.Println(deleteUI.RenderDeletingMessage(uuid))
		if err := client.DeleteOverride(uuid); err != nil {
			if logging.GetLogLevel() == logging.LogLevelDebug {
				logging.Error("Error deleting override", "error", err, "uuid", uuid)
			}
			fmt.Println(
				deleteUI.RenderError(
					fmt.Errorf("error deleting override: %v", err),
				),
			)
			os.Exit(1)
		}

		// Apply changes
		fmt.Println(deleteUI.RenderApplyingMessage())
		if err := client.ApplyChanges(); err != nil {
			if logging.GetLogLevel() == logging.LogLevelDebug {
				logging.Error("Error applying changes", "error", err)
			}
			fmt.Println(
				deleteUI.RenderError(
					fmt.Errorf(
						"error applying changes: %v\nThe override was deleted but changes were not applied",
						err,
					),
				),
			)
			os.Exit(1)
		}

		fmt.Println(deleteUI.RenderSuccess(uuid))
		if logging.GetLogLevel() == logging.LogLevelDebug {
			logging.Info("DNS override deleted successfully", "uuid", uuid)
		}
	},
}

type deleteUI struct {
	*ui.BaseUI
}

func newDeleteUI() *deleteUI {
	return &deleteUI{ui.NewBaseUI()}
}

func (ui *deleteUI) RenderOverrideDetails(override api.DNSOverride) string {
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

func (ui *deleteUI) RenderDeletingMessage(uuid string) string {
	return ui.RenderInfo(fmt.Sprintf("Deleting DNS override with UUID: %s", uuid))
}

func (ui *deleteUI) RenderApplyingMessage() string {
	return ui.RenderInfo("Applying configuration...")
}

func (ui *deleteUI) RenderConfirmation(override api.DNSOverride) string {
	return ui.RenderInfo(fmt.Sprintf("Are you sure you want to delete the DNS override for %s.%s? (UUID: %s)", override.Host, override.Domain, override.UUID))
}

func (ui *deleteUI) RenderSuccess(uuid string) string {
	return ui.RenderSuccess(fmt.Sprintf("DNS override deleted successfully: %s", uuid))
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	// Add flags
	deleteCmd.Flags().BoolVarP(&force, "force", "f", false, "Force deletion without confirmation")
}
