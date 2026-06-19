// Package varresolvers implements variable interpolation for ork manifests.
// Expressions of the form ${...} in configuration values are resolved at
// runtime by a chain of resolvers that handle component outputs, environment
// variables, and user-supplied inputs.
package varresolvers

import (
	"context"
	"fmt"
	"regexp"
)

// Resolver resolves a single interpolation expression (the content between
// ${ and }) to its string value. Implementations return an error if the
// expression is not handled by this resolver.
type Resolver interface {
	Resolve(ctx context.Context, path string) (string, error)
}

var re = regexp.MustCompile(`\$\{([^}]+)}`)

// InterpolateString replaces all ${...} expressions in s using the provided
// Resolver. It returns the interpolated string or the first resolution error.
func InterpolateString(ctx context.Context, s string, resolver Resolver) (string, error) {
	var firstErr error
	out := re.ReplaceAllStringFunc(s, func(match string) string {
		key := match[2 : len(match)-1] // strip ${ and }
		val, err := resolver.Resolve(ctx, key)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("resolving %q: %w", key, err)
			}
			return match
		}
		return val
	})
	if firstErr != nil {
		return "", firstErr
	}
	return out, nil
}

func deepInterpolateValue(
	ctx context.Context,
	val interface{},
	resolver Resolver,
) (interface{}, error) {

	switch v := val.(type) {

	case string:
		// Only strings can contain ${...}
		return InterpolateString(ctx, v, resolver)

	case map[string]interface{}:
		m := make(map[string]interface{}, len(v))
		for k, vv := range v {
			iv, err := deepInterpolateValue(ctx, vv, resolver)
			if err != nil {
				return nil, err
			}
			m[k] = iv
		}
		return m, nil

	case []interface{}:
		arr := make([]interface{}, len(v))
		for i, vv := range v {
			iv, err := deepInterpolateValue(ctx, vv, resolver)
			if err != nil {
				return nil, err
			}
			arr[i] = iv
		}
		return arr, nil

	default:
		// Numbers, bools, nil, etc. pass through unchanged
		return v, nil
	}
}

// DeepInterpolate recursively walks a map and interpolates all string values
// containing ${...} expressions using the provided Resolver.
func DeepInterpolate(
	ctx context.Context,
	in map[string]interface{},
	resolver Resolver,
) (map[string]interface{}, error) {

	out := make(map[string]interface{}, len(in))

	for k, v := range in {
		iv, err := deepInterpolateValue(ctx, v, resolver)
		if err != nil {
			return nil, fmt.Errorf("interpolating key %q: %w", k, err)
		}
		out[k] = iv
	}

	return out, nil
}
