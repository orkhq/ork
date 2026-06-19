// Package adapters provides the component adapter layer for ork. Each adapter
// knows how to apply and destroy a specific infrastructure type (scripts,
// Docker Compose, Terraform, CloudFormation). Adapters are registered at init
// time and looked up by component type during orchestration.
package adapters

import (
	"context"
	"path"

	"ork/pkg/events"
	"ork/pkg/logging"
)

const AdapterContextKey = "__adapter.context"

// AdapterContext carries environment-scoped state that adapters need during
// apply and destroy operations, including the environment ID, a debug logger,
// and an event emitter.
type AdapterContext struct {
	envID   string
	logger  logging.DebugLogger
	emitter events.Emitter
}

// GetComponentWorkDirInOrkLocalWorkDir returns the local working directory path
// for a component within the .ork state directory (e.g., .ork/<envID>/<component>).
func (a AdapterContext) GetComponentWorkDirInOrkLocalWorkDir(c string) string {
	return path.Join(".ork", a.envID, c)
}

func (a AdapterContext) EnvID() string {
	return a.envID
}

// BuildRunnerWorkDir constructs the working directory path on the runner
// for a component, namespaced by environment ID to avoid collisions.
func (a AdapterContext) BuildRunnerWorkDir(baseWorkDir, componentName string) string {
	return path.Join(baseWorkDir, "ork", a.envID, componentName)
}

// NewAdapterContext creates an AdapterContext for the given environment ID.
func NewAdapterContext(id string, logger logging.DebugLogger, emitter events.Emitter) AdapterContext {
	return AdapterContext{
		envID:   id,
		logger:  logger,
		emitter: emitter,
	}
}

// WithAdapterContext stores an AdapterContext in the given context.
func WithAdapterContext(ctx context.Context, aCtx AdapterContext) context.Context {
	return context.WithValue(ctx, AdapterContextKey, aCtx)
}

// AdapterContextFromContext extracts the AdapterContext from a context, returning
// false if none is present.
func AdapterContextFromContext(ctx context.Context) (AdapterContext, bool) {
	aCtx, ok := ctx.Value(AdapterContextKey).(AdapterContext)
	return aCtx, ok
}
