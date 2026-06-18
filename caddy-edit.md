# Caddy File Editor — Design Plan

## Goal

Add a "Caddy Editor" feature to CaddySync that lets you visually add, edit, and remove
reverse-proxy entries from your Caddyfile — then commit, push, and deploy, all from the
web UI. CaddySync must be installed on (or have SSH access to) the Caddy box.

---

## User's Setup

```
/root/homelab/caddy/          ← git repo
  Caddyfile                   ← main config (imports below)
  services/*.conf             ← one file per service (where new entries go)
  conf.d/*.conf               ← additional config
  Makefile                    ← stop / pull / start / check targets

/etc/caddy/                   ← deployed copy
  Caddyfile
  services/*.conf
  conf.d/*.conf

Deployment:
  make stop pull start        ← on the Caddy box
  (git pull → cp -r * /etc/caddy → chown → systemctl restart caddy)
```

---

## New Config Section

Add to `~/.caddy-dns-sync.json` under a `"caddy_editor"` key:

```json
{
  "caddy_editor": {
    "enabled": true,
    "repo_path": "/root/homelab/caddy",
    "services_dir": "services",
    "deploy_command": "make -C /root/homelab/caddy stop pull start",
    "validate_command": "make -C /root/homelab/caddy check",
    "git_auto_commit": true,
    "git_auto_push": false,
    "git_remote": "origin",
    "git_branch": "main",
    "entry_template": "default"
  }
}
```

| Field | Purpose |
|---|---|
| `repo_path` | Path to the git repo root |
| `services_dir` | Subdirectory where per-service conf files live (relative to repo_path) |
| `deploy_command` | Shell command to deploy (e.g. `make stop pull start`) |
| `validate_command` | Shell command to validate before deploying (e.g. `make check`) |
| `git_auto_commit` | Commit after each write |
| `git_auto_push` | Push after commit (requires SSH keys or stored credentials) |
| `git_remote` / `git_branch` | Which remote/branch to push to |
| `entry_template` | Template style for new entries (see below) |

---

## Caddyfile Entry Templates

### `default` — bare reverse proxy
```caddyfile
service.vookie.net {
    reverse_proxy 10.0.0.112:3000
}
```

### `cf-tls` — with Cloudflare DNS-01 TLS
```caddyfile
service.vookie.net {
    tls {
        dns cloudflare {env.CLOUDFLARE_API_TOKEN}
    }
    reverse_proxy 10.0.0.112:3000
}
```

### `headers` — pass real IP + strip HTTPS redirect
```caddyfile
service.vookie.net {
    reverse_proxy 10.0.0.112:3000 {
        header_up Host {upstream_hostport}
        header_up X-Real-IP {remote_host}
    }
}
```

Templates are stored as Go text/template strings and rendered at write time.
New templates can be added without code changes (loaded from `repo_path/templates/*.caddytemplate`).

---

## Architecture

### New packages

```
internal/caddyeditor/
  config.go          — EditorConfig struct, LoadEditorConfig()
  parser.go          — parse .conf files into SiteBlock structs
  writer.go          — add / remove / update SiteBlock in a file
  validator.go       — run validate_command, capture stdout/stderr
  deployer.go        — run deploy_command, stream output
  git.go             — diff, add, commit, push wrappers
  templates.go       — built-in and custom entry templates
```

### SiteBlock (in-memory representation)

```go
type SiteBlock struct {
    Hostname    string            // "service.vookie.net"
    Upstream    string            // "10.0.0.112:3000"
    Directives  []string          // raw extra lines (tls block, headers, etc.)
    SourceFile  string            // absolute path of the .conf file
    LineStart   int               // for targeted editing
    LineEnd     int
    Raw         string            // original text, for diff display
}
```

### Parser approach

Caddy config is not trivially parseable (nested braces, imports, snippets).
Start with a **pragmatic line-scanner** that handles the common case:

```
<hostname> {         ← opens a site block
    reverse_proxy …  ← first directive = upstream
    …                ← other directives collected verbatim
}                    ← closes block
```

Edge cases deferred to v2:
- `import` statements
- Named matchers (`@name`)
- Multi-hostname blocks (`a.com b.com { … }`)

### Writer approach

- **Add**: append a rendered template block to `services/<hostname>.conf`
  (one file per service keeps diffs clean and avoids merge conflicts).
- **Remove**: delete `services/<hostname>.conf`.
- **Edit**: rewrite the file in-place (replace lines `LineStart..LineEnd`).

### Validate → Commit → Deploy flow

```
User clicks Deploy
  │
  ├─ run validate_command         → fail: show error log, stop
  │
  ├─ git diff repo_path           → show what changed in UI
  │
  ├─ git add repo_path            → stage all changes
  │
  ├─ git commit -m "caddy: …"     → commit with auto-generated message
  │
  ├─ git push (if auto_push)      → optional
  │
  └─ run deploy_command           → stream stdout to UI log panel
```

