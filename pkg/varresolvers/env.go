package varresolvers

import (
	"context"
	"fmt"
	"os"
)

// EnvResolver resolves expressions by looking them up as environment variables
// in the current process. It treats the entire expression as the variable name.
type EnvResolver struct{}

// NewEnvResolver creates an EnvResolver.
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
