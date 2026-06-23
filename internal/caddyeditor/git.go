package caddyeditor

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GitDiff returns the git diff output for the repo.
func GitDiff(cfg EditorConfig) (string, error) {
	out, err := runGit(cfg.RepoPath, "diff", "HEAD")
	if err != nil {
		// If there's nothing committed yet, try without HEAD.
		out2, err2 := runGit(cfg.RepoPath, "diff")
		if err2 != nil {
			return "", fmt.Errorf("git diff: %w", err)
		}
		return out2, nil
	}
	return out, nil
}

// GitStatus returns a short git status summary.
func GitStatus(cfg EditorConfig) (string, error) {
	return runGit(cfg.RepoPath, "status", "--short")
}

// GitAdd stages all changes in the repo.
func GitAdd(cfg EditorConfig) error {
	_, err := runGit(cfg.RepoPath, "add", ".")
	return err
}

// GitCommit creates a commit with the given message.
func GitCommit(cfg EditorConfig, message string) error {
	_, err := runGit(cfg.RepoPath, "commit", "-m", message)
	return err
}

// GitPush pushes the branch to the configured remote.
func GitPush(cfg EditorConfig) error {
	remote := cfg.GitRemote
	if remote == "" {
		remote = "origin"
	}
	branch := cfg.GitBranch
	if branch == "" {
		branch = "main"
	}
	_, err := runGit(cfg.RepoPath, "push", remote, branch)
	return err
}

// runGit runs a git sub-command in the given directory and returns stdout.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...) //nolint:gosec
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		combined := strings.TrimSpace(stderr.String())
		if combined == "" {
			combined = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), combined)
	}
	return stdout.String(), nil
}
