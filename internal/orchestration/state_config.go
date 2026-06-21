package orchestration

import (
	"context"
	"fmt"

	manifestcore "ork/pkg/manifest/core"
	"ork/pkg/varresolvers"
)

func resolveStateConfig(ctx context.Context, cfg *manifestcore.StateConfig, resolver varresolvers.Resolver) (*manifestcore.StateConfig, error) {
	if cfg == nil {
		return nil, nil
	}

	resolved := &manifestcore.StateConfig{Backend: cfg.Backend}
	if cfg.Auth != nil {
		auth, err := varresolvers.DeepInterpolate(ctx, cfg.Auth, resolver)
		if err != nil {
			return nil, fmt.Errorf("state auth interpolation failed: %w", err)
		}
		resolved.Auth = auth
	}
	if cfg.Config != nil {
		backendConfig, err := varresolvers.DeepInterpolate(ctx, cfg.Config, resolver)
		if err != nil {
			return nil, fmt.Errorf("state config interpolation failed: %w", err)
		}
		resolved.Config = backendConfig
	}
	return resolved, nil
}

func stateConfigResolver(inputsResolver *varresolvers.InputsResolver) varresolvers.Resolver {
	return &varresolvers.ChainResolver{
		Resolvers: []varresolvers.Resolver{
			inputsResolver,
			varresolvers.NewEnvResolver(),
		},
	}
}
