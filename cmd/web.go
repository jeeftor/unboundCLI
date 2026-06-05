package cmd

import (
	"fmt"
	"net/http"
	"time"

	runtimeapp "github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	webui "github.com/jeeftor/caddy-dns-sync/internal/web"
	"github.com/spf13/cobra"
)

var (
	webHost            string
	webPort            int
	webCaddyServerIP   string
	webCaddyServerPort int
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the local web GUI",
	Long: `Start a local web GUI for viewing DNS sync status, previewing sync actions,
and running dry-run sync apply operations through the shared service layer.`,
	RunE: runWeb,
}

func init() {
	rootCmd.AddCommand(webCmd)

	webCmd.Flags().StringVar(&webHost, "host", "127.0.0.1", "host interface for the web server")
	webCmd.Flags().IntVar(&webPort, "port", 8080, "port for the web server")
	webCmd.Flags().StringVar(&webCaddyServerIP, "caddy-ip", runtimeapp.DefaultCaddyServerIP, "Caddy server IP")
	webCmd.Flags().IntVar(&webCaddyServerPort, "caddy-port", runtimeapp.DefaultCaddyServerPort, "Caddy admin API port")
}

func runWeb(cmd *cobra.Command, args []string) error {
	runtime, err := runtimeapp.LoadRuntime(runtimeapp.RuntimeOptions{
		CaddyServerIP:     webCaddyServerIP,
		CaddyServerPort:   webCaddyServerPort,
		IncludeUnbound:    true,
		IncludeDNSMasq:    true,
		IncludeAdguard:    true,
		IncludeCloudflare: true,
	})
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", webHost, webPort)
	server := &http.Server{
		Addr:              addr,
		Handler:           webui.NewServer(runtime),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logging.Info("Starting web GUI", "addr", addr)
	fmt.Fprintf(cmd.OutOrStdout(), "Web GUI listening on http://%s\n", addr)
	return server.ListenAndServe()
}
