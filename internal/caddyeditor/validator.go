package caddyeditor

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ValidationResult holds the outcome of running the validate command.
type ValidationResult struct {
	OK     bool   `json:"ok"`
	Output string `json:"output"`
}

// Validate runs the configured validate_command and returns the result.
func Validate(cfg EditorConfig) ValidationResult {
	if cfg.ValidateCommand == "" {
		return ValidationResult{OK: true, Output: "no validate_command configured"}
	}
	cmd := exec.Command("sh", "-c", cfg.ValidateCommand) //nolint:gosec
	cmd.Dir = cfg.RepoPath
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	output := strings.TrimSpace(buf.String())
	if err != nil {
		return ValidationResult{OK: false, Output: fmt.Sprintf("%s\n%s", err.Error(), output)}
	}
	return ValidationResult{OK: true, Output: output}
}
