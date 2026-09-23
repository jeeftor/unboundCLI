package cmd

import (
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/ui"
	"github.com/spf13/cobra"
)

var force bool

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:     "delete [uuid]",
	Short:   "Delete a DNS override",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"del", "remove", "rm"},
	Long: `Delete a caddy-dns-sync-managed DNS override from Unbound DNS.

This command deletes only an override that is explicitly marked as managed by
caddy-dns-sync. It refuses manual and unproven records. You must specify the
UUID of the override to delete. Use the 'list' command to find UUIDs.`,
	RunE: runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
	deleteUI := newDeleteUI()

	uuid := args[0]
	if logging.GetLogLevel() == logging.LogLevelDebug {
		logging.Debug("Delete command called", "uuid", uuid, "force", force)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		if logging.GetLogLevel() == logging.LogLevelDebug {
			logging.Error("Error loading configuration", "error", err)
		}
		return fmt.Errorf("error loading configuration: %w\nPlease run 'config' command to set up API access", err)
	}

	client := api.NewClient(cfg)

	overrides, err := client.GetOverrides()
	if err != nil {
		if logging.GetLogLevel() == logging.LogLevelDebug {
			logging.Error("Error fetching overrides", "error", err)
		}
		return fmt.Errorf("error fetching overrides: %w", err)
	}
	targetOverride, err := managedOverrideForDelete(overrides, uuid)
	if err != nil {
		return err
	}

	if !force {
		fmt.Fprintln(cmd.OutOrStdout(), deleteUI.RenderConfirmation(*targetOverride))
		fmt.Fprint(cmd.OutOrStdout(), "Confirm deletion (y/N): ")
		var confirm string
		if _, err := fmt.Fscanln(cmd.InOrStdin(), &confirm); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), deleteUI.RenderWarning("Deletion cancelled (no input)"))
			return nil
		}
		if confirm != "y" && confirm != "Y" {
			if logging.GetLogLevel() == logging.LogLevelDebug {
				logging.Info("Delete operation cancelled by user", "uuid", uuid)
			}
			fmt.Fprintln(cmd.OutOrStdout(), deleteUI.RenderWarning("Deletion cancelled"))
			return nil
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), deleteUI.RenderDeletingMessage(uuid))
	if err := client.DeleteOverride(uuid); err != nil {
		if logging.GetLogLevel() == logging.LogLevelDebug {
			logging.Error("Error deleting override", "error", err, "uuid", uuid)
		}
		return fmt.Errorf("error deleting override: %w", err)
	}
	current, err := client.GetOverrides()
	if err != nil {
		return fmt.Errorf("verify deletion of override %s: %w", uuid, err)
	}
	for _, override := range current {
		if override.UUID == uuid {
			return fmt.Errorf("verify deletion of override %s: record remains; review provider state before retrying", uuid)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), deleteUI.RenderApplyingMessage())
	if err := client.ApplyChanges(); err != nil {
		if logging.GetLogLevel() == logging.LogLevelDebug {
			logging.Error("Error applying changes", "error", err)
		}
		return fmt.Errorf("error applying changes: %w\nThe override was deleted but changes were not applied", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), deleteUI.RenderSuccess(uuid))
	if logging.GetLogLevel() == logging.LogLevelDebug {
		logging.Info("DNS override deleted successfully", "uuid", uuid)
	}
	return nil
}

func managedOverrideForDelete(overrides []api.DNSOverride, uuid string) (*api.DNSOverride, error) {
	for index := range overrides {
		override := &overrides[index]
		if override.UUID != uuid {
			continue
		}
		if !app.IsManagedUnboundDescription(override.Description) {
			return nil, fmt.Errorf("refusing to delete unowned Unbound override %s; explicitly adopt it before syncing or remove it in OPNSense", uuid)
		}
		return override, nil
	}
	return nil, fmt.Errorf("no override found with UUID %s", uuid)
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
