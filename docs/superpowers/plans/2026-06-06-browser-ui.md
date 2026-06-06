# Browser UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Build a tested browser UI for `go run main.go web` that provides status visibility, sync preview, and safe apply workflows over the existing runtime/status/syncplan layers.

**Architecture:** Keep the backend in `internal/web` and keep the first production UI dependency-free: Go `embed` for static assets, vanilla JavaScript modules for browser behavior, and existing Go services for data. Real mutation stays disabled until a local session token, CSRF-style checks, loopback binding constraints, and server-validated plan/action IDs are implemented and tested.

**Tech Stack:** Go stdlib HTTP server, `embed`, `testing`/`httptest`, existing `internal/status` and `internal/syncplan`, vanilla HTML/CSS/JS, Chrome Headless Shell browser smoke tests.

**Implementation review adjustments:** The final implementation intentionally differs from some early snippets below. Startup output prints only the local URL, because mutation tokens are inactive until real apply support exists. Browser E2E query hooks are enabled only by a test-only server option. The web API returns stable action IDs, rejects wildcard hosts for future mutation mode, and filters the combined "all" plan to runtime-enabled DNS targets.

---

## File Structure

- Modify: `cmd/web.go` to print the web URL and later a local session token when mutation is enabled.
- Modify: `internal/web/server.go` to route static assets, API routes, health checks, plan-service validation, and later token/plan validation.
- Delete: `internal/web/assets.go`; move embedded asset loading to `internal/web/server.go`.
- Create: `internal/web/static/index.html` for the app shell.
- Create: `internal/web/static/app.js` for API calls, state, rendering, filtering, and sync preview interactions.
- Create: `internal/web/static/styles.css` for the browser UI.
- Create: `internal/web/static/icons.svg` only if repeated inline status icons become unwieldy; prefer text/status chips first.
- Modify: `internal/web/server_test.go` for API, static asset, safety, and route tests.
- Modify: `internal/web/browser_test.go` for end-to-end DOM assertions and screenshots with fake Caddy data.
- Modify: `internal/status/loader.go` only when web needs new status fields that should also benefit TUI/CLI.
- Modify: `internal/syncplan/plan.go` only when web needs shared planning behavior.

## Task 1: Static Asset Split

**Files:**
- Modify: `internal/web/server.go`
- Delete: `internal/web/assets.go`
- Create: `internal/web/static/index.html`
- Create: `internal/web/static/app.js`
- Create: `internal/web/static/styles.css`
- Test: `internal/web/server_test.go`

- [x] **Step 1: Write the failing static asset test**

Add this test to `internal/web/server_test.go`:

```go
func TestStaticAssetsAreServedWithExpectedContentTypes(t *testing.T) {
	server := NewServer(&app.Runtime{})

	tests := []struct {
		path        string
		contentType string
		contains    []byte
	}{
		{path: "/", contentType: "text/html; charset=utf-8", contains: []byte(`<div id="app"`)},
		{path: "/static/app.js", contentType: "text/javascript; charset=utf-8", contains: []byte("async function refreshEntries")},
		{path: "/static/styles.css", contentType: "text/css; charset=utf-8", contains: []byte(".status-chip")},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tt.path, rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Content-Type"); got != tt.contentType {
			t.Fatalf("%s: expected content type %q, got %q", tt.path, tt.contentType, got)
		}
		if !bytes.Contains(rec.Body.Bytes(), tt.contains) {
			t.Fatalf("%s: response missing %q", tt.path, tt.contains)
		}
	}
}
```

- [x] **Step 2: Run the test and verify it fails**

Run:

```bash
rtk go test ./internal/web -run TestStaticAssetsAreServedWithExpectedContentTypes -count=1
```

Expected: FAIL because `/static/app.js` and `/static/styles.css` are not routed.

- [x] **Step 3: Implement embedded static assets**

Create `internal/web/static/index.html`:

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Caddy DNS Sync</title>
  <link rel="stylesheet" href="/static/styles.css">
</head>
<body>
  <div id="app">
    <header class="topbar">
      <h1>Caddy DNS Sync</h1>
      <div id="runtime" class="runtime">Loading runtime...</div>
    </header>
    <main>
      <section id="summary" class="summary" aria-live="polite"></section>
      <section class="toolbar">
        <button id="refresh" type="button">Refresh</button>
        <select id="status-filter" aria-label="Status filter">
          <option value="all">All entries</option>
          <option value="out_of_sync">Out of sync</option>
          <option value="caddy_only">Caddy only</option>
          <option value="stale">Stale</option>
        </select>
        <select id="service-filter" aria-label="Service filter">
          <option value="all">All services</option>
          <option value="unbound">Unbound</option>
          <option value="adguard">AdGuard</option>
          <option value="cloudflare">Cloudflare</option>
        </select>
        <label class="field-label" for="search">Search</label>
        <input id="search" type="search" aria-label="Search hostnames" placeholder="Search hostnames">
      </section>
      <section id="entries-panel" class="panel">
        <table>
          <thead>
            <tr>
              <th>Hostname</th><th>Source</th><th>Caddy</th><th>DNS</th><th>Unbound</th><th>AdGuard</th><th>Cloudflare</th><th>Status</th>
            </tr>
          </thead>
          <tbody id="entries"></tbody>
        </table>
      </section>
      <section id="sync-panel" class="panel">
        <div class="sync-toolbar">
          <select id="sync-service" aria-label="Sync target">
            <option value="all">Unbound + AdGuard</option>
            <option value="unbound">Unbound</option>
            <option value="adguard">AdGuard</option>
            <option value="dhcp">DHCP preview</option>
          </select>
          <button id="preview-sync" type="button">Preview sync</button>
          <button id="dry-run-sync" type="button" disabled>Dry-run apply</button>
        </div>
        <div id="sync-log" class="log" role="status" aria-live="polite">No actions planned.</div>
      </section>
    </main>
  </div>
  <script src="/static/app.js"></script>
