package cmd

import (
	"fmt"
	"os"

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
	Run: func(cmd *cobra.Command, args []string) {
		wizard := tui.NewCloudflareSetupWizard()
		if err := wizard.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running Cloudflare setup wizard: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(cloudflareSetupCmd)
}
