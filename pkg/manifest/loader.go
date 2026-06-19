// Package manifest handles loading and version-dispatched parsing of ork
// manifest files. A manifest declares the components, runners, inputs, and
// metadata that define an ephemeral environment.
package manifest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"ork/pkg/logging"
	manifestcore "ork/pkg/manifest/core"
	manifestparsers "ork/pkg/manifest/parsers"
)

// VersionedManifest is a minimal struct used to extract the version field from a
// raw manifest file before selecting the appropriate version-specific parser.
type VersionedManifest struct {
	Version string `yaml:"version"`
}

// Load reads a YAML manifest file from disk, determines its version, and
// delegates parsing to the appropriate version-specific parser. It returns the
// fully parsed Manifest or an error if the file cannot be read or parsed.
func Load(path string, logger logging.Logger) (*manifestcore.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var v VersionedManifest
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}

	if v.Version == "" {
		return nil, fmt.Errorf("manifest missing version")
	}

	p, err := manifestparsers.Get(v.Version)
	if err != nil {
		return nil, err
	}

	m := &manifestcore.Manifest{}
	m, err = p.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	logger.Debug("Loaded manifest", logging.Field{Key: "path", Value: path}, logging.Field{Key: "version", Value: v.Version})

	return m, nil
}
