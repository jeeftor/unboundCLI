package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/tui"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
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
		applyUI := newApplyUI()

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

type applyUI struct {
	Styles tui.StyleConfig
}

func newApplyUI() *applyUI {
	return &applyUI{
		Styles: tui.DefaultStyles(),
	}
}

func (ui *applyUI) RenderHeader() string {
	return ui.Styles.Header.Render(" ⚙️ Apply DNS Changes ⚙️ ") + "\n\n"
}

func (ui *applyUI) RenderSuccess() string {
	var sb strings.Builder
	sb.WriteString(ui.Styles.Success.Render(" ✅ DNS changes applied successfully "))
	return sb.String()
}

func (ui *applyUI) RenderError(err error) string {
	return ui.Styles.Error.Render(fmt.Sprintf(" ❌ Error: %s ", err))
}

func (ui *applyUI) RenderApplyingMessage() string {
	return ui.Styles.Info.Render(" 💾 Applying DNS changes... ")
}

func init() {
	rootCmd.AddCommand(applyCmd)
}
