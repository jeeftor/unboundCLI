# Changelog

All notable changes to this project are documented in this file. The format
follows [Keep a Changelog](https://keepachangelog.com/) and this project adheres
to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- **Caddyfile editor mutex** — all file mutations and deploy operations are now
  serialized via a `sync.Mutex`, preventing concurrent edits from corrupting the
  Caddyfile or git state.
- **Deploy timeout** — deploy commands now have a 5-minute timeout via
  `context.WithTimeout`, preventing hung deploys from blocking indefinitely.
- **Path traversal protection** — template names are validated to reject `..`,
  `/`, and `\` sequences, preventing arbitrary file reads.
- **CLI sync lock** — CLI sync commands now acquire an exclusive file lock
  (`~/.local/share/caddy-dns-sync/sync.lock`) to prevent concurrent syncs from
  racing on API writes.
- **Sync failure tracking** — `SyncResult` and `AdguardSyncResult` now include
  `FailedHostnames` and `ApplyFailed` fields, tracking individual operation
  failures during apply instead of silently continuing.
- **Frontend error boundary** — React `ErrorBoundary` component wraps the app,
  preventing white-screen crashes on unhandled component errors.
- **SSE auto-reconnection** — entries SSE stream now auto-reconnects with
  exponential backoff (1s → 2s → 5s → ... → 30s max) on connection loss.
- **AbortController support** — `postJSON`, `putJSON`, and `deleteJSON` API
  client methods now accept optional `AbortSignal` for request cancellation.
- **Sync confirmation dialog** — SyncModal now shows a confirmation banner
  before applying changes, preventing accidental syncs.
- **Race-detection in CI** — `go test` in CI now uses `-race` flag.
- **Web UI build in CI** — build workflow now builds the React frontend before
  `go build`, ensuring embedded assets are current.
- **Node.js in lint CI** — lint workflow now sets up Node.js and installs web
  deps so the `web-lint` pre-commit hook works in CI.

### Fixed

- **ApplyChanges() failure semantics** — Unbound `ApplyChanges()` (service
  restart) failure no longer returns `false` for `changesApplied`; entries are
  already written, only the restart failed. New `ApplyFailed` field tracks this.
- **Nil pointer dereference in `runSyncAll`** — added nil check for `result`
  before accessing `result.HostnameMap`.
- **Authentik enrichment failure** — hosts in `unknown` state are now
  re-classified when Authentik API fails, matching the CF Access failure path.
- **SLSA release config** — fixed binary name from `binary-linux-amd64` to
  `caddy-dns-sync-linux-amd64`, removed template placeholders.
- **Cosign v3 compatibility** — updated `.goreleaser.yaml` signing config to
  use `--bundle` flag instead of deprecated `--output-signature`/`--output-certificate`.

### Changed

- **Split monolithic `server.go`** (2507 lines) into 7 focused files:
  `server.go` (core), `server_config.go`, `server_entries.go`,
  `server_cloudflare.go`, `server_diagnostics.go`, `server_probe.go`,
  `server_auth.go`.
- **Consolidated duplicate wildcard matching** into exported
  `api.IsWildcardMatch` / `api.IsWildcardDomain` / `api.IsWildcardOrRootHostname`.
- **Refactored `LoadDataWithReport`** (227 lines) into `fetchAllData()` and
  `enrichWithCloudflare()` sub-methods.
- **Refactored `classifyAuth`** (109 lines) into `classifyWANAuth`,
  `classifyAPIAuth`, `classifyLANAuth`, and `classifyStatus` sub-functions.
- **Improved `stripScheme`** to use `strings.TrimPrefix` idiomatically.
- **Added context propagation to all API clients** — `CloudflareClient`,
  `AuthentikClient`, `Client` (OPNSense), and `AdguardClient` now support
  `WithContext(ctx)` for cancellation and timeout support. API calls in the
  data loader and auth discovery now respect the request/server context.
- **Added panic recovery to all goroutines** — `logging.Recover()` helper
  prevents server crashes from goroutine panics. Applied to all 12+ goroutine
  sites (auth discovery, data loader, DNS resolution, upstream probing, cache
  refresh).
- **Improved `refreshAuthCache`** — uses server context instead of
  `context.Background()`, deduplicates concurrent refresh calls with
  `refreshMu` mutex, and tracks goroutines via `WaitGroup` for graceful
  shutdown.
- **Improved DNS resolution** — now logs resolution failures and respects
  context cancellation (skips pending resolutions when context is cancelled).
- **CLI commands use signal-aware context** — `status` and
  `caddy-push-cloudflare` commands now use `signal.NotifyContext` so Ctrl+C
  cancels pending API calls.
- **Graceful shutdown** — web server now catches SIGINT/SIGTERM, drains
  in-flight HTTP requests, and stops background goroutines cleanly.
- **Refactored diagnostics** — extracted `runDiagnostics()` shared between
  blocking and streaming handlers.
- **`handleCloudflareRepairDNS`** now supports SSE streaming (via Accept
  header) with per-hostname progress events, and uses request context for
  cancellation.
- **Frontend diagnostics tab** now uses SSE stream endpoint with timeout
  handling.

### Added

- **`HasDNSMismatch()`** method on `models.Entry` — compares `DNSResolved`
  against `CaddyServerIP` to detect DNS pointing to the wrong server.
- **Hostname format validation** on `handleSyncRemove` and
  `handleCloudflareSetRoute` API handlers.
- **Service URL validation** on `handleCloudflareSetRoute` — must start with
  `http://` or `https://`.
- **Periodic auth cache background refresh** — auth inventory cache now
  auto-refreshes every 5 minutes via a background ticker, ensuring the UI
  shows current auth state without manual refresh.
- **Short-lived entries cache (30s TTL)** — `loadEntries` now caches results
  for 30 seconds, so rapid successive calls from multiple endpoints (entries,
  diagnostics, plan, auth) don't re-fetch from all APIs. Invalidated on
  mutations.
- **`Server.Shutdown()`** method — cancels background goroutines and waits
  for them to finish, enabling graceful shutdown.
- **`logging.Recover()`** helper — deferred panic recovery for goroutines.
- **`logging.GoSafe()`** helper — launches a goroutine with panic recovery.
- **`sendSSEEvent()`** helper — shared SSE event writer for streaming handlers.
- **`GET /api/diagnostics/stream`** — SSE endpoint for diagnostics with
  loading/done/error events.
- **`POST /api/cloudflare/repair-dns`** with SSE — streams per-hostname
  progress when Accept header includes `text/event-stream`.
- **12 table-driven tests** for `classifyAuth()` covering all auth patterns
  (A–F, IdP bypass, open host, LAN-only, service_auth, Authentik provider).
- **6 unit tests** for `buildCloudflareAction` edge cases (direct mode,
  non-default tunnel skip, TLS verify change, no-op, stale delete).

### Fixed

- **Deploy handler error swallowing** — `handleCaddyDeploy` now returns 400
  on malformed JSON instead of silently ignoring the decode error.
- **AdGuard config load error swallowing** — `list_sources.go` now logs
  load errors instead of discarding them.
- **Tabwriter flush error** — `cmd/status.go` now logs flush errors.
- **Delete confirmation error** — `cmd/delete.go` now handles EOF/read
  errors gracefully instead of silently ignoring them.

### Removed

- **Unused wrapper functions** in `cmd/util.go` (`NewClient`,
  `NewCaddyClient`, `NewAdguardClient`) — callers use `api.*` directly.

## [0.4.0] - 2026-07-27

### Added

- **Auth Flows tab** — new top-level tab with a read-only inventory of each
  hostname's WAN/LAN/API authentication configuration. Cross-references Caddy
  (forward_auth detection), Cloudflare Access (apps, policies, service tokens),
  and Authentik (proxy providers, applications, outposts) to classify every
  host into the WAN/LAN auth model and flag misconfigurations (double-login
  risk, WAN-exposed with no auth, split auth).
- **Tab-based navigation** — the web UI now uses three top-level tabs
  (Dashboard | Caddyfile | Auth Flows) with hash-based routing
  (`#/dashboard`, `#/caddyfile`, `#/auth`). Bookmarkable URLs, browser
  back/forward support.
- **Authentik API client** (`internal/api/authentik.go`) — manages proxy
  providers, applications, policy bindings, and outposts via the
  `goauthentik.io/api/v3` SDK. Includes high-level helpers
  (`EnsureProxyApp`, `RemoveProxyApp`) for provisioning forward_auth.
- **Cloudflare Access API wrappers** (`internal/api/cloudflare_access.go`) —
  manages CF Access applications, policies, service tokens, and groups via
  the `cloudflare-go` SDK. Includes helpers for bypass policies, service_auth
  policies, and wildcard app matching.
- **Auth discovery layer** (`internal/auth/discovery.go`) — queries all auth
  sources in parallel and classifies each hostname's auth mode and status.
- **Authentik config integration** — `AuthentikConfig` struct with env vars
  (`AUTHENTIK_ENABLED`, `AUTHENTIK_API_TOKEN`, `AUTHENTIK_BASE_URL`,
  `AUTHENTIK_INSECURE`) and config file support (`~/.caddy-dns-sync.json`).

### Changed

- `ClientSet` now includes an optional `AuthentikClient`.
- `RuntimeOptions` has a new `IncludeAuthentik` flag.
- `NewRuntimeFromConfigs` takes an additional `AuthentikConfig` parameter.
- The web server (`cmd/web.go`) now loads Authentik config at startup.
- `Entry` model has a new `Auth *HostAuth` field for auth discovery results.

## [0.3.0] - 2026-07-27

### Changed

- **Repository rename**: `unboundCLI` → `caddy-dns-sync` on GitHub to match the
  Go module path (`github.com/jeeftor/caddy-dns-sync`), binary name, config
  file (`~/.caddy-dns-sync.json`), and Homebrew formula. GitHub auto-redirects
  the old repo URL, so existing clones keep working. Update your local remote
  with `git remote set-url origin git@github.com:jeeftor/caddy-dns-sync.git`.
- **Environment variables renamed**. The primary env var names are now
  `CADDY_DNS_SYNC_API_KEY`, `CADDY_DNS_SYNC_API_SECRET`, `CADDY_DNS_SYNC_BASE_URL`,
  and `CADDY_DNS_SYNC_INSECURE`. The previous `UNBOUND_CLI_*` names are still
  accepted as deprecated fallbacks (new names take precedence when both are
  set). The deprecated names will be removed in a future release.
- Browser smoke-test flag renamed to `CADDY_DNS_SYNC_BROWSER_TESTS`
  (`UNBOUNDCLI_BROWSER_TESTS` still accepted).
- Live service-test flag renamed to `CADDY_DNS_SYNC_LIVE_TESTS`
  (`UNBOUNDCLI_LIVE_TESTS` still accepted).

### Fixed

- **Release pipeline ldflags targeted the wrong package**. `.slsa-goreleaser.yml`
  and `.goreleaser.yaml` set `main.Version`/`main.version` etc., but the version
  variables live in the `cmd` package (`cmd.Version`/`cmd.Commit`/`cmd.Date`).
  Released binaries built via the SLSA workflow would show `"dev"` instead of
  the real version. Fixed to target `github.com/jeeftor/caddy-dns-sync/cmd.*`.
- **Release workflow Go version mismatch**. `release.yml` pinned
  `go-version: 1.24.2` while `go.mod` declares `go 1.25.0`, which would fail
  the release build on module resolution. Bumped to `1.25`.
- **Sync `--legacy-desc` default did not include `unboundCLI` description
  strings**, so entries tagged with old descriptions like `"Entry created by
  unboundCLI caddy-sync-all"` were treated as foreign (not sync-managed). This
  caused the sync engine to attempt re-adding hostnames that already existed,
  hitting "already exists" errors, and prevented updates/cleanup of those
  entries. The default now includes all known legacy description strings.
- **Sync `--description` default changed** from `"Entry created by CaddySync"`
  to `"Managed by caddy-dns-sync"` to match the description already used by
  the majority of existing entries.
- **Web UI config response leaked Cloudflare tunnel ID**. The `/api/config`
  endpoint included the actual `tunnel_id` value in its `details` map, exposing
  it to anyone who could reach the web UI. Removed — the `tunnel_id_set`
  boolean in `fields` is sufficient for the UI to know whether it's configured.
- **5 pre-existing test failures fixed**:
  - 3 mock OPNSense servers were missing the `/api/core/firmware/backup` path
    (called by `persistConfig()` after `ApplyChanges()`).
  - DHCP lease test fixture had `is_reserved` as a string instead of an array
    (the real OPNSense API returns `["hwaddr"]` or `[]`).
  - Cloudflare plan test mock was missing the DNS records list path.

### Added

- **Auto-migration of Unbound DNS override descriptions at startup**. When the
  Unbound client is constructed (via `LoadRuntime`), the app now scans existing
  overrides for entries with known legacy description strings (`unboundCLI`,
  `CaddySync`, `Route via Caddy`) and rewrites them to the current
  `"Managed by caddy-dns-sync"` value. This is best-effort: failures are logged
  but non-fatal. This ensures the description-based ownership model continues
  to work transparently after the rename — no user action required.

### Documentation

- Rewrote `AGENTS.md` to reflect the actual multi-service scope (Unbound,
  AdGuard, dnsmasq, Cloudflare, Caddyfile editor), correct binary name, Go 1.25,
  full command list, and new env var names.
- Updated `README.md` command list, feature list, examples, and browser-test
  instructions.
- Marked `plan.md` as superseded/completed (the CF tunnel push-sync feature it
  tracked has shipped).
- Updated `claude.md` env var references and stale "planned" markers.

### Known follow-ups (not in this release)

- The internal web UI wire protocol still uses `UNBOUNDCLI_*` identifiers:
  `window.UNBOUNDCLI_WEB_CONFIG`, `window.UNBOUNDCLI_TEST_HOOKS`, and the
  `X-UnboundCLI-Token` HTTP header. These are internal to the Go↔React coupling
  and require a coordinated frontend rebuild (`vite build` to regenerate
  `internal/web/static/app.js`) plus matching changes in `web/src/`. Left for a
  follow-up to avoid shipping a broken web UI.
- The Homebrew formula in `jeeftor/homebrew-tap` should be updated to point at
  the new repo URL (GitHub redirects, so this is non-urgent).
- `changelog.yml` triggers on push to `main`, but the default branch is
  `master` — so git-cliff has never auto-run. Either rename the default branch
  or update the workflow trigger.


