UnboundCLI
A CLI tool for managing Unbound DNS on OPNSense routers. This application allows you to create, read, update, and delete DNS overrides through the OPNSense API.
Features

Modern CLI interface with color output using Cobra and Viper
Interactive TUI (Text User Interface) using Bubble Tea and Lipgloss
CRUD operations for DNS overrides
Secure configuration management
Cross-platform support (macOS, Linux, Windows)

Installation
Using Homebrew (macOS and Linux)
brew tap jeeftor/tap
brew install unboundcli

Manual Installation
Download the latest binary from the Releases page.
Building from Source
# Clone the repository
git clone https://github.com/jeeftor/unboundCLI.git
cd unboundCLI

# Build the application
make build

# Install to your GOPATH/bin
make install

Usage
Initial Configuration
Before using UnboundCLI, you need to configure it with your OPNSense API credentials:
unboundCLI config

Follow the prompts to enter your API key, API secret, and OPNSense URL.
Commands
Usage:
unboundCLI [command]

Available Commands:
add         Add a DNS override
apply       Apply pending DNS changes
caddy-sync  Synchronize DNS entries with Caddy server
completion  Generate the autocompletion script for the specified shell
config      Configure API connection settings
delete      Delete a DNS override
edit        Edit a DNS override
find        Find DNS overrides by host, domain, or both
help        Help about any command
list        List DNS overrides
tui         Launch the Text User Interface

Flags:
--config string      config file (default is $HOME/.unboundCLI.yaml)
-h, --help               help for unboundCLI
--log-level string   set logging level (debug, info, warn, error) (default "info")
-v, --verbose            enable verbose output
--version            version for unboundCLI

Use "unboundCLI [command] --help" for more information about a command.

Development
Prerequisites

Go 1.18 or higher
Make
GoReleaser (optional, for releases)

Makefile Commands
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

Release Process
This project uses GoReleaser with SLSA Level 3 provenance for secure releases.
To create a new release:

Tag the commit:
git tag -a v0.1.0 -m "First release"
git push origin v0.1.0


GitHub Actions will automatically build and publish the release with SLSA provenance.


SLSA Provenance
This project follows SLSA Level 3 security practices for its releases, providing:

Source verification
Build integrity guarantees
Provenance generation
Tamper resistance

License
MIT License
