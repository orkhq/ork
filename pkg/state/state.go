package state

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ork/pkg/logging"
	manifestcore "ork/pkg/manifest/core"
	"ork/pkg/runners"
)

// Status records Ork's durable belief about a component lifecycle outcome.
// It does not imply that no resources exist after a failure.
type Status string

const (
	StatusApplying   Status = "applying"
	StatusApplied    Status = "applied"
	StatusFailed     Status = "failed"
	StatusDestroying Status = "destroying"
	StatusDestroyed  Status = "destroyed"
)

// Stage identifies the lifecycle checkpoint associated with a component status.
type Stage string

const (
	StageConfig      Stage = "config"
	StagePreApply    Stage = "pre_apply"
	StageApply       Stage = "apply"
	StageOutputs     Stage = "outputs"
	StageArtifacts   Stage = "artifacts"
	StagePostApply   Stage = "post_apply"
	StagePreDestroy  Stage = "pre_destroy"
	StageDestroy     Stage = "destroy"
	StagePostDestroy Stage = "post_destroy"
)

// IsDestroyStage reports whether retry must continue through ork down rather
// than ork up.
func (s Stage) IsDestroyStage() bool {
	switch s {
	case StagePreDestroy, StageDestroy, StagePostDestroy:
		return true
	default:
		return false
	}
}

// RunnerRef identifies the execution context without persisting credentials.
type RunnerRef struct {
	Name string             `json:"name"`
	Type runners.RunnerType `json:"type"`
}

