// Package state manages the persistent operational state of ork environments.
// It tracks the status of each component through its lifecycle (applying,
// applied, destroying, destroyed, failed), persists outputs and artifacts, and
// enables idempotent re-runs and teardowns.
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

// Status represents the current lifecycle status of a component.
type Status string

const (
	StatusApplying   Status = "applying"
	StatusApplied    Status = "applied"
	StatusFailed     Status = "failed"
	StatusDestroying Status = "destroying"
	StatusDestroyed  Status = "destroyed"
)

// Stage identifies the specific lifecycle phase a component is in within its
// current status (e.g. pre_apply, apply, outputs).
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

// ComponentStateData holds adapter-produced data for a component including
// the working directory, arbitrary payload, and declared artifacts.
type ComponentStateData struct {
	WorkDir   string                 `json:"workdir"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Artifacts []Artifact             `json:"artifacts,omitempty"`
}

// Artifact references a file produced by a component that should be captured
// in the state backend for later restoration during destroy.
type Artifact struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Required  bool   `json:"required,omitempty"`
	Sensitive bool   `json:"sensitive,omitempty"`
}

// DestroyHooks captures the destroy-phase hooks from the manifest so they
// remain available during teardown even if the manifest changes.
type DestroyHooks struct {
	PreDestroy  []manifestcore.Hook `json:"pre_destroy,omitempty"`
	PostDestroy []manifestcore.Hook `json:"post_destroy,omitempty"`
}

// ComponentState represents the state of a single provisioned component
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

// OrkState represents the state of an entire ork environment
type OrkState struct {
	EnvID      string           `json:"env_id"`
	ManifestID string           `json:"manifest_id"`
	Components []ComponentState `json:"components"`
	CreatedAt  string           `json:"created_at"`
	UpdatedAt  string           `json:"updated_at"`

	logger logging.DebugLogger
}

// Manager handles persistence of ork state
type Manager struct {
	envID   string
	backend Backend
}

func NewManager(envID string, backend Backend) *Manager {
	return &Manager{envID: envID, backend: backend}
}

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

// Load reads the state file and returns the ork state
func (sm *Manager) Load() (*OrkState, error) {
	return sm.backend.Load(context.Background(), sm.envID)
}

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

func (s *OrkState) FindComponent(name string) (ComponentState, bool) {
	for _, component := range s.Components {
		if component.Name == name {
			return component, true
		}
	}
	return ComponentState{}, false
}

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

func (s *OrkState) MarkComponentFailed(name string, stage Stage) {
	s.markComponentStatus(name, StatusFailed, stage)
}

func (s *OrkState) MarkComponentApplying(name string, stage Stage) {
	s.markComponentStatus(name, StatusApplying, stage)
}

func (s *OrkState) MarkComponentApplied(name string, stage Stage) {
	s.markComponentStatus(name, StatusApplied, stage)
}

func (s *OrkState) MarkComponentDestroying(name string, stage Stage) {
	s.markComponentStatus(name, StatusDestroying, stage)
}

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

// Save writes the ork state to the state file
func (sm *Manager) Save(state *OrkState) error {
	return sm.backend.Save(context.Background(), sm.envID, state)
}

// Exists checks if a state file exists for this environment
func (sm *Manager) Exists(ctx context.Context) (bool, error) {
	return sm.backend.Exists(ctx, sm.envID)
}

// Delete removes the state file
func (sm *Manager) Delete() error {
	return sm.backend.Delete(context.Background(), sm.envID)
}

// NewComponentState constructs a ComponentState for a successfully applied
// component, combining manifest declarations with adapter-produced data.
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

// NewComponentStateData creates a ComponentStateData by marshaling the payload
// into a generic map and attaching the provided artifacts.
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

// EmptyComponentStateData returns a ComponentStateData with no payload or
// artifacts, useful for components that produce no state data.
func EmptyComponentStateData(workDir string) ComponentStateData {
	return ComponentStateData{
		WorkDir: workDir,
		Payload: make(map[string]interface{}),
	}
}

// ToMap converts an arbitrary value to a map[string]interface{} via JSON
// round-tripping. It returns an empty map for nil input.
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

// SanitizeMap recursively redacts values whose keys match known sensitive
// patterns (e.g. "password", "secret", "token") before persisting state.
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

// IsSensitiveKey returns true if the given key contains a substring that
// indicates it holds a secret value (e.g. "password", "token", "secret").
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
