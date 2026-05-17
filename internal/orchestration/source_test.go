package orchestration

import (
	"strings"
	"testing"

	"orch.io/internal/adapters"
	manifestcore "orch.io/pkg/manifest/core"
)

func TestValidateComponentSourceRejectsMultipleSourceKinds(t *testing.T) {
	component := &manifestcore.Component{
		Name: "script",
		Type: "script",
		Source: manifestcore.ComponentSource{
			Embedded: "echo hi",
			Files: map[string]string{
				"script.sh": "./script.sh",
			},
		},
	}

	err := validateComponentSource(component, &adapters.ScriptAdapter{})
	if err == nil {
		t.Fatal("expected multiple source kinds to fail")
	}
	if !strings.Contains(err.Error(), "multiple source types") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateComponentSourceRejectsUnsupportedSource(t *testing.T) {
	component := &manifestcore.Component{
		Name: "script",
		Type: "script",
		Source: manifestcore.ComponentSource{
			Path: "./scripts",
		},
	}

	err := validateComponentSource(component, &adapters.ScriptAdapter{})
	if err == nil {
		t.Fatal("expected unsupported source to fail")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}
