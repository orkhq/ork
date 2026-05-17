package orchestration

import (
	"testing"

	"orch.io/internal/adapters"
	"orch.io/pkg/events"
	manifestcore "orch.io/pkg/manifest/core"
)

type noopEmitter struct{}

func (noopEmitter) Emit(events.Event) {}

func TestValidateApplyOutputsRequiresDeclaredOutputsByDefault(t *testing.T) {
	component := &manifestcore.Component{
		Name: "script",
		Type: "script",
		Outputs: []manifestcore.Output{
			{Name: "token"},
		},
	}

	err := validateApplyOutputs(component, adapters.ComponentApplyOutput{}, noopEmitter{})
	if err == nil {
		t.Fatal("expected missing required output to fail")
	}
}

func TestValidateApplyOutputsAllowsMissingOptionalOutputs(t *testing.T) {
	required := false
	component := &manifestcore.Component{
		Name: "script",
		Type: "script",
		Outputs: []manifestcore.Output{
			{Name: "token", Required: &required},
		},
	}

	if err := validateApplyOutputs(component, adapters.ComponentApplyOutput{}, noopEmitter{}); err != nil {
		t.Fatalf("expected missing optional output to pass, got %v", err)
	}
}

func TestValidateOutputDeclarationsRejectsDuplicateNames(t *testing.T) {
	component := &manifestcore.Component{
		Name: "script",
		Outputs: []manifestcore.Output{
			{Name: "token"},
			{Name: "token"},
		},
	}

	if err := validateOutputDeclarations(component); err == nil {
		t.Fatal("expected duplicate output declarations to fail")
	}
}

func TestFilterDeclaredOutputsRemovesExtras(t *testing.T) {
	component := &manifestcore.Component{
		Name: "script",
		Outputs: []manifestcore.Output{
			{Name: "token"},
		},
	}

	filtered := filterDeclaredOutputs(component, adapters.ComponentApplyOutput{
		"token": "abc",
		"extra": "ignored",
	})
	if len(filtered) != 1 {
		t.Fatalf("expected one output, got %d", len(filtered))
	}
	if filtered["token"] != "abc" {
		t.Fatalf("unexpected token output: %q", filtered["token"])
	}
	if _, ok := filtered["extra"]; ok {
		t.Fatal("expected extra output to be removed")
	}
}

func TestFilterStateOutputsRemovesSensitiveOutputs(t *testing.T) {
	component := &manifestcore.Component{
		Name: "script",
		Outputs: []manifestcore.Output{
			{Name: "token", Sensitive: true},
			{Name: "url"},
		},
	}

	filtered := filterStateOutputs(component, adapters.ComponentApplyOutput{
		"token": "secret",
		"url":   "http://localhost:8080",
	})
	if len(filtered) != 1 {
		t.Fatalf("expected one output, got %d", len(filtered))
	}
	if _, ok := filtered["token"]; ok {
		t.Fatal("expected sensitive token output to be removed")
	}
	if filtered["url"] != "http://localhost:8080" {
		t.Fatalf("unexpected url output: %q", filtered["url"])
	}
}
