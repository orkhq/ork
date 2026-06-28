package statebackends

import (
	"context"
	"fmt"

	"ork/pkg/logging"
	manifestcore "ork/pkg/manifest/core"
	"ork/pkg/state"
)

// FromManifest constructs a backend with a background context and no-op logger.
func FromManifest(cfg *manifestcore.StateConfig) (state.Backend, error) {
	return FromManifestContext(context.Background(), cfg, &logging.NoopDebugLogger{})
}

// FromManifestContext constructs and validates the backend selected by cfg. A
// nil config selects local state at .ork.
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
		if len(cfg.Auth) > 0 {
			return nil, fmt.Errorf("local state backend does not support auth config")
		}
		return NewLocalFromConfig(cfg.Config, logger)
	case "s3":
		return NewS3FromConfig(ctx, cfg.Config, cfg.Auth, logger)
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
