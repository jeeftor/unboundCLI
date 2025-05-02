package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jeeftor/unboundCLI/internal/exec/list"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/config"
	"github.com/jeeftor/unboundCLI/internal/logging"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	quietMode  bool
)

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
		listUI := list.NewUI()

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

// printOverridesTable prints the overrides in a table format
func printOverridesTable(overrides []api.DNSOverride) {
	// Print header
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "UUID\tHOST\tDOMAIN\tIP ADDRESS\tDESCRIPTION\tENABLED")
	for _, o := range overrides {
		enabled := "No"
		if o.Enabled == "1" {
			enabled = "Yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			o.UUID, o.Host, o.Domain, o.Server, o.Description, enabled)
	}
	w.Flush()
}

// printOverridesJSON prints the overrides in JSON format
func printOverridesJSON(overrides []api.DNSOverride, ui *list.UI) {
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
