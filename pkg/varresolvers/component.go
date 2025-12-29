package varresolvers

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type ComponentResolver struct {
	outputs map[string]string // componentName -> outputName -> value
	mutex   sync.RWMutex      // thread-safe for concurrent component execution
}

func NewComponentResolver() *ComponentResolver {
	return &ComponentResolver{
		outputs: make(map[string]string),
	}
}

// Resolve an expression like "component.outputs.db_url"
func (r *ComponentResolver) Resolve(ctx context.Context, expr string) (string, error) {
	parts := strings.Split(expr, ".")
	if len(parts) != 3 || parts[1] != "outputs" {
		return "", fmt.Errorf("invalid component output reference: %q", expr)
	}
	compName, outputName := parts[0], parts[2]

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if comp, ok := r.outputs[expr]; ok {
		return comp, fmt.Errorf("output %q not yet available for component %q", outputName, compName)
	}
	return "", fmt.Errorf("component %q not yet executed", compName)
}

// RegisterComponentOutput Called by adapters after a component finishes
func (r *ComponentResolver) RegisterComponentOutput(componentName, outputName, value string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := componentName + ".outputs." + outputName
	r.outputs[key] = value
}
