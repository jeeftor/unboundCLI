# Cloudflare Tunnel Sync — Implementation Plan

## Background & Context

### Current state (as of commit dbf6479)

The codebase has:
- `github.com/cloudflare/cloudflare-go v0.115.0` SDK in `go.mod`
- `internal/api/cloudflare.go` — `CloudflareClient` with:
  - `GetTunnelHostnames()` — reads ingress rules from an existing tunnel ✅
  - `ListTunnels()` — lists all account tunnels ✅
  - `ListZones()` — lists all zones accessible by the token ✅
  - `NewCloudflareClientWithBaseURL()` — testable constructor ✅
  - `AddTunnelHostname()` / `DeleteTunnelHostname()` — **stubs only** (log but do nothing)
- `internal/config/config.go` — `CloudflareConfig` wired into `ExtendedConfig` ✅
  - `LoadCloudflareConfig()` with env → viper → file precedence ✅
  - Env vars: `CF_ENABLED`, `CF_API_TOKEN`, `CF_ACCOUNT_ID`, `CF_ZONE_ID`, `CF_TUNNEL_ID`, `CF_CADDY_SERVICE_URL` ✅
- `cmd/cloudflare-setup` — interactive Bubble Tea wizard (token → account → tunnel picker → zone picker → Caddy URL → save) ✅
- `cmd/config` — non-TUI config with optional Cloudflare section; preserves all existing fields on save ✅
- `cmd/config-tui` — real `ConfigWizard` TUI (ctrl+s save, ctrl+t test connections, ctrl+p toggle password visibility) ✅
- `internal/widgets/config_editor.go` — `TogglePasswordVisibility()` for show/hide secrets ✅
- Connection test output shows all 3 services (UnboundDNS / AdguardHome / Cloudflare) with disabled/success/fail states ✅
- 57 unit tests passing ✅

### What's still missing

The "push to Cloudflare" direction: reading Caddy hostnames and writing them as ingress rules into the CF tunnel plus creating the corresponding CF DNS CNAME records.

---

## Wildcard vs. Individual Host — Recommendation

| | Wildcard `*.vookie.net` | Individual hosts |
|---|---|---|
| CF tunnel config | One ingress rule | One rule per hostname |
| CF DNS records | One wildcard CNAME | One CNAME per hostname |
| Selective exposure | No — all subdomains route through | Yes — only listed hosts |
| Safety | Caddy still 404s unknown hosts, so functionally safe | Explicit, auditable |
| Management | Nothing to do after setup | Managed by this tool |

**Recommendation: individual hosts.** Wildcard is safe (Caddy rejects unknown hosts), but individual entries give explicit control over what's publicly reachable and a clean audit trail. Use this tool to automate the management burden.

---

## Architecture

### Data flow

```
Caddy Admin API  →  extract hostnames
                        ↓
          compare against current CF tunnel ingress config
                        ↓
     diff: add / keep / remove ingress rules
                        ↓
     UpdateTunnelConfiguration (full replace, atomically)
                        ↓
     for each new hostname: CreateDNSRecord (CNAME → tunnel UUID)
     for each removed hostname: DeleteDNSRecord
```

### CF DNS record structure

Each exposed hostname needs:
- A **CNAME** record: `app.vookie.net` → `<tunnel-uuid>.cfargotunnel.com`
- A corresponding **ingress rule** in the tunnel: `hostname: app.vookie.net, service: http://caddy-ip:port`

The tunnel's catch-all rule (`service: http_status:404`) must always be the last ingress rule.

---

## Implementation Steps

### Step 1 — Cloudflare config in `ExtendedConfig` ✅ DONE

`CloudflareConfig` struct added to `internal/config/config.go`, wired into `ExtendedConfig`, with `LoadCloudflareConfig()` and all env vars. See commit dbf6479.

---

### Step 2 — Implement `CloudflareClient` write methods

**File**: `internal/api/cloudflare.go`

Replace the stub `AddTunnelHostname`/`DeleteTunnelHostname` with a proper full-replacement approach (safer than incremental):

```go
// SetTunnelIngress replaces the entire ingress rule list atomically.
// rules: map of hostname → internal service URL (e.g. "http://192.168.1.15:80")
// The catch-all rule is appended automatically.
func (c *CloudflareClient) SetTunnelIngress(rules map[string]string) error

// EnsureDNSRecord creates a CNAME for hostname → tunnelUUID.cfargotunnel.com
// if it doesn't already exist; updates it if the target changed.
func (c *CloudflareClient) EnsureDNSRecord(hostname string) error

// DeleteDNSRecord removes the CNAME for hostname, if present.
func (c *CloudflareClient) DeleteDNSRecord(hostname string) error

// ListManagedDNSRecords returns all CNAME records in the zone pointing to cfargotunnel.com
func (c *CloudflareClient) ListManagedDNSRecords() (map[string]string, error)
```

Key SDK calls:
- `c.api.UpdateTunnelConfiguration(ctx, accountRC, tunnelID, cloudflare.TunnelConfigurationParams{...})`
- `c.api.CreateDNSRecord(ctx, zoneRC, cloudflare.CreateDNSRecordParams{...})`
- `c.api.UpdateDNSRecord(ctx, zoneRC, recordID, ...)`
- `c.api.ListDNSRecords(ctx, zoneRC, cloudflare.ListDNSRecordsParams{Type: "CNAME"})`
- `c.api.DeleteDNSRecord(ctx, zoneRC, recordID)`

---

### Step 3 — Sync engine: Caddy → Cloudflare tunnel

**New file**: `internal/exec/sync/caddy_push_cloudflare.go`

