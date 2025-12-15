package manifest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	manifestcore "orch.io/pkg/manifest/core"
	manifestparsers "orch.io/pkg/manifest/parsers"
)

type VersionedManifest struct {
	Version string `yaml:"version"`
}

func Load(path string) (*manifestcore.Manifest, error) {
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
	fmt.Println("Successfully loaded manifest")

	return m, nil
}
