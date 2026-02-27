package cmd

import (
	"fmt"
	"os"

	"github.com/jeeftor/caddy-dns-sync/internal/commands"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/spf13/cobra"
)

var (
	listJsonOutput bool
	listQuietMode  bool
)

// listCmd is the parent command for listing entries from various services
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List entries from various services",
	Long: `List DNS entries, DHCP leases, or routes from different services.

Available subcommands:
  all      - Show 3-way sync status across all services
  unbound  - List UnboundDNS host overrides
  adguard  - List AdguardHome DNS rewrites
  dhcp     - List DNSMasq DHCP leases
  caddy    - List Caddy reverse proxy routes`,
}

// allCmd shows 3-way sync status
var allCmd = &cobra.Command{
	Use:     "all",
	Aliases: []string{"status"},
	Short:   "Show 3-way sync status across all services",
	Long: `Show the sync status of all hostnames across Caddy, UnboundDNS, and AdguardHome.

This command displays a table showing which services are configured in Caddy and whether
they are properly synchronized to UnboundDNS and AdguardHome. It's similar to the
interactive dashboard but outputs a static table for quick status checks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		source := commands.NewAllDataSource()
		runner := commands.NewListCommandRunner(source)
		runner.SetJSONOutput(listJsonOutput)
		runner.SetQuietMode(listQuietMode)

		if err := runner.Run(); err != nil {
			logging.Error("Error listing sync status", "error", err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}
		return nil
	},
}

// unboundCmd lists Unbound DNS overrides
var unboundCmd = &cobra.Command{
	Use:     "unbound",
	Aliases: []string{"u"},
	Short:   "List UnboundDNS host overrides",
	Long: `List all DNS host overrides from UnboundDNS.

This command retrieves all host override entries from the OPNSense UnboundDNS API
and displays them in a table format. You can also output the results in JSON format
using the --json flag.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		source := commands.NewUnboundDataSource()
		runner := commands.NewListCommandRunner(source)
		runner.SetJSONOutput(listJsonOutput)
		runner.SetQuietMode(listQuietMode)

		if err := runner.Run(); err != nil {
			logging.Error("Error listing Unbound overrides", "error", err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}
		return nil
	},
}

// adguardCmd lists AdguardHome DNS rewrites
var adguardCmd = &cobra.Command{
	Use:     "adguard",
	Aliases: []string{"a"},
	Short:   "List AdguardHome DNS rewrites",
	Long: `List all DNS rewrite rules from AdguardHome.

This command retrieves all DNS rewrite rules from the AdguardHome API and displays
them in a table format. You can also output the results in JSON format using the
--json flag.

Note: AdguardHome must be enabled in the configuration for this command to work.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		source := commands.NewAdguardDataSource()
		runner := commands.NewListCommandRunner(source)
		runner.SetJSONOutput(listJsonOutput)
		runner.SetQuietMode(listQuietMode)

		if err := runner.Run(); err != nil {
			logging.Error("Error listing Adguard rewrites", "error", err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}
		return nil
	},
}

// dhcpCmd lists DNSMasq DHCP leases
var dhcpCmd = &cobra.Command{
	Use:     "dhcp",
	Aliases: []string{"dnsmasq", "d"},
	Short:   "List DNSMasq DHCP leases",
	Long: `List all DHCP leases from DNSMasq.

This command retrieves all DHCP leases from the OPNSense DNSMasq API and displays them
in a table format. You can also output the results in JSON format using the --json flag.

The table shows both static reservations and dynamic leases, along with their MAC addresses
and expiration times (for dynamic leases).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		source := commands.NewDHCPDataSource()
		runner := commands.NewListCommandRunner(source)
		runner.SetJSONOutput(listJsonOutput)
		runner.SetQuietMode(listQuietMode)

		if err := runner.Run(); err != nil {
			logging.Error("Error listing DHCP leases", "error", err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}
		return nil
	},
}

// caddyCmd lists Caddy reverse proxy routes
var caddyCmd = &cobra.Command{
	Use:     "caddy",
	Aliases: []string{"c"},
	Short:   "List Caddy reverse proxy routes",
	Long: `List all reverse proxy routes from Caddy.

This command retrieves all configured reverse proxy routes from the Caddy Admin API
and displays them in a table format showing the hostname and upstream target (IP:Port).
You can also output the results in JSON format using the --json flag.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		source := commands.NewCaddyDataSource()
		runner := commands.NewListCommandRunner(source)
		runner.SetJSONOutput(listJsonOutput)
		runner.SetQuietMode(listQuietMode)

		if err := runner.Run(); err != nil {
			logging.Error("Error listing Caddy routes", "error", err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Add subcommands
	listCmd.AddCommand(allCmd)
	listCmd.AddCommand(unboundCmd)
	listCmd.AddCommand(adguardCmd)
	listCmd.AddCommand(dhcpCmd)
	listCmd.AddCommand(caddyCmd)

	// Persistent flags (apply to all subcommands)
	listCmd.PersistentFlags().BoolVar(&listJsonOutput, "json", false, "Output in JSON format")
	listCmd.PersistentFlags().BoolVarP(&listQuietMode, "quiet", "q", false, "Suppress informational output")
}
