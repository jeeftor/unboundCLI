package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

const unitTemplate = `[Unit]
Description=CaddySync Web UI
After=network.target

[Service]
Type=simple
Environment=HOME={{.HomeDir}}
ExecStart={{.BinPath}} web --host {{.Host}} --port {{.Port}}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`

const unitPath = "/etc/systemd/system/caddy-sync.service"

var (
	installHost  string
	installPort  int
	installStart bool
)

var installServiceCmd = &cobra.Command{
	Use:   "install-service",
	Short: "Install caddy-sync as a systemd service",
	Long:  `Writes a systemd unit file and enables caddy-sync web as a service.`,
	RunE:  runInstallService,
}

var uninstallServiceCmd = &cobra.Command{
	Use:   "uninstall-service",
	Short: "Remove the caddy-sync systemd service",
	Long:  `Stops, disables, and removes the caddy-sync systemd unit file.`,
	RunE:  runUninstallService,
}

func init() {
	rootCmd.AddCommand(installServiceCmd)
	rootCmd.AddCommand(uninstallServiceCmd)

	installServiceCmd.Flags().StringVar(&installHost, "host", "127.0.0.1", "host interface for the web server")
	installServiceCmd.Flags().IntVar(&installPort, "port", 8080, "port for the web server")
	installServiceCmd.Flags().BoolVar(&installStart, "start", false, "start the service immediately after installing")
}

func runInstallService(cmd *cobra.Command, args []string) error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate binary: %w", err)
	}
	binPath, err = filepath.EvalSymlinks(binPath)
	if err != nil {
		return fmt.Errorf("resolve binary path: %w", err)
	}

	tmpl, err := template.New("unit").Parse(unitTemplate)
	if err != nil {
		return fmt.Errorf("parse unit template: %w", err)
	}

	f, err := os.OpenFile(unitPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("write unit file (are you root?): %w", err)
	}
	defer f.Close()

	homeDir, _ := os.UserHomeDir()
	if err := tmpl.Execute(f, struct {
		BinPath string
		Host    string
		Port    int
		HomeDir string
	}{binPath, installHost, installPort, homeDir}); err != nil {
		return fmt.Errorf("render unit file: %w", err)
	}
	o := cmd.OutOrStdout()
	fmt.Fprintf(o, "  %s  Wrote %s\n", SymOK, StyleCode.Render(unitPath))

	for _, sc := range [][]string{
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", "caddy-sync"},
	} {
		if out, err := exec.Command(sc[0], sc[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("run %v: %w\n%s", sc, err, out)
		}
		fmt.Fprintf(o, "  %s  %s\n", SymOK, StyleMuted.Render(strings.Join(sc, " ")))
	}

	if installStart {
		// Use restart so the new unit file is always picked up, even if already running
		if out, err := exec.Command("systemctl", "restart", "caddy-sync").CombinedOutput(); err != nil {
			return fmt.Errorf("restart service: %w\n%s", err, out)
		}
		fmt.Fprintf(o, "  %s  %s\n", SymOK, StyleMuted.Render("systemctl restart caddy-sync"))
	}

	fmt.Fprintf(o, "\n  %s  Service installed.  To start: %s\n",
		SymOK, StyleCode.Render("systemctl start caddy-sync"))
	return nil
}

func runUninstallService(cmd *cobra.Command, args []string) error {
	o := cmd.OutOrStdout()
	for _, sc := range [][]string{
		{"systemctl", "stop", "caddy-sync"},
		{"systemctl", "disable", "caddy-sync"},
	} {
		out, err := exec.Command(sc[0], sc[1:]...).CombinedOutput()
		if err != nil {
			fmt.Fprintf(o, "  %s  %s: %s\n", SymWarn, StyleMuted.Render(strings.Join(sc, " ")), StyleMuted.Render(strings.TrimSpace(string(out))))
		} else {
			fmt.Fprintf(o, "  %s  %s\n", SymOK, StyleMuted.Render(strings.Join(sc, " ")))
		}
	}

	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove unit file: %w", err)
	}
	fmt.Fprintf(o, "  %s  Removed %s\n", SymOK, StyleCode.Render(unitPath))

	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload: %w\n%s", err, out)
	}
	fmt.Fprintf(o, "  %s  %s\n", SymOK, StyleMuted.Render("systemctl daemon-reload"))
	return nil
}
