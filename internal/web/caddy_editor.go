package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jeeftor/caddy-dns-sync/internal/caddyeditor"
	"github.com/jeeftor/caddy-dns-sync/internal/logging"
)

// CaddyEntryRequest is the payload for POST/PUT caddy entry endpoints.
type CaddyEntryRequest struct {
	Hostname      string          `json:"hostname"`
	Upstream      string          `json:"upstream"`
	Template      string          `json:"template"`
	Options       map[string]bool `json:"options,omitempty"`
	CommitMessage string          `json:"commit_message,omitempty"`
}

// CaddyEntryResponse represents a single parsed Caddyfile site block.
type CaddyEntryResponse struct {
	Hostname   string   `json:"hostname"`
	Upstream   string   `json:"upstream"`
	Directives []string `json:"directives"`
	SourceFile string   `json:"source_file"`
	Raw        string   `json:"raw"`
}

// CaddyEntriesResponse is the list response for GET /api/caddy/entries.
type CaddyEntriesResponse struct {
	Entries []CaddyEntryResponse     `json:"entries"`
	Editor  caddyeditor.EditorConfig `json:"editor"`
}

// CaddyDiffResponse holds the current git diff for the repo.
type CaddyDiffResponse struct {
	Diff   string `json:"diff"`
	Status string `json:"status"`
}

// CaddyTemplatesResponse lists available entry templates.
type CaddyTemplatesResponse struct {
	Templates []string `json:"templates"`
	Default   string   `json:"default"`
}

// CaddyPreviewResponse holds a rendered template preview.
type CaddyPreviewResponse struct {
	Content string `json:"content"`
}

// caddyEditorConfig returns the caddy editor config from the server's loaded config.
func (s *Server) caddyEditorConfig() (caddyeditor.EditorConfig, error) {
	configPath, err := s.configPath()
	if err != nil {
		return caddyeditor.EditorConfig{}, err
	}
	data, err := readFileBytes(configPath)
	if err != nil {
		return caddyeditor.DefaultEditorConfig(), nil
	}
	cfg, err := parseEditorConfig(data)
	if err != nil {
		return caddyeditor.DefaultEditorConfig(), nil
	}
	return cfg, nil
}

