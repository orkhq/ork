package adapters

import (
	"context"
	"fmt"

	"orch.io/pkg/events"
	manifestcore "orch.io/pkg/manifest/core"
	"orch.io/pkg/targets"
)

type ComponentConfig interface{}

type Adapter interface {
	Apply(ctx context.Context, c *manifestcore.Component, t targets.Target) error
	Destroy(ctx context.Context, c *manifestcore.Component, t targets.Target) error
	RequiredCapabilities() targets.Capabilities
	// ValidateAndLoadConfig validates the component config and loads it into a structured format.
	// It returns the loaded config, any warning events generated during validation, and an error if validation fails.
	ValidateAndLoadConfig(c *manifestcore.Component) (ComponentConfig, []events.Event, error)
}

var registry = map[string]Adapter{}

func Register(name string, a Adapter) {
	registry[name] = a
}

func Get(name string) (Adapter, error) {
	a, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for type %s", name)
	}
	return a, nil
}
