package web

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
)

func TestBrowserSmokeWithFakeData(t *testing.T) {
	if os.Getenv("UNBOUNDCLI_BROWSER_TESTS") != "1" {
		t.Skip("set UNBOUNDCLI_BROWSER_TESTS=1 to run browser smoke checks")
	}

	chromePath := chromeHeadlessShellPath(t)
	caddy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config/" {
			t.Fatalf("unexpected Caddy path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"apps": {
				"http": {
					"servers": {
						"srv0": {
							"routes": [
								{
									"match": [{"host": ["browser.example.test"]}],
									"handle": [
										{
											"handler": "reverse_proxy",
											"upstreams": [{"dial": "10.0.0.5:8080"}]
										}
									]
								}
							]
						}
					}
				}
			}
		}`)
	}))
	defer caddy.Close()

	host, port := splitBrowserTestServerHostPort(t, caddy.URL)
	webServer := httptest.NewServer(NewServer(&app.Runtime{
		CaddyEndpoint: app.CaddyEndpoint{ServerIP: host, ServerPort: port},
		Clients: app.ClientSet{
			Caddy: api.NewCaddyClient(host, port),
		},
	}))
	defer webServer.Close()

	screenshotPath := filepath.Join(t.TempDir(), "web-smoke.png")
	cmd := exec.Command(chromePath,
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--hide-scrollbars",
		"--window-size=1280,900",
		"--virtual-time-budget=5000",
		"--dump-dom",
		"--screenshot="+screenshotPath,
		webServer.URL,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("browser smoke failed: %v\n%s", err, string(output))
	}
	dom := string(output)
	if !strings.Contains(dom, "Caddy DNS Sync") || !strings.Contains(dom, "browser.example.test") {
		t.Fatalf("browser DOM did not render fake entry:\n%s", dom)
	}
	if !strings.Contains(dom, `id="state">Loaded</span>`) || strings.Contains(dom, "Failed to fetch") {
		t.Fatalf("browser DOM shows stale loading or fetch failure:\n%s", dom)
	}
	info, err := os.Stat(screenshotPath)
	if err != nil {
		t.Fatalf("browser screenshot missing: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("browser screenshot was empty")
	}
}

func chromeHeadlessShellPath(t *testing.T) string {
	t.Helper()
	if path := os.Getenv("CHROME_HEADLESS_SHELL"); path != "" {
		return path
	}
	for _, path := range []string{
		"/Users/jstein/chrome-headless-shell/mac_arm-148.0.7778.167/chrome-headless-shell-mac-arm64/chrome-headless-shell",
		"/Users/jstein/.cache/puppeteer/chrome-headless-shell/mac_arm-148.0.7778.167/chrome-headless-shell-mac-arm64/chrome-headless-shell",
	} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	t.Skip("set CHROME_HEADLESS_SHELL to a chrome-headless-shell binary")
	return ""
}

func splitBrowserTestServerHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}
	host, portString, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("failed to split server host/port: %v", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatalf("failed to parse server port: %v", err)
	}
	return host, port
}
