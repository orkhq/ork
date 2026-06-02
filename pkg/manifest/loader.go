package manifest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"ork/pkg/logging"
	manifestcore "ork/pkg/manifest/core"
	manifestparsers "ork/pkg/manifest/parsers"
)

type VersionedManifest struct {
	Version string `yaml:"version"`
}

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
