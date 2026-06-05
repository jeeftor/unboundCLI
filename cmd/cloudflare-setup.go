package cmd

import (
	"fmt"

	"github.com/jeeftor/caddy-dns-sync/internal/tui"
	"github.com/spf13/cobra"
)

var cloudflareSetupCmd = &cobra.Command{
	Use:   "cloudflare-setup",
	Short: "Interactive Cloudflare tunnel configuration wizard",
	Long: `Interactive wizard for configuring Cloudflare tunnel integration.

The wizard will guide you through:
  1. Entering your Cloudflare API token
  2. Entering your Cloudflare account ID
  3. Selecting a tunnel from your account
  4. Selecting a DNS zone from your account
  5. Entering the Caddy service URL
  6. Confirming and saving the configuration

The configuration is saved to ~/.caddy-dns-sync.json and used by
the caddy-push-cloudflare command.`,
	RunE: runCloudflareSetup,
}

func runCloudflareSetup(cmd *cobra.Command, args []string) error {
	wizard := tui.NewCloudflareSetupWizard()
	if err := wizard.Run(); err != nil {
		return fmt.Errorf("error running Cloudflare setup wizard: %w", err)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(cloudflareSetupCmd)
}
