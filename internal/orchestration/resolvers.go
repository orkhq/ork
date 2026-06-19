package orchestration

import "ork/pkg/varresolvers"

// shellCommandResolver builds a resolver chain for interpolating hook and
// lifecycle commands. It intentionally excludes the EnvResolver so that
// environment variables are not eagerly substituted into shell text—commands
// receive env values via the runner exec environment instead.
func shellCommandResolver(resolvers ...varresolvers.Resolver) varresolvers.Resolver {
	// Omit EnvResolver for command strings so environment values are not eagerly
	// interpolated into shell text. Commands still receive env through the runner.
	return &varresolvers.ChainResolver{Resolvers: resolvers}
}
