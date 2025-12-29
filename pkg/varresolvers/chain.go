package varresolvers

import (
	"context"
	"fmt"
)

type ChainResolver struct {
	Resolvers []Resolver
}

func (c *ChainResolver) Resolve(ctx context.Context, expr string) (string, error) {
	for _, r := range c.Resolvers {
		val, err := r.Resolve(ctx, expr)
		if err == nil {
			return val, nil
		}
	}
	return "", fmt.Errorf("could not resolve %q", expr)
}
