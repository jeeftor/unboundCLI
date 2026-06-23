package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jeeftor/caddy-dns-sync/internal/caddyeditor"
	"github.com/spf13/cobra"
)

var caddyEditorCheckCmd = &cobra.Command{
	Use:           "doctor",
	Aliases:       []string{"caddy-editor-check"},
	Short:         "Check config, paths, git, and commands — no files written",
	Long:          `Verifies the caddy_editor config is correct and all paths/commands work.\nRuns validation and checks git connectivity, but makes NO edits, commits, or deploys.`,
	RunE:          runCaddyEditorCheck,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(caddyEditorCheckCmd)
}

// ── Extra styles only used in doctor ─────────────────────────────────────────
var (
	docKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Width(28)

	docValOK = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2"))

	docValBad = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1"))

	docHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Italic(true)

	docFixStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	docEntryHost = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Bold(true)

	docEntryUpstream = lipgloss.NewStyle().
				Foreground(lipgloss.Color("6"))

	docSummaryOK = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("2"))

	docSummaryFail = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("1"))

	docBanner = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.ThickBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("12"))
)

func runCaddyEditorCheck(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	pass := true
	failCount := 0
	var failures []string

	// Print the title banner
	fmt.Fprintln(out)
	fmt.Fprintln(out, docBanner.Render("caddy-dns-sync  doctor"))
	fmt.Fprintln(out)

	check := func(label string, ok bool, fix string) {
		sym := StyleOK.Render("✓")
		lbl := StyleBold.Render(label)
		if ok {
			fmt.Fprintf(out, "  %s  %s\n", sym, lbl)
		} else {
			fmt.Fprintf(out, "  %s  %s\n", StyleFail.Render("✗"), lbl)
			if fix != "" {
				fmt.Fprintf(out, "       %s %s\n", StyleWarn.Render("→"), docFixStyle.Render(fix))
			}
			pass = false
			failCount++
			failures = append(failures, label)
		}
	}

	kv := func(key, val string, ok bool) {
		valStyle := docValOK
		if !ok {
			valStyle = docValBad
		}
		fmt.Fprintf(out, "     %s %s\n", docKeyStyle.Render(key), valStyle.Render(val))
	}

	info := func(msg string) {
		fmt.Fprintf(out, "  %s  %s\n", StyleInfo.Render("ℹ"), msg)
	}

	section := func(title string) {
		bar := strings.Repeat("─", max(0, 46-len(title)))
		fmt.Fprintf(out, "\n%s\n", StyleSection.Render("── "+title+" "+bar))
	}

	// ── 1. Config ─────────────────────────────────────────────────────────────
	section("Config")
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".caddy-dns-sync.json")
	info("config file: " + StyleCode.Render(configPath))
	fmt.Fprintln(out)

	data, err := os.ReadFile(configPath)
	check("config file exists", err == nil, configPath)

	var cfg caddyeditor.EditorConfig
	if err == nil {
		var wrapper struct {
			CaddyEditor caddyeditor.EditorConfig `json:"caddy_editor"`
		}
		if jsonErr := json.Unmarshal(data, &wrapper); jsonErr != nil {
			check("config file is valid JSON", false, jsonErr.Error())
			return nil
		}
		cfg = wrapper.CaddyEditor
		check("config file is valid JSON", true, "")
	} else {
		cfg = caddyeditor.DefaultEditorConfig()
	}
	fmt.Fprintln(out)

	check("caddy_editor.enabled", cfg.Enabled, `set "enabled": true in caddy_editor config`)
	kv("enabled", fmt.Sprintf("%v", cfg.Enabled), cfg.Enabled)

	check("caddy_editor.repo_path", cfg.RepoPath != "", "repo_path is empty")
	kv("repo_path", or(cfg.RepoPath, "(not set)"), cfg.RepoPath != "")

	caddyfileSet := cfg.CaddyfilePath != ""
	check("caddy_editor.caddyfile", caddyfileSet, `set "caddyfile": "caddy/Caddyfile" in caddy_editor config`)
	kv("caddyfile", or(cfg.CaddyfilePath, "(not set)"), caddyfileSet)

	check("caddy_editor.deploy_command", cfg.DeployCommand != "", "deploy_command is empty")
	kv("deploy_command", or(cfg.DeployCommand, "(not set)"), cfg.DeployCommand != "")

	check("caddy_editor.validate_command", cfg.ValidateCommand != "", "validate_command is empty")
	kv("validate_command", or(cfg.ValidateCommand, "(not set)"), cfg.ValidateCommand != "")

	kv("git_auto_commit", fmt.Sprintf("%v", cfg.GitAutoCommit), true)
	kv("git_auto_push", fmt.Sprintf("%v", cfg.GitAutoPush), true)
	kv("git_remote", or(cfg.GitRemote, "origin"), true)
	kv("git_branch", or(cfg.GitBranch, "(auto-detect)"), true)

	if cfg.RepoPath == "" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, docSummaryFail.Render("  Cannot continue without repo_path  "))
		return exitCode(1)
	}

	// ── 2. Paths ──────────────────────────────────────────────────────────────
	section("Paths")
	fmt.Fprintln(out)

	repoStat, statErr := os.Stat(cfg.RepoPath)
	check("repo_path exists", statErr == nil, cfg.RepoPath)
	kv("repo_path", cfg.RepoPath, statErr == nil)

	if statErr == nil {
		check("repo_path is a directory", repoStat.IsDir(), cfg.RepoPath+" is not a directory")
	}

	gitDir := filepath.Join(cfg.RepoPath, ".git")
	_, gitErr := os.Stat(gitDir)
	check("repo is a git repo (.git exists)", gitErr == nil, "cd "+cfg.RepoPath+" && git init")
	kv(".git", gitDir, gitErr == nil)

	caddyfilePath := caddyeditor.AbsCaddyfilePath(cfg)
	_, cfErr := os.Stat(caddyfilePath)
	check("caddyfile exists", cfErr == nil, "check caddyfile path in config")
	kv("caddyfile", caddyfilePath, cfErr == nil)

	// ── 3. Entries ────────────────────────────────────────────────────────────
	section("Caddyfile entries (read-only)")
	fmt.Fprintln(out)

	entries, parseErr := caddyeditor.ParseCaddyfileFromConfig(cfg)
	check("caddyfile parseable", parseErr == nil, errStr(parseErr))

	if parseErr == nil {
		if len(entries) == 0 {
			info(StyleMuted.Render("no @matcher entries found in caddyfile"))
		} else {
			info(fmt.Sprintf("found %s entr%s", StyleCode.Render(fmt.Sprintf("%d", len(entries))), pluralY(len(entries))))
			fmt.Fprintln(out)
			for _, e := range entries {
				fmt.Fprintf(out, "     %s  %s  %s  %s\n",
					StyleMuted.Render("•"),
					docEntryHost.Render(e.Hostname),
					StyleMuted.Render("→"),
					docEntryUpstream.Render(e.Upstream),
				)
			}
		}
	}

	// ── 4. Git ────────────────────────────────────────────────────────────────
	section("Git")
	fmt.Fprintln(out)

	remote := or(cfg.GitRemote, "origin")

	// Always detect the actual current branch
	var detectedBranch string
	brCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD") //nolint:gosec
	brCmd.Dir = cfg.RepoPath
	if brOut, brErr := brCmd.Output(); brErr == nil {
		detectedBranch = strings.TrimSpace(string(brOut))
	}

	// Use config branch as push target override; otherwise use detected branch
	pushBranch := cfg.GitBranch
	if pushBranch == "" {
		pushBranch = detectedBranch
	}
	if pushBranch == "" {
		pushBranch = "main"
	}

	// Get remote URL
	remoteURLCmd := exec.Command("git", "remote", "get-url", remote) //nolint:gosec
	remoteURLCmd.Dir = cfg.RepoPath
	remoteURL := ""
	if urlOut, urlErr := remoteURLCmd.Output(); urlErr == nil {
		remoteURL = strings.TrimSpace(string(urlOut))
	}

	kv("remote", remote, true)
	if remoteURL != "" {
		kv("remote url", remoteURL, true)
	}
	kv("current branch", or(detectedBranch, "(unknown)"), detectedBranch != "")
	if cfg.GitBranch != "" && cfg.GitBranch != detectedBranch {
		kv("push branch (config)", cfg.GitBranch, false)
		fmt.Fprintf(out, "       %s\n", docHintStyle.Render(
			fmt.Sprintf("  config git_branch=%q differs from current branch %q — update config or checkout the right branch", cfg.GitBranch, detectedBranch)))
	} else {
		kv("push branch", pushBranch, true)
	}
	fmt.Fprintln(out)

	status, gitStatusErr := caddyeditor.GitStatus(cfg)
	check("git status runs", gitStatusErr == nil, errStr(gitStatusErr))

	if gitStatusErr == nil {
		trimmed := strings.TrimSpace(status)
		if trimmed == "" {
			info(StyleOK.Render("working tree clean"))
		} else {
			info(StyleWarn.Render("uncommitted changes:"))
			for _, line := range strings.Split(trimmed, "\n") {
				added := strings.HasPrefix(line, "A") || strings.HasPrefix(line, "M")
				untracked := strings.HasPrefix(line, "??")
				var lineStyle lipgloss.Style
				switch {
				case added:
					lineStyle = docValOK
				case untracked:
					lineStyle = docHintStyle
				default:
					lineStyle = StyleMuted
				}
				fmt.Fprintf(out, "       %s\n", lineStyle.Render(line))
			}
		}
	}
	fmt.Fprintln(out)

	lsRemote := exec.Command("git", "ls-remote", "--exit-code", remote, pushBranch) //nolint:gosec
	lsRemote.Dir = cfg.RepoPath
	lsOut, lsErr := lsRemote.CombinedOutput()
	reachable := lsErr == nil
	lsMsg := strings.TrimSpace(string(lsOut))
	// Distinguish "remote unreachable" from "branch not found on remote"
	branchMissing := lsErr != nil && lsMsg == ""
	var lsFix string
	if branchMissing {
		lsFix = fmt.Sprintf(`branch %q not found on remote — update git_branch in config or push branch first`, pushBranch)
	} else {
		lsFix = "check SSH keys / git remote -v"
	}
	check(fmt.Sprintf("git remote %s reachable (branch: %s)",
		StyleCode.Render(remote), StyleCode.Render(pushBranch)),
		reachable, lsFix)
	if lsErr != nil && lsMsg != "" {
		fmt.Fprintf(out, "       %s\n", StyleMuted.Render(lsMsg))
	}

	// ── 5. Validate ───────────────────────────────────────────────────────────
	section("Validate command (read-only)")
	fmt.Fprintln(out)
	info("running: " + StyleMuted.Render(cfg.ValidateCommand))
	fmt.Fprintln(out)

	result := caddyeditor.Validate(cfg)
	check("validate_command succeeded", result.OK, "check validate_command in config")

	if result.Output != "" {
		fmt.Fprintln(out)
		for _, line := range strings.Split(strings.TrimSpace(result.Output), "\n") {
			switch {
			case strings.HasPrefix(line, "Error:"):
				fmt.Fprintf(out, "       %s\n", StyleFail.Render(line))
			case strings.Contains(line, `"level":"warn"`):
				// Extract just the "msg" field from the JSON log line for brevity
				msg := extractJSONField(line, "msg")
				if msg == "" {
					msg = line
				}
				fmt.Fprintf(out, "       %s  %s\n", SymWarn, StyleWarn.Render(msg))
			case line == "Valid configuration":
				fmt.Fprintf(out, "       %s  %s\n", SymOK, StyleOK.Render(line))
			case strings.Contains(line, `"level":"debug"`):
				// skip — too verbose
			case strings.Contains(line, `"level":"info"`) && strings.Contains(line, `"msg":"adapted config to JSON"`):
				// skip — not useful
			case strings.Contains(line, `"level":"info"`):
				msg := extractJSONField(line, "msg")
				if msg == "" {
					msg = line
				}
				fmt.Fprintf(out, "       %s  %s\n", SymInfo, StyleMuted.Render(msg))
			}
		}
	}

	// ── 6. Deploy preview ─────────────────────────────────────────────────────
	section("Deploy command (NOT run)")
	fmt.Fprintln(out)
	info("would run:")
	fmt.Fprintf(out, "       %s\n", docFixStyle.Render(cfg.DeployCommand))
	fmt.Fprintln(out)
	kv("git_auto_commit", fmt.Sprintf("%v", cfg.GitAutoCommit), true)
	kv("git_auto_push", fmt.Sprintf("%v", cfg.GitAutoPush), true)

	// ── Summary ───────────────────────────────────────────────────────────────
	fmt.Fprintln(out)
	fmt.Fprintln(out, StyleSection.Render(strings.Repeat("─", 50)))
	fmt.Fprintln(out)
	if pass {
		fmt.Fprintln(out, docSummaryOK.Render("  ✓  All checks passed — Caddy Editor is ready  "))
	} else {
		lines := fmt.Sprintf("  ✗  %d check%s failed:\n", failCount, pluralS(failCount))
		for _, f := range failures {
			lines += fmt.Sprintf("       • %s\n", f)
		}
		lines = strings.TrimRight(lines, "\n")
		fmt.Fprintln(out, docSummaryFail.Render(lines))
	}
	fmt.Fprintln(out)

	if !pass {
		return exitCode(1)
	}
	return nil
}

// extractJSONField pulls a string field value from a single-line JSON log entry.
func extractJSONField(line, field string) string {
	needle := `"` + field + `":"`
	idx := strings.Index(line, needle)
	if idx < 0 {
		return ""
	}
	rest := line[idx+len(needle):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func pluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// or returns a if non-empty, else b.
func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
