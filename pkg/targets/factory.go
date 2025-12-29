package targets

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	manifestcore "orch.io/pkg/manifest/core"
)

func FromManifest(name string, mt manifestcore.TargetManifest) (Target, error) {
	switch mt.Type {

	case "local":
		return &LocalTarget{name: name}, nil

	case "ssh":
		var cfg SSHTargetConfig
		if err := mapstructure.Decode(mt.Config, &cfg); err != nil {
			return nil, fmt.Errorf("failed to decode ssh target \"%s\" config: %w", name, err)
		}
		return &SSHTarget{name: name, config: cfg}, nil

	case "aws":
		var cfg AWSTargetConfig
		if err := mapstructure.Decode(mt.Config, &cfg); err != nil {
			return nil, fmt.Errorf("failed to decode aws target \"%s\" config: %w", name, err)
		}
		return &AWSTarget{config: cfg, name: name}, nil

	default:
		return nil, fmt.Errorf("unknown target type: %s", mt.Type)
	}
}

func FromManifestTargetsMap(targetsMap map[string]manifestcore.TargetManifest) (map[string]Target, error) {
	targets := make(map[string]Target)
	for name, mt := range targetsMap {
		target, err := FromManifest(name, mt)
		if err != nil {
			return nil, fmt.Errorf("failed to create target \"%s\": %w", name, err)
		}

		if err := target.ValidateAndInitialize(); err != nil {
			return nil, fmt.Errorf("] target \"%s\" failed to create: %w", name, err)
		}

		targets[name] = target
	}
	return targets, nil
}
