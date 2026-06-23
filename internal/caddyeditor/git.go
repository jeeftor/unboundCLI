package caddyeditor

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// RemoteStatus describes how many commits the remote is ahead/behind local HEAD.
type RemoteStatus struct {
	RemoteAhead int    `json:"remote_ahead"` // commits in remote not yet in local
	LocalAhead  int    `json:"local_ahead"`  // local commits not yet pushed
	Branch      string `json:"branch"`
	Remote      string `json:"remote"`
	FetchError  string `json:"fetch_error,omitempty"`
}

// GitFetchRemoteStatus fetches from remote and returns the commit delta.
func GitFetchRemoteStatus(cfg EditorConfig) RemoteStatus {
	remote := cfg.GitRemote
	if remote == "" {
		remote = "origin"
	}
	branch := cfg.GitBranch
	if branch == "" {
		if out, err := runGit(cfg.RepoPath, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
			branch = strings.TrimSpace(out)
		}
	}
	if branch == "" {
		branch = "main"
	}

	st := RemoteStatus{Remote: remote, Branch: branch}

	// Fetch quietly — this is the network call.
	if _, err := runGit(cfg.RepoPath, "fetch", "--quiet", remote); err != nil {
		st.FetchError = err.Error()
		return st
	}

	upstream := remote + "/" + branch

	// Commits in remote not yet in local (remote ahead).
	if out, err := runGit(cfg.RepoPath, "rev-list", "--count", "HEAD.."+upstream); err == nil {
		fmt.Sscanf(strings.TrimSpace(out), "%d", &st.RemoteAhead)
	}
	// Local commits not yet pushed (local ahead).
	if out, err := runGit(cfg.RepoPath, "rev-list", "--count", upstream+"..HEAD"); err == nil {
		fmt.Sscanf(strings.TrimSpace(out), "%d", &st.LocalAhead)
	}

	return st
}

// GitPull pulls from the configured remote.
func GitPull(cfg EditorConfig) (string, error) {
	remote := cfg.GitRemote
	if remote == "" {
		remote = "origin"
	}
	out, err := runGit(cfg.RepoPath, "pull", remote)
	return out, err
}

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
