package statebackends

import (
	"testing"

	"orch.io/pkg/logging"
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
