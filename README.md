# 🌐 caddy-dns-sync

[![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
[![Release](https://img.shields.io/github/v/release/jeeftor/caddy-dns-sync?style=for-the-badge&logo=github)](https://github.com/jeeftor/caddy-dns-sync/releases)
[![SLSA 3](https://img.shields.io/badge/SLSA-Level%203-green?style=for-the-badge)](https://slsa.dev)

> 🚀 A powerful CLI tool for managing Unbound DNS on OPNSense routers with a beautiful interactive interface

caddy-dns-sync provides seamless DNS override management through the OPNSense API, featuring both command-line operations and an intuitive Text User Interface (TUI).

## ✨ Features

- 🎨 **Modern CLI interface** with color output using Cobra and Viper
- 🖥️ **Interactive TUI** powered by Bubble Tea and Lipgloss
- 📝 **Complete CRUD operations** for DNS overrides
- 🔐 **Secure configuration management**
- 🌍 **Cross-platform support** (macOS, Linux, Windows)
- 🔄 **Caddy integration** for automatic DNS synchronization
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

## 📖 Usage

```
Usage:
  caddy-dns-sync [command]

Available Commands:
  add         ➕ Add a DNS override
  apply       ✅ Apply pending DNS changes
  caddy-sync  🔄 Synchronize DNS entries with Caddy server
  completion  📝 Generate shell autocompletion script
  config      ⚙️  Configure API connection settings
  delete      🗑️  Delete a DNS override
  edit        ✏️  Edit a DNS override
  find        🔍 Find DNS overrides by host, domain, or both
  help        ❓ Help about any command
  list        📋 List DNS overrides
  tui         💻 Launch the Text User Interface

Flags:
  --config string      config file (default: $HOME/.caddy-dns-sync.yaml)
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

# Launch interactive mode
caddy-dns-sync tui
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
