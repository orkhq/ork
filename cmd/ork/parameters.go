package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParametersSource holds parameters loaded from a file and from CLI flags.
// CLI parameters take precedence over file parameters when merged.
type ParametersSource struct {
	FileParameters map[string]string
	CLIParameters  map[string]string
}

// Merge Returns merged map: CLIParameters override FileParameters
func (s *ParametersSource) Merge() map[string]string {
	merged := make(map[string]string)
	for k, v := range s.FileParameters {
		merged[k] = v
	}
	for k, v := range s.CLIParameters {
		merged[k] = v
	}
	return merged
}

// LoadParametersFile reads parameters from a YAML (.yml/.yaml) or env-style file
// and returns them as a flat key-value map.
func LoadParametersFile(filePath string) (map[string]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	parameters := make(map[string]string)
	if strings.HasSuffix(filePath, ".yml") || strings.HasSuffix(filePath, ".yaml") {
		// Simple YAML unmarshal
		err = yaml.Unmarshal(data, &parameters)
		if err != nil {
			return nil, err
		}
	} else {
		// Treat as env file
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid parameter line: %s", line)
			}
			parameters[parts[0]] = parts[1]
		}
	}
	return parameters, nil
}

// ParseCLIParameters parses a slice of "key=value" strings into a map.
func ParseCLIParameters(cliParameters []string) (map[string]string, error) {
	parsed := make(map[string]string)
	for _, s := range cliParameters {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid parameter: %s", s)
		}
		parsed[parts[0]] = parts[1]
	}
	return parsed, nil
}

// LoadParameters combines file-based and CLI-based parameters into a single
// ParametersSource. If parametersFile is empty, only CLI parameters are used.
func LoadParameters(parametersFile string, cliParameters []string) (*ParametersSource, error) {
	var parsedFileParameters map[string]string
	var err error

	if parametersFile != "" {
		parsedFileParameters, err = LoadParametersFile(parametersFile)
		if err != nil {
			return nil, err
		}
	} else {
		parsedFileParameters = make(map[string]string)
	}

	parsedCLIParameters, err := ParseCLIParameters(cliParameters)
	if err != nil {
		return nil, err
	}

	return &ParametersSource{
		FileParameters: parsedFileParameters,
		CLIParameters:  parsedCLIParameters,
	}, nil
}
