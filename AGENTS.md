# Repository Guidelines

## Project Structure & Module Organization

This repository is a Go CLI for managing Unbound DNS overrides on OPNSense. The entry point is [main.go](/Users/jstein/devel/unboundCLI/main.go), CLI commands live in [cmd/](/Users/jstein/devel/unboundCLI/cmd), and private implementation packages live in [internal/](/Users/jstein/devel/unboundCLI/internal). Notable packages include `internal/api` for OPNSense, Cloudflare, and Caddy API clients, `internal/config` for credential/config loading, `internal/exec/sync` for sync workflows, and `internal/tui`/`internal/ui` for Bubble Tea and Lipgloss UI code. Build and release configuration is kept in `Makefile`, `.goreleaser.yaml`, `.slsa-goreleaser.yml`, and `.github/workflows/`.

## Build, Test, and Development Commands

- `make help`: list available Make targets.
- `make build`: compile the `unboundCLI` binary with version metadata.
- `make test`: run `go test -v ./...`.
- `make vet`: run `go vet ./...`.
- `make fmt`: run `go fmt ./...`.
- `make check`: run formatting, vetting, and tests.
- `make cross-build`: build platform binaries into `dist/`.
- `make release-dry-run`: run a snapshot GoReleaser release without publishing.

Use `go run . --help` or `go run . <command> --help` for local command exploration.

## Coding Style & Naming Conventions

Use Go 1.23 as declared in `go.mod`. Keep package names short, lowercase, and aligned with their directory names. Add new Cobra commands as focused files in `cmd/`, following the existing `add.go`, `list.go`, and `cloudflare-sync.go` pattern. Run `go fmt` and keep imports organized with `goimports`; the pre-commit setup also runs `golines` for Go files. Exported functions, types, and constants should have concise godoc comments.

## Testing Guidelines

Place tests beside the code they cover with `_test.go` suffixes, and prefer table-driven tests for command parsing, config loading, and API behavior. Use the standard `testing` package unless an existing test introduces another dependency. Run `make test` for normal validation and `make check` before opening a PR.

## Commit & Pull Request Guidelines

Commit history and hooks expect Conventional Commit-style messages, such as `feat: add dns override sync` or `fix: handle missing config`. Pre-commit hooks enforce commit message format, whitespace, YAML checks, Go formatting, `go mod tidy`, and GitHub Actions linting.

For PRs, include a short description, the commands you ran, any linked issue, and screenshots or terminal output when changing TUI or CLI presentation. Note configuration or credential handling changes explicitly.

## Security & Configuration Tips

Do not commit real OPNSense, Cloudflare, or Caddy credentials. Local configuration is loaded from environment variables such as `UNBOUND_CLI_API_KEY`, `UNBOUND_CLI_API_SECRET`, `UNBOUND_CLI_BASE_URL`, or from the user config file created by `unboundCLI config`.