func parseEditorConfig(data []byte) (caddyeditor.EditorConfig, error) {
	var wrapper struct {
		CaddyEditor caddyeditor.EditorConfig `json:"caddy_editor"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return caddyeditor.EditorConfig{}, err
	}
	return wrapper.CaddyEditor, nil
}

// handleCaddyEntries handles GET/POST /api/caddy/entries.
func (s *Server) handleCaddyEntries(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getCaddyEntries(w, r)
	case http.MethodPost:
		s.createCaddyEntry(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

// handleCaddyEntry handles PUT/DELETE /api/caddy/entries/{host}.
func (s *Server) handleCaddyEntry(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		s.updateCaddyEntry(w, r)
	case http.MethodDelete:
		s.deleteCaddyEntry(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

func (s *Server) getCaddyEntries(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.caddyEditorConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !cfg.Enabled || cfg.RepoPath == "" {
		writeJSON(w, http.StatusOK, CaddyEntriesResponse{Entries: []CaddyEntryResponse{}, Editor: cfg})
		return
	}
	blocks, err := caddyeditor.ParseCaddyfileFromConfig(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("parsing caddyfile: %w", err))
		return
	}
	entries := make([]CaddyEntryResponse, 0, len(blocks))
	for _, b := range blocks {
		entries = append(entries, blockToResponse(b))
	}
	writeJSON(w, http.StatusOK, CaddyEntriesResponse{Entries: entries, Editor: cfg})
}

func (s *Server) createCaddyEntry(w http.ResponseWriter, r *http.Request) {
	if err := s.allowMutation(r); err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req CaddyEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request: %w", err))
		return
	}
	if req.Hostname == "" || req.Upstream == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("hostname and upstream are required"))
		return
	}
	cfg, err := s.caddyEditorConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	tmpl := req.Template
	if tmpl == "" {
		tmpl = cfg.EntryTemplate
	}
	if tmpl == "" {
		tmpl = "default"
	}
	block := caddyeditor.SiteBlock{Hostname: req.Hostname, Upstream: req.Upstream}
	if err := caddyeditor.AddEntry(cfg, block, tmpl); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("writing entry: %w", err))
		return
	}
	if cfg.GitAutoCommit {
		msg := req.CommitMessage
		if msg == "" {
			msg = fmt.Sprintf("caddy: add %s", req.Hostname)
		}
		_ = caddyeditor.GitAdd(cfg)
		_ = caddyeditor.GitCommit(cfg, msg)
		if cfg.GitAutoPush {
			_ = caddyeditor.GitPush(cfg)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "hostname": req.Hostname})
}

func (s *Server) updateCaddyEntry(w http.ResponseWriter, r *http.Request) {
	if err := s.allowMutation(r); err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	hostname := hostnameFromPath(r.URL.Path, "/api/caddy/entries/")
	if hostname == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("hostname required in path"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req CaddyEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request: %w", err))
		return
	}
	if req.Upstream == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("upstream is required"))
		return
	}
	cfg, err := s.caddyEditorConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	tmpl := req.Template
	if tmpl == "" {
		tmpl = cfg.EntryTemplate
	}
	if tmpl == "" {
		tmpl = "default"
	}
	block := caddyeditor.SiteBlock{Hostname: hostname, Upstream: req.Upstream}
	data := caddyeditor.TemplateData{
		Hostname: hostname,
		Upstream: req.Upstream,
		Options:  req.Options,
	}
	if err := caddyeditor.UpdateEntry(cfg, block, tmpl, data); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("updating entry: %w", err))
		return
	}
	if cfg.GitAutoCommit {
		msg := req.CommitMessage
		if msg == "" {
			msg = fmt.Sprintf("caddy: update %s", hostname)
		}
		_ = caddyeditor.GitAdd(cfg)
		_ = caddyeditor.GitCommit(cfg, msg)
		if cfg.GitAutoPush {
			_ = caddyeditor.GitPush(cfg)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "hostname": hostname})
}

func (s *Server) deleteCaddyEntry(w http.ResponseWriter, r *http.Request) {
	if err := s.allowMutation(r); err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	hostname := hostnameFromPath(r.URL.Path, "/api/caddy/entries/")
	if hostname == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("hostname required in path"))
		return
	}
	cfg, err := s.caddyEditorConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := caddyeditor.RemoveEntry(cfg, hostname); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("removing entry: %w", err))
		return
	}
	if cfg.GitAutoCommit {
		msg := fmt.Sprintf("caddy: remove %s", hostname)
		_ = caddyeditor.GitAdd(cfg)
		_ = caddyeditor.GitCommit(cfg, msg)
		if cfg.GitAutoPush {
			_ = caddyeditor.GitPush(cfg)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "hostname": hostname})
}

func (s *Server) handleCaddyDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	cfg, err := s.caddyEditorConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	diff, _ := caddyeditor.GitDiff(cfg)
	status, _ := caddyeditor.GitStatus(cfg)
	writeJSON(w, http.StatusOK, CaddyDiffResponse{Diff: diff, Status: status})
}

// handleCaddyGitStatus runs git fetch and returns remote-ahead/local-ahead counts.
// GET /api/caddy/git/status
func (s *Server) handleCaddyGitStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	cfg, err := s.caddyEditorConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	st := caddyeditor.GitFetchRemoteStatus(cfg)
	writeJSON(w, http.StatusOK, st)
}

// handleCaddyGitPull runs git pull and returns the output.
// POST /api/caddy/git/pull  (requires mutation token)
func (s *Server) handleCaddyGitPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	if err := s.allowMutation(r); err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	cfg, err := s.caddyEditorConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logging.Info("caddy: git pull starting", "repo", cfg.RepoPath)
	out, err := caddyeditor.GitPull(cfg)
	if err != nil {
		logging.Error("caddy: git pull failed", "error", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logging.Info("caddy: git pull completed", "output", out)
	writeJSON(w, http.StatusOK, map[string]string{"output": out, "status": "ok"})
}

func (s *Server) handleCaddyValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	cfg, err := s.caddyEditorConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	result := caddyeditor.Validate(cfg)
	writeJSON(w, http.StatusOK, result)
}

// handleCaddyValidateDraft validates a draft entry without writing to disk.
func (s *Server) handleCaddyValidateDraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req CaddyEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request: %w", err))
		return
	}
	if req.Hostname == "" || req.Upstream == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("hostname and upstream are required"))
		return
	}
	cfg, err := s.caddyEditorConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	block := caddyeditor.SiteBlock{Hostname: req.Hostname, Upstream: req.Upstream}
	result := caddyeditor.ValidateDraft(cfg, block, req.Template)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCaddyDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	if err := s.allowMutation(r); err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	var opts struct {
		CommitMessage string `json:"commit_message"`
		SkipValidate  bool   `json:"skip_validate"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	_ = json.NewDecoder(r.Body).Decode(&opts)

	cfg, err := s.caddyEditorConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Stream deploy output via Server-Sent Events.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, canFlush := w.(http.Flusher)

	pr := &sseWriter{w: w, flusher: flusher, canFlush: canFlush}

	result := caddyeditor.DeployPipeline(cfg, caddyeditor.DeployPipelineOptions{
		SkipValidate:  opts.SkipValidate,
		CommitMessage: opts.CommitMessage,
	}, pr)

	// Send final done event.
	status := "ok"
	if !result.OK {
		status = "error"
	}
	_, _ = fmt.Fprintf(w, "data: {\"done\":true,\"status\":%q}\n\n", status)
	if canFlush {
		flusher.Flush()
	}
}

func (s *Server) handleCaddyTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	cfg, err := s.caddyEditorConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	templates := caddyeditor.ListTemplates(cfg.RepoPath)
	def := cfg.EntryTemplate
	if def == "" {
		def = "default"
	}
	writeJSON(w, http.StatusOK, CaddyTemplatesResponse{Templates: templates, Default: def})
}

