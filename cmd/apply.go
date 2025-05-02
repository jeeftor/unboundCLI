package cmd

import (
	"fmt"
	"os"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/config"
	"github.com/jeeftor/unboundCLI/internal/exec/apply"
	"github.com/jeeftor/unboundCLI/internal/logging"
	"github.com/spf13/cobra"
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply pending DNS changes",
	Long: `Apply pending DNS changes to Unbound DNS.

This command applies any pending changes to Unbound DNS. Changes made with
the add, edit, or delete commands are not applied immediately. You must use
this command to apply the changes.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create UI component
		applyUI := apply.NewUI()

		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			logging.Error("Error loading configuration", "error", err)
			fmt.Println(
				applyUI.RenderError(
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

		// Apply changes
		fmt.Println(applyUI.RenderApplyingMessage())
		if err := client.ApplyChanges(); err != nil {
			logging.Error("Error applying changes", "error", err)
			fmt.Println(
				applyUI.RenderError(
					fmt.Errorf("error applying changes: %v", err),
				),
			)
			os.Exit(1)
		}

		fmt.Println(applyUI.RenderSuccess())
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
}