```go
type CaddyToCloudflareSyncOptions struct {
    DryRun           bool
    CaddyServerIP    string
    CaddyServerPort  int
    CaddyServiceURL  string   // target for tunnel ingress rules, e.g. "http://192.168.1.15:80"
    EntryDescription string   // tag used in CF DNS record comments
    // Optional filter: if non-empty, only sync hostnames matching these suffixes
    HostFilter       []string // e.g. ["vookie.net"]
    Verbose          bool
}

type CaddyToCloudflareSyncResult struct {
    CaddyHostnames  []string
    TunnelAdded     []string
    TunnelRemoved   []string
    DNSAdded        []string
    DNSRemoved      []string
    DryRun          bool
    ChangesApplied  bool
}

func SyncCaddyToCloudflare(
    cfClient *api.CloudflareClient,
    options CaddyToCloudflareSyncOptions,
) (*CaddyToCloudflareSyncResult, error)
```

Logic:
1. Fetch Caddy hostname map via `api.NewCaddyClient(...).GetHostnameMap()`
2. Apply `HostFilter` (skip hostnames that don't match configured domain)
3. Fetch current tunnel ingress via `cfClient.GetTunnelHostnames()`
4. Fetch current CF DNS records via `cfClient.ListManagedDNSRecords()`
5. Compute diff (add/keep/remove)
6. If not dry-run: call `cfClient.SetTunnelIngress(newRules)`, then reconcile DNS records

---

### Step 4 — CLI command

**New file**: `cmd/caddy-push-cloudflare.go`

```
caddy-dns-sync caddy-push-cloudflare [flags]

Flags:
  --dry-run               Show changes without applying
  --caddy-ip              Caddy admin API IP (default from config)
  --caddy-port            Caddy admin API port (default 2019)
  --caddy-service-url     Internal service URL for tunnel ingress target
  --host-filter           Only sync hosts matching this domain suffix (repeatable)
  --verbose
```

Config is loaded from `ExtendedConfig.Cloudflare`; flags override.

---

### Step 5 — Config TUI support ✅ DONE

`internal/widgets/config_editor.go` has Cloudflare section in `config-tui` wizard with all fields (enabled, api_token masked, account_id, zone_id, tunnel_id, caddy_service_url). Password visibility toggled with `ctrl+p`. See commit dbf6479.

---

### Step 6 — Unified sync integration (optional, Phase 2)

Extend `caddy-sync-all` / `SyncAll()` with a `--cloudflare` flag so a single invocation can push to UnboundDNS, AdguardHome, **and** the Cloudflare tunnel simultaneously.

---

## Existing Command Clarification

The existing `caddy-sync-cloudflare` command is **not** a CF tunnel write operation — it creates dual-mode local DNS entries (`service.dev.example.com` → direct IP, `service.caddy.example.com` → Caddy IP) to support split-horizon routing alongside an existing CF tunnel. It remains useful as-is and should not be renamed.

The new command (`caddy-push-cloudflare`) is the complement: it manages the Cloudflare side.

---

## Go Library Notes

`github.com/cloudflare/cloudflare-go v0.115.0` is already a dependency.

Relevant types/methods:
```go
// Tunnel config update
cloudflare.TunnelConfigurationParams{
    Config: cloudflare.TunnelConfiguration{
        Ingress: []cloudflare.UnvalidatedIngressRule{
            {Hostname: "app.vookie.net", Service: "http://192.168.1.15:80"},
            {Service: "http_status:404"}, // catch-all — must be last
        },
    },
}

// DNS record creation
cloudflare.CreateDNSRecordParams{
    Type:    "CNAME",
    Name:    "app.vookie.net",
    Content: "<tunnel-uuid>.cfargotunnel.com",
    Proxied: cloudflare.BoolPtr(true), // proxied = goes through CF
    TTL:     1, // auto
}
```

Use `cloudflare.ResourceIdentifier(accountID)` and `cloudflare.ResourceIdentifier(zoneID)` for the resource container params.

---

## Testing Plan

- Unit tests for `SetTunnelIngress` / DNS record methods using a mock HTTP server (follow `internal/api/cloudflare_test.go` pattern — `NewCloudflareClientWithBaseURL` + `httptest.NewServer`)
- Unit tests for `SyncCaddyToCloudflare` with stubbed clients
- Integration test notes: requires real CF credentials — mark with `//go:build integration`

---

## File Checklist

| File | Status | Notes |
|------|--------|-------|
| `internal/config/config.go` | ✅ Done | `CloudflareConfig`, `ExtendedConfig`, `LoadCloudflareConfig()` |
| `internal/api/cloudflare.go` | ⏳ Partial | `ListTunnels`, `ListZones` done; need `SetTunnelIngress`, `EnsureDNSRecord`, `DeleteDNSRecord`, `ListManagedDNSRecords` |
| `cmd/cloudflare-setup.go` | ✅ Done | Interactive wizard with tunnel/zone picker |
| `internal/tui/config_wizard.go` | ✅ Done | Real config TUI with ctrl+s/t/p |
| `internal/widgets/config_editor.go` | ✅ Done | Cloudflare section + password toggle |
| `internal/api/cloudflare_test.go` | ✅ Done | `TestListTunnels`, `TestListZones` |
| `internal/config/cloudflare_config_test.go` | ✅ Done | Env/file loading tests |
| `internal/tui/config_wizard_test.go` | ✅ Done | Connection test logic, config building |
| `internal/widgets/config_editor_test.go` | ✅ Done | Password toggle, values round-trip |
| `internal/exec/sync/caddy_push_cloudflare.go` | ❌ TODO | `SyncCaddyToCloudflare` |
| `cmd/caddy-push-cloudflare.go` | ❌ TODO | CLI command |
| `internal/exec/sync/caddy_push_cloudflare_test.go` | ❌ TODO | Sync engine tests |
