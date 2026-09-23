package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/ui"
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
	RunE: runEdit,
}

func runEdit(cmd *cobra.Command, args []string) error {
	uuid := args[0]
	logging.Debug("Edit command called", "uuid", uuid)
	releaseLock, err := acquireSyncLockWithWait()
	if err != nil {
		return err
	}
	defer releaseLock()

	editUI := newEditUI()

	cfg, err := config.LoadConfig()
	if err != nil {
		logging.Error("Error loading configuration", "error", err)
		return fmt.Errorf("error loading configuration: %w\nPlease run 'config' command to set up API access", err)
	}

	client := api.NewClient(cfg)

	logging.Debug("Fetching overrides to find target")
	overrides, err := client.GetOverrides()
	if err != nil {
		logging.Error("Error fetching overrides", "error", err)
		return fmt.Errorf("error fetching overrides: %w", err)
	}

	targetOverride, err := managedOverrideForEdit(overrides, uuid)
	if err != nil {
		logging.Error("Override cannot be edited", "uuid", uuid, "error", err)
		return err
	}

	logging.Debug("Found override to edit",
		"uuid", uuid,
		"host", targetOverride.Host,
		"domain", targetOverride.Domain,
		"server", targetOverride.Server)

	editedOverride := *targetOverride
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

	if !editNoPrompt {
		if err := promptForOverrideEdits(cmd, &editedOverride); err != nil {
			return err
		}
	}

	if editedOverride.Host == "" || editedOverride.Domain == "" || editedOverride.Server == "" {
		logging.Error("Missing required fields",
			"host", editedOverride.Host,
			"domain", editedOverride.Domain,
			"server", editedOverride.Server)
		return fmt.Errorf("host, domain, and server are required")
	}
	if !app.IsManagedUnboundDescription(editedOverride.Description) {
		return fmt.Errorf("refusing to remove the ownership marker from Unbound override %s", uuid)
	}

	fmt.Fprintln(cmd.OutOrStdout(),
		editUI.RenderUpdatingMessage(
			editedOverride.Host,
			editedOverride.Domain,
			editedOverride.Server,
		),
	)
	if err := client.UpdateOverride(editedOverride); err != nil {
		logging.Error("Error updating override", "error", err)
		return fmt.Errorf("error updating override: %w", err)
	}
	if err := verifyManagedOverride(client, uuid, editedOverride); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), editUI.RenderSuccess("DNS override updated successfully"))

	if editApplyAfter {
		fmt.Fprintln(cmd.OutOrStdout(), editUI.RenderApplyingMessage())
		if err := client.ApplyChanges(); err != nil {
			logging.Error("Error applying changes", "error", err)
			return fmt.Errorf("error applying changes: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), editUI.RenderApplySuccess())
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), editUI.RenderChangesNotApplied())
	}

	logging.Info("DNS override updated successfully", "uuid", uuid)
	return nil
}

func managedOverrideForEdit(overrides []api.DNSOverride, uuid string) (*api.DNSOverride, error) {
	for index := range overrides {
		override := &overrides[index]
		if override.UUID != uuid {
			continue
		}
		if !app.IsManagedUnboundDescription(override.Description) {
			return nil, fmt.Errorf("refusing to edit unowned Unbound override %s; explicitly adopt it before syncing or edit it in OPNSense", uuid)
		}
		return override, nil
	}
	return nil, fmt.Errorf("no override found with UUID %s", uuid)
}

func verifyManagedOverride(client *api.Client, uuid string, expected api.DNSOverride) error {
	overrides, err := client.GetOverrides()
	if err != nil {
		return fmt.Errorf("read back Unbound override %s: %w", uuid, err)
	}
	for _, override := range overrides {
		if override.UUID != uuid {
			continue
		}
		if override.Host == expected.Host && override.Domain == expected.Domain && override.Server == expected.Server && override.Enabled == expected.Enabled && override.Description == expected.Description {
			return nil
		}
		return fmt.Errorf("verify Unbound override %s: provider returned different values", uuid)
	}
	return fmt.Errorf("verify Unbound override %s: record not found", uuid)
}

func promptForOverrideEdits(cmd *cobra.Command, override *api.DNSOverride) error {
	scanner := bufio.NewScanner(cmd.InOrStdin())

	fmt.Fprintf(cmd.OutOrStdout(), "Host [%s]: ", override.Host)
	if scanner.Scan() && scanner.Text() != "" {
		override.Host = scanner.Text()
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Domain [%s]: ", override.Domain)
	if scanner.Scan() && scanner.Text() != "" {
		override.Domain = scanner.Text()
	}

	fmt.Fprintf(cmd.OutOrStdout(), "IP Address [%s]: ", override.Server)
	if scanner.Scan() && scanner.Text() != "" {
		override.Server = scanner.Text()
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Description [%s]: ", override.Description)
	if scanner.Scan() && scanner.Text() != "" {
		override.Description = scanner.Text()
	}

	enabled := "Yes"
	if override.Enabled == "0" {
		enabled = "No"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Enable (y/n) [%s]: ", enabled)
	if scanner.Scan() && scanner.Text() != "" {
		switch strings.ToLower(scanner.Text()) {
		case "y":
			override.Enabled = "1"
		case "n":
			override.Enabled = "0"
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading edit prompts: %w", err)
	}
	return nil
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
