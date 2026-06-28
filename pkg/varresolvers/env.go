package varresolvers

import (
	"context"
	"fmt"
	"os"
)

// EnvResolver resolves an expression as an operating-system environment name.
type EnvResolver struct{}

// NewEnvResolver creates an environment resolver.
func NewEnvResolver() *EnvResolver {
	return &EnvResolver{}
}

func (r *EnvResolver) Resolve(ctx context.Context, expr string) (string, error) {
	val, ok := os.LookupEnv(expr)
	if !ok {
		return "", fmt.Errorf("env variable %q not set", expr)
	}
	return val, nil
}
