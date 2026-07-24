package caddyeditor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// writeAndValidate validates newContent by writing it to a temp file and running
// "caddy adapt" (pure Caddyfile syntax check — no env-var resolution, no network).
// Only if syntax is valid does it atomically replace the real file.
// A bad Caddyfile will NEVER be written to the real path.
func writeAndValidate(cfg EditorConfig, absPath, newContent string) error {
	dir := filepath.Dir(absPath)
	tmp, err := os.CreateTemp(dir, ".caddyfile-validate-*")
	if err != nil {
		return fmt.Errorf("creating temp file for validation: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(newContent); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	tmp.Close()

	// "caddy adapt" converts Caddyfile → JSON without resolving env vars or
	// connecting to anything — it only checks syntax and structure.
	var buf bytes.Buffer
	cmd := exec.Command("caddy", "adapt", "--config", tmpPath, "--adapter", "caddyfile") //nolint:gosec
	cmd.Dir = cfg.RepoPath
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		// Surface the most informative error line.
		output := strings.TrimSpace(buf.String())
		for _, line := range strings.Split(output, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(strings.ToLower(line), "error") || strings.Contains(line, "malformed") {
				return fmt.Errorf("caddyfile syntax error: %s", line)
			}
		}
		if output != "" {
			return fmt.Errorf("caddyfile validation failed: %s", output)
		}
		return fmt.Errorf("caddyfile validation failed: %w", err)
	}

	// Syntax valid — atomically replace the real file.
	if err := os.Rename(tmpPath, absPath); err != nil {
		// Fallback for cross-device rename.
		return os.WriteFile(absPath, []byte(newContent), 0o644)
	}
	return nil
}

// matcherNameFromHostname derives a safe matcher name from a hostname.
// e.g. "sonarr.vookie.net" → "sonarr_vookie_net"
func matcherNameFromHostname(hostname string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return re.ReplaceAllString(hostname, "_")
}

// renderMatcherBlock renders the @matcher + handle block for insertion into the Caddyfile.
func renderMatcherBlock(matcherName, hostname, upstream, templateName string, repoPath string, params map[string]string) (string, error) {
	tmplStr, err := lookupTemplate(repoPath, templateName)
	if err != nil {
		return "", err
	}
	content, err := renderTemplateStr(tmplStr, TemplateData{
		Hostname:    hostname,
		Upstream:    upstream,
		MatcherName: matcherName,
		Params:      params,
	})
	if err != nil {
		return "", err
	}
	return content, nil
}

// ValidateDraft renders the entry (add or update) into a temp copy of the Caddyfile
// and runs caddy adapt to check syntax — without touching the real file.
func ValidateDraft(cfg EditorConfig, block SiteBlock, templateName string) ValidationResult {
	if templateName == "" {
		templateName = cfg.EntryTemplate
	}
	if templateName == "" {
		templateName = "default"
	}

	path := AbsCaddyfilePath(cfg)
	raw, err := os.ReadFile(path)
	if err != nil {
		return ValidationResult{OK: false, Output: fmt.Sprintf("reading caddyfile: %s", err)}
	}

	matcherName := matcherNameFromHostname(block.Hostname)
	snippet, err := renderMatcherBlock(matcherName, block.Hostname, block.Upstream, templateName, cfg.RepoPath, block.Params)
	if err != nil {
		return ValidationResult{OK: false, Output: fmt.Sprintf("rendering template: %s", err)}
	}

	// For updates, remove the existing entry first.
	content := string(raw)
	existing, _ := ParseCaddyfile(path)
	for _, b := range existing {
		if b.Hostname == block.Hostname {
			content = deleteEntry(content, b.MatcherName, b.Hostname)
			break
		}
	}

	draft, err := insertEntry(content, matcherName, block.Hostname, snippet)
	if err != nil {
		return ValidationResult{OK: false, Output: err.Error()}
	}

	// Write draft to a temp file and validate syntax only.
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".caddyfile-draft-*")
	if err != nil {
		return ValidationResult{OK: false, Output: fmt.Sprintf("creating temp file: %s", err)}
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(draft); err != nil {
		tmp.Close()
		return ValidationResult{OK: false, Output: fmt.Sprintf("writing temp file: %s", err)}
	}
	tmp.Close()

	var buf bytes.Buffer
	cmd := exec.Command("caddy", "adapt", "--config", tmpPath, "--adapter", "caddyfile") //nolint:gosec
	cmd.Dir = cfg.RepoPath
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		output := strings.TrimSpace(buf.String())
		for _, line := range strings.Split(output, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(strings.ToLower(line), "error") || strings.Contains(line, "malformed") {
				return ValidationResult{OK: false, Output: "caddyfile syntax error: " + line}
			}
		}
		if output != "" {
			return ValidationResult{OK: false, Output: "caddyfile validation failed: " + output}
		}
		return ValidationResult{OK: false, Output: "caddyfile validation failed: " + err.Error()}
	}
	return ValidationResult{OK: true, Output: ""}
}

// AddEntry inserts a new @matcher + handle block into the Caddyfile.
// It inserts just before the fallback "handle {" block (if present), or at the
// end of the outermost wildcard block.
func AddEntry(cfg EditorConfig, block SiteBlock, templateName string) error {
	if templateName == "" {
		templateName = cfg.EntryTemplate
	}
	if templateName == "" {
		templateName = "default"
	}

	path := AbsCaddyfilePath(cfg)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading caddyfile: %w", err)
	}

	matcherName := matcherNameFromHostname(block.Hostname)
	snippet, err := renderMatcherBlock(matcherName, block.Hostname, block.Upstream, templateName, cfg.RepoPath, block.Params)
	if err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}

	updated, err := insertEntry(string(data), matcherName, block.Hostname, snippet)
	if err != nil {
		return err
	}

	return writeAndValidate(cfg, path, updated)
}

