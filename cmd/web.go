package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
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

	token, err := newWebToken()
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", webHost, webPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return serveWeb(listener, runtime, token, webHost, cmd.OutOrStdout())
}

func serveWebForTest(listener net.Listener, token string, out io.Writer) error {
	return serveWeb(listener, &runtimeapp.Runtime{}, token, "127.0.0.1", out)
}

func serveWeb(listener net.Listener, runtime *runtimeapp.Runtime, token, boundHost string, out io.Writer) error {
	actualAddr := listener.Addr().String()
	server := &http.Server{
		Handler: webui.NewServerWithOptions(runtime, webui.Options{
			ApplyToken:    token,
			AllowedOrigin: "http://" + actualAddr,
			BoundHost:     boundHost,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logging.Info("Starting web GUI", "addr", actualAddr)
	fmt.Fprintf(out, "Web GUI listening on http://%s\n", actualAddr)
	return server.Serve(listener)
}

func newWebToken() (string, error) {
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate web token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}
