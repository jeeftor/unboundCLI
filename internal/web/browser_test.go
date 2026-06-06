package web

import (
	"encoding/json"
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
	"github.com/jeeftor/caddy-dns-sync/internal/config"
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

	opnsense := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/unbound/settings/searchHostOverride":
			fmt.Fprint(w, `{"rows":[]}`)
		case "/api/unbound/settings/addHostOverride":
			fmt.Fprint(w, `{"result":"saved","uuid":"new-uuid"}`)
		case "/api/unbound/service/reconfigure":
			fmt.Fprint(w, `{"result":"saved"}`)
		default:
			t.Fatalf("unexpected OPNSense path %s", r.URL.Path)
		}
	}))
	defer opnsense.Close()

	host, port := splitBrowserTestServerHostPort(t, caddy.URL)
	configPath := filepath.Join(t.TempDir(), "browser-config.json")
	webHandler := NewServerWithOptions(&app.Runtime{
		UnboundConfig: api.Config{
			APIKey:    "fixture-browser-key",
			APISecret: "fixture-browser-secret",
			BaseURL:   opnsense.URL,
			Insecure:  true,
		},
		CaddyEndpoint: app.CaddyEndpoint{ServerIP: host, ServerPort: port},
		Clients: app.ClientSet{
			Caddy: api.NewCaddyClient(host, port),
			Unbound: api.NewClient(api.Config{
				APIKey:    "fixture-key",
				APISecret: "fixture-secret",
				BaseURL:   opnsense.URL,
				Insecure:  true,
			}),
		},
	}, Options{
		ApplyToken:      "browser-token",
		AllowMutations:  true,
		BoundHost:       "127.0.0.1",
		EnableTestHooks: true,
		ConfigPath:      configPath,
	})
	webServer := httptest.NewServer(webHandler)
	webHandler.options.AllowedOrigin = webServer.URL
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
	if !strings.Contains(dom, `data-mutation-enabled="true"`) {
		t.Fatalf("browser DOM should report backend-supported sync session:\n%s", dom)
	}
	if !strings.Contains(dom, `id="config-panel"`) || !strings.Contains(dom, "Configuration") || !strings.Contains(dom, "OPNSense / Unbound") || !strings.Contains(dom, "API Key Set") {
		t.Fatalf("browser DOM should render sanitized configuration summary:\n%s", dom)
	}
	if !strings.Contains(dom, "Save target: "+configPath) ||
		!strings.Contains(dom, "Set OPNSense") ||
		!strings.Contains(dom, "Test OPNSense") ||
		!strings.Contains(dom, "Test Caddy") ||
		!strings.Contains(dom, "Defaults") {
		t.Fatalf("browser DOM should render config source, save target, set buttons, and test buttons:\n%s", dom)
	}
	if strings.Contains(dom, "fixture-browser-key") || strings.Contains(dom, "fixture-browser-secret") {
		t.Fatalf("browser DOM leaked sensitive config values:\n%s", dom)
	}
	if !strings.Contains(dom, `data-loading="false"`) ||
		!strings.Contains(dom, `id="top-progress"`) ||
		!strings.Contains(dom, `id="top-progress-title"`) ||
		!strings.Contains(dom, `class="progress-track"`) {
		t.Fatalf("browser DOM should expose completed loading state and clear progress bar structure:\n%s", dom)
	}
	if !strings.Contains(dom, `class="service-card`) || !strings.Contains(dom, `Cloudflare</span>`) {
		t.Fatalf("browser DOM should render service health cards:\n%s", dom)
	}
	if !strings.Contains(dom, `class="row-preview"`) || !strings.Contains(dom, `Not routed`) {
		t.Fatalf("browser DOM should render row preview controls and Cloudflare route status:\n%s", dom)
	}
	if !strings.Contains(dom, `id="sync-now"`) || !strings.Contains(dom, `class="row-sync"`) {
		t.Fatalf("browser DOM should expose disabled global and row sync buttons:\n%s", dom)
	}
	if !strings.Contains(dom, `dns-result bad`) {
		t.Fatalf("browser DOM should color failed DNS resolution as bad:\n%s", dom)
	}

	loadingDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=holdloading", 1280, 900)
	if !strings.Contains(loadingDOM, `data-loading="true"`) ||
		!strings.Contains(loadingDOM, "Loading service status") ||
		!strings.Contains(loadingDOM, "Reading Caddy, DNS targets, and runtime config") {
		t.Fatalf("loading DOM should keep a visible labeled loading bar for long refreshes:\n%s", loadingDOM)
	}

	configTestDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=testconfig:unbound", 1280, 900)
	if !strings.Contains(configTestDOM, "Connected to OPNSense Unbound API.") ||
		!strings.Contains(configTestDOM, `id="config-test-unbound"`) {
		t.Fatalf("config test should call backend and render result:\n%s", configTestDOM)
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
	if !strings.Contains(previewDOM, `data-sync-enabled="true"`) {
		t.Fatalf("sync button should be enabled after backend-issued planned actions:\n%s", previewDOM)
	}
	if !strings.Contains(previewDOM, `id="sync-progress-title"`) || !strings.Contains(previewDOM, `id="sync-progress-detail"`) {
		t.Fatalf("preview DOM should include labeled sync progress structure:\n%s", previewDOM)
	}

	dryRunDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=preview:unbound,dryrun", 1280, 900)
	if !strings.Contains(dryRunDOM, "All operations completed successfully") || !strings.Contains(dryRunDOM, "added=2") {
		t.Fatalf("dry-run result was not rendered:\n%s", dryRunDOM)
	}

	syncDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=preview:unbound,sync", 1280, 900)
	if !strings.Contains(syncDOM, "All operations completed successfully") || !strings.Contains(syncDOM, "added=2") {
		t.Fatalf("backend-backed sync result was not rendered:\n%s", syncDOM)
	}

	rowPreviewDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=rowpreview:browser.example.test:unbound", 1280, 900)
	if !strings.Contains(rowPreviewDOM, "ADD unbound browser.example.test") || strings.Contains(rowPreviewDOM, "ADD unbound hidden.example.test") {
		t.Fatalf("row preview should render only the selected hostname action:\n%s", rowPreviewDOM)
	}

	closedConfigDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=toggleconfig:closed", 1280, 900)
	if !strings.Contains(closedConfigDOM, `id="config-panel" class="config-summary panel"`) || strings.Contains(closedConfigDOM, `id="config-panel" class="config-summary panel" open`) {
		t.Fatalf("config panel should be closable:\n%s", closedConfigDOM)
	}

	mobileDOM := runChromeSmoke(t, chromePath, webServer.URL, 390, 844)
	if !strings.Contains(mobileDOM, `data-mobile="true"`) {
		t.Fatalf("mobile DOM did not mark mobile layout:\n%s", mobileDOM)
	}
	if !strings.Contains(mobileDOM, `data-table-scrolls="false"`) || !strings.Contains(mobileDOM, `id="host-inspector"`) {
		t.Fatalf("mobile DOM should avoid horizontal table scrolling and render the inspector:\n%s", mobileDOM)
	}

	configSaveDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=setconfig:unbound", 1280, 900)
	if !strings.Contains(configSaveDOM, "Saved unbound config.") || !strings.Contains(configSaveDOM, "https://saved.example.test") {
		t.Fatalf("config save should update the browser summary:\n%s", configSaveDOM)
	}
	savedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected browser config save to write %s: %v", configPath, err)
	}
	var savedConfig config.ExtendedConfig
	if err := json.Unmarshal(savedData, &savedConfig); err != nil {
		t.Fatalf("failed to decode browser-saved config: %v", err)
	}
	if savedConfig.BaseURL != "https://saved.example.test" || savedConfig.APIKey != "saved-key" || savedConfig.APISecret != "fixture-browser-secret" {
		t.Fatalf("unexpected browser-saved OPNSense config: %#v", savedConfig.Config)
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
