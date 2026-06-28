package statebackends

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-viper/mapstructure/v2"
	"ork/pkg/logging"
	"ork/pkg/state"
)

// LocalConfig configures filesystem-backed state persistence.
type LocalConfig struct {
	Path string `mapstructure:"path"`
}

// Local stores one directory per environment on the Ork machine.
type Local struct {
	root   string
	logger logging.DebugLogger
}

// NewLocalFromConfig strictly decodes manifest backend configuration.
func NewLocalFromConfig(config map[string]interface{}, logger logging.DebugLogger) (*Local, error) {
	var cfg LocalConfig
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      &cfg,
		TagName:     "mapstructure",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize local state backend decoder: %w", err)
	}
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("invalid local state backend config: %w", err)
	}

	return NewLocal(cfg.Path, logger), nil
}

// NewLocal creates a local backend, defaulting an empty root to .ork.
func NewLocal(root string, logger logging.DebugLogger) *Local {
	if root == "" {
		root = ".ork"
	}
	return &Local{root: root, logger: normalizeLogger(logger)}
}

func (b *Local) Load(ctx context.Context, envID string) (*state.OrkState, error) {
	stateFile := b.stateFile(envID)
	b.logger.Debug("loading local state", logging.Field{Key: "path", Value: stateFile})
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("state file not found: %s (environment may not exist or was never created)", stateFile)
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var orkState state.OrkState
	if err := json.Unmarshal(data, &orkState); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &orkState, nil
}

func (b *Local) Save(ctx context.Context, envID string, orkState *state.OrkState) error {
	stateFile := b.stateFile(envID)
	b.logger.Debug("saving local state", logging.Field{Key: "path", Value: stateFile})
	stateDir := filepath.Dir(stateFile)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(orkState, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (b *Local) Exists(ctx context.Context, envID string) (bool, error) {
	_, err := os.Stat(b.stateFile(envID))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (b *Local) Delete(ctx context.Context, envID string) error {
	envDir := b.envDir(envID)
	b.logger.Debug("deleting local state environment", logging.Field{Key: "path", Value: envDir})
	if err := os.RemoveAll(envDir); err != nil {
		return fmt.Errorf("failed to delete state environment: %w", err)
	}
	return nil
}

func (b *Local) SaveArtifact(ctx context.Context, envID string, componentName string, artifact state.Artifact, sourcePath string) error {
	destination, err := b.artifactFile(envID, componentName, artifact)
	if err != nil {
		return err
	}
	b.logger.Debug("saving local state artifact", logging.Field{Key: "path", Value: destination})
	return copyLocalFile(sourcePath, destination, 0600)
}

func (b *Local) RestoreArtifact(ctx context.Context, envID string, componentName string, artifact state.Artifact, destinationPath string) error {
	source, err := b.artifactFile(envID, componentName, artifact)
	if err != nil {
		return err
	}
	b.logger.Debug("restoring local state artifact", logging.Field{Key: "path", Value: source})
	return copyLocalFile(source, destinationPath, 0600)
}

func (b *Local) ArtifactExists(ctx context.Context, envID string, componentName string, artifact state.Artifact) (bool, error) {
	artifactFile, err := b.artifactFile(envID, componentName, artifact)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(artifactFile)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (b *Local) stateFile(envID string) string {
	return filepath.Join(b.envDir(envID), "state.json")
}

func (b *Local) artifactFile(envID string, componentName string, artifact state.Artifact) (string, error) {
	artifactPath, err := state.CleanArtifactPath(artifact.Path)
	if err != nil {
		return "", err
	}
	return filepath.Join(b.envDir(envID), "artifacts", componentName, filepath.FromSlash(artifactPath)), nil
}

func (b *Local) envDir(envID string) string {
	return filepath.Join(b.root, envID)
}

func copyLocalFile(source, destination string, mode os.FileMode) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	return os.WriteFile(destination, data, mode)
}
