package manifestparsers

import (
	"gopkg.in/yaml.v3"
	manifestcore "orch.io/pkg/manifest/core"
)

type V1Manifest struct {
	Version    string                        `yaml:"version"`
	Inputs     map[string]manifestcore.Input `yaml:"inputs,omitempty"`
	Metadata   manifestcore.Metadata         `yaml:"metadata"`
	Cloud      manifestcore.CloudSpec        `yaml:"cloud,omitempty"`
	Components []manifestcore.Component      `yaml:"components"`
}

type V1Parser struct{}

func (p *V1Parser) Parse(data []byte) (*manifestcore.Manifest, error) {

	var m V1Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	return &manifestcore.Manifest{
		Version:    m.Version,
		Inputs:     m.Inputs,
		Metadata:   m.Metadata,
		Cloud:      m.Cloud,
		Components: m.Components,
	}, nil
}
