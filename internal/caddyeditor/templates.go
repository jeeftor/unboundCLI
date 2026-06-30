package caddyeditor

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateData is passed to every entry template during rendering.
type TemplateData struct {
	Hostname    string
	Upstream    string
	MatcherName string          // auto-derived from hostname if empty
	Options     map[string]bool // optional flags (e.g. tls_insecure_skip_verify)
}

// builtinTemplates generate a @matcher + handle block for insertion into a wildcard Caddyfile.
// The Caddy {placeholder} syntax conflicts with Go templates, so we use [[ ]] delimiters.
var builtinTemplates = map[string]string{
	// default: uses the shared proxy_headers snippet (import proxy_headers)
	// which must be defined in your Caddyfile as a snippet, e.g.:
	//   (proxy_headers) {
	//       header_up Host {upstream_hostport}
	//       header_up X-Real-IP {remote_host}
	//   }
	"default": `@[[ .MatcherName ]] host [[ .Hostname ]]
handle @[[ .MatcherName ]] {
	reverse_proxy [[ .Upstream ]] {
		import proxy_headers
	}
}`,

	"no-tls-verify": `@[[ .MatcherName ]] host [[ .Hostname ]]
handle @[[ .MatcherName ]] {
	reverse_proxy [[ .Upstream ]] {
		import proxy_headers
		transport http {
			tls_insecure_skip_verify
		}
	}
}`,

	"compression": `@[[ .MatcherName ]] host [[ .Hostname ]]
handle @[[ .MatcherName ]] {
	encode gzip zstd
	reverse_proxy [[ .Upstream ]] {
		import proxy_headers
	}
}`,

	"simple": `@[[ .MatcherName ]] host [[ .Hostname ]]
handle @[[ .MatcherName ]] {
	reverse_proxy [[ .Upstream ]]
}`,

	"headers-inline": `@[[ .MatcherName ]] host [[ .Hostname ]]
handle @[[ .MatcherName ]] {
	reverse_proxy [[ .Upstream ]] {
		header_up Host {upstream_hostport}
		header_up X-Real-IP {remote_host}
	}
}`,
}

// ListTemplates returns the names of available templates (built-in + custom).
func ListTemplates(repoPath string) []string {
	names := make([]string, 0, len(builtinTemplates))
	for name := range builtinTemplates {
		names = append(names, name)
	}
	if repoPath != "" {
		names = append(names, loadCustomTemplateNames(repoPath)...)
	}
	return names
}

// RenderTemplate renders the named template with the given data.
// If MatcherName is empty it is derived from Hostname.
func RenderTemplate(repoPath, templateName string, data TemplateData) (string, error) {
	if data.MatcherName == "" {
		data.MatcherName = matcherNameFromHostname(data.Hostname)
	}
	tmplStr, err := lookupTemplate(repoPath, templateName)
	if err != nil {
		return "", err
	}
	return renderTemplateStr(tmplStr, data)
}

// renderTemplateStr executes a template string with [[ ]] delimiters.
func renderTemplateStr(tmplStr string, data TemplateData) (string, error) {
	tmpl, err := template.New("entry").Delims("[[", "]]").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering template: %w", err)
	}
	return buf.String(), nil
}

func lookupTemplate(repoPath, name string) (string, error) {
	if s, ok := builtinTemplates[name]; ok {
		return s, nil
	}
	if repoPath != "" {
		path := filepath.Join(repoPath, "templates", name+".caddytemplate")
		data, err := readFileString(path)
		if err == nil {
			return data, nil
		}
	}
	return "", fmt.Errorf("unknown template %q", name)
}

func loadCustomTemplateNames(repoPath string) []string {
	dir := filepath.Join(repoPath, "templates")
	entries, err := readdirNames(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, name := range entries {
		if strings.HasSuffix(name, ".caddytemplate") {
			names = append(names, strings.TrimSuffix(name, ".caddytemplate"))
		}
	}
	return names
}