</body>
</html>
```

Create `internal/web/static/app.js` with a minimal boot path:

```javascript
let entries = [];
let plannedActions = [];

const el = (id) => document.getElementById(id);

async function getJSON(path) {
  const response = await fetch(path);
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error || response.statusText);
  }
  return data;
}

function escapeHTML(value) {
  return String(value || '').replace(/[&<>"']/g, (char) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
  }[char]));
}

function statusClass(label) {
  const normalized = String(label || '').toLowerCase();
  if (normalized.includes('stale') || normalized.includes('out')) return 'bad';
  if (normalized.includes('sync')) return 'ok';
  return 'warn';
}

function renderSummary() {
  const total = entries.length;
  const out = entries.filter((entry) => String(entry.status_label || '').toLowerCase().includes('out')).length;
  el('summary').innerHTML = [
    `<span class="metric"><strong>${total}</strong> entries</span>`,
    `<span class="metric"><strong>${out}</strong> out of sync</span>`
  ].join('');
}

function renderEntries() {
  el('entries').innerHTML = entries.map((entry) => `
    <tr>
      <td>${escapeHTML(entry.hostname)}</td>
      <td>${escapeHTML(entry.data_source || '-')}</td>
      <td>${escapeHTML(entry.caddy_upstream || '-')}</td>
      <td>${escapeHTML(entry.dns_resolved || '-')}</td>
      <td>${escapeHTML(entry.unbound_status?.ip || '-')}</td>
      <td>${escapeHTML(entry.adguard_status?.ip || '-')}</td>
      <td>${entry.cloudflare_status?.configured ? 'Configured' : '-'}</td>
      <td><span class="status-chip ${statusClass(entry.status_label)}">${escapeHTML(entry.status_label || 'Unknown')}</span></td>
    </tr>
  `).join('');
}

async function refreshEntries() {
  el('runtime').textContent = 'Loading...';
  const config = await getJSON('/api/config');
  el('runtime').textContent = `Caddy ${config.caddy.server_ip}:${config.caddy.server_port}`;
  const data = await getJSON('/api/entries');
  entries = data.entries || [];
  renderSummary();
  renderEntries();
}

