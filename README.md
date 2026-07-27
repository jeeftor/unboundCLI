# 🌐 caddy-dns-sync

[![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
[![Release](https://img.shields.io/github/v/release/jeeftor/caddy-dns-sync?style=for-the-badge&logo=github)](https://github.com/jeeftor/caddy-dns-sync/releases)
[![SLSA 3](https://img.shields.io/badge/SLSA-Level%203-green?style=for-the-badge)](https://slsa.dev)

> 🚀 A CLI + local web UI that uses Caddy as the source of truth for hostnames and synchronizes them across Unbound DNS (OPNSense), AdGuard Home, dnsmasq/DHCP, and Cloudflare Tunnels — with a git-backed Caddyfile editor.

caddy-dns-sync reads hostname data from the Caddy Admin API and reconciles it
against your DNS providers and Cloudflare tunnel ingress. It ships as a Cobra
CLI, a Bubble Tea TUI, and an embedded React web UI for status review and
dry-run sync previews.

## ✨ Features

- 🎨 **Modern CLI interface** with color output using Cobra and Viper
- 🖥️ **Interactive TUI** powered by Bubble Tea and Lipgloss
- 🌐 **Local browser UI** (React 19 + Vite, embedded in the binary) for status review and dry-run sync previews
- 📝 **Complete CRUD operations** for DNS overrides
- 🔄 **Multi-target sync**: Unbound DNS (OPNSense), AdGuard Home, dnsmasq/DHCP, Cloudflare Tunnels + DNS
- 📄 **Git-backed Caddyfile editor** with validate → commit → push → deploy pipeline
- 🛡️ **Auth-bypass detection** for Cloudflare tunnel ingress rules that skip Caddy/Authentik
- 🔐 **Secure configuration management**
- 🌍 **Cross-platform support** (macOS, Linux, Windows)
- 🛡️ **SLSA Level 3** security compliance

## 📦 Installation

### Using Homebrew (Recommended for macOS and Linux)

```bash
brew tap jeeftor/tap
brew install caddy-dns-sync
```

### Manual Installation

1. Download the latest binary from the [Releases page](https://github.com/jeeftor/caddy-dns-sync/releases)
2. Extract and move to your `$PATH`

### Building from Source

```bash
# Clone the repository
git clone https://github.com/jeeftor/caddy-dns-sync.git
cd caddy-dns-sync

# Build the application
make build

# Install to your GOPATH/bin
make install
```

## 🚀 Quick Start

### Initial Setup

Configure caddy-dns-sync with your OPNSense API credentials:

```bash
caddy-dns-sync config
```

Follow the interactive prompts to enter:
- 🔑 API Key
- 🔐 API Secret
- 🌐 OPNSense URL

### Launch the TUI

Experience the beautiful interactive interface:

```bash
caddy-dns-sync tui
```

### Launch the Web UI

Start the local browser interface:

```bash
caddy-dns-sync web
```

The web UI is bound to `127.0.0.1:8080` by default. It shows Caddy/DNS status and supports sync previews plus dry-run apply. Real browser-triggered mutations remain disabled until local token and server-side plan validation are enabled.

## 📖 Usage

```
Usage:
  caddy-dns-sync [command]

Available Commands:
  add                      ➕ Add a DNS override
  apply                    ✅ Apply pending DNS changes
  caddy-editor-check       📝 Validate the configured Caddyfile
  caddy-push-cloudflare    ☁️  Push Caddy hostnames into a Cloudflare tunnel + DNS
  caddy-sync-cloudflare    🔄 Create dual-mode local DNS entries alongside a CF tunnel
  cf-tunnel-backup         💾 Backup Cloudflare tunnel ingress config
  cloudflare-setup         🔧 Interactive Cloudflare setup wizard
  completion               📝 Generate shell autocompletion script
  config                   ⚙️  Configure API connection settings (non-TUI)
  config-tui               🖥️  Interactive TUI-based configuration setup
  dashboard                📊 Show a unified status dashboard
  delete                   🗑️  Delete a DNS override
  edit                     ✏️  Edit a DNS override
  find                     🔍 Find DNS overrides by host, domain, or both
  help                     ❓ Help about any command
  install-service          🔧 Install a systemd unit for periodic sync
  list                     📋 List DNS overrides
  list-sources             📋 List hostname sources (Caddy, CF tunnels)
  status                   📊 Show live service status
  sync                     🔄 Synchronize DNS entries with Caddy server
  tui                      💻 Launch the Text User Interface
  web                      🌐 Start the local web GUI

Flags:
  --config string      config file (default: $HOME/.caddy-dns-sync.json)
  -h, --help          help for caddy-dns-sync
  --log-level string  set logging level (debug, info, warn, error) (default: "info")
  -v, --verbose       enable verbose output
  --version           version for caddy-dns-sync

Use "caddy-dns-sync [command] --help" for more information about a command.
```

### Examples

```bash
# List all DNS overrides
caddy-dns-sync list

# Add a new DNS override
caddy-dns-sync add --host myserver --domain local.lan --ip 192.168.1.100

# Find specific overrides
caddy-dns-sync find --host myserver

# Sync Caddy hostnames into Unbound + AdGuard
caddy-dns-sync sync

# Push Caddy hostnames into a Cloudflare tunnel (dry-run first)
caddy-dns-sync caddy-push-cloudflare --dry-run

# Launch interactive mode
caddy-dns-sync tui

# Start the local web UI
caddy-dns-sync web
```

## 🛠️ Development

### Prerequisites

- Go 1.18 or higher
- Make
- GoReleaser (optional, for releases)

### Available Make Commands

```bash
make build          # 🔨 Build the application
make test           # 🧪 Run tests
make check          # 🔍 Format code and run linters
make cross-build    # 🌍 Cross-compile for multiple platforms
make release-dry-run # 🚀 Test GoReleaser configuration
make help           # 📚 Show all available commands
```

### Browser UI Checks

```bash
go run . web --help
CADDY_DNS_SYNC_BROWSER_TESTS=1 go test ./internal/web -count=1
```

Set `CHROME_HEADLESS_SHELL=/path/to/chrome-headless-shell` only if the browser
smoke test cannot auto-discover a local Chrome Headless Shell binary. The legacy
flag name `UNBOUNDCLI_BROWSER_TESTS` is still accepted.

## 🚢 Release Process

This project uses **GoReleaser** with **SLSA Level 3** provenance for secure, automated releases.

### Creating a New Release

1. **Tag the commit:**
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

2. **Automated build:** GitHub Actions automatically builds and publishes the release with SLSA provenance

## 🛡️ Security & SLSA Provenance

This project follows **SLSA Level 3** security practices, providing:

- ✅ **Source verification**
- 🔒 **Build integrity guarantees**
- 📋 **Provenance generation**
- 🛡️ **Tamper resistance**

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**Made with ❤️ by [jeeftor](https://github.com/jeeftor)**

⭐ If you find this project helpful, please give it a star!

</div>
