# UnboundCLI

A CLI tool for managing Unbound DNS on OPNSense routers. This application allows you to create, read, update, and delete DNS overrides through the OPNSense API.

## Features

- Modern CLI interface with color output using Cobra and Viper
- Interactive TUI (Text User Interface) using Bubble Tea and Lipgloss
- CRUD operations for DNS overrides
- Secure configuration management
- Cross-platform support (macOS, Linux, Windows)

## Installation

### Using Homebrew (macOS and Linux)

```bash
brew tap jeeftor/tap
brew install unboundcli
```

### Manual Installation

Download the latest binary from the [Releases](https://github.com/jeeftor/unboundCLI/releases) page.

### Building from Source

```bash
# Clone the repository
git clone https://github.com/jeeftor/unboundCLI.git
cd unboundCLI

# Build the application
make build

# Install to your GOPATH/bin
make install
```

## Usage

### Initial Configuration

Before using UnboundCLI, you need to configure it with your OPNSense API credentials:

```bash
unboundCLI config
```

Follow the prompts to enter your API key, API secret, and OPNSense URL.

### Commands

- `unboundCLI list` - List all DNS overrides
- `unboundCLI add` - Add a new DNS override
- `unboundCLI edit <uuid>` - Edit an existing DNS override
- `unboundCLI delete <uuid>` - Delete a DNS override
- `unboundCLI apply` - Apply pending changes
- `unboundCLI tui` - Launch the interactive Text User Interface

For detailed help on any command:

```bash
unboundCLI help [command]
```

## Development

### Prerequisites

- Go 1.18 or higher
- Make
- GoReleaser (optional, for releases)

### Makefile Commands

```bash
# Build the application
make build

# Run tests
make test

# Format code and run linters
make check

# Cross-compile for multiple platforms
make cross-build

# Run GoReleaser in dry-run mode
make release-dry-run

# Show all available commands
make help
```

### Release Process

This project uses GoReleaser with SLSA Level 3 provenance for secure releases.

To create a new release:

1. Tag the commit:
   ```bash
   git tag -a v0.1.0 -m "First release"
   git push origin v0.1.0
   ```

2. GitHub Actions will automatically build and publish the release with SLSA provenance.

## SLSA Provenance

This project follows [SLSA Level 3](https://slsa.dev/spec/v1.0/levels) security practices for its releases, providing:

- Source verification
- Build integrity guarantees
- Provenance generation
- Tamper resistance

## License

MIT License
