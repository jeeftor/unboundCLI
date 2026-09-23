package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jeeftor/caddy-dns-sync/internal/api"
	"github.com/jeeftor/caddy-dns-sync/internal/config"
	"github.com/jeeftor/caddy-dns-sync/internal/ownership"
)

type adoptionCandidate struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
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

// handleAdoption previews and confirms explicit ownership adoption.
func (s *Server) handleAdoption(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		switch r.URL.Query().Get("provider") {
		case "adguard":
			s.previewAdguardAdoption(w, r)
		case "cloudflare":
			s.previewCloudflareAdoption(w, r)
		default:
			writeError(w, http.StatusBadRequest, fmt.Errorf("provider must be adguard or cloudflare"))
		}
	case http.MethodPost:
		s.confirmAdoption(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

func (s *Server) previewAdguardAdoption(w http.ResponseWriter, r *http.Request) {
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
		response.Candidates = append(response.Candidates, adoptionCandidate{ID: rewrite.Domain, Kind: "rewrite", Current: rewrite.Answer})
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

func (s *Server) confirmAdoption(w http.ResponseWriter, r *http.Request) {
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
	current, kinds, err := s.currentAdoptionResources(preview.provider)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	state, path, err := s.loadOwnershipState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	for _, id := range request.IDs {
		expected, ok := preview.candidates[id]
		if !ok || current[id] != expected {
			writeError(w, http.StatusConflict, fmt.Errorf("resource %s changed; preview adoption again", id))
			return
		}
		if err := state.Record(ownership.Resource{Provider: preview.provider, Kind: kinds[id], ID: adoptionResourceID(id), Expected: ownership.Fingerprint(expected)}); err != nil {
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

func (s *Server) previewCloudflareAdoption(w http.ResponseWriter, r *http.Request) {
	state, path, err := s.loadOwnershipState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	candidates, kinds, err := s.currentAdoptionResources("cloudflare")
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	response := adoptionResponse{Candidates: make([]adoptionCandidate, 0, len(candidates))}
	for id, current := range candidates {
		kind := kinds[id]
		resourceID := adoptionResourceID(id)
		if state.Owns("cloudflare", kind, resourceID) || state.HasIntent("cloudflare", kind, resourceID) {
			continue
		}
		response.Candidates = append(response.Candidates, adoptionCandidate{ID: id, Kind: kind, Current: current})
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
	s.adoptions[id] = adoptionPreview{provider: "cloudflare", candidates: candidates, configPath: path, configRevision: revision, createdAt: time.Now()}
	s.adoptionMu.Unlock()
	response.PreviewID = id
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) currentAdoptionResources(provider string) (map[string]string, map[string]string, error) {
	runtime := s.runtimeSnapshot()
	if provider == "adguard" {
		if runtime.Clients.Adguard == nil {
			return nil, nil, fmt.Errorf("AdGuard is unavailable in this web session")
		}
		rewrites, err := runtime.Clients.Adguard.ListRewrites()
		if err != nil {
			return nil, nil, fmt.Errorf("list AdGuard rewrites: %w", err)
		}
		resources, kinds := map[string]string{}, map[string]string{}
		for _, rewrite := range rewrites {
			if _, duplicate := resources[rewrite.Domain]; duplicate {
				return nil, nil, fmt.Errorf("multiple AdGuard rewrites exist for %s", rewrite.Domain)
			}
			resources[rewrite.Domain], kinds[rewrite.Domain] = rewrite.Answer, "rewrite"
		}
		return resources, kinds, nil
	}
	if runtime.Clients.Cloudflare == nil {
		return nil, nil, fmt.Errorf("Cloudflare is unavailable in this web session")
	}
	ingress, err := runtime.Clients.Cloudflare.GetAllTunnelsDetails()
	if err != nil {
		return nil, nil, fmt.Errorf("list Cloudflare ingress: %w", err)
	}
	dnsRecords, err := runtime.Clients.Cloudflare.ListTunnelDNSRecords()
	if err != nil {
		return nil, nil, fmt.Errorf("list Cloudflare DNS records: %w", err)
	}
	resources, kinds := map[string]string{}, map[string]string{}
	for _, entry := range ingress {
		id := "ingress:" + entry.TunnelID + ":" + entry.Hostname + ":" + entry.Path
		resources[id], kinds[id] = cloudflareIngressValue(entry), "ingress"
	}
	for _, record := range dnsRecords {
		id := "dns:" + record.ID
		resources[id], kinds[id] = record.Target, "dns"
	}
	return resources, kinds, nil
}

func adoptionResourceID(id string) string {
	return strings.TrimPrefix(strings.TrimPrefix(id, "ingress:"), "dns:")
}

func cloudflareIngressValue(entry api.CloudflareIngressEntry) string {
	return entry.TunnelID + "|" + entry.Hostname + "|" + entry.Path + "|" + entry.Service + "|" + entry.HTTPHostHeader + "|" + entry.OriginServerName
}
