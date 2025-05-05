package cmd

import (
	"encoding/json"
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
	jsonOutput bool
	quietMode  bool
)

type listUI struct {
	*ui.BaseUI
}

func newListUI() *listUI {
	return &listUI{ui.NewBaseUI()}
}

func (ui *listUI) RenderTable(overrides []api.DNSOverride) string {
	var sb strings.Builder
	// Table header
	sb.WriteString(fmt.Sprintf("%-36s %-20s %-20s %-15s %-30s %-10s\n",
		"UUID", "HOST", "DOMAIN", "IP ADDRESS", "DESCRIPTION", "ENABLED"))
	sb.WriteString(strings.Repeat("─", 140))
	sb.WriteString("\n")
	// Table rows
	for _, o := range overrides {
		enabled := "No"
		if o.Enabled == "1" {
			enabled = "Yes"
		}
		sb.WriteString(fmt.Sprintf("%-36s %-20s %-20s %-15s %-30s %-10s\n",
			o.UUID, o.Host, o.Domain, o.Server, o.Description, enabled))
	}
	// Add summary
	sb.WriteString("\n")
	sb.WriteString(ui.RenderInfo(fmt.Sprintf("Total DNS Overrides: %d", len(overrides))))
	return sb.String()
}

func (ui *listUI) RenderEmpty() string {
	return ui.RenderWarning("No DNS overrides found.")
}

func (ui *listUI) RenderError(err error) string {
	return ui.BaseUI.RenderError(err)
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List DNS overrides",
	Long: `List all DNS overrides configured in Unbound DNS.

This command retrieves all DNS overrides from the OPNSense API and displays them
in a table format. You can also output the results in JSON format using the --json flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create UI component
		listUI := newListUI()

		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			logging.Error("Error loading configuration", "error", err)
			fmt.Println(
				listUI.RenderError(
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

		// Get overrides
		if !quietMode {
			fmt.Println("Fetching DNS overrides...")
		}
		overrides, err := client.GetOverrides()
		if err != nil {
			logging.Error("Error fetching overrides", "error", err)
			fmt.Println(listUI.RenderError(err))
			os.Exit(1)
		}

		if len(overrides) == 0 {
			if !quietMode {
				fmt.Println(listUI.RenderEmpty())
			}
			return
		}

		// Display results
		if jsonOutput {
			// Output as JSON
			printOverridesJSON(overrides, listUI)
		} else {
			// Output as table
			fmt.Println(listUI.RenderTable(overrides))
		}
		logging.Info("Successfully displayed overrides", "count", len(overrides))
	},
}

// printOverridesJSON prints the overrides in JSON format
func printOverridesJSON(overrides []api.DNSOverride, ui *listUI) {
	jsonData, err := json.MarshalIndent(overrides, "", "  ")
	if err != nil {
		logging.Error("Error formatting JSON", "error", err)
		fmt.Println(ui.RenderError(fmt.Errorf("error formatting JSON: %v", err)))
		return
	}
	fmt.Println(string(jsonData))
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Add flags
	listCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	listCmd.Flags().BoolVarP(&quietMode, "quiet", "q", false, "Suppress informational output")
}
