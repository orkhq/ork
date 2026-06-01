package statebackends

import (
	"context"
	"fmt"

	"orch/pkg/logging"
	manifestcore "orch/pkg/manifest/core"
	"orch/pkg/state"
)

func FromManifest(cfg *manifestcore.StateConfig) (state.Backend, error) {
	return FromManifestContext(context.Background(), cfg, &logging.NoopDebugLogger{})
}

func FromManifestContext(ctx context.Context, cfg *manifestcore.StateConfig, logger logging.DebugLogger) (state.Backend, error) {
	logger = normalizeLogger(logger)
	if cfg == nil {
		return NewLocal("", logger), nil
	}

	backend := cfg.Backend
	if backend == "" {
		backend = "local"
	}

	switch backend {
	case "local":
		return NewLocalFromConfig(cfg.Config, logger)
	case "s3":
		return NewS3FromConfig(ctx, cfg.Config, logger)
	default:
		return nil, fmt.Errorf("unsupported state backend %q; supported backends: local, s3", backend)
	}
}

func normalizeLogger(logger logging.DebugLogger) logging.DebugLogger {
	if logger == nil {
		return &logging.NoopDebugLogger{}
	}
	return logger
}
