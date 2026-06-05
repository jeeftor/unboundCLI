package cmd

import (
	"fmt"
	"os"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	"github.com/spf13/cobra"
)

// cf-tunnel-backup: save a snapshot of the current CF tunnel ingress rules
var cfTunnelBackupCmd = &cobra.Command{
	Use:   "cf-tunnel-backup",
	Short: "Save a backup snapshot of the Cloudflare tunnel ingress rules",
	Long: `Fetches the current Cloudflare tunnel ingress configuration and writes it to
~/.caddy-dns-sync-backups/cf-tunnel-<id>-<timestamp>.json.

Backups are also written automatically before every cf-tunnel-restore or TUI edit.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfCfg, err := config.LoadCloudflareConfig()
		if err != nil {
			logging.Error("Error loading Cloudflare configuration", "error", err)
			fmt.Fprintf(os.Stderr, "Error loading Cloudflare configuration: %v\n", err)
			os.Exit(1)
		}
		if !cfCfg.Enabled {
			fmt.Fprintln(os.Stderr, "Cloudflare integration is disabled.")
			os.Exit(1)
		}

		cfClient, err := api.NewCloudflareClient(cfCfg.GetCloudflareAPIConfig())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Cloudflare client: %v\n", err)
			os.Exit(1)
		}

		path, err := cfClient.BackupTunnelConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Backup failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Backup saved:", path)
	},
}

// cf-tunnel-restore: restore a backup snapshot to Cloudflare
var (
	restoreBackupFile string
	restoreListOnly   bool
)

var cfTunnelRestoreCmd = &cobra.Command{
	Use:   "cf-tunnel-restore",
	Short: "Restore a Cloudflare tunnel ingress backup",
	Long: `Lists available backups or restores a specific backup file to Cloudflare.

A pre-restore backup of the current state is saved automatically before applying.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfCfg, err := config.LoadCloudflareConfig()
		if err != nil {
			logging.Error("Error loading Cloudflare configuration", "error", err)
			fmt.Fprintf(os.Stderr, "Error loading Cloudflare configuration: %v\n", err)
			os.Exit(1)
		}
		if !cfCfg.Enabled {
			fmt.Fprintln(os.Stderr, "Cloudflare integration is disabled.")
			os.Exit(1)
		}

		cfClient, err := api.NewCloudflareClient(cfCfg.GetCloudflareAPIConfig())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Cloudflare client: %v\n", err)
			os.Exit(1)
		}

		// --list: show available backups
		if restoreListOnly || restoreBackupFile == "" {
			backups, err := cfClient.ListTunnelBackups()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error listing backups: %v\n", err)
				os.Exit(1)
			}
			if len(backups) == 0 {
				fmt.Println("No backups found in ~/.caddy-dns-sync-backups/")
				return
			}
			fmt.Printf("Available backups (%d), newest first:\n", len(backups))
			for _, p := range backups {
				fmt.Println(" ", p)
			}
			if restoreBackupFile == "" {
				fmt.Println("\nRun with --file <path> to restore one of the above.")
			}
			return
		}

		// Save a pre-restore backup first
		fmt.Println("Saving pre-restore backup...")
		prePath, err := cfClient.BackupTunnelConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save pre-restore backup: %v\n", err)
		} else {
			fmt.Println("Pre-restore backup saved:", prePath)
		}

		fmt.Println("Restoring from:", restoreBackupFile)
		if err := cfClient.RestoreTunnelConfig(restoreBackupFile); err != nil {
			fmt.Fprintf(os.Stderr, "Restore failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Tunnel configuration restored successfully.")
	},
}

func init() {
	rootCmd.AddCommand(cfTunnelBackupCmd)
	rootCmd.AddCommand(cfTunnelRestoreCmd)

	cfTunnelRestoreCmd.Flags().StringVar(&restoreBackupFile, "file", "", "Backup file to restore (from cf-tunnel-backup output or --list)")
	cfTunnelRestoreCmd.Flags().BoolVar(&restoreListOnly, "list", false, "List available backup files")
}