window.addEventListener('DOMContentLoaded', () => {
  el('refresh').addEventListener('click', () => refreshEntries().catch((err) => {
    el('runtime').textContent = err.message;
  }));
  refreshEntries().catch((err) => {
    el('runtime').textContent = err.message;
  });
});
```

Create `internal/web/static/styles.css`:

```css
:root {
  color-scheme: dark;
  --bg: #0f1218;
  --panel: #171b24;
  --panel-2: #202635;
  --text: #e8edf6;
  --muted: #94a3b8;
  --border: #303848;
  --ok: #8fd694;
  --warn: #ffd166;
  --bad: #ff6b6b;
}
* { box-sizing: border-box; }
body { margin: 0; background: var(--bg); color: var(--text); font: 14px/1.45 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
.topbar { display: flex; align-items: center; justify-content: space-between; padding: 16px 20px; border-bottom: 1px solid var(--border); background: #121620; }
h1 { margin: 0; font-size: 18px; letter-spacing: 0; }
main { padding: 18px 20px; display: grid; gap: 14px; }
.summary, .toolbar, .sync-toolbar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.panel { border: 1px solid var(--border); background: var(--panel); border-radius: 8px; overflow: auto; }
.metric, button, select, input { border: 1px solid var(--border); border-radius: 6px; background: var(--panel-2); color: var(--text); padding: 7px 10px; font: inherit; }
button:disabled { opacity: .55; }
table { width: 100%; border-collapse: collapse; min-width: 860px; }
th, td { padding: 10px 12px; border-bottom: 1px solid var(--border); text-align: left; white-space: nowrap; }
th { color: var(--muted); font-size: 12px; text-transform: uppercase; background: #151923; }
.status-chip { display: inline-block; border-radius: 999px; padding: 2px 8px; border: 1px solid var(--border); }
.status-chip.ok { color: var(--ok); }
.status-chip.warn { color: var(--warn); }
.status-chip.bad { color: var(--bad); }
.runtime, .log { color: var(--muted); }
.log { padding: 12px; min-height: 44px; border-top: 1px solid var(--border); white-space: pre-wrap; }
@media (max-width: 760px) {
  .topbar { align-items: flex-start; flex-direction: column; gap: 8px; }
  main { padding: 12px; }
}
```

Modify `internal/web/server.go`:

```go
import (
	"embed"
	"io/fs"
)

//go:embed static/*
var staticFiles embed.FS

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleIndex)
	staticRoot, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticRoot))))
	s.mux.HandleFunc("/api/config", s.handleConfig)
	s.mux.HandleFunc("/api/entries", s.handleEntries)
	s.mux.HandleFunc("/api/sync/plan", s.handlePlan)
	s.mux.HandleFunc("/api/sync/apply", s.handleApply)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFileFS(w, r, staticFiles, "static/index.html")
}
```

- [x] **Step 4: Run targeted tests**

Run:

```bash
rtk go test ./internal/web -run TestStaticAssetsAreServedWithExpectedContentTypes -count=1
```

Expected: PASS.

- [x] **Step 5: Commit**

Run:

```bash
git add internal/web
git commit -m "refactor: split web UI static assets"
```

## Task 2: Status Dashboard Interactions

**Files:**
- Modify: `internal/web/static/app.js`
- Modify: `internal/web/static/styles.css`
- Test: `internal/web/browser_test.go`

- [x] **Step 1: Write the browser interaction test**

Extend `internal/web/browser_test.go` with a second fake Caddy route and an interaction URL. The test must assert behavior, not just static control IDs:

```go
dom := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=filter:caddy_only,search:browser", 1280, 900)
if !strings.Contains(dom, `data-e2e="done"`) {
	t.Fatalf("browser e2e hook did not complete:\n%s", dom)
}
if !strings.Contains(dom, "browser.example.test") {
	t.Fatalf("filtered DOM missing browser.example.test:\n%s", dom)
}
if strings.Contains(dom, "hidden.example.test") {
	t.Fatalf("filtered DOM should hide hidden.example.test:\n%s", dom)
}
if !strings.Contains(dom, `status-chip warn`) && !strings.Contains(dom, `status-chip bad`) {
	t.Fatalf("filtered DOM missing status chip class:\n%s", dom)
}
```

- [x] **Step 2: Run and verify failure or current gap**

Run:

```bash
rtk env UNBOUNDCLI_BROWSER_TESTS=1 go test ./internal/web -run TestBrowserSmokeWithFakeData -count=1
```

Expected: FAIL until the app supports filter/search interaction and marks the e2e hook complete. If Chrome Headless Shell is not auto-discovered, set `CHROME_HEADLESS_SHELL` to the validated local binary and rerun the same command.

- [x] **Step 3: Implement filtering**

In `internal/web/static/app.js`, add:

```javascript
function matchesStatus(entry, filter) {
  const label = String(entry.status_label || '').toLowerCase();
  if (filter === 'all') return true;
  if (filter === 'out_of_sync') return label.includes('out');
  if (filter === 'caddy_only') return label.includes('caddy');
  if (filter === 'stale') return label.includes('stale');
  return true;
}

function matchesService(entry, filter) {
  if (filter === 'all') return true;
  if (filter === 'unbound') return Boolean(entry.unbound_status?.configured);
  if (filter === 'adguard') return Boolean(entry.adguard_status?.configured);
  if (filter === 'cloudflare') return Boolean(entry.cloudflare_status?.configured);
  return true;
}

