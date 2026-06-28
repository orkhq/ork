package manifestparsers

import (
	"fmt"

	manifestcore "ork/pkg/manifest/core"
)

// Parser converts one manifest version into the canonical core model.
type Parser interface {
	Parse([]byte) (*manifestcore.Manifest, error)
}

var registry = map[string]Parser{
	"ork/1.0": &V1Parser{},
}

// Get returns the parser registered for an exact manifest version.
func Get(version string) (Parser, error) {
	p, ok := registry[version]
	if !ok {
		return nil, fmt.Errorf("no parser registered for version %s", version)
	}
	return p, nil
}
