package manifestparsers

import (
	"fmt"

	manifestcore "ork/pkg/manifest/core"
)

type Parser interface {
	Parse([]byte) (*manifestcore.Manifest, error)
}

var registry = map[string]Parser{
	"ork/1.0": &V1Parser{},
}

func Get(version string) (Parser, error) {
	p, ok := registry[version]
	if !ok {
		return nil, fmt.Errorf("no parser registered for version %s", version)
	}
	return p, nil
}
