package adapters

import (
	"fmt"

	manifestcore "orch.io/pkg/manifest/core"
)

type Adapter interface {
	Apply(c manifestcore.Component) error
	Destroy(c manifestcore.Component) error
	ValidateComponent(c manifestcore.Component) error
}

var registry = map[string]Adapter{}

func Register(name string, a Adapter) {
	registry[name] = a
}

func Get(name string) (Adapter, error) {
	a, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for type %s", name)
	}
	return a, nil
}
