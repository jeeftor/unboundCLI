package cmd

import (
	"fmt"
	"os"

	"github.com/jeeftor/unboundCLI/internal/config"
	"github.com/jeeftor/unboundCLI/internal/tui"
	"github.com/spf13/cobra"
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the Text User Interface",
	Long: `Launch the interactive Text User Interface (TUI) for managing DNS overrides.

This command starts an interactive terminal interface using Bubble Tea and Lipgloss
for a modern, colorful experience. Navigate through your DNS overrides, add new ones,
edit existing ones, and delete them all from a single interface.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Println(
				UI.RenderError(
					fmt.Errorf(
						"error loading configuration: %v\nPlease run 'config' command to set up API access",
						err,
					),
				),
			)
			os.Exit(1)
		}

		// Create client
		client := NewClient(cfg)

		// Initialize and start the TUI
		model := tui.NewModel(client)
		if err := model.Start(); err != nil {
			fmt.Println(UI.RenderError(fmt.Errorf("error starting TUI: %v", err)))
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