// ComponentStateData is adapter-owned operational data returned after apply.
// Payload must not contain secrets; Artifacts are persisted separately.
type ComponentStateData struct {
	WorkDir   string                 `json:"workdir"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Artifacts []Artifact             `json:"artifacts,omitempty"`
}

// Artifact describes a tool-state file that must survive until teardown.
// Path is relative to the component work directory.
type Artifact struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Required  bool   `json:"required,omitempty"`
	Sensitive bool   `json:"sensitive,omitempty"`
}

// DestroyHooks preserves the hooks that belonged to the applied component so a
// later manifest edit cannot silently replace its teardown contract.
type DestroyHooks struct {
	PreDestroy  []manifestcore.Hook `json:"pre_destroy,omitempty"`
	PostDestroy []manifestcore.Hook `json:"post_destroy,omitempty"`
}

// ComponentState represents the durable lifecycle state of one component.
type ComponentState struct {
	Name               string                       `json:"name"`
	Type               string                       `json:"type"`
	Runner             RunnerRef                    `json:"runner"`
	Source             manifestcore.ComponentSource `json:"source,omitempty"`
	WorkDir            string                       `json:"workdir,omitempty"`
	NonSensitiveConfig map[string]interface{}       `json:"non_sensitive_config,omitempty"`
	Outputs            map[string]string            `json:"outputs,omitempty"`
	Payload            map[string]interface{}       `json:"payload,omitempty"`
	Artifacts          []Artifact                   `json:"artifacts,omitempty"`
	DestroyHooks       DestroyHooks                 `json:"destroy_hooks,omitempty"`
	Status             Status                       `json:"status"`
	Stage              Stage                        `json:"stage,omitempty"`
	ProvisionedAt      string                       `json:"provisioned_at"`
	UpdatedAt          string                       `json:"updated_at"`
	Env                map[string]string            `json:"-"`
}

// OrkState represents the durable state of an entire Ork environment.
type OrkState struct {
	EnvID      string           `json:"env_id"`
	ManifestID string           `json:"manifest_id"`
	Components []ComponentState `json:"components"`
	CreatedAt  string           `json:"created_at"`
	UpdatedAt  string           `json:"updated_at"`

	logger logging.DebugLogger
}

// Manager binds a backend to one environment ID and coordinates state and
// artifact persistence.
type Manager struct {
	envID   string
	backend Backend
}

// NewManager creates a state manager scoped to envID.
func NewManager(envID string, backend Backend) *Manager {
	return &Manager{envID: envID, backend: backend}
}

// New creates an empty environment state document.
func New(envID, manifestID string, logger logging.DebugLogger) *OrkState {
	now := time.Now().UTC().Format(time.RFC3339)
	return &OrkState{
		EnvID:      envID,
		ManifestID: manifestID,
		Components: make([]ComponentState, 0),
		CreatedAt:  now,
		UpdatedAt:  now,
		logger:     logger,
	}
}

// LoadOrNew loads an existing environment or creates an unsaved empty state.
func (sm *Manager) LoadOrNew(manifestID string, logger logging.DebugLogger) (*OrkState, error) {
	exists, err := sm.Exists(context.Background())
	if err != nil {
		return nil, err
	}
	if !exists {
		return New(sm.envID, manifestID, logger), nil
	}

	current, err := sm.Load()
	if err == nil {
		return current, nil
	}

	return nil, err
}

// Load reads the environment state from the configured backend.
func (sm *Manager) Load() (*OrkState, error) {
	return sm.backend.Load(context.Background(), sm.envID)
}

// UpsertComponent adds or replaces component state while preserving its
// original provisioning timestamp.
func (s *OrkState) UpsertComponent(component ComponentState) {
	now := time.Now().UTC().Format(time.RFC3339)
	component.UpdatedAt = now

	for i, existing := range s.Components {
		if existing.Name == component.Name {
			if component.ProvisionedAt == "" {
				component.ProvisionedAt = existing.ProvisionedAt
			}
			s.Components[i] = component
			s.UpdatedAt = now
			return
		}
	}

	if component.ProvisionedAt == "" {
		component.ProvisionedAt = now
	}
	s.Components = append(s.Components, component)
	s.UpdatedAt = now
}

// FindComponent returns a copy of the named component state.
func (s *OrkState) FindComponent(name string) (ComponentState, bool) {
	for _, component := range s.Components {
		if component.Name == name {
			return component, true
		}
	}
	return ComponentState{}, false
}

// BeginComponentApply records destroyable manifest facts before apply-side work
// begins. Existing payload and artifacts are preserved until replacement state
// is safely available.
func (s *OrkState) BeginComponentApply(component *manifestcore.Component, runnerType runners.RunnerType, workDir string, stage Stage) {
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range s.Components {
		if s.Components[i].Name == component.Name {
			s.Components[i].Type = component.Type
			s.Components[i].Runner = RunnerRef{
				Name: component.Runner,
				Type: runnerType,
			}
			s.Components[i].Source = component.Source
			s.Components[i].WorkDir = workDir
			s.Components[i].NonSensitiveConfig = SanitizeMap(component.Config)
			s.Components[i].DestroyHooks = destroyHooksFromComponent(component)
			s.Components[i].Status = StatusApplying
			s.Components[i].Stage = stage
			s.Components[i].UpdatedAt = now
			s.UpdatedAt = now
			return
		}
	}

	// create new if not found.
	s.Components = append(s.Components, ComponentState{
		Name: component.Name,
		Type: component.Type,
		Runner: RunnerRef{
			Name: component.Runner,
			Type: runnerType,
		},
		Source:             component.Source,
		WorkDir:            workDir,
		NonSensitiveConfig: SanitizeMap(component.Config),
		Outputs:            make(map[string]string),
		Payload:            make(map[string]interface{}),
		DestroyHooks:       destroyHooksFromComponent(component),
		Status:             StatusApplying,
		Stage:              stage,
		ProvisionedAt:      now,
		UpdatedAt:          now,
	})
	s.UpdatedAt = now
}

// MarkComponentFailed records an interrupted or failed lifecycle checkpoint.
func (s *OrkState) MarkComponentFailed(name string, stage Stage) {
	s.markComponentStatus(name, StatusFailed, stage)
}

// MarkComponentApplying records an in-progress apply checkpoint.
func (s *OrkState) MarkComponentApplying(name string, stage Stage) {
	s.markComponentStatus(name, StatusApplying, stage)
}

// MarkComponentApplied records a completed apply checkpoint.
func (s *OrkState) MarkComponentApplied(name string, stage Stage) {
	s.markComponentStatus(name, StatusApplied, stage)
}

// MarkComponentDestroying records an in-progress destroy checkpoint.
func (s *OrkState) MarkComponentDestroying(name string, stage Stage) {
	s.markComponentStatus(name, StatusDestroying, stage)
}

// MarkComponentDestroyed records a completed destroy checkpoint.
func (s *OrkState) MarkComponentDestroyed(name string, stage Stage) {
	s.markComponentStatus(name, StatusDestroyed, stage)
}

func (s *OrkState) markComponentStatus(name string, status Status, stage Stage) {
	logger := s.logger
	if logger == nil {
		logger = &logging.NoopDebugLogger{}
	}
	logger.Debug(
		"component status transitioned",
		logging.Field{Key: "name", Value: name},
		logging.Field{Key: "status", Value: status},
		logging.Field{Key: "stage", Value: stage},
	)
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range s.Components {
		if s.Components[i].Name == name {
			s.Components[i].Status = status
			s.Components[i].Stage = stage
			s.Components[i].UpdatedAt = now
			s.UpdatedAt = now
			return
		}
	}
}

// Save writes the environment state to the configured backend.
func (sm *Manager) Save(state *OrkState) error {
	return sm.backend.Save(context.Background(), sm.envID, state)
}

// Exists reports whether the configured backend contains this environment.
func (sm *Manager) Exists(ctx context.Context) (bool, error) {
	return sm.backend.Exists(ctx, sm.envID)
}

// Delete removes the complete environment state bundle, including artifacts.
func (sm *Manager) Delete() error {
	return sm.backend.Delete(context.Background(), sm.envID)
}

// NewComponentState builds the durable applied state returned by an adapter.
func NewComponentState(
	component *manifestcore.Component,
	runnerType runners.RunnerType,
	outputs map[string]string,
	data ComponentStateData,
) ComponentState {
	return ComponentState{
		Name: component.Name,
		Type: component.Type,
		Runner: RunnerRef{
			Name: component.Runner,
			Type: runnerType,
		},
		Source:             component.Source,
		WorkDir:            data.WorkDir,
		NonSensitiveConfig: SanitizeMap(component.Config),
		Outputs:            outputs,
		Payload:            data.Payload,
		Artifacts:          data.Artifacts,
		DestroyHooks:       destroyHooksFromComponent(component),
		Status:             StatusApplied,
		Stage:              StagePostApply,
	}
}

func destroyHooksFromComponent(component *manifestcore.Component) DestroyHooks {
	return DestroyHooks{
		PreDestroy:  append([]manifestcore.Hook(nil), component.Hooks.PreDestroy...),
		PostDestroy: append([]manifestcore.Hook(nil), component.Hooks.PostDestroy...),
	}
}

// NewComponentStateData converts a typed adapter payload into a persistable map.
func NewComponentStateData(workDir string, payload interface{}, artifacts ...Artifact) (ComponentStateData, error) {
	mapped, err := ToMap(payload)
	if err != nil {
		return ComponentStateData{}, err
	}

	return ComponentStateData{
		WorkDir:   workDir,
		Payload:   mapped,
		Artifacts: artifacts,
	}, nil
}

// EmptyComponentStateData returns valid adapter state with no payload.
func EmptyComponentStateData(workDir string) ComponentStateData {
	return ComponentStateData{
		WorkDir: workDir,
		Payload: make(map[string]interface{}),
	}
}

// ToMap converts a typed adapter payload through JSON so backend serialization
// remains independent of adapter-specific Go types.
func ToMap(in interface{}) (map[string]interface{}, error) {
	if in == nil {
		return make(map[string]interface{}), nil
	}

	if mapped, ok := in.(map[string]interface{}); ok {
		return mapped, nil
	}

	data, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state data: %w", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state data: %w", err)
	}

	if out == nil {
		out = make(map[string]interface{})
	}
	return out, nil
}

// SanitizeMap recursively redacts values whose keys resemble credentials before
// configuration is copied into state. It is a safety net, not a secret scanner.
func SanitizeMap(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}

	out := make(map[string]interface{}, len(in))
	for key, value := range in {
		if IsSensitiveKey(key) {
			out[key] = "<redacted>"
			continue
		}

		switch typed := value.(type) {
		case map[string]interface{}:
			out[key] = SanitizeMap(typed)
		case map[interface{}]interface{}:
			nested := make(map[string]interface{}, len(typed))
			for nestedKey, nestedValue := range typed {
				nested[fmt.Sprint(nestedKey)] = nestedValue
			}
			out[key] = SanitizeMap(nested)
		default:
			out[key] = typed
		}
	}
	return out
}

// IsSensitiveKey reports whether a key name matches the conservative redaction
// heuristic used for persisted configuration.
func IsSensitiveKey(key string) bool {
	normalized := strings.ToLower(key)
	sensitiveParts := []string{
		"password",
		"passwd",
		"secret",
		"token",
		"credential",
		"private_key",
		"access_key",
	}

	for _, part := range sensitiveParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}

	return false
}
