// Package ownership persists explicit provider-resource ownership.
package ownership

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const stateVersion = 1

// Resource identifies one explicitly adopted or tool-created provider resource.
// Identity is provider-specific and intentionally never inferred from a hostname
// or a destination value.
type Resource struct {
	Provider string    `json:"provider"`
	Kind     string    `json:"kind"`
	ID       string    `json:"id"`
	Expected string    `json:"expected,omitempty"`
	OwnedAt  time.Time `json:"owned_at"`
}

// Intent records a provider mutation before it is attempted. An intent left on
// disk signals an interrupted or ambiguous operation and must be reviewed
// instead of being silently retried.
type Intent struct {
	Operation string    `json:"operation"`
	Provider  string    `json:"provider"`
	Kind      string    `json:"kind"`
	ID        string    `json:"id"`
	Expected  string    `json:"expected,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// State is the versioned on-disk ownership document.
type State struct {
	Version   int                 `json:"version"`
	Resources map[string]Resource `json:"resources"`
	Intents   map[string]Intent   `json:"intents,omitempty"`
}

// Key returns a stable key for a provider-specific resource identity.
func Key(provider, kind, id string) string {
	return strings.ToLower(strings.TrimSpace(provider)) + ":" +
		strings.ToLower(strings.TrimSpace(kind)) + ":" + strings.TrimSpace(id)
}

// Load returns an empty state for a missing file and rejects unknown versions.
func Load(path string) (State, error) {
	state := State{Version: stateVersion, Resources: map[string]Resource{}, Intents: map[string]Intent{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("read ownership state: %w", err)
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse ownership state: %w", err)
	}
	if state.Version != stateVersion {
		return State{}, fmt.Errorf("unsupported ownership state version %d", state.Version)
	}
	if state.Resources == nil {
		state.Resources = map[string]Resource{}
	}
	if state.Intents == nil {
		state.Intents = map[string]Intent{}
	}
	return state, nil
}

// Owns reports whether the exact provider resource was explicitly recorded.
func (s State) Owns(provider, kind, id string) bool {
	if _, unresolved := s.Intents[Key(provider, kind, id)]; unresolved {
		return false
	}
	_, ok := s.Resources[Key(provider, kind, id)]
	return ok
}

// Record adds or refreshes an explicit ownership record after verified readback.
func (s *State) Record(resource Resource) error {
	resource.Provider = strings.ToLower(strings.TrimSpace(resource.Provider))
	resource.Kind = strings.ToLower(strings.TrimSpace(resource.Kind))
	resource.ID = strings.TrimSpace(resource.ID)
	if resource.Provider == "" || resource.Kind == "" || resource.ID == "" {
		return errors.New("provider, kind, and resource ID are required")
	}
	if resource.OwnedAt.IsZero() {
		resource.OwnedAt = time.Now().UTC()
	}
	if s.Version == 0 {
		s.Version = stateVersion
	}
	if s.Version != stateVersion {
		return fmt.Errorf("unsupported ownership state version %d", s.Version)
	}
	if s.Resources == nil {
		s.Resources = map[string]Resource{}
	}
	if s.Intents == nil {
		s.Intents = map[string]Intent{}
	}
	s.Resources[Key(resource.Provider, resource.Kind, resource.ID)] = resource
	return nil
}

// Forget removes a resource only after the provider mutation has been verified.
func (s *State) Forget(provider, kind, id string) {
	delete(s.Resources, Key(provider, kind, id))
}

// Begin records a mutation intent before a provider write.
func (s *State) Begin(intent Intent) error {
	intent.Operation = strings.ToLower(strings.TrimSpace(intent.Operation))
	intent.Provider = strings.ToLower(strings.TrimSpace(intent.Provider))
	intent.Kind = strings.ToLower(strings.TrimSpace(intent.Kind))
	intent.ID = strings.TrimSpace(intent.ID)
	if intent.Operation == "" || intent.Provider == "" || intent.Kind == "" || intent.ID == "" {
		return errors.New("operation, provider, kind, and resource ID are required")
	}
	if intent.CreatedAt.IsZero() {
		intent.CreatedAt = time.Now().UTC()
	}
	if s.Version == 0 {
		s.Version = stateVersion
	}
	if s.Version != stateVersion {
		return fmt.Errorf("unsupported ownership state version %d", s.Version)
	}
	if s.Intents == nil {
		s.Intents = map[string]Intent{}
	}
	s.Intents[Key(intent.Provider, intent.Kind, intent.ID)] = intent
	return nil
}

// Resolve clears an intent only after provider readback confirms the outcome.
func (s *State) Resolve(provider, kind, id string) {
	delete(s.Intents, Key(provider, kind, id))
}

// HasIntent reports an operation that requires recovery rather than retry.
func (s State) HasIntent(provider, kind, id string) bool {
	_, ok := s.Intents[Key(provider, kind, id)]
	return ok
}

// Save atomically replaces the state file with restrictive permissions.
func Save(path string, state State) error {
	if state.Version == 0 {
		state.Version = stateVersion
	}
	if state.Version != stateVersion {
		return fmt.Errorf("unsupported ownership state version %d", state.Version)
	}
	if state.Resources == nil {
		state.Resources = map[string]Resource{}
	}
	if state.Intents == nil {
		state.Intents = map[string]Intent{}
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal ownership state: %w", err)
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create ownership state directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".ownership-*")
	if err != nil {
		return fmt.Errorf("create ownership state temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("set ownership state permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write ownership state: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync ownership state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close ownership state: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace ownership state: %w", err)
	}
	return nil
}

// PathForConfig keeps ownership state beside its selected configuration file.
func PathForConfig(configPath string) string {
	dir := filepath.Dir(configPath)
	base := strings.TrimSuffix(filepath.Base(configPath), filepath.Ext(configPath))
	return filepath.Join(dir, base+".ownership.json")
}

// Fingerprint returns a stable value fingerprint suitable for drift detection
// without persisting sensitive values verbatim.
func Fingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("sha256:%x", sum[:])
}

// ResourcesForProvider returns deterministic records for status and adoption UIs.
func (s State) ResourcesForProvider(provider string) []Resource {
	provider = strings.ToLower(strings.TrimSpace(provider))
	resources := make([]Resource, 0)
	for _, resource := range s.Resources {
		if resource.Provider == provider {
			resources = append(resources, resource)
		}
	}
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Kind == resources[j].Kind {
			return resources[i].ID < resources[j].ID
		}
		return resources[i].Kind < resources[j].Kind
	})
	return resources
}
