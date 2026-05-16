package statebackends

import (
	"testing"

	manifestcore "orch.io/pkg/manifest/core"
)

func TestFromManifest(t *testing.T) {
	t.Run("defaults to local when state is omitted", func(t *testing.T) {
		backend, err := FromManifest(nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		local, ok := backend.(*Local)
		if !ok {
			t.Fatalf("expected *Local, got %T", backend)
		}
		if local.root != ".orch" {
			t.Fatalf("expected default root .orch, got %q", local.root)
		}
	})

	t.Run("uses explicit local config", func(t *testing.T) {
		backend, err := FromManifest(&manifestcore.StateConfig{
			Backend: "local",
			Config:  map[string]interface{}{"path": ".state"},
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		local, ok := backend.(*Local)
		if !ok {
			t.Fatalf("expected *Local, got %T", backend)
		}
		if local.root != ".state" {
			t.Fatalf("expected root .state, got %q", local.root)
		}
	})
}