function filteredEntries() {
  const status = el('status-filter').value;
  const service = el('service-filter').value;
  const query = el('search').value.trim().toLowerCase();
  return entries.filter((entry) =>
    matchesStatus(entry, status) &&
    matchesService(entry, service) &&
    (!query || String(entry.hostname || '').toLowerCase().includes(query))
  );
}
```

Update `renderEntries()` and `renderSummary()` to use `filteredEntries()`.

- [x] **Step 4: Wire controls**

In the `DOMContentLoaded` handler, add:

```javascript
['status-filter', 'service-filter', 'search'].forEach((id) => {
  el(id).addEventListener('input', () => {
    renderSummary();
    renderEntries();
  });
});
```

- [x] **Step 5: Add the browser e2e hook**

Add this helper to `internal/web/static/app.js` and call it at the end of `refreshEntries()`:

```javascript
async function runE2EActions() {
  const params = new URLSearchParams(window.location.search);
  const script = params.get('e2e');
  if (!script) return;
  for (const action of script.split(',')) {
    const [name, value] = action.split(':');
    if (name === 'filter') {
      el('status-filter').value = value;
      el('status-filter').dispatchEvent(new Event('input', {bubbles: true}));
    }
    if (name === 'search') {
      el('search').value = value;
      el('search').dispatchEvent(new Event('input', {bubbles: true}));
    }
  }
  el('app').setAttribute('data-e2e', 'done');
}
```

At the end of `refreshEntries()`:

```javascript
await runE2EActions();
```

- [x] **Step 6: Run browser test**

Run:

```bash
rtk env UNBOUNDCLI_BROWSER_TESTS=1 go test ./internal/web -run TestBrowserSmokeWithFakeData -count=1
```

Expected: PASS.

- [x] **Step 7: Commit**

Run:

```bash
git add internal/web
git commit -m "feat: add browser status dashboard filters"
```

## Task 3: Sync Preview UI

**Files:**
- Modify: `internal/web/static/app.js`
- Modify: `internal/web/static/styles.css`
- Modify: `internal/web/server.go`
- Test: `internal/web/server_test.go`
- Test: `internal/web/browser_test.go`

- [x] **Step 1: Add API tests for plan service selection and validation**

Add to `internal/web/server_test.go`:

```go
func TestPlanRouteSupportsServiceSelection(t *testing.T) {
	caddy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config/" {
			t.Fatalf("unexpected Caddy path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"apps":{"http":{"servers":{"srv0":{"routes":[{"match":[{"host":["plan.example.test"]}],"handle":[{"handler":"reverse_proxy","upstreams":[{"dial":"10.0.0.5:8080"}]}]}]}}}}}`)
	}))
	defer caddy.Close()

	host, port := splitWebTestServerHostPort(t, caddy.URL)
	server := NewServer(&app.Runtime{
		CaddyEndpoint: app.CaddyEndpoint{ServerIP: host, ServerPort: port},
		Clients: app.ClientSet{Caddy: api.NewCaddyClient(host, port)},
	})

	planResp := getJSON[PlanResponse](t, server, "/api/sync/plan?service=adguard")
	if len(planResp.Actions) != 1 {
		t.Fatalf("expected one action, got %#v", planResp.Actions)
	}
	if planResp.Actions[0].Service != "adguard" {
		t.Fatalf("expected adguard action, got %#v", planResp.Actions[0])
	}
}

