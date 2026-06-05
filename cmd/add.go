package cmd

import (
	"fmt"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/jeeftor/caddy-dns-sync/internal/ui"
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
	RunE: runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	addUI := newAddUI()

	if host == "" || domain == "" || server == "" {
		logging.Error("Missing required flags",
			"host", host,
			"domain", domain,
			"server", server)
		_ = cmd.Help()
		return fmt.Errorf("host, domain, and server are required")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		logging.Error("Error loading configuration", "error", err)
		return fmt.Errorf("error loading configuration: %w\nPlease run 'config' command to set up API access", err)
	}

	client := api.NewClient(cfg)

	if !forceAdd {
		if logging.GetLogLevel() != logging.LogLevelDebug {
			fmt.Fprintln(cmd.OutOrStdout(), addUI.RenderCheckingMessage())
		}
		exists, uuid, err := client.IsOverrideExists(host, domain)
		if err != nil {
			logging.Error("Error checking existing overrides", "error", err)
			return fmt.Errorf("error checking existing overrides: %w", err)
		}

		if exists {
			logging.Warn("DNS override already exists",
				"host", host,
				"domain", domain,
				"uuid", uuid)
			fmt.Fprintln(cmd.OutOrStdout(), addUI.RenderDuplicateWarning(host, domain, uuid))
			return nil
		}
	}

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

	fmt.Fprintln(cmd.OutOrStdout(), addUI.RenderAddingMessage())
	uuid, err := client.AddOverride(override)
	if err != nil {
		logging.Error("Error adding override", "error", err)
		if strings.Contains(err.Error(), "already exists") {
			fmt.Fprintln(cmd.OutOrStdout(), addUI.RenderWarning(err.Error()))
			fmt.Fprintln(cmd.OutOrStdout(), addUI.RenderWarning("Use --force flag to add anyway"))
			return nil
		}
		return fmt.Errorf("error adding override: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), addUI.RenderApplyingMessage())
	if err := client.ApplyChanges(); err != nil {
		logging.Error("Error applying changes", "error", err)
		return fmt.Errorf("error applying changes: %w\nThe override was added but changes were not applied", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), addUI.RenderSuccess(fmt.Sprintf("DNS override added successfully with UUID: %s", uuid)))
	if logging.GetLogLevel() == logging.LogLevelDebug {
		logging.Info("DNS override added successfully", "uuid", uuid)
	}
	return nil
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
