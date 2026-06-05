# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, Test, and Development Commands

```bash
# Build the application
make build              # Builds caddy-dns-sync binary with version info from git
go build               # Simple build without version ldflags

# Run tests
make test              # Run all tests with verbose output
go test ./...          # Run tests without make
go test -v ./internal/api  # Run tests for specific package

# Code quality
make check             # Run fmt, vet, and test
make fmt               # Format code
make vet               # Run go vet

# Cross-platform builds
make cross-build       # Build for linux/darwin/windows (amd64 and arm64)

# Release testing
make release-dry-run   # Test GoReleaser config without publishing
```

## Architecture Overview

`caddy-dns-sync` synchronizes DNS entries across multiple systems in a split-horizon DNS setup: it reads hostnames from Caddy reverse proxy (and optionally Cloudflare tunnels) and keeps UnboundDNS (OPNSense) and AdguardHome in sync.

### Package Structure

**API Clients** (`internal/api/`):
- `client.go`: OPNSense UnboundDNS client — CRUD for DNS host overrides
- `adguard.go`: AdguardHome client — CRUD for DNS rewrite rules
- `caddy.go`: Caddy Admin API client — extracts hostnames from reverse proxy config
- `cloudflare.go`: Cloudflare API client — reads tunnel ingress rules (`GetTunnelHostnames`); write methods (`AddTunnelHostname`/`DeleteTunnelHostname`) are stubs pending implementation
- `dnsmasq.go`: DNSMasq client — reads DHCP leases from OPNSense

**Sync Engine** (`internal/exec/sync/`): Low-level sync implementations. Each function takes API clients + options and returns a typed result struct.
- `caddy.go` → UnboundDNS
- `caddy_adguard.go` → AdguardHome
- `caddy_cloudflare.go` → UnboundDNS with dual-mode entries (direct + caddy subdomain) for LAN/CF routing
- `cloudflare.go` → UnboundDNS from CF tunnel hostnames (CF tunnel as source, not destination)
- `cloudflare_ui.go` → UI rendering helpers for cloudflare sync output
- `unified.go` → both UnboundDNS + AdguardHome targets in parallel
- `sync_exec.go` → shared logic (diff/apply helpers)

**Sync Abstraction** (`internal/sync/`): Higher-level wrapper around `internal/exec/sync/`.
- `options.go`: `SyncOptions` struct with `DefaultSyncOptions()`
- `executor.go`: `SyncExecutor` — takes `SyncOptions`, exposes `SyncToUnbound()`, `SyncToAdguard()`, `SyncAll()`

**Data Layer** (`internal/loader/`, `internal/models/`):
- `loader/data_loader.go`: `SyncDataLoader` — fetches from all services at once, supports progress callbacks
- `models/entry.go`: `Entry` — unified view of a hostname across Caddy, UnboundDNS, AdguardHome, and DHCP
- `models/service_status.go`, `models/sync_status.go`: status types

**TUI** (`internal/tui/`, `internal/widgets/`, `internal/tuitypes/`):
- `tui/app.go`: `AppModel` — main Bubble Tea model, owns all widget instances and API clients
- `tui/sync_executor.go`: async sync execution within the TUI
- `tui/data_loader.go`: async data loading for the TUI
- `widgets/`: individual widget components (`TableWidget`, `StatusWidget`, `LogWidget`, `SyncDialog`, `ConfigEditorWidget`, `HelpWidget`)
- `tuitypes/types.go`, `tuitypes/keybindings.go`: shared TUI type definitions

**Commands** (`cmd/`):
- Sync: `sync.go` (unified Unbound+Adguard), `caddy-sync-cloudflare.go` (dual-mode local DNS for CF routing)
- View: `dashboard.go` (3-way comparison table), `list.go`, `tui.go`
- DNS ops: `add.go`, `delete.go`, `edit.go`, `find.go`, `apply.go`
- Config: `config.go`, `config-tui.go`, `status.go`
- Util: `colors.go` (color/style helpers), `util.go`

**Support packages**:
- `internal/commands/`: business logic for list operations shared between CLI and TUI (`list.go`, `list_sources.go`)
- `internal/tables/renderer.go`: table rendering utilities
- `internal/ui/ui.go`: global UI/styling instance
- `internal/logging/logging.go`: leveled logger
- `internal/tui/types.go`: internal TUI message/state types

