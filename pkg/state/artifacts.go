package state

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"ork/pkg/runners"
)

// CaptureArtifacts copies declared files from a runner into the state backend.
func (sm *Manager) CaptureArtifacts(ctx context.Context, component ComponentState, runner runners.Runner) error {
	for _, artifact := range component.Artifacts {
		if err := sm.captureArtifact(ctx, component, artifact, runner); err != nil {
			return err
		}
	}
	return nil
}

func (sm *Manager) captureArtifact(ctx context.Context, component ComponentState, artifact Artifact, runner runners.Runner) error {
	runnerPath, err := artifactRunnerPath(component.WorkDir, artifact)
	if err != nil {
		return fmt.Errorf("invalid artifact %q for component %q: %w", artifact.Name, component.Name, err)
	}

	localPath, cleanup, err := tempArtifactFile()
	if err != nil {
		return err
	}
	defer cleanup()

	copyRes, err := runner.CopyFile(ctx, runners.FileCopyRequest{
		Source:      runnerPath,
		Destination: localPath,
		ToRunner:    false,
		Overwrite:   true,
		Recursive:   false,
	})
	if err != nil || copyRes.Error != nil {
		if !artifact.Required {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to copy required artifact %q from runner: %w", artifact.Name, err)
		}
		return fmt.Errorf("failed to copy required artifact %q from runner: %w", artifact.Name, copyRes.Error)
	}

	if err := sm.backend.SaveArtifact(ctx, sm.envID, component.Name, artifact, localPath); err != nil {
		return fmt.Errorf("failed to save artifact %q for component %q: %w", artifact.Name, component.Name, err)
	}
	return nil
}

// RestoreArtifacts materializes captured files on a runner before destroy.
func (sm *Manager) RestoreArtifacts(ctx context.Context, component ComponentState, runner runners.Runner) error {
	for _, artifact := range component.Artifacts {
		if err := sm.restoreArtifact(ctx, component, artifact, runner); err != nil {
			return err
		}
	}
	return nil
}

func (sm *Manager) restoreArtifact(ctx context.Context, component ComponentState, artifact Artifact, runner runners.Runner) error {
	runnerPath, err := artifactRunnerPath(component.WorkDir, artifact)
	if err != nil {
		return fmt.Errorf("invalid artifact %q for component %q: %w", artifact.Name, component.Name, err)
	}

	exists, err := sm.backend.ArtifactExists(ctx, sm.envID, component.Name, artifact)
	if err != nil {
		return fmt.Errorf("failed to check artifact %q for component %q: %w", artifact.Name, component.Name, err)
	}
	if !exists {
		if artifact.Required {
			return fmt.Errorf("required artifact %q for component %q is missing", artifact.Name, component.Name)
		}
		return nil
	}

	localPath, cleanup, err := tempArtifactFile()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := sm.backend.RestoreArtifact(ctx, sm.envID, component.Name, artifact, localPath); err != nil {
		return fmt.Errorf("failed to restore artifact %q for component %q: %w", artifact.Name, component.Name, err)
	}

	copyRes, err := runner.CopyFile(ctx, runners.FileCopyRequest{
		Source:      localPath,
		Destination: runnerPath,
		ToRunner:    true,
		Overwrite:   true,
		Recursive:   false,
	})
	if err != nil {
		return fmt.Errorf("failed to copy artifact %q to runner: %w", artifact.Name, err)
	}
	if copyRes.Error != nil {
		return fmt.Errorf("failed to copy artifact %q to runner: %w", artifact.Name, copyRes.Error)
	}
	return nil
}

func tempArtifactFile() (string, func(), error) {
	// The current runner and backend APIs both speak in filesystem paths, not
	// streams. A temp file is the bridge between "copy from runner" and "save to
	// backend" until we can guarantee direct runner-to-backend streaming.
	file, err := os.CreateTemp("", "ork-artifact-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to create temporary artifact file: %w", err)
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(name)
		return "", func() {}, fmt.Errorf("failed to close temporary artifact file: %w", err)
	}
	return name, func() { _ = os.Remove(name) }, nil
}

func artifactRunnerPath(workDir string, artifact Artifact) (string, error) {
	cleaned, err := CleanArtifactPath(artifact.Path)
	if err != nil {
		return "", err
	}
	return path.Join(workDir, cleaned), nil
}

// CleanArtifactPath validates and normalizes a path relative to a component
// work directory, rejecting absolute paths and traversal.
func CleanArtifactPath(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("path is required")
	}
	if path.IsAbs(value) {
		return "", fmt.Errorf("path %q must be relative", value)
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("path %q escapes component workdir", value)
	}
	return cleaned, nil
}
