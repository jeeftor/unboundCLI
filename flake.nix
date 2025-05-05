{
  description = "Development environment with syft, goreleaser, golang, gh, and pre-commit";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          # Enable Go cross-compilation support
          config = {
            permittedInsecurePackages = [ ];
            allowUnfree = true;
          };
        };
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Golang
            go_1_24

            # Full toolchain
            gcc
            binutils

            # GitHub CLI
            gh

            # Signing tool (Cosign only)
            cosign

            # Syft for SBOM generation
            syft

            # GoReleaser for Go project releases
            goreleaser

            # Pre-commit for git hooks
            pre-commit

            # Additional useful tools
            git
          ];

          # Explicitly set environment variables for proper cross-compilation
          env = {
            CGO_ENABLED = "0";
            # Make sure Go uses its own toolchain for cross-compilation
            GOROOT_FINAL = "${pkgs.go_1_24}/share/go";
            # Ensure Go can find its tools
            GOTOOLDIR = "${pkgs.go_1_24}/share/go/pkg/tool/${system}-amd64";
          };

          shellHook = ''
            # Ensure Go toolchain is configured correctly
            export GOROOT=$(go env GOROOT)
            export PATH=$GOROOT/bin:$PATH

            # Print environment info in a shell-agnostic way
            echo "Development environment loaded with the following tools:"
            echo "- Go $(go version | cut -d ' ' -f 3)"
            echo "- GitHub CLI $(gh --version | head -n 1)"
            echo "- Cosign $(cosign version 2>/dev/null || echo 'installed')"
            echo "- Syft $(syft version 2>/dev/null || echo 'installed')"
            echo "- GoReleaser $(goreleaser --version 2>/dev/null || echo 'installed')"
            echo "- Pre-commit $(pre-commit --version 2>/dev/null || echo 'installed')"
            echo ""
            echo "Cross-compilation is enabled with CGO_ENABLED=0"
            echo "Ready to develop!"
          '';
        };
      }
    );
}