### Key Architecture Patterns

**Description-Based Ownership**: Each sync operation tags created DNS entries with a unique description string. Syncs only modify entries they created:
```go
options.EntryDescription = "Entry created by CaddySync"
// Legacy descriptions supported for upgrade paths
options.LegacyDescriptions = []string{"Entry created by unboundCLI caddy-sync-unbound"}
```

**Three-Phase Sync**: identify changes (add/update/remove) → apply changes → reconfigure service (`client.ApplyChanges()` for UnboundDNS).

**Unified Sync** (`caddy-sync-all`): fetches Caddy hostnames once, then syncs to both UnboundDNS and AdguardHome via goroutines. `--unbound-only` / `--adguard-only` flags for selective targeting.

**TUI Widget Architecture**: `AppModel` holds widget pointers and switches `currentView` between `ViewModeTable`, `ViewModeSync`, `ViewModeConfig`. Widgets are pure Bubble Tea models returned by their constructors.

### Configuration System

**Precedence** (highest first):
1. Environment variables
2. Viper config
3. JSON config file (`~/.caddy-dns-sync.json`)

**Environment Variables**:
```bash
UNBOUND_CLI_API_KEY, UNBOUND_CLI_API_SECRET, UNBOUND_CLI_BASE_URL, UNBOUND_CLI_INSECURE
ADGUARD_ENABLED, ADGUARD_USERNAME, ADGUARD_PASSWORD, ADGUARD_BASE_URL, ADGUARD_INSECURE
CF_ENABLED, CF_API_TOKEN, CF_ACCOUNT_ID, CF_ZONE_ID, CF_TUNNEL_ID, CF_CADDY_SERVICE_URL  # planned
```

**Config structs** (`internal/config/config.go`):
- `ExtendedConfig` embeds `api.Config` (Unbound) + `CaddyConfig` + `AdguardConfig`; `CloudflareConfig` is planned
- `LoadConfig()` / `LoadAdguardConfig()` / `LoadExtendedConfig()` follow the same env → viper → file pattern

### Version Information

Injected at build time via ldflags into `cmd/root.go`:
```go
var Version, Commit, Date string  // defaults: "dev", "none", "unknown"
```

### When Adding New Sync Commands

1. Add sync function in `internal/exec/sync/` returning a typed result struct
2. Add command in `cmd/` using `SyncExecutor` or calling exec/sync directly
3. Tag entries with a unique `EntryDescription`; list legacy descriptions for migration
4. Support `--dry-run`; call `client.ApplyChanges()` after mutations on UnboundDNS

### When Adding New API Clients

- Struct with config fields + `*http.Client`; support `Insecure` TLS skip
- Basic Auth (base64 `username:password`), JSON marshal/unmarshal
- Debug logging via `internal/logging`

### Cloudflare Tunnel Sync (Planned — see plan.md)

Two distinct Cloudflare-related sync directions exist:

1. **`caddy-sync-cloudflare`** (existing): Reads Caddy hostnames, writes *local* UnboundDNS entries with `dev`/`caddy` subdomain variants to enable split-horizon routing alongside an existing CF tunnel. Does **not** write to Cloudflare.

2. **`caddy-push-cloudflare`** (planned): Reads Caddy hostnames, writes to the **Cloudflare tunnel** ingress rules and creates CF DNS CNAME records. This is the "push" direction. Implementation tracked in `plan.md`.

The `cloudflare-go` SDK is already a dependency. `internal/api/cloudflare.go` has read support; write methods need implementation.

### Testing

- `internal/api/adguard_test.go`, `internal/api/caddy_test.go`: API client tests (mock HTTP servers)
- `internal/config/config_test.go`: config loading
- `internal/tui/sync_executor_test.go`: TUI sync executor
- `test/` directory: test data for DNSMasq work

<!-- rtk-instructions v2 -->
# RTK (Rust Token Killer) - Token-Optimized Commands

## Golden Rule

**Always prefix commands with `rtk`**. If RTK has a dedicated filter, it uses it. If not, it passes through unchanged. This means RTK is always safe to use.

**Important**: Even in command chains with `&&`, use `rtk`:
```bash
# ❌ Wrong
git add . && git commit -m "msg" && git push

# ✅ Correct
rtk git add . && rtk git commit -m "msg" && rtk git push
```