// UpdateEntry replaces the existing entry for the given hostname in the Caddyfile.
func UpdateEntry(cfg EditorConfig, block SiteBlock, templateName string, data TemplateData) error {
	if err := RemoveEntry(cfg, block.Hostname); err != nil {
		return fmt.Errorf("removing old entry: %w", err)
	}
	block.Upstream = data.Upstream
	return AddEntry(cfg, block, templateName)
}

// RemoveEntry deletes the @matcher line and handle block for the given hostname.
func RemoveEntry(cfg EditorConfig, hostname string) error {
	path := AbsCaddyfilePath(cfg)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading caddyfile: %w", err)
	}

	// Find the entry to get its matcher name.
	blocks, err := ParseCaddyfile(path)
	if err != nil {
		return fmt.Errorf("parsing caddyfile: %w", err)
	}

	var target *SiteBlock
	for i := range blocks {
		if blocks[i].Hostname == hostname {
			target = &blocks[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("entry %q not found in caddyfile", hostname)
	}

	updated := deleteEntry(string(data), target.MatcherName, target.Hostname)
	return writeAndValidate(cfg, path, updated)
}

// insertEntry inserts snippet into the caddyfile content before the fallback handle block.
// snippet is indented to match the surrounding style.
func insertEntry(content, matcherName, hostname, snippet string) (string, error) {
	lines := strings.Split(content, "\n")

	// Check for duplicate.
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "@"+matcherName+" host ") || t == "@"+matcherName+" host "+hostname {
			return "", fmt.Errorf("entry for %q already exists (matcher @%s)", hostname, matcherName)
		}
	}

	// Detect indentation from surrounding content (look for a handle line).
	indent := "\t"
	for _, line := range lines {
		if strings.Contains(line, "handle") && strings.HasSuffix(strings.TrimSpace(line), "{") {
			for _, ch := range line {
				if ch == ' ' || ch == '\t' {
					indent = string(ch)
				} else {
					break
				}
			}
			break
		}
	}

	// Indent the snippet.
	indented := indentBlock(snippet, indent)

	// Find insertion point: just before "handle {" (fallback, no @name).
	// Pattern: line is just "indent handle {" with nothing between handle and {.
	fallbackRe := regexp.MustCompile(`^\s+handle\s*\{`)
	for i, line := range lines {
		// Must be a plain "handle {" — not "handle @name {"
		t := strings.TrimSpace(line)
		if fallbackRe.MatchString(line) && !strings.Contains(t, "@") {
			// Insert snippet + blank line before this line.
			newLines := make([]string, 0, len(lines)+len(strings.Split(indented, "\n"))+1)
			newLines = append(newLines, lines[:i]...)
			newLines = append(newLines, strings.Split(indented, "\n")...)
			newLines = append(newLines, "")
			newLines = append(newLines, lines[i:]...)
			return strings.Join(newLines, "\n"), nil
		}
	}

	// No fallback found — insert before the last closing brace of the outermost block.
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "}" {
			newLines := make([]string, 0, len(lines)+len(strings.Split(indented, "\n"))+1)
			newLines = append(newLines, lines[:i]...)
			newLines = append(newLines, "")
			newLines = append(newLines, strings.Split(indented, "\n")...)
			newLines = append(newLines, lines[i:]...)
			return strings.Join(newLines, "\n"), nil
		}
	}

	return content + "\n" + indented + "\n", nil
}

// deleteEntry removes the @matcherName line and handle @matcherName {} block.
func deleteEntry(content, matcherName, hostname string) string {
	lines := strings.Split(content, "\n")
	var out []string

	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		// Skip the @matcher line.
		if trimmed == "@"+matcherName+" host "+hostname ||
			strings.HasPrefix(trimmed, "@"+matcherName+" host ") {
			// Also skip a blank line that immediately follows.
			i++
			if i < len(lines) && strings.TrimSpace(lines[i]) == "" {
				i++
			}
			continue
		}

		// Skip the handle @matcherName { ... } block.
		if strings.HasPrefix(trimmed, "handle @"+matcherName) &&
			(strings.Contains(trimmed, "{") || i+1 < len(lines) && strings.Contains(strings.TrimSpace(lines[i+1]), "{")) {
			depth := 0
			for _, ch := range trimmed {
				if ch == '{' {
					depth++
				} else if ch == '}' {
					depth--
				}
			}
			i++
			for i < len(lines) && depth > 0 {
				for _, ch := range strings.TrimSpace(lines[i]) {
					if ch == '{' {
						depth++
					} else if ch == '}' {
						depth--
					}
				}
				i++
			}
			// Skip trailing blank line.
			if i < len(lines) && strings.TrimSpace(lines[i]) == "" {
				i++
			}
			continue
		}

		out = append(out, lines[i])
		i++
	}

	return strings.Join(out, "\n")
}

// indentBlock prefixes every non-empty line of block with the given indent string.
func indentBlock(block, indent string) string {
	lines := strings.Split(block, "\n")
	var out []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			out = append(out, "")
		} else {
			out = append(out, indent+line)
		}
	}
	return strings.Join(out, "\n")
}
