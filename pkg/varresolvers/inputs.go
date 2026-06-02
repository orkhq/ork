package varresolvers

import (
	"context"
	"fmt"
	"strings"

	manifestcore "ork/pkg/manifest/core"
)

type InputsResolver struct {
	Inputs map[string]string
}

func NewInputsResolver(inputs map[string]string, schema map[string]manifestcore.Input) (*InputsResolver, error) {
	for key := range schema {
		if _, ok := inputs[key]; !ok {
			if def, ok := schema[key]; ok {
				if def.Default == "" && def.Required {
					return nil, fmt.Errorf("input variable %q is required but not provided", key)
				}

				inputs[key] = def.Default
			}
		}
	}

	return &InputsResolver{Inputs: inputs}, nil
}

func (r *InputsResolver) Resolve(ctx context.Context, expr string) (string, error) {
	parts := strings.Split(expr, ".")
	if len(parts) != 2 || parts[0] != "inputs" {
		return "", fmt.Errorf("input variable %q not found", expr)
	}

	if val, ok := r.Inputs[parts[1]]; ok {
		return val, nil
	}

	fmt.Println(expr, r.Inputs)
	return "", fmt.Errorf("input variable %q not found", expr)
}
