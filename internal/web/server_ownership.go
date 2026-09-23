package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/ownership"
)

type adoptionCandidate struct {
	ID      string `json:"id"`
	Current string `json:"current"`
}

type adoptionPreview struct {
	provider       string
	candidates     map[string]string
	configPath     string
	configRevision string
	createdAt      time.Time
	completed      bool
}

type adoptionResponse struct {
	PreviewID  string              `json:"preview_id"`
	Candidates []adoptionCandidate `json:"candidates"`
}

// handleAdoption previews and confirms explicit ownership adoption. It only
// supports AdGuard rewrites until Cloudflare resource IDs are available.
func (s *Server) handleAdoption(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.previewAdguardAdoption(w, r)
	case http.MethodPost:
		s.confirmAdguardAdoption(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

func (s *Server) previewAdguardAdoption(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("provider") != "adguard" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("only provider=adguard is currently supported"))
		return
	}
	runtime := s.runtimeSnapshot()
	if runtime.Clients.Adguard == nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("AdGuard is unavailable in this web session"))
		return
	}
	state, path, err := s.loadOwnershipState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	rewrites, err := runtime.Clients.Adguard.ListRewrites()
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("list AdGuard rewrites: %w", err))
		return
	}
	candidates := make(map[string]string)
	response := adoptionResponse{Candidates: make([]adoptionCandidate, 0)}
	for _, rewrite := range rewrites {
		if state.Owns("adguard", "rewrite", rewrite.Domain) || state.HasIntent("adguard", "rewrite", rewrite.Domain) {
			continue
		}
		if _, duplicate := candidates[rewrite.Domain]; duplicate {
			writeError(w, http.StatusConflict, fmt.Errorf("multiple AdGuard rewrites exist for %s; resolve the ambiguity before adoption", rewrite.Domain))
			return
		}
		candidates[rewrite.Domain] = rewrite.Answer
		response.Candidates = append(response.Candidates, adoptionCandidate{ID: rewrite.Domain, Current: rewrite.Answer})
	}
	revision, err := config.Revision(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	id, err := newPlanID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.adoptionMu.Lock()
	s.adoptions[id] = adoptionPreview{provider: "adguard", candidates: candidates, configPath: path, configRevision: revision, createdAt: time.Now()}
	s.adoptionMu.Unlock()
	response.PreviewID = id
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) confirmAdguardAdoption(w http.ResponseWriter, r *http.Request) {
	if err := s.allowMutation(r); err != nil {
		writeError(w, http.StatusForbidden, err)
		return
	}
	var request struct {
		PreviewID string   `json:"preview_id"`
		IDs       []string `json:"ids"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil || request.PreviewID == "" || len(request.IDs) == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("preview_id and ids are required"))
		return
	}
	s.adoptionMu.Lock()
	preview, ok := s.adoptions[request.PreviewID]
	if !ok || preview.completed || time.Since(preview.createdAt) > planTTL {
		s.adoptionMu.Unlock()
		writeError(w, http.StatusBadRequest, fmt.Errorf("unknown, completed, or expired adoption preview"))
		return
	}
	preview.completed = true
	s.adoptions[request.PreviewID] = preview
	s.adoptionMu.Unlock()
	if current, err := config.Revision(preview.configPath); err != nil || current != preview.configRevision {
		writeError(w, http.StatusConflict, fmt.Errorf("configuration changed; preview adoption again"))
		return
	}
	runtime := s.runtimeSnapshot()
	if runtime.Clients.Adguard == nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("AdGuard is unavailable in this web session"))
		return
	}
	rewrites, err := runtime.Clients.Adguard.ListRewrites()
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Errorf("verify AdGuard rewrites: %w", err))
		return
	}
	current := map[string]string{}
	for _, rewrite := range rewrites {
		if _, duplicate := current[rewrite.Domain]; duplicate {
			writeError(w, http.StatusConflict, fmt.Errorf("multiple AdGuard rewrites exist for %s; preview adoption again after resolving the ambiguity", rewrite.Domain))
			return
		}
		current[rewrite.Domain] = rewrite.Answer
	}
	state, path, err := s.loadOwnershipState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	for _, id := range request.IDs {
		expected, ok := preview.candidates[id]
		if !ok || current[id] != expected {
			writeError(w, http.StatusConflict, fmt.Errorf("rewrite %s changed; preview adoption again", id))
			return
		}
		if err := state.Record(ownership.Resource{Provider: "adguard", Kind: "rewrite", ID: id, Expected: ownership.Fingerprint(expected)}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if err := ownership.Save(path, state); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"adopted": len(request.IDs)})
}
