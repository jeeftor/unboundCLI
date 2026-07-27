# Repository Guidelines

## Project Structure & Module Organization

This repository is `caddy-dns-sync` — a Go CLI and local web UI that uses a
Caddy server as the source of truth for hostnames and synchronizes them across
Unbound DNS (OPNSense), AdGuard Home, dnsmasq/DHCP, and Cloudflare Tunnels. It
also includes a git-backed Caddyfile editor with a validate → commit → push →
deploy pipeline.

The entry point is [main.go](main.go), CLI commands live in [cmd/](cmd), and
private implementation packages live in [internal/](internal). Notable packages
include `internal/api` for OPNSense/Unbound, Caddy, AdGuard, Cloudflare, and
dnsmasq API clients; `internal/config` for credential/config loading;
`internal/exec/sync` and `internal/sync` for sync workflows; `internal/syncplan`
for the plan/apply model; `internal/caddyeditor` for the Caddyfile editor;
`internal/status` for live service health; and `internal/tui`/`internal/ui`/
`internal/widgets` for Bubble Tea and Lipgloss UI code. The embedded web UI
backend lives in `internal/web` and its React 19 + Vite frontend in `web/`.
Build and release configuration is kept in `Makefile`, `.goreleaser.yaml`,
`.slsa-goreleaser.yml`, and `.github/workflows/`.

## Build, Test, and Development Commands

- `make help`: list available Make targets.
- `make build`: compile the `caddy-dns-sync` binary with version metadata.
- `make test`: run `go test -v ./...`.
- `make vet`: run `go vet ./...`.
- `make fmt`: run `go fmt ./...`.
- `make check`: run formatting, vetting, and tests.
- `make cross-build`: build platform binaries into `dist/`.
- `make release-dry-run`: run a snapshot GoReleaser release without publishing.
- `go run . web --help`: inspect local web UI flags.
- `CADDY_DNS_SYNC_BROWSER_TESTS=1 go test ./internal/web -count=1`: run browser
  smoke checks; set `CHROME_HEADLESS_SHELL` only if auto-discovery fails. The
  legacy flag name `UNBOUNDCLI_BROWSER_TESTS` is still accepted.

Use `go run . --help` or `go run . <command> --help` for local command
exploration.

## Coding Style & Naming Conventions

Use Go 1.25 as declared in `go.mod`. Keep package names short, lowercase, and
aligned with their directory names. Add new Cobra commands as focused files in
`cmd/`, following the existing `add.go`, `list.go`, and `caddy-push-cloudflare.go`
pattern. Run `go fmt` and keep imports organized with `goimports`; the pre-commit
setup also runs `golines` for Go files. Exported functions, types, and constants
should have concise godoc comments.

## Testing Guidelines

Place tests beside the code they cover with `_test.go` suffixes, and prefer
table-driven tests for command parsing, config loading, and API behavior. Use
the standard `testing` package unless an existing test introduces another
dependency. Run `make test` for normal validation and `make check` before
opening a PR.

Browser UI changes should also run
`CADDY_DNS_SYNC_BROWSER_TESTS=1 go test ./internal/web -count=1` so the Chrome
Headless Shell smoke test validates rendering, filtering, sync preview, dry-run
behavior, and mobile layout markers.

## Commit & Pull Request Guidelines

Commit history and hooks expect Conventional Commit-style messages, such as
`feat: add dns override sync` or `fix: handle missing config`. Pre-commit hooks
enforce commit message format, whitespace, YAML checks, Go formatting,
`go mod tidy`, and GitHub Actions linting.

For PRs, include a short description, the commands you ran, any linked issue,
and screenshots or terminal output when changing TUI or CLI presentation. Note
configuration or credential handling changes explicitly.

## Security & Configuration Tips

Do not commit real OPNSense, Cloudflare, Caddy, or AdGuard credentials. Local
configuration is loaded from environment variables such as
`CADDY_DNS_SYNC_API_KEY`, `CADDY_DNS_SYNC_API_SECRET`, `CADDY_DNS_SYNC_BASE_URL`,
or from the user config file created by `caddy-dns-sync config`. The deprecated
`UNBOUND_CLI_*` env var names are still accepted as fallbacks for backwards
compatibility.
