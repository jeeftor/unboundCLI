// Package caddyeditor provides file-based editing, validation, and deployment
// of Caddyfile reverse-proxy entries from within CaddySync.
package caddyeditor

// EditorConfig holds the user-supplied caddy_editor section from the config file.
type EditorConfig struct {
	Enabled         bool   `json:"enabled" mapstructure:"enabled"`
	RepoPath        string `json:"repo_path" mapstructure:"repo_path"`
	CaddyfilePath   string `json:"caddyfile" mapstructure:"caddyfile"` // relative to repo_path, e.g. "caddy/Caddyfile"
	DeployCommand   string `json:"deploy_command" mapstructure:"deploy_command"`
	ValidateCommand string `json:"validate_command" mapstructure:"validate_command"`
	GitAutoCommit   bool   `json:"git_auto_commit" mapstructure:"git_auto_commit"`
	GitAutoPush     bool   `json:"git_auto_push" mapstructure:"git_auto_push"`
	GitRemote       string `json:"git_remote" mapstructure:"git_remote"`
	GitBranch       string `json:"git_branch" mapstructure:"git_branch"`
	EntryTemplate   string `json:"entry_template" mapstructure:"entry_template"`
}

// DefaultEditorConfig returns an EditorConfig with sensible defaults.
func DefaultEditorConfig() EditorConfig {
	return EditorConfig{
		CaddyfilePath: "caddy/Caddyfile",
		GitRemote:     "origin",
		EntryTemplate: "default",
	}
}
