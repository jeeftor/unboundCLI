package cmd

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	runtimeapp "github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
	webui "github.com/jeeftor/caddy-dns-sync/internal/web"
	"github.com/spf13/cobra"
)

var (
	webHost            string
	webPort            int
	webOrigin          string
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
	webCmd.Flags().StringVar(&webOrigin, "origin", "", "allowed Origin header for browser mutations (e.g. https://caddy-sync.example.com); empty = no origin check")
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
		IncludeAuthentik:  true,
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
	return serveWeb(listener, runtime, token, webHost, webOrigin, cmd.OutOrStdout())
}

func serveWebForTest(listener net.Listener, token string, out io.Writer) error {
	return serveWebInternal(listener, &runtimeapp.Runtime{}, token, "127.0.0.1", "", out, false)
}

// serveWebInternal is the core server setup. When installSignalHandler is true,
// SIGINT/SIGTERM trigger graceful shutdown. When false, the caller manages
// lifecycle (used by tests which close the listener to stop the server).
func serveWebInternal(listener net.Listener, runtime *runtimeapp.Runtime, token, boundHost, allowedOrigin string, out io.Writer, installSignalHandler bool) error {
	addr := listener.Addr().String()
	webServer := webui.NewServerWithOptions(runtime, webui.Options{
		ApplyToken:     token,
		AllowMutations: true,
		AllowedOrigin:  allowedOrigin,
		BoundHost:      boundHost,
		Version:        Version,
		Commit:         Commit,
		BuildDate:      Date,
	})
	server := &http.Server{
		Handler:           webServer,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logging.Info("Starting web GUI", "addr", addr)
	fmt.Fprintf(out, "Web GUI listening on http://%s\n", addr)

	if !installSignalHandler {
		err := server.Serve(listener)
		webServer.Shutdown()
		return err
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var serveErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer logging.Recover("web: http server")
		serveErr = server.Serve(listener)
	}()

	<-ctx.Done()
	logging.Info("Shutting down web GUI...")
	stop() // restore default signal handling

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logging.Warn("HTTP server shutdown error", "error", err)
	}
	webServer.Shutdown()
	wg.Wait()

	if serveErr != nil && serveErr != http.ErrServerClosed {
		return serveErr
	}
	logging.Info("Web GUI stopped")
	return nil
}

func serveWeb(listener net.Listener, runtime *runtimeapp.Runtime, token, boundHost, allowedOrigin string, out io.Writer) error {
	return serveWebInternal(listener, runtime, token, boundHost, allowedOrigin, out, true)
}

func newWebToken() (string, error) {
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate web token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}