func TestPlanRouteRejectsUnknownService(t *testing.T) {
	server := NewServer(&app.Runtime{})
	req := httptest.NewRequest(http.MethodGet, "/api/sync/plan?service=bogus", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
```

- [x] **Step 2: Run and verify current API behavior**

Run:

```bash
rtk go test ./internal/web -run 'TestPlanRoute(SupportsServiceSelection|RejectsUnknownService)' -count=1
```

Expected: one test may already pass, and `TestPlanRouteRejectsUnknownService` should fail until service validation is implemented.

- [x] **Step 3: Implement service validation**

Add to `internal/web/server.go`:

```go
func validPlanService(service string) bool {
	switch service {
	case "", "all", "unbound", "adguard", "dhcp":
		return true
	default:
		return false
	}
}
```

In `handlePlan`, before loading entries:

```go
service := r.URL.Query().Get("service")
if !validPlanService(service) {
	writeError(w, http.StatusBadRequest, fmt.Errorf("invalid sync service %q", service))
	return
}
```

- [x] **Step 4: Implement sync preview rendering**

Add to `internal/web/static/app.js`:

```javascript
function renderActions(actions) {
  if (!actions.length) {
    el('sync-log').textContent = 'No actions needed.';
    el('dry-run-sync').disabled = true;
    return;
  }
  const lines = actions.map((action) => {
    const target = action.new_ip ? ` -> ${action.new_ip}` : '';
    return `${action.type.toUpperCase()} ${action.service} ${action.hostname}${target}`;
  });
  el('sync-log').textContent = lines.join('\n');
  el('dry-run-sync').disabled = false;
}

async function previewSync() {
  const service = el('sync-service').value;
  if (service === 'dhcp') {
    plannedActions = [];
    el('sync-log').textContent = 'DHCP apply is not implemented; preview only.';
    el('dry-run-sync').disabled = true;
    return;
  }
  const data = await getJSON(`/api/sync/plan?service=${encodeURIComponent(service)}`);
  plannedActions = data.actions || [];
  renderActions(plannedActions);
}
```

Wire the button:

```javascript
el('preview-sync').addEventListener('click', () => previewSync().catch((err) => {
  el('sync-log').textContent = err.message;
}));
```

- [x] **Step 5: Add browser interaction assertion for preview**

In `internal/web/browser_test.go`, use the e2e hook to click preview and assert the rendered log:

```go
dom := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=preview:adguard", 1280, 900)
if !strings.Contains(dom, `data-e2e="done"`) {
	t.Fatalf("preview e2e hook did not complete:\n%s", dom)
}
if !strings.Contains(dom, "ADD adguard browser.example.test") {
	t.Fatalf("preview did not render expected action:\n%s", dom)
}
if strings.Contains(dom, `id="dry-run-sync" disabled`) {
	t.Fatalf("dry-run button should be enabled after planned actions:\n%s", dom)
}
```

Extend `runE2EActions()` in `internal/web/static/app.js`:

```javascript
if (name === 'preview') {
  el('sync-service').value = value;
  await previewSync();
}
```

- [x] **Step 6: Run tests**

Run:

```bash
rtk go test ./internal/web -count=1
rtk env UNBOUNDCLI_BROWSER_TESTS=1 go test ./internal/web -run TestBrowserSmokeWithFakeData -count=1
```

Expected: PASS.

- [x] **Step 7: Commit**

Run:

```bash
git add internal/web
git commit -m "feat: add web sync preview"
```

## Task 4: Dry-Run Apply UI

**Files:**
- Modify: `internal/web/static/app.js`
- Modify: `internal/web/server.go`
- Test: `internal/web/server_test.go`
- Test: `internal/web/browser_test.go`

- [x] **Step 1: Add server test for dry-run apply response**

Add to `internal/web/server_test.go`:

```go
func TestApplyRouteAllowsDryRunOnly(t *testing.T) {
	server := NewServer(&app.Runtime{})
	action := syncplan.Action{
		Type: "add", Service: "unbound", Hostname: "dryrun.example.test", NewIP: "192.168.1.15", Enabled: true,
	}
	body, err := json.Marshal(ApplyRequest{DryRun: true, Actions: []syncplan.Action{action}})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body, err = json.Marshal(ApplyRequest{DryRun: false, Actions: []syncplan.Action{action}})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestApplyRouteRejectsUnsupportedDHCPDryRun(t *testing.T) {
	server := NewServer(&app.Runtime{})
	action := syncplan.Action{
		Type: "add", Service: "dhcp", Hostname: "dhcp.example.test", NewIP: "192.168.1.55", Enabled: true,
	}
	body, err := json.Marshal(ApplyRequest{DryRun: true, Actions: []syncplan.Action{action}})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported DHCP apply, got %d: %s", rec.Code, rec.Body.String())
	}
}
```

- [x] **Step 2: Run test**

Run:

```bash
rtk go test ./internal/web -run 'TestApplyRoute(AllowsDryRunOnly|RejectsUnsupportedDHCPDryRun)' -count=1
```

Expected: `TestApplyRouteRejectsUnsupportedDHCPDryRun` fails until apply validation rejects DHCP.

- [x] **Step 3: Reject unsupported apply services**

Add to `internal/web/server.go`:

```go
func validateApplyActions(actions []syncplan.Action) error {
	for _, action := range actions {
		switch action.Service {
		case "unbound", "adguard":
			continue
		case "dhcp":
			return fmt.Errorf("DHCP apply is not implemented")
		default:
			return fmt.Errorf("invalid sync service %q", action.Service)
		}
	}
	return nil
}
```

In `handleApply`, after decoding `request`:

```go
if err := validateApplyActions(request.Actions); err != nil {
	writeError(w, http.StatusBadRequest, err)
	return
}
```

- [x] **Step 4: Implement dry-run button**

Add to `internal/web/static/app.js`:

```javascript
async function dryRunSync() {
  if (el('sync-service').value === 'dhcp') {
    el('sync-log').textContent = 'DHCP apply is not implemented.';
    return;
  }
  if (!plannedActions.length) {
    el('sync-log').textContent = 'Preview sync before running a dry run.';
    return;
  }
  const response = await fetch('/api/sync/apply', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({dry_run: true, actions: plannedActions})
  });
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.error || response.statusText);
  }
  const result = data.result;
  el('sync-log').textContent = `${result.message}\nadded=${result.items_added} updated=${result.items_updated} deleted=${result.items_deleted}`;
}
```

Wire the button:

```javascript
el('dry-run-sync').addEventListener('click', () => dryRunSync().catch((err) => {
  el('sync-log').textContent = err.message;
}));
```

- [x] **Step 5: Add browser interaction assertion for dry-run**

In `internal/web/browser_test.go`, assert the e2e dry-run path updates the log:

```go
dom := runChromeSmoke(t, chromePath, webServer.URL+"?e2e=preview:unbound,dryrun", 1280, 900)
if !strings.Contains(dom, `data-e2e="done"`) {
	t.Fatalf("dry-run e2e hook did not complete:\n%s", dom)
}
if !strings.Contains(dom, "All operations completed successfully") || !strings.Contains(dom, "added=1") {
	t.Fatalf("dry-run result was not rendered:\n%s", dom)
}
```

Extend `runE2EActions()`:

```javascript
if (name === 'dryrun') {
  await dryRunSync();
}
```

- [x] **Step 6: Run tests**

Run:

```bash
rtk go test ./internal/web -count=1
rtk env UNBOUNDCLI_BROWSER_TESTS=1 go test ./internal/web -run TestBrowserSmokeWithFakeData -count=1
```

Expected: PASS.

- [x] **Step 7: Commit**

Run:

```bash
git add internal/web
git commit -m "feat: add web dry-run sync flow"
```

## Task 5: Local Session Token and Server-Validated Apply Plumbing

**Files:**
- Modify: `cmd/web.go`
- Modify: `internal/web/server.go`
- Modify: `internal/web/server_test.go`
- Create: `cmd/web_test.go`

- [x] **Step 1: Add tests for token, origin, and forged-action rejection**

Add to `internal/web/server_test.go`:

```go
func TestMutatingApplyRequiresTokenAndAllowedOrigin(t *testing.T) {
	server := NewServerWithOptions(&app.Runtime{}, Options{
		ApplyToken:     "test-token",
		AllowMutations: true,
		AllowedOrigin:  "http://127.0.0.1:8080",
		BoundHost:      "127.0.0.1",
	})
	body, err := json.Marshal(ApplyRequest{
		DryRun: false,
		PlanID: "missing-plan",
		ActionIDs: []string{"action-1"},
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected missing token to be forbidden, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-UnboundCLI-Token", "test-token")
	req.Header.Set("Origin", "http://evil.example")
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected disallowed origin to be forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMutatingApplyRejectsPostedActionsEvenWithToken(t *testing.T) {
	server := NewServerWithOptions(&app.Runtime{}, Options{
		ApplyToken:     "test-token",
		AllowMutations: true,
		AllowedOrigin:  "http://127.0.0.1:8080",
		BoundHost:      "127.0.0.1",
	})
	body, err := json.Marshal(map[string]any{
		"dry_run": false,
		"actions": []syncplan.Action{
			{Type: "add", Service: "unbound", Hostname: "forged.example.test", NewIP: "192.168.1.15", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal forged request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sync/apply", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-UnboundCLI-Token", "test-token")
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected forged action request to be rejected, got %d: %s", rec.Code, rec.Body.String())
	}
}
```

- [x] **Step 2: Run and verify failure**

Run:

```bash
rtk go test ./internal/web -run 'TestMutatingApply(RequiresTokenAndAllowedOrigin|RejectsPostedActionsEvenWithToken)' -count=1
```

Expected: FAIL because `NewServerWithOptions`, `Options`, token validation, origin validation, and server-side plan IDs do not exist.

- [x] **Step 3: Change the apply request contract**

Modify `internal/web/server.go`:

```go
type ApplyRequest struct {
	PlanID    string   `json:"plan_id"`
	ActionIDs []string `json:"action_ids"`
	DryRun    bool     `json:"dry_run"`

	// Actions is accepted for dry-run compatibility only; mutating requests must use PlanID/ActionIDs.
	Actions []syncplan.Action `json:"actions"`
}

type PlanResponse struct {
	PlanID  string            `json:"plan_id"`
	Actions []syncplan.Action `json:"actions"`
	Report  status.LoadReport `json:"report"`
}
```

- [x] **Step 4: Implement options and request safety checks**

Add to `internal/web/server.go`:

```go
type Options struct {
	ApplyToken      string
	AllowMutations  bool
	AllowedOrigin   string
	AllowUnsafeBind bool
	BoundHost       string
}

type Server struct {
	runtime *app.Runtime
	options Options
	mux     *http.ServeMux
}

func NewServer(runtime *app.Runtime) *Server {
	return NewServerWithOptions(runtime, Options{})
}

func NewServerWithOptions(runtime *app.Runtime, options Options) *Server {
	if runtime == nil {
		runtime = &app.Runtime{}
	}
	server := &Server{runtime: runtime, options: options, mux: http.NewServeMux()}
	server.routes()
	return server
}

func (s *Server) allowMutation(r *http.Request) error {
	if !s.options.AllowMutations {
		return fmt.Errorf("web apply mutations are disabled")
	}
	if !s.options.AllowUnsafeBind && !isLoopbackHost(s.options.BoundHost) {
		return fmt.Errorf("web apply mutations require a loopback bind address")
	}
	if s.options.ApplyToken == "" || r.Header.Get("X-UnboundCLI-Token") != s.options.ApplyToken {
		return fmt.Errorf("web apply requires a valid local session token")
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != s.options.AllowedOrigin {
		return fmt.Errorf("web apply rejected origin %q", origin)
	}
	return nil
}

func isLoopbackHost(host string) bool {
	return host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1"
}
```

Update the non-dry-run branch in `handleApply`:

```go
if !request.DryRun {
	if err := s.allowMutation(r); err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	if len(request.Actions) > 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("mutating apply must use server-issued plan/action IDs"))
		return
	}
	if request.PlanID == "" || len(request.ActionIDs) == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("mutating apply requires plan_id and action_ids"))
		return
	}
	writeError(w, http.StatusNotImplemented, fmt.Errorf("mutating apply plan validation is not enabled yet"))
	return
}
```

- [x] **Step 5: Generate token in web command without enabling mutations yet**

In `cmd/web.go`, add:

```go
func newWebToken() (string, error) {
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate web token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}
```

Print the token as informational only:

```go
token, err := newWebToken()
if err != nil {
	return err
}
fmt.Fprintf(cmd.OutOrStdout(), "Web GUI listening on http://%s\n", addr)
fmt.Fprintf(cmd.OutOrStdout(), "Local session token: %s\n", token)
```

Keep `AllowMutations: false` when constructing the server until a later task explicitly enables real apply.

- [x] **Step 6: Use `net.Listen` so command smoke can bind port 0**

In `cmd/web.go`, replace `server.ListenAndServe()` with explicit listen:

```go
listener, err := net.Listen("tcp", addr)
if err != nil {
	return err
}
actualAddr := listener.Addr().String()
fmt.Fprintf(cmd.OutOrStdout(), "Web GUI listening on http://%s\n", actualAddr)
return server.Serve(listener)
```

Pass `BoundHost: webHost` and `AllowedOrigin: "http://" + actualAddr` into `webui.NewServerWithOptions`.

- [x] **Step 7: Add command-level token and server wiring tests**

Create `cmd/web_test.go`:

```go
package cmd

import (
	"bytes"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewWebTokenReturnsURLSafeToken(t *testing.T) {
	token, err := newWebToken()
	if err != nil {
		t.Fatalf("newWebToken failed: %v", err)
	}
	if len(token) < 24 {
		t.Fatalf("expected token length >= 24, got %d", len(token))
	}
	if strings.ContainsAny(token, "+/=") {
		t.Fatalf("expected raw URL-safe token, got %q", token)
	}
}

func TestServeWebPrintsBoundAddressAndServesIndex(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	var output bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- serveWebForTest(listener, "test-token", &output)
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case <-errCh:
		case <-time.After(time.Second):
			t.Fatal("server did not stop after listener close")
		}
	})

	resp, err := http.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatalf("GET web index failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(output.String(), "Web GUI listening on http://") {
		t.Fatalf("startup output missing URL: %q", output.String())
	}
	if !strings.Contains(output.String(), "Local session token: test-token") {
		t.Fatalf("startup output missing token: %q", output.String())
	}
}
```

Implement `serveWebForTest` by extracting the listener/server setup from `runWeb` into a small helper that accepts `net.Listener`, token, and output writer. The helper must construct `webui.NewServerWithOptions` using the actual listener address for `AllowedOrigin`.

- [x] **Step 8: Run tests**

Run:

```bash
rtk go test ./cmd ./internal/web -count=1
```

Expected: PASS. Real mutation is still not enabled; this task only adds the safe plumbing required before a later mutation task.

- [x] **Step 9: Commit**

Run:

```bash
git add cmd/web.go internal/web
git commit -m "feat: add web session token plumbing"
```

## Task 6: Error, Empty, Loading, and Disabled-Service States

**Files:**
- Modify: `internal/web/static/index.html`
- Modify: `internal/web/static/app.js`
- Modify: `internal/web/static/styles.css`
- Modify: `internal/web/browser_test.go`

- [x] **Step 1: Add browser assertions for empty/error containers**

In `internal/web/browser_test.go`, assert:

```go
if !strings.Contains(dom, `id="message"`) {
	t.Fatalf("browser DOM missing message region:\n%s", dom)
}
if !strings.Contains(dom, `aria-live="polite"`) {
	t.Fatalf("browser DOM missing polite live region:\n%s", dom)
}
if !strings.Contains(dom, `data-service="adguard" disabled`) {
	t.Fatalf("browser DOM should mark unavailable AdGuard controls disabled:\n%s", dom)
}
```

- [x] **Step 2: Add message region to HTML**

Add below the toolbar in `internal/web/static/index.html`:

```html
<section id="message" class="message" aria-live="polite"></section>
```

- [x] **Step 3: Track enabled services from `/api/config`**

Add to `internal/web/static/app.js`:

```javascript
let enabledServices = {};

