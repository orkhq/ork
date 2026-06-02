package orchestration

import (
	"fmt"
	"strings"

	"ork/internal/adapters"
	"ork/pkg/events"
	manifestcore "ork/pkg/manifest/core"
)

func validateOutputDeclarations(component *manifestcore.Component) error {
	seen := make(map[string]struct{}, len(component.Outputs))
	for _, output := range component.Outputs {
		if output.Name == "" {
			return fmt.Errorf("component %q has an output with no name", component.Name)
		}
		if isReservedOutputName(output.Name) {
			return fmt.Errorf("component %q declares reserved output %q", component.Name, output.Name)
		}
		if _, ok := seen[output.Name]; ok {
			return fmt.Errorf("component %q declares output %q more than once", component.Name, output.Name)
		}
		seen[output.Name] = struct{}{}
	}
	return nil
}

func validateApplyOutputs(component *manifestcore.Component, outputs adapters.ComponentApplyOutput, emitter events.Emitter) error {
	declared := make(map[string]manifestcore.Output, len(component.Outputs))
	for _, output := range component.Outputs {
		declared[output.Name] = output
		if output.IsRequired() {
			if _, ok := outputs[output.Name]; !ok {
				return fmt.Errorf("component %q did not provide required output %q", component.Name, output.Name)
			}
		}
	}

	for outputName := range outputs {
		if _, ok := declared[outputName]; ok || isReservedOutputName(outputName) {
			continue
		}
		emitter.Emit(events.Event{
			Type:      events.EventWarning,
			Message:   fmt.Sprintf("Component produced undeclared output %q; it will not be available for interpolation.", outputName),
			Adapter:   component.Type,
			Runner:    component.Runner,
			Component: component.Name,
		})
	}

	return nil
}

func filterDeclaredOutputs(component *manifestcore.Component, outputs adapters.ComponentApplyOutput) adapters.ComponentApplyOutput {
	filtered := make(adapters.ComponentApplyOutput)
	for outputName, value := range outputs {
		if isReservedOutputName(outputName) {
			filtered[outputName] = value
		}
	}
	for _, output := range component.Outputs {
		if value, ok := outputs[output.Name]; ok {
			filtered[output.Name] = value
		}
	}
	return filtered
}

func filterStateOutputs(component *manifestcore.Component, outputs adapters.ComponentApplyOutput) adapters.ComponentApplyOutput {
	if len(outputs) == 0 {
		return adapters.ComponentApplyOutput{}
	}

	filtered := make(adapters.ComponentApplyOutput)
	for outputName, value := range outputs {
		if isReservedOutputName(outputName) {
			filtered[outputName] = value
		}
	}
	for _, output := range component.Outputs {
		if output.Sensitive {
			continue
		}
		if value, ok := outputs[output.Name]; ok {
			filtered[output.Name] = value
		}
	}
	return filtered
}

func isReservedOutputName(name string) bool {
	return name == "_meta" || strings.HasPrefix(name, "_meta.")
}
