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

// ComponentSourceSupport declares which source types an adapter supports (local
// path, named files, or inline embedded content).
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

// ComponentConfig is a marker interface for adapter-specific configuration
// structs returned by ValidateAndLoadConfig.
type ComponentConfig interface{}
// ComponentApplyOutput is a set of key-value output pairs produced by a
// successful adapter Apply call.
type ComponentApplyOutput map[string]string

// ComponentApplyResult bundles the outputs and state data returned by an
// adapter after a successful apply. State is persisted for later destroy.
type ComponentApplyResult struct {
	Outputs ComponentApplyOutput
	State   state.ComponentStateData
}

// Adapter is the interface that infrastructure adapters must implement to
// participate in ork's apply/destroy lifecycle.
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

// Register adds an adapter to the global registry under the given name.
// Adapters call this in their init() functions.
func Register(name string, a Adapter) {
	registry[name] = a
}

// Get retrieves a registered adapter by name, returning an error if none is found.
func Get(name string) (Adapter, error) {
	a, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for type %s", name)
	}
	return a, nil
}
