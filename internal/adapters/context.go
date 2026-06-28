package adapters

import (
	"context"
	"path"

	"ork/pkg/events"
	"ork/pkg/logging"
)

const AdapterContextKey = "__adapter.context"

// AdapterContext carries environment-scoped paths, events, and diagnostics into
// adapter operations without expanding the public Adapter method signatures.
type AdapterContext struct {
	envID   string
	logger  logging.DebugLogger
	emitter events.Emitter
}

func (a AdapterContext) GetComponentWorkDirInOrkLocalWorkDir(c string) string {
	return path.Join(".ork", a.envID, c)
}

func (a AdapterContext) EnvID() string {
	return a.envID
}

func (a AdapterContext) BuildRunnerWorkDir(baseWorkDir, componentName string) string {
	return path.Join(baseWorkDir, "ork", a.envID, componentName)
}

// NewAdapterContext creates context shared by adapters in one environment run.
func NewAdapterContext(id string, logger logging.DebugLogger, emitter events.Emitter) AdapterContext {
	return AdapterContext{
		envID:   id,
		logger:  logger,
		emitter: emitter,
	}
}

// WithAdapterContext attaches adapter lifecycle context to ctx.
func WithAdapterContext(ctx context.Context, aCtx AdapterContext) context.Context {
	return context.WithValue(ctx, AdapterContextKey, aCtx)
}

// AdapterContextFromContext retrieves adapter lifecycle context from ctx.
func AdapterContextFromContext(ctx context.Context) (AdapterContext, bool) {
	aCtx, ok := ctx.Value(AdapterContextKey).(AdapterContext)
	return aCtx, ok
}
