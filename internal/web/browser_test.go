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
								},
								{
									"match": [{"host": ["hidden.example.test"]}],
									"handle": [
										{
											"handler": "reverse_proxy",
											"upstreams": [{"dial": "10.0.0.6:8080"}]
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
	webServer := httptest.NewServer(NewServerWithOptions(&app.Runtime{
		CaddyEndpoint: app.CaddyEndpoint{ServerIP: host, ServerPort: port},
		Clients: app.ClientSet{
			Caddy: api.NewCaddyClient(host, port),
		},
	}, Options{EnableTestHooks: true}))
	defer webServer.Close()

	dom := runChromeSmoke(t, chromePath, webServer.URL, 1280, 900)
	if !strings.Contains(dom, "Caddy DNS Sync") || !strings.Contains(dom, "browser.example.test") {
		t.Fatalf("browser DOM did not render fake entry:\n%s", dom)
	}
	if !strings.Contains(dom, `id="message"`) || !strings.Contains(dom, `aria-live="polite"`) {
		t.Fatalf("browser DOM missing live message region:\n%s", dom)
	}
	if strings.Contains(dom, "Failed to fetch") {
		t.Fatalf("browser DOM shows stale loading or fetch failure:\n%s", dom)
	}
	if !strings.Contains(dom, `data-adguard-enabled="false"`) {
		t.Fatalf("browser DOM should mark unavailable AdGuard controls disabled:\n%s", dom)
	}
	if !strings.Contains(dom, `data-loading="false"`) || !strings.Contains(dom, `id="top-progress"`) {
		t.Fatalf("browser DOM should expose completed loading state and progress bar:\n%s", dom)
	}
	if !strings.Contains(dom, `class="service-card`) || !strings.Contains(dom, `Cloudflare</span>`) {
		t.Fatalf("browser DOM should render service health cards:\n%s", dom)
	}
	if !strings.Contains(dom, `class="row-preview"`) || !strings.Contains(dom, `Not routed`) {
		t.Fatalf("browser DOM should render row preview controls and Cloudflare route status:\n%s", dom)
	}
	if !strings.Contains(dom, `dns-result bad`) {
		t.Fatalf("browser DOM should color failed DNS resolution as bad:\n%s", dom)
	}

	filteredDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=filter:caddy_only,search:browser", 1280, 900)
	if !strings.Contains(filteredDOM, `data-e2e="done"`) {
		t.Fatalf("browser e2e hook did not complete:\n%s", filteredDOM)
	}
	if !strings.Contains(filteredDOM, "browser.example.test") {
		t.Fatalf("filtered DOM missing browser.example.test:\n%s", filteredDOM)
	}
	if strings.Contains(filteredDOM, "hidden.example.test") {
		t.Fatalf("filtered DOM should hide hidden.example.test:\n%s", filteredDOM)
	}

	previewDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=preview:unbound", 1280, 900)
	if !strings.Contains(previewDOM, "ADD unbound browser.example.test") {
		t.Fatalf("preview did not render expected action:\n%s", previewDOM)
	}
	if !strings.Contains(previewDOM, `data-dry-run-enabled="true"`) {
		t.Fatalf("dry-run button should be enabled after planned actions:\n%s", previewDOM)
	}

	dryRunDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=preview:unbound,dryrun", 1280, 900)
	if !strings.Contains(dryRunDOM, "All operations completed successfully") || !strings.Contains(dryRunDOM, "added=2") {
		t.Fatalf("dry-run result was not rendered:\n%s", dryRunDOM)
	}

	rowPreviewDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=rowpreview:browser.example.test:unbound", 1280, 900)
	if !strings.Contains(rowPreviewDOM, "ADD unbound browser.example.test") || strings.Contains(rowPreviewDOM, "ADD unbound hidden.example.test") {
		t.Fatalf("row preview should render only the selected hostname action:\n%s", rowPreviewDOM)
	}

	mobileDOM := runChromeSmoke(t, chromePath, webServer.URL, 390, 844)
	if !strings.Contains(mobileDOM, `data-mobile="true"`) {
		t.Fatalf("mobile DOM did not mark mobile layout:\n%s", mobileDOM)
	}
	if !strings.Contains(mobileDOM, `data-table-scrolls="true"`) {
		t.Fatalf("mobile DOM should expose horizontal table scrolling:\n%s", mobileDOM)
	}
}

func runChromeSmoke(t *testing.T, chromePath, targetURL string, width, height int) string {
	t.Helper()
	screenshotPath := filepath.Join(t.TempDir(), fmt.Sprintf("web-smoke-%dx%d.png", width, height))
	cmd := exec.Command(chromePath,
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--hide-scrollbars",
		fmt.Sprintf("--window-size=%d,%d", width, height),
		"--virtual-time-budget=5000",
		"--dump-dom",
		"--screenshot="+screenshotPath,
		targetURL,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("browser smoke failed: %v\n%s", err, string(output))
	}
	info, err := os.Stat(screenshotPath)
	if err != nil {
		t.Fatalf("browser screenshot missing: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("browser screenshot was empty")
	}
	return string(output)
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
