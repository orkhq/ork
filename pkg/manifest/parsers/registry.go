// Package manifestparsers provides version-specific parsers for ork manifest
// files. Each parser knows how to transform raw YAML bytes into the canonical
// manifestcore.Manifest structure.
package manifestparsers

import (
	"fmt"

	manifestcore "ork/pkg/manifest/core"
)

// Parser converts raw manifest bytes into the canonical Manifest structure.
// Each manifest version has a corresponding Parser implementation.
type Parser interface {
	Parse([]byte) (*manifestcore.Manifest, error)
}

var registry = map[string]Parser{
	"ork/1.0": &V1Parser{},
}

// Get returns the registered Parser for the given manifest version string, or
// an error if no parser is registered for that version.
func Get(version string) (Parser, error) {
	p, ok := registry[version]
	if !ok {
		return nil, fmt.Errorf("no parser registered for version %s", version)
	}
	return p, nil
}
