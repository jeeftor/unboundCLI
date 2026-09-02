package caddyeditor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SiteBlock is an in-memory representation of a single reverse-proxy entry
// expressed as an @matcher + handle block inside a wildcard Caddyfile block.
type SiteBlock struct {
	Hostname    string            // e.g. "sonarr.vookie.net"
	Upstream    string            // e.g. "http://10.0.0.112:8989"
	MatcherName string            // e.g. "sonarr" (the @name used in the file)
	Directives  []string          // extra raw directives inside the handle block
	SourceFile  string            // absolute path of the Caddyfile
	LineStart   int               // 1-based line of the @matcher definition
	LineEnd     int               // 1-based line of the closing } of the handle block
	Raw         string            // raw text of the @matcher line + handle block
	Params      map[string]string // extra string parameters passed to the entry template
}

// AbsCaddyfilePath returns the absolute path to the Caddyfile.
func AbsCaddyfilePath(cfg EditorConfig) string {
	if filepath.IsAbs(cfg.CaddyfilePath) {
		return cfg.CaddyfilePath
	}
	return filepath.Join(cfg.RepoPath, cfg.CaddyfilePath)
}

// ParseCaddyfile parses a Caddyfile that uses the wildcard+matcher pattern:
//
//	*.domain.tld {
//	    @name host service.domain.tld
//	    handle @name {
//	        reverse_proxy http://upstream { ... }
//	    }
//	}
//
// It returns one SiteBlock per @matcher/handle pair found anywhere in the file.
func ParseCaddyfile(caddyfilePath string) ([]SiteBlock, error) {
	data, err := os.ReadFile(caddyfilePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", caddyfilePath, err)
	}

	lines := strings.Split(string(data), "\n")

	// Pass 1: collect @name → hostname mappings and their line numbers.
	type matcherDef struct {
		hostname string
		lineNum  int // 1-based
	}
	matchers := map[string]matcherDef{}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "@") {
			continue
		}
		// e.g. "@sonarr host sonarr.vookie.net" or "@sonarr host sonarr.vookie.net sonarr.local"
		parts := strings.Fields(trimmed)
		if len(parts) >= 3 && parts[1] == "host" {
			name := strings.TrimPrefix(parts[0], "@")
			hostname := parts[2] // first hostname is canonical
			matchers[name] = matcherDef{hostname: hostname, lineNum: i + 1}
		}
	}

	// Pass 2: find "handle @name {" blocks.
	var blocks []SiteBlock
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		// Match "handle @name {" or "handle @name{"
		if !strings.HasPrefix(trimmed, "handle @") {
			i++
			continue
		}

		rest := strings.TrimPrefix(trimmed, "handle @")
		nameEnd := strings.IndexAny(rest, " {")
		if nameEnd < 0 {
			i++
			continue
		}
		name := rest[:nameEnd]
		def, ok := matchers[name]
		if !ok {
			i++
			continue
		}

		// Scan forward to collect the full handle block.
		depth := 0
		for _, ch := range trimmed {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
			}
		}

		rawLines := []string{lines[i]}
		upstream := ""
		var directives []string
		j := i + 1
		for j < len(lines) && depth > 0 {
			rawLines = append(rawLines, lines[j])
			t := strings.TrimSpace(lines[j])
			for _, ch := range t {
				if ch == '{' {
					depth++
				} else if ch == '}' {
					depth--
				}
			}
			if depth > 0 && t != "" && !strings.HasPrefix(t, "#") {
				if strings.HasPrefix(t, "reverse_proxy") {
					parts := strings.Fields(t)
					// parts[1] is the upstream URL. Skip path-pattern args (e.g.
					// "reverse_proxy /outpost.goauthentik.io/* host:port") so we
					// don't accidentally store a path as the upstream.
					if len(parts) >= 2 && upstream == "" && !strings.HasPrefix(parts[1], "/") {
						upstream = parts[1]
					}
				} else if depth == 1 {
					directives = append(directives, t)
				}
			}
			j++
		}

		blocks = append(blocks, SiteBlock{
			Hostname:    def.hostname,
			Upstream:    upstream,
			MatcherName: name,
			Directives:  directives,
			SourceFile:  caddyfilePath,
			LineStart:   def.lineNum,
			LineEnd:     j,
			Raw:         strings.Join(rawLines, "\n"),
		})
		i = j
	}

	return blocks, nil
}

// ParseCaddyfileFromConfig is a convenience wrapper that resolves the path from config.
func ParseCaddyfileFromConfig(cfg EditorConfig) ([]SiteBlock, error) {
	path := AbsCaddyfilePath(cfg)
	return ParseCaddyfile(path)
}