## RTK Commands by Workflow

### Build & Compile (80-90% savings)
```bash
rtk cargo build         # Cargo build output
rtk cargo check         # Cargo check output
rtk cargo clippy        # Clippy warnings grouped by file (80%)
rtk tsc                 # TypeScript errors grouped by file/code (83%)
rtk lint                # ESLint/Biome violations grouped (84%)
rtk prettier --check    # Files needing format only (70%)
rtk next build          # Next.js build with route metrics (87%)
```

### Test (90-99% savings)
```bash
rtk cargo test          # Cargo test failures only (90%)
rtk vitest run          # Vitest failures only (99.5%)
rtk playwright test     # Playwright failures only (94%)
rtk test <cmd>          # Generic test wrapper - failures only
```

### Git (59-80% savings)
```bash
rtk git status          # Compact status
rtk git log             # Compact log (works with all git flags)
rtk git diff            # Compact diff (80%)
rtk git show            # Compact show (80%)
rtk git add             # Ultra-compact confirmations (59%)
rtk git commit          # Ultra-compact confirmations (59%)
rtk git push            # Ultra-compact confirmations
rtk git pull            # Ultra-compact confirmations
rtk git branch          # Compact branch list
rtk git fetch           # Compact fetch
rtk git stash           # Compact stash
rtk git worktree        # Compact worktree
```

Note: Git passthrough works for ALL subcommands, even those not explicitly listed.

### GitHub (26-87% savings)
```bash
rtk gh pr view <num>    # Compact PR view (87%)
rtk gh pr checks        # Compact PR checks (79%)
rtk gh run list         # Compact workflow runs (82%)
rtk gh issue list       # Compact issue list (80%)
rtk gh api              # Compact API responses (26%)
```

### JavaScript/TypeScript Tooling (70-90% savings)
```bash
rtk pnpm list           # Compact dependency tree (70%)
rtk pnpm outdated       # Compact outdated packages (80%)
rtk pnpm install        # Compact install output (90%)
rtk npm run <script>    # Compact npm script output
rtk npx <cmd>           # Compact npx command output
rtk prisma              # Prisma without ASCII art (88%)
```

### Files & Search (60-75% savings)
```bash
rtk ls <path>           # Tree format, compact (65%)
rtk read <file>         # Code reading with filtering (60%)
rtk grep <pattern>      # Search grouped by file (75%)
rtk find <pattern>      # Find grouped by directory (70%)
```

### Analysis & Debug (70-90% savings)
```bash
rtk err <cmd>           # Filter errors only from any command
rtk log <file>          # Deduplicated logs with counts
rtk json <file>         # JSON structure without values
rtk deps                # Dependency overview
rtk env                 # Environment variables compact
rtk summary <cmd>       # Smart summary of command output
rtk diff                # Ultra-compact diffs
```

### Infrastructure (85% savings)
```bash
rtk docker ps           # Compact container list
rtk docker images       # Compact image list
rtk docker logs <c>     # Deduplicated logs
rtk kubectl get         # Compact resource list
rtk kubectl logs        # Deduplicated pod logs
```

### Network (65-70% savings)
```bash
rtk curl <url>          # Compact HTTP responses (70%)
rtk wget <url>          # Compact download output (65%)
```

### Meta Commands
```bash
rtk gain                # View token savings statistics
rtk gain --history      # View command history with savings
rtk discover            # Analyze Claude Code sessions for missed RTK usage
rtk proxy <cmd>         # Run command without filtering (for debugging)
rtk init                # Add RTK instructions to CLAUDE.md
rtk init --global       # Add RTK to ~/.claude/CLAUDE.md
```

## Token Savings Overview

| Category | Commands | Typical Savings |
|----------|----------|-----------------|
| Tests | vitest, playwright, cargo test | 90-99% |
| Build | next, tsc, lint, prettier | 70-87% |
| Git | status, log, diff, add, commit | 59-80% |
| GitHub | gh pr, gh run, gh issue | 26-87% |
| Package Managers | pnpm, npm, npx | 70-90% |
| Files | ls, read, grep, find | 60-75% |
| Infrastructure | docker, kubectl | 85% |
| Network | curl, wget | 65-70% |

Overall average: **60-90% token reduction** on common development operations.
<!-- /rtk-instructions -->
