package statebackends

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"orch/pkg/logging"
	"orch/pkg/state"
)

func TestNewLocalFromConfig(t *testing.T) {
	t.Run("defaults when config omitted", func(t *testing.T) {
		backend, err := NewLocalFromConfig(nil, &logging.NoopDebugLogger{})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if backend.root != ".orch" {
			t.Fatalf("expected default root .orch, got %q", backend.root)
		}
	})

	t.Run("accepts path", func(t *testing.T) {
		backend, err := NewLocalFromConfig(map[string]interface{}{"path": ".state"}, &logging.NoopDebugLogger{})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if backend.root != ".state" {
			t.Fatalf("expected root .state, got %q", backend.root)
		}
	})

	t.Run("rejects unknown config", func(t *testing.T) {
		_, err := NewLocalFromConfig(map[string]interface{}{"bucket": "not-for-local"}, &logging.NoopDebugLogger{})
		if err == nil {
			t.Fatal("expected error for unknown local backend config")
		}
	})
}

func TestLocalDeleteRemovesEnvironmentBundle(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	envID := "pr-123"
	backend := NewLocal(root, &logging.NoopDebugLogger{})

	if err := backend.Save(ctx, envID, state.New(envID, "test", &logging.NoopDebugLogger{})); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	artifactSource := filepath.Join(t.TempDir(), "terraform.tfstate")
	if err := os.WriteFile(artifactSource, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to write artifact source: %v", err)
	}
	if err := backend.SaveArtifact(ctx, envID, "tf", state.Artifact{
		Name: "tfstate",
		Path: "terraform.tfstate",
	}, artifactSource); err != nil {
		t.Fatalf("failed to save artifact: %v", err)
	}

	if err := backend.Delete(ctx, envID); err != nil {
		t.Fatalf("failed to delete state bundle: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, envID)); !os.IsNotExist(err) {
		t.Fatalf("expected environment directory to be deleted, got err=%v", err)
	}
}
