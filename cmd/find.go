package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jeeftor/unboundCLI/internal/api"
	"github.com/jeeftor/unboundCLI/internal/config"
	"github.com/jeeftor/unboundCLI/internal/exec/find"
	"github.com/jeeftor/unboundCLI/internal/logging"
	"github.com/spf13/cobra"
)

var (
	findHost       string
	findDomain     string
	findJsonOutput bool
	scriptOutput   bool
)

// findCmd represents the find command
var findCmd = &cobra.Command{
	Use:   "find",
	Short: "Find DNS overrides by host, domain, or both",
	Long: `Find DNS overrides by host, domain, or both.

This command searches for DNS overrides based on the specified criteria.
It can be used to find the UUID of an entry for use in other commands.

Examples:
  unboundCLI find --host test
  unboundCLI find --domain vookie.net
  unboundCLI find --host test --domain vookie.net
  unboundCLI find --host test --json
  unboundCLI find --host test --script`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create UI component
		findUI := find.NewUI()

		// Validate that at least one search parameter is provided
		if findHost == "" && findDomain == "" {
			fmt.Println(
				findUI.RenderError(
					fmt.Errorf(
						"at least one search parameter (--host or --domain) must be provided",
					),
				),
			)
			os.Exit(1)
		}

		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			if logging.GetLogLevel() == logging.LogLevelDebug {
				logging.Error("Error loading configuration", "error", err)
			}
			fmt.Println(
				findUI.RenderError(
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

		// Get all overrides
		if !findJsonOutput && !scriptOutput {
			fmt.Println(findUI.RenderSearchingMessage())
		}
		overrides, err := client.GetOverrides()
		if err != nil {
			if logging.GetLogLevel() == logging.LogLevelDebug {
				logging.Error("Error fetching overrides", "error", err)
			}
			fmt.Println(
				findUI.RenderError(
					fmt.Errorf("error fetching overrides: %v", err),
				),
			)
			os.Exit(1)
		}

		// Filter overrides based on search criteria
		var matches []api.DNSOverride
		for _, override := range overrides {
			if findHost != "" && findDomain != "" {
				// Match both host and domain
				if strings.EqualFold(override.Host, findHost) &&
					strings.EqualFold(override.Domain, findDomain) {
					matches = append(matches, override)
				}
			} else if findHost != "" {
				// Match only host
				if strings.EqualFold(override.Host, findHost) {
					matches = append(matches, override)
				}
			} else if findDomain != "" {
				// Match only domain
				if strings.EqualFold(override.Domain, findDomain) {
					matches = append(matches, override)
				}
			}
		}

		// Display results
		if len(matches) == 0 {
			if findJsonOutput {
				fmt.Println("[]")
			} else if scriptOutput {
				// For script mode, just exit silently with error code 1 when no matches are found
				os.Exit(1)
			} else {
				fmt.Println(findUI.RenderNoMatches())
			}
			if !scriptOutput {
				os.Exit(0)
			} else {
				os.Exit(1) // Exit with error code for scripting
			}
		}

		if findJsonOutput {
			// Output in JSON format
			fmt.Println(findUI.RenderJSON(matches))
		} else if scriptOutput {
			// Output just the UUIDs for scripting
			for _, override := range matches {
				fmt.Println(override.UUID)
			}
		} else {
			// Output in human-readable format
			fmt.Println(findUI.RenderMatches(matches))
		}
	},
}

func init() {
	rootCmd.AddCommand(findCmd)

	// Add flags
	findCmd.Flags().StringVar(&findHost, "host", "", "Host name to search for")
	findCmd.Flags().StringVar(&findDomain, "domain", "", "Domain name to search for")
	findCmd.Flags().BoolVar(&findJsonOutput, "json", false, "Output results in JSON format")
	findCmd.Flags().
		BoolVar(&scriptOutput, "script", false, "Output just UUIDs for scripting (exits with code 1 if no matches)")
}
