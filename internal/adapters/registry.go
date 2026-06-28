package adapters

import (
	"context"
	"fmt"
	"strings"

	"ork/pkg/events"
	manifestcore "ork/pkg/manifest/core"
	"ork/pkg/runners"
	"ork/pkg/state"
)

// ComponentSourceSupport declares which mutually exclusive source shapes an
// adapter accepts.
type ComponentSourceSupport struct {
	Path     bool
	Files    bool
	Embedded bool
}

func (s ComponentSourceSupport) String() string {
	var supported []string
	if s.Path {
		supported = append(supported, "Path")
	}
	if s.Files {
		supported = append(supported, "Files")
	}
	if s.Embedded {
		supported = append(supported, "Embedded")
	}
	return "[" + strings.Join(supported, ", ") + "]"
}

func (s ComponentSourceSupport) SatisfiedBy(c manifestcore.ComponentSource) bool {
	if c.Path != "" && !s.Path {
		return false
	}
	if len(c.Files) > 0 && !s.Files {
		return false
	}
	if c.Embedded != "" && !s.Embedded {
		return false
	}

	if c.Type() == "none" && (s.Path || s.Files || s.Embedded) {
		return false
	}

	return true
}

// ComponentConfig is an adapter's validated, structured configuration.
type ComponentConfig interface{}

// ComponentApplyOutput contains raw adapter outputs before declaration and
// sensitivity filtering by orchestration.
type ComponentApplyOutput map[string]string

// ComponentApplyResult returns both downstream outputs and the operational
// state required for later artifact capture and teardown.
type ComponentApplyResult struct {
	Outputs ComponentApplyOutput
	State   state.ComponentStateData
}

// Adapter implements one component type. Implementations must perform all
// execution and file transfer through the supplied runner.
type Adapter interface {
	Apply(ctx context.Context, c *manifestcore.Component, r runners.Runner) (ComponentApplyResult, error)
	Destroy(ctx context.Context, c state.ComponentState, r runners.Runner) error
	RequiredCapabilities() runners.Capabilities
	SupportedSources() ComponentSourceSupport

	// ValidateAndLoadConfig validates the component config and loads it into a structured format.
	// It returns the loaded config, any warning events generated during validation, and an error if validation fails.
	ValidateAndLoadConfig(ctx context.Context, c *manifestcore.Component) (ComponentConfig, []events.Event, error)
}

var registry = map[string]Adapter{}

// Register associates a manifest component type with its adapter.
func Register(name string, a Adapter) {
	registry[name] = a
}

// Get returns the adapter registered for a manifest component type.
func Get(name string) (Adapter, error) {
	a, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for type %s", name)
	}
	return a, nil
}
