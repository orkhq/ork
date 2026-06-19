package orchestration

import (
	"context"

	manifestcore "ork/pkg/manifest/core"
	"ork/pkg/varresolvers"
)

// componentExecutionEnv builds the full environment map for a component
// execution by merging the base resolved env with standard ork-injected
// variables (ORK_ENV_ID, ORK_COMPONENT_NAME, etc.).
func componentExecutionEnv(envID string, component *manifestcore.Component, runnerName string, workDir string, base map[string]string) map[string]string {
	env := make(map[string]string, len(base)+5)
	for key, value := range base {
		env[key] = value
	}

	env["ORK_ENV_ID"] = envID
	env["ORK_COMPONENT_NAME"] = component.Name
	env["ORK_COMPONENT_TYPE"] = component.Type
	env["ORK_RUNNER_NAME"] = runnerName
	env["ORK_WORKDIR"] = workDir

	return env
}

// interpolateEnv resolves variable references (e.g., ${inputs.foo}) in each
// environment variable value using the provided resolver chain.
func interpolateEnv(ctx context.Context, env map[string]string, resolver varresolvers.Resolver) (map[string]string, error) {
	resolved := make(map[string]string, len(env))
	for key, value := range env {
		interpolated, err := varresolvers.InterpolateString(ctx, value, resolver)
		if err != nil {
			return nil, err
		}
		resolved[key] = interpolated
	}
	return resolved, nil
}
