package varresolvers

import (
	"context"
	"fmt"
	"strings"
)

type InputsResolver struct {
	Inputs map[string]string
}

func NewInputsResolver(inputs map[string]string) *InputsResolver {
	return &InputsResolver{Inputs: inputs}
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
