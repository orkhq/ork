package orchestration

import (
	"context"
	"fmt"
	"strings"

	"orch.io/internal/adapters"
	"orch.io/pkg/events"
	"orch.io/pkg/logging"
	manifestcore "orch.io/pkg/manifest/core"
	"orch.io/pkg/targets"
	"orch.io/pkg/varresolvers"
)

func RunUp(envID string, m *manifestcore.Manifest, logger logging.Logger, inputs map[string]string) error {
	componentResolver := varresolvers.NewComponentResolver()
	resolvers := &varresolvers.ChainResolver{
		Resolvers: []varresolvers.Resolver{
			varresolvers.NewInputsResolver(inputs), //todo: support inputs schema
			varresolvers.NewEnvResolver(),
			componentResolver,
		},
	}

	emitter := events.NewRendererEmitter()
	ctx := adapters.NewAdapterContext(context.Background(), envID, logger.AsDebugLogger(), emitter)

	for key, value := range m.Targets {
		cfg, err := varresolvers.DeepInterpolate(ctx, value.Config, resolvers)
		if err != nil {
			return fmt.Errorf("failed to interpolate target \"%s\" config: %w", key, err)
		}
		value.Config = cfg
		m.Targets[key] = value
	}

	allTargets, err := targets.FromManifestTargetsMap(m.Targets)
	if err != nil {
		return err
	}

	componentsInOrder, err := TopologicallySortComponents(m.Components)
	if err != nil {
		return fmt.Errorf("failed to establish logical apply order: %w", err)
	}

	for i := range componentsInOrder {
		c := &m.Components[i]
		t, ok := allTargets[c.Target]
		if !ok {
			return fmt.Errorf("component \"%s\" references an unknown target \"%s\"",
				c.Name, c.Target)
		}

		if yes, list := t.UsesNonAmbientCredentials(); yes {
			emitter.Emit(events.Event{
				Type: events.EventWarning,
				Message: fmt.Sprintf(
					"Target uses non-ambient credentials (%v). This component cannot be reliably torn down by Orch.",
					strings.Join(list, ", "),
				),
				Hint:      "Use ambient authentication for the target to enable safe teardown of this component. Learn more at https://orch.io/docs/guides/authentication",
				Target:    t.Name(),
				Component: c.Name,
				Adapter:   c.Type,
			})
		}

		adapter, err := adapters.Get(c.Type)
		if err != nil {
			return err
		}

		if !adapter.RequiredCapabilities().SatisfiedBy(t.Capabilities()) {
			return fmt.Errorf("component \"%s\" requires capabilities %v which are not satisfied by target \"%s\" capabilities %v",
				c.Name, adapter.RequiredCapabilities(), t.Name(), t.Capabilities())
		}

		// Interpolate variables in component properties
		for key, value := range c.Config {
			resolvedValue, err := varresolvers.InterpolateString(ctx, value, resolvers)
			if err == nil {
				c.Config[key] = resolvedValue
			}
		}

		// Interpolate variables in component environment variables
		for key, value := range c.Env {
			resolvedValue, err := varresolvers.InterpolateString(ctx, value, resolvers)
			if err == nil {
				c.Env[key] = resolvedValue
			}
		}

		cfg, warnings, err := adapter.ValidateAndLoadConfig(c)
		if err != nil {
			return fmt.Errorf("component \"%s\" config validation failed: %w", c.Name, err)
		}

		for _, warning := range warnings {
			emitter.Emit(warning)
		}

		c.LoadedConfig = cfg

		emitter.Emit(events.Event{
			Type:      events.EventStart,
			Message:   fmt.Sprintf("starting apply for component"),
			Adapter:   c.Type,
			Target:    c.Target,
			Component: c.Name,
		})
		if err := adapter.Apply(ctx, c, t); err != nil {
			emitter.Emit(events.Event{
				Type:      events.EventFailure,
				Message:   fmt.Sprintf("failed to apply component"),
				Adapter:   c.Type,
				Target:    c.Target,
				Component: c.Name,
				Err:       err,
			})

			return fmt.Errorf("component \"%s\" failed to apply", c.Name)
		}

		componentResolver.RegisterComponentOutput(c.Name, "test", "value") //todo: fix
	}

	// Disconnect all targets
	for _, t := range allTargets {
		if err := t.Disconnect(); err != nil {
			emitter.Emit(events.Event{
				Type:    events.EventWarning,
				Message: fmt.Sprintf("failed to disconnect from target \"%s\": %v", t.Name(), err),
				Target:  t.Name(),
			})
		}
	}

	fmt.Printf("Sandbox created successfully\n")
	return nil
}
