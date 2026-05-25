package orchestration

import (
	"testing"

	"orch.io/internal/adapters"
	"orch.io/pkg/events"
	manifestcore "orch.io/pkg/manifest/core"
)

type noopEmitter struct{}

func (noopEmitter) Emit(events.Event) {}

type recordingEmitter struct {
	events []events.Event
}

func (r *recordingEmitter) Emit(event events.Event) {
	r.events = append(r.events, event)
}

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

func TestValidateOutputDeclarationsRejectsReservedMetaOutput(t *testing.T) {
	component := &manifestcore.Component{
		Name: "script",
		Outputs: []manifestcore.Output{
			{Name: "_meta.ports.services.web.80"},
		},
	}

	if err := validateOutputDeclarations(component); err == nil {
		t.Fatal("expected reserved _meta output declaration to fail")
	}
}

func TestValidateApplyOutputsAllowsReservedMetaOutputs(t *testing.T) {
	component := &manifestcore.Component{Name: "compose", Type: "docker-compose"}
	emitter := &recordingEmitter{}

	err := validateApplyOutputs(component, adapters.ComponentApplyOutput{
		"_meta.ports.services.web.80": "49153",
	}, emitter)
	if err != nil {
		t.Fatalf("expected reserved _meta output to pass: %v", err)
	}
	if len(emitter.events) != 0 {
		t.Fatalf("expected no warning for reserved _meta output, got %d events", len(emitter.events))
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

func TestFilterDeclaredOutputsKeepsReservedMetaOutputs(t *testing.T) {
	component := &manifestcore.Component{Name: "compose"}

	filtered := filterDeclaredOutputs(component, adapters.ComponentApplyOutput{
		"_meta.ports.services.web.80": "49153",
		"extra":                       "ignored",
	})
	if len(filtered) != 1 {
		t.Fatalf("expected one output, got %d", len(filtered))
	}
	if filtered["_meta.ports.services.web.80"] != "49153" {
		t.Fatalf("unexpected _meta output: %q", filtered["_meta.ports.services.web.80"])
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

func TestFilterStateOutputsKeepsReservedMetaOutputs(t *testing.T) {
	component := &manifestcore.Component{
		Name: "compose",
		Outputs: []manifestcore.Output{
			{Name: "token", Sensitive: true},
		},
	}

	filtered := filterStateOutputs(component, adapters.ComponentApplyOutput{
		"token":                       "secret",
		"_meta.ports.services.web.80": "49153",
	})
	if len(filtered) != 1 {
		t.Fatalf("expected one output, got %d", len(filtered))
	}
	if filtered["_meta.ports.services.web.80"] != "49153" {
		t.Fatalf("unexpected _meta output: %q", filtered["_meta.ports.services.web.80"])
	}
}