function applyEnabledServices(config) {
  enabledServices = config.enabled || {};
  document.querySelectorAll('[data-service]').forEach((node) => {
    const service = node.getAttribute('data-service');
    const available = enabledServices[service] !== false;
    node.disabled = !available;
    node.classList.toggle('disabled-service', !available);
  });
}
```

Update `refreshEntries()` after fetching `/api/config`:

```javascript
applyEnabledServices(config);
```

Add `data-service` attributes in `internal/web/static/index.html`:

```html
<option data-service="unbound" value="unbound">Unbound</option>
<option data-service="adguard" value="adguard">AdGuard</option>
<option data-service="cloudflare" value="cloudflare">Cloudflare</option>
```

For sync target options:

```html
<option data-service="unbound" value="unbound">Unbound</option>
<option data-service="adguard" value="adguard">AdGuard</option>
```

- [x] **Step 4: Add state helpers**

Add to `internal/web/static/app.js`:

```javascript
function setMessage(text, kind = 'info') {
  const node = el('message');
  node.textContent = text;
  node.className = `message ${kind}`;
}

function setLoading(isLoading) {
  el('refresh').disabled = isLoading;
  el('preview-sync').disabled = isLoading;
  if (isLoading) setMessage('Loading service status...', 'info');
}
```

Use `setLoading(true)` before refresh and `setLoading(false)` in a `finally` block.

- [x] **Step 5: Add CSS**

Add:

```css
.message { min-height: 20px; color: var(--muted); }
.message.error { color: var(--bad); }
.message.info { color: var(--muted); }
.disabled-service { color: var(--muted); }
```

- [x] **Step 6: Run browser test**

Run:

```bash
rtk env UNBOUNDCLI_BROWSER_TESTS=1 go test ./internal/web -run TestBrowserSmokeWithFakeData -count=1
```

Expected: PASS.

- [x] **Step 7: Commit**

Run:

```bash
git add internal/web
git commit -m "feat: improve web UI states"
```

## Task 7: Visual QA and Responsive Layout

**Files:**
- Modify: `internal/web/browser_test.go`
- Modify: `internal/web/static/styles.css`

- [x] **Step 1: Add desktop and mobile screenshot checks**

In `internal/web/browser_test.go`, extend the Chrome invocation into a helper:

```go
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
```

Call it for `1280x900` and `390x844`.

- [x] **Step 2: Run browser smoke**

Run:

```bash
rtk env UNBOUNDCLI_BROWSER_TESTS=1 go test ./internal/web -run TestBrowserSmokeWithFakeData -count=1
```

Expected: PASS and non-empty screenshots in test temp dirs.

- [x] **Step 3: Add mobile layout assertions**

After the `390x844` run, assert that the mobile DOM records responsive state. Add this browser-side marker in `internal/web/static/app.js`:

```javascript
function markResponsiveState() {
  const app = el('app');
  app.setAttribute('data-mobile', window.innerWidth <= 760 ? 'true' : 'false');
  const panel = el('entries-panel');
  app.setAttribute('data-table-scrolls', panel.scrollWidth > panel.clientWidth ? 'true' : 'false');
}
window.addEventListener('resize', markResponsiveState);
```

Call `markResponsiveState()` after `renderEntries()`.

In `internal/web/browser_test.go`, assert the mobile dump:

```go
mobileDOM := runChromeSmoke(t, chromePath, webServer.URL, 390, 844)
if !strings.Contains(mobileDOM, `data-mobile="true"`) {
	t.Fatalf("mobile DOM did not mark mobile layout:\n%s", mobileDOM)
}
if !strings.Contains(mobileDOM, `data-table-scrolls="true"`) {
	t.Fatalf("mobile DOM should expose horizontal table scrolling:\n%s", mobileDOM)
}
```

- [x] **Step 4: Polish responsive CSS**

Add:

```css
@media (max-width: 760px) {
  .summary, .toolbar, .sync-toolbar { align-items: stretch; flex-direction: column; }
  button, select, input { width: 100%; }
  .panel { border-left: 0; border-right: 0; border-radius: 0; }
}
```

- [x] **Step 5: Commit**

Run:

```bash
git add internal/web
git commit -m "test: add responsive web browser smoke"
```

## Task 8: Final Verification and Documentation

**Files:**
- Modify: `AGENTS.md`
- Modify: `README.md` if it exists by the time this task runs; otherwise skip it and document in `AGENTS.md`.

- [x] **Step 1: Add web verification commands to contributor guide**

Add to `AGENTS.md` under build/test commands:

```markdown
- `rtk go run main.go web --help`: inspect local web UI flags.
- `rtk env UNBOUNDCLI_BROWSER_TESTS=1 go test ./internal/web -count=1`: run browser smoke checks, using the test's Chrome Headless Shell auto-discovery.
```

- [x] **Step 2: Run final verification**

Run:

```bash
rtk proxy git diff --check
rtk make check
rtk env UNBOUNDCLI_BROWSER_TESTS=1 go test ./internal/web -count=1
rtk make build
rtk make
rtk go run main.go web --help
rtk proxy rm -f caddy-dns-sync
```

Expected:
- `git diff --check`: no output, exit 0.
- `make check`: `go fmt`, `go vet`, and `go test -v ./...` pass.
- Browser tests: pass.
- `make build`: creates `caddy-dns-sync`, then it is removed.
- `make`: prints help.
- `rtk go run main.go web --help`: prints web flags.

- [x] **Step 3: Commit final docs**

Run:

```bash
git add AGENTS.md README.md
git commit -m "docs: document web UI verification"
```

If `README.md` does not exist, run:

```bash
git add AGENTS.md
git commit -m "docs: document web UI verification"
```

## Execution Notes

- Keep `.envrc` untracked.
- Do not enable real web mutations until Task 5 token checks are present and tested.
- Prefer small commits after each task so regressions can be bisected.
- Keep DHCP apply out of default `all` until a real DHCP applier exists.
- Do not add a frontend package manager unless vanilla JS becomes a bottleneck. If a package manager is introduced later, add explicit Make targets and browser build verification in the same commit.

## Self-Review

- Spec coverage: The plan covers static assets, dashboard, filtering, sync preview, dry-run apply, future token safety, browser tests, responsive checks, and contributor docs.
- Placeholder scan: No `TBD`, `TODO`, or unspecified "add tests" steps remain; every task names files, commands, and expected results.
- Type consistency: API names match the current code: `NewServer`, `ApplyRequest`, `PlanResponse`, `syncplan.Action`, `status.LoadReport`, and `app.Runtime`.
