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
	"sync/atomic"
	"testing"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/app"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
)

func TestBrowserSmokeWithFakeData(t *testing.T) {
	// CADDY_DNS_SYNC_BROWSER_TESTS is the preferred flag; UNBOUNDCLI_BROWSER_TESTS
	// remains supported as a deprecated alias.
	if os.Getenv("CADDY_DNS_SYNC_BROWSER_TESTS") != "1" && os.Getenv("UNBOUNDCLI_BROWSER_TESTS") != "1" {
		t.Skip("set CADDY_DNS_SYNC_BROWSER_TESTS=1 to run browser smoke checks")
	}
	for _, name := range []string{
		config.EnvAPIKey, config.EnvAPISecret, config.EnvBaseURL, config.EnvInsecure,
		config.EnvAPIKeyDeprecated, config.EnvAPISecretDeprecated, config.EnvBaseURLDeprecated, config.EnvInsecureDeprecated,
	} {
		t.Setenv(name, "")
	}

	chromePath := chromeHeadlessShellPath(t)
	fixtureHostnames := []string{
		"browser.example.test", "hidden.example.test", "media.example.test", "notes.example.test",
		"photos.example.test", "status.example.test", "vault.example.test", "wiki.example.test",
	}
	caddy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config/" {
			t.Fatalf("unexpected Caddy path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		routes := make([]map[string]any, 0, len(fixtureHostnames))
		for index, hostname := range fixtureHostnames {
			routes = append(routes, map[string]any{
				"match":  []map[string][]string{{"host": {hostname}}},
				"handle": []map[string]any{{"handler": "reverse_proxy", "upstreams": []map[string]string{{"dial": fmt.Sprintf("10.0.0.%d:8080", index+5)}}}},
			})
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"apps": map[string]any{"http": map[string]any{"servers": map[string]any{"srv0": map[string]any{"routes": routes}}}}}); err != nil {
			t.Fatalf("encode Caddy fixture: %v", err)
		}
	}))
	defer caddy.Close()

	var unboundWrites atomic.Int32
	opnsense := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/unbound/settings/searchHostOverride":
			fmt.Fprint(w, `{"rows":[]}`)
		case "/api/unbound/settings/addHostOverride":
			unboundWrites.Add(1)
			fmt.Fprint(w, `{"result":"saved","uuid":"new-uuid"}`)
		case "/api/unbound/service/reconfigure":
			fmt.Fprint(w, `{"result":"saved"}`)
		case "/api/core/firmware/backup":
			fmt.Fprint(w, `{"status":"ok"}`)
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

	dom := runChromeSmoke(t, chromePath, webServer.URL, 1440, 1000)
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
	if !strings.Contains(dom, `id="config-panel"`) || !strings.Contains(dom, "Configuration") {
		t.Fatalf("browser DOM should render the configuration dialog:\n%s", dom)
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
	if !strings.Contains(dom, `class="service-progress-chips"`) || !strings.Contains(dom, `Cloudflare —`) {
		t.Fatalf("browser DOM should render service health progress:\n%s", dom)
	}
	if !strings.Contains(dom, `data-hostname="browser.example.test"`) || !strings.Contains(dom, `Not routed`) {
		t.Fatalf("browser DOM should render hostname rows and Cloudflare route status:\n%s", dom)
	}
	if !strings.Contains(dom, `data-visible-hostname-rows="8"`) {
		t.Fatalf("desktop viewport should show eight hostname rows:\n%s", dom)
	}
	if !strings.Contains(dom, `class="btn-primary btn-sm toolbar-sync-all"`) || !strings.Contains(dom, `class="row-sync-btn"`) {
		t.Fatalf("browser DOM should expose global and row sync buttons:\n%s", dom)
	}
	if !strings.Contains(dom, `dns-result bad`) {
		t.Fatalf("browser DOM should color failed DNS resolution as bad:\n%s", dom)
	}

	loadingDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=holdloading", 1280, 900)
	if !strings.Contains(loadingDOM, `data-loading="true"`) ||
		!strings.Contains(loadingDOM, `id="top-progress-title"`) ||
		!strings.Contains(loadingDOM, "Loading service status...") ||
		!strings.Contains(loadingDOM, `class="topbar-activity"`) {
		t.Fatalf("loading DOM should keep a visible labeled loading bar for long refreshes:\n%s", loadingDOM)
	}

	configTestDOM := runChromeSmoke(t, chromePath, webServer.URL+"?configtab=unbound&e2e=toggleconfig:open,testconfig:unbound", 1280, 900)
	if !strings.Contains(configTestDOM, "Connected to OPNSense Unbound API.") ||
		!strings.Contains(configTestDOM, `data-config-editor="unbound"`) ||
		!strings.Contains(configTestDOM, "OPNSense / Unbound") ||
		!strings.Contains(configTestDOM, "Test OPNSense") ||
		!strings.Contains(configTestDOM, "Save OPNSense") {
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
	if !strings.Contains(previewDOM, `data-dry-run-enabled="true"`) {
		t.Fatalf("dry-run button should be enabled after planned actions:\n%s", previewDOM)
	}
	if !strings.Contains(previewDOM, `data-sync-enabled="true"`) {
		t.Fatalf("sync button should be enabled after backend-issued planned actions:\n%s", previewDOM)
	}

	_ = runChromeSmoke(t, chromePath, webServer.URL+"?e2e=preview:unbound,dryrun", 1280, 900)
	if writes := unboundWrites.Load(); writes != 0 {
		t.Fatalf("dry-run made %d Unbound writes", writes)
	}

	_ = runChromeSmoke(t, chromePath, webServer.URL+"?e2e=preview:unbound,sync", 1280, 900)
	if writes := unboundWrites.Load(); writes != int32(len(fixtureHostnames)) {
		t.Fatalf("sync made %d Unbound writes, want %d", writes, len(fixtureHostnames))
	}

	rowPreviewDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=rowpreview:browser.example.test:unbound", 1280, 900)
	if !strings.Contains(rowPreviewDOM, `data-dry-run-enabled="true"`) || !strings.Contains(rowPreviewDOM, `aria-selected="true"`) {
		t.Fatalf("row preview should select the hostname and issue a plan:\n%s", rowPreviewDOM)
	}

	closedConfigDOM := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=toggleconfig:closed", 1280, 900)
	if !strings.Contains(closedConfigDOM, `id="config-panel" class="config-modal " hidden`) {
		t.Fatalf("config modal should be closable and hidden:\n%s", closedConfigDOM)
	}

	mobileDOM := runChromeSmoke(t, chromePath, webServer.URL, 390, 844)
	if !strings.Contains(mobileDOM, `data-mobile="true"`) {
		t.Fatalf("mobile DOM did not mark mobile layout:\n%s", mobileDOM)
	}
	if !strings.Contains(mobileDOM, `data-table-scrolls="false"`) || !strings.Contains(mobileDOM, `id="entries-panel"`) {
		t.Fatalf("mobile DOM should avoid horizontal table scrolling and render hostname entries:\n%s", mobileDOM)
	}
	if !strings.Contains(mobileDOM, `data-search-visible="true"`) {
		t.Fatal("mobile viewport should show search without scrolling")
	}
	if !strings.Contains(mobileDOM, `data-first-hostname-visible="true"`) {
		t.Fatal("mobile viewport should show the first hostname without scrolling")
	}
	if !strings.Contains(mobileDOM, `tabindex="0"`) || !strings.Contains(mobileDOM, `class="row-sync-btn"`) {
		t.Fatalf("entry rows should be keyboard-selectable:\n%s", mobileDOM)
	}
	if !strings.Contains(mobileDOM, `data-label="Hostname"`) || !strings.Contains(mobileDOM, `data-label="Cloudflare route"`) {
		t.Fatalf("mobile table cells should expose labels when headers are hidden:\n%s", mobileDOM)
	}

	configSaveDOM := runChromeSmoke(t, chromePath, webServer.URL+"?configtab=unbound&e2e=toggleconfig:open,setconfig:unbound", 1280, 900)
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
	// The browser fixture starts with no config file. A blank secret must not
	// copy the runtime-only secret into this new file.
	if savedConfig.BaseURL != "https://saved.example.test" || savedConfig.APIKey != "saved-key" || savedConfig.APISecret != "" {
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