---

## Web UI Changes

### New tab: "Caddy" (in top nav, next to Settings)

```
┌─────────────────────────────────────────────────────────────────────┐
│  Caddy Editor                               [+ Add entry]  [Deploy] │
├─────────────────────────────────────────────────────────────────────┤
│  service.vookie.net        10.0.0.112:3000   [Edit]  [Remove]    │
│  other.vookie.net          10.0.0.100:8080   [Edit]  [Remove]    │
│  …                                                                  │
├─────────────────────────────────────────────────────────────────────┤
│  Git status:  2 files changed                                       │
│  ┌ diff ───────────────────────────────────────────────────────┐    │
│  │ + new.vookie.net {                                          │    │
│  │ +   reverse_proxy 10.0.0.5:9090                         │    │
│  │ + }                                                         │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

### Add / Edit modal

```
Hostname:   [_________________________]   e.g. myapp.vookie.net
Upstream:   [_________________________]   e.g. 10.0.0.112:3000
Template:   [default ▾]
Options:    [✓] No TLS verify   [ ] Disable chunked encoding
            [ ] HTTP/2 origin

Preview ────────────────────────────────────
myapp.vookie.net {
    reverse_proxy 10.0.0.112:3000
}
────────────────────────────────────────────

[Cancel]                            [Write to file]
```

### Deploy panel (slides up on Deploy click)

```
Validating…   ✓ Config OK
Committing…   ✓ caddy: add myapp.vookie.net
Deploying…
  ● make stop pull start
  Stopping caddy...
  git pull: Already up to date.
  Copying files...
  Starting caddy...
  ✓ Done
```

---

## Caddy Admin API vs File Editing

Two approaches could work; we use **both**:

| Operation | Method | Why |
|---|---|---|
| Read current routes | Admin API (`/config/`) | Always reflects what's running |
| Add/remove entries | File edit + deploy | Tracked in git, survives restarts |
| Validate | `caddy validate --config …` | Catches syntax errors before deploy |
| Reload (dev mode) | Admin API PATCH | Instant preview without full deploy |

"Dev mode" (future): use Admin API to hot-reload immediately, then offer "Commit & Deploy" to make permanent.

---

## New API Endpoints

```
GET  /api/caddy/entries          — list parsed entries from repo files
POST /api/caddy/entries          — write a new entry to file
PUT  /api/caddy/entries/:host    — update an existing entry
DEL  /api/caddy/entries/:host    — remove entry file
GET  /api/caddy/diff             — git diff of repo
POST /api/caddy/validate         — run validate_command, return stdout/stderr
POST /api/caddy/deploy           — validate + commit + (push) + deploy_command (SSE stream)
GET  /api/caddy/templates        — list available templates
```

---

## Phased Implementation

### Phase 1 — Read + Parse (no writes yet)
- `EditorConfig` struct and config loading
- File parser → list of SiteBlocks
- `/api/caddy/entries` GET
- Caddy tab in UI showing parsed entries (read-only)
- Reconcile with Admin API entries (flag any mismatch)

### Phase 2 — Write + Validate
- Writer (add / remove / edit files)
- Template renderer
- `/api/caddy/entries` POST/PUT/DELETE
- Add/Edit modal in UI
- `/api/caddy/validate` + validate output panel

### Phase 3 — Git + Deploy
- `git.go` wrapper
- `/api/caddy/diff` — show what changed
- `/api/caddy/deploy` — full pipeline, SSE-streamed output
- Deploy panel in UI with live log

### Phase 4 — Polish
- Custom templates from `repo_path/templates/`
- "Dev mode" hot-reload via Admin API
- Rollback: `git revert HEAD` + redeploy
- Entry history from `git log -- services/<hostname>.conf`

---

## Open Questions

1. **SSH vs local**: If CaddySync runs on the Caddy box, all file operations are local.
   If it runs elsewhere, needs SSH or a remote agent. Start local-only.

2. **Permissions**: CaddySync process needs write access to `repo_path` and execute
   permission for `deploy_command`. Run as the same user as the git repo owner (`root`
   in your setup) or grant specific sudo rules.

3. **Concurrent edits**: If multiple people use the UI simultaneously, last-write-wins.
   Acceptable for a homelab; a future file lock or optimistic-concurrency check could help.

4. **Caddy import chains**: Your Caddyfile uses `import services/*.conf`. The parser
   needs to follow imports to build a complete picture. Phase 1 will parse files directly;
   the Admin API provides the ground truth for what's actually running.

5. **Secret handling**: The `CLOUDFLARE_API_TOKEN` in the Makefile is currently plaintext.
   CaddySync should not echo it in logs. The deploy command can reference env vars rather
   than inlining secrets.
