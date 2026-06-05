package cmd

import (
	"fmt"

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
	RunE: runCFTunnelBackup,
}

func runCFTunnelBackup(cmd *cobra.Command, args []string) error {
	cfClient, err := loadCloudflareClient()
	if err != nil {
		return err
	}

	path, err := cfClient.BackupTunnelConfig()
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Backup saved:", path)
	return nil
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
	RunE: runCFTunnelRestore,
}

func runCFTunnelRestore(cmd *cobra.Command, args []string) error {
	cfClient, err := loadCloudflareClient()
	if err != nil {
		return err
	}

	if restoreListOnly || restoreBackupFile == "" {
		backups, err := cfClient.ListTunnelBackups()
		if err != nil {
			return fmt.Errorf("error listing backups: %w", err)
		}
		if len(backups) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No backups found in ~/.caddy-dns-sync-backups/")
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Available backups (%d), newest first:\n", len(backups))
		for _, p := range backups {
			fmt.Fprintln(cmd.OutOrStdout(), " ", p)
		}
		if restoreBackupFile == "" {
			fmt.Fprintln(cmd.OutOrStdout(), "\nRun with --file <path> to restore one of the above.")
		}
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Saving pre-restore backup...")
	prePath, err := cfClient.BackupTunnelConfig()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not save pre-restore backup: %v\n", err)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Pre-restore backup saved:", prePath)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Restoring from:", restoreBackupFile)
	if err := cfClient.RestoreTunnelConfig(restoreBackupFile); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Tunnel configuration restored successfully.")
	return nil
}

func loadCloudflareClient() (*api.CloudflareClient, error) {
	cfCfg, err := config.LoadCloudflareConfig()
	if err != nil {
		logging.Error("Error loading Cloudflare configuration", "error", err)
		return nil, fmt.Errorf("error loading Cloudflare configuration: %w", err)
	}
	if !cfCfg.Enabled {
		return nil, fmt.Errorf("Cloudflare integration is disabled")
	}

	cfClient, err := api.NewCloudflareClient(cfCfg.GetCloudflareAPIConfig())
	if err != nil {
		return nil, fmt.Errorf("error creating Cloudflare client: %w", err)
	}
	return cfClient, nil
}

func init() {
	rootCmd.AddCommand(cfTunnelBackupCmd)
	rootCmd.AddCommand(cfTunnelRestoreCmd)

	cfTunnelRestoreCmd.Flags().StringVar(&restoreBackupFile, "file", "", "Backup file to restore (from cf-tunnel-backup output or --list)")
	cfTunnelRestoreCmd.Flags().BoolVar(&restoreListOnly, "list", false, "List available backup files")
}
