package caddyeditor

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/jeeftor/caddy-dns-sync/internal/logging"
)

// DeployResult holds the final outcome of a deploy pipeline.
type DeployResult struct {
	OK     bool   `json:"ok"`
	Output string `json:"output"`
}

// DeployPipelineOptions controls the deploy pipeline behaviour.
type DeployPipelineOptions struct {
	// SkipValidate skips the validate step.
	SkipValidate bool
	// CommitMessage overrides the auto-generated git commit message.
	CommitMessage string
}

// DeployPipeline runs validate → git add → commit → (push) → deploy.
// Progress lines are written to w as they arrive so the caller can stream them.
func DeployPipeline(cfg EditorConfig, opts DeployPipelineOptions, w io.Writer) DeployResult {
	writeLine := func(line string) {
		_, _ = fmt.Fprintln(w, line)
		logging.Info("deploy: " + line)
	}
	writeError := func(line string) {
		_, _ = fmt.Fprintln(w, line)
		logging.Error("deploy: " + line)
	}

	// 1. Validate
	if !opts.SkipValidate && cfg.ValidateCommand != "" {
		writeLine("Validating config...")
		result := Validate(cfg)
		if result.Output != "" {
			writeLine(result.Output)
		}
		if !result.OK {
			writeError("FAILED: validation error")
			return DeployResult{OK: false, Output: "validation failed"}
		}
		writeLine("OK: config valid")
	}

	// 2. Git add
	if cfg.GitAutoCommit {
		writeLine("Staging changes...")
		if err := GitAdd(cfg); err != nil {
			writeError(fmt.Sprintf("FAILED: %v", err))
			return DeployResult{OK: false, Output: err.Error()}
		}

		// Check if there is anything to commit.
		status, _ := GitStatus(cfg)
		if status == "" {
			writeLine("Nothing to commit.")
		} else {
			msg := opts.CommitMessage
			if msg == "" {
				msg = "caddy: update Caddyfile entries"
			}
			writeLine(fmt.Sprintf("Committing: %s", msg))
			if err := GitCommit(cfg, msg); err != nil {
				writeError(fmt.Sprintf("FAILED: %v", err))
				return DeployResult{OK: false, Output: err.Error()}
			}
			writeLine("OK: committed")
		}

		if cfg.GitAutoPush {
			writeLine("Pushing to remote...")
			if err := GitPush(cfg); err != nil {
				writeError(fmt.Sprintf("FAILED: %v", err))
				return DeployResult{OK: false, Output: err.Error()}
			}
			writeLine("OK: pushed")
		}
	}

	// 3. Deploy
	if cfg.DeployCommand == "" {
		writeLine("No deploy_command configured — skipping deploy step.")
		return DeployResult{OK: true}
	}

	writeLine(fmt.Sprintf("Running: %s", cfg.DeployCommand))
	cmd := exec.Command("sh", "-c", cfg.DeployCommand) //nolint:gosec
	cmd.Dir = cfg.RepoPath
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		writeError(fmt.Sprintf("FAILED: %v", err))
		return DeployResult{OK: false, Output: err.Error()}
	}
	writeLine("Done.")
	return DeployResult{OK: true}
}