func (s *Server) handleCaddyPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	q := r.URL.Query()
	hostname := strings.TrimSpace(q.Get("hostname"))
	upstream := strings.TrimSpace(q.Get("upstream"))
	tmplName := strings.TrimSpace(q.Get("template"))
	if tmplName == "" {
		tmplName = "default"
	}
	cfg, err := s.caddyEditorConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	content, err := caddyeditor.RenderTemplate(cfg.RepoPath, tmplName, caddyeditor.TemplateData{
		Hostname: hostname,
		Upstream: upstream,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, CaddyPreviewResponse{Content: content})
}

// sseWriter wraps http.ResponseWriter to write SSE data: lines.
type sseWriter struct {
	w        http.ResponseWriter
	flusher  http.Flusher
	canFlush bool
}

func (s *sseWriter) Write(p []byte) (int, error) {
	text := strings.TrimRight(string(p), "\n")
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		payload, _ := json.Marshal(map[string]string{"line": line})
		_, _ = fmt.Fprintf(s.w, "data: %s\n\n", payload)
	}
	if s.canFlush {
		s.flusher.Flush()
	}
	return len(p), nil
}

// hostnameFromPath extracts the trailing path component after prefix.
func hostnameFromPath(path, prefix string) string {
	after := strings.TrimPrefix(path, prefix)
	decoded, err := url.PathUnescape(after)
	if err != nil {
		return after
	}
	return decoded
}

// readFileBytes reads the file at path.
func readFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func blockToResponse(b caddyeditor.SiteBlock) CaddyEntryResponse {
	return CaddyEntryResponse{
		Hostname:   b.Hostname,
		Upstream:   b.Upstream,
		Directives: b.Directives,
		SourceFile: b.SourceFile,
		Raw:        b.Raw,
	}
}
