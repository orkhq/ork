package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type InitOptions struct {
	Path  string
	ID    string
	Force bool
}

func RunInit(options InitOptions) error {
	if options.Path == "" {
		options.Path = "orch.yaml"
	}
	if options.ID == "" {
		options.ID = inferManifestID()
	} else {
		options.ID = sanitizeManifestID(options.ID)
	}

	if !options.Force {
		if _, err := os.Stat(options.Path); err == nil {
			return fmt.Errorf("%s already exists; use --force to overwrite", options.Path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check %s: %w", options.Path, err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(normalizeInitPath(options.Path)), 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	if err := os.WriteFile(options.Path, []byte(starterManifest(options.ID)), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", options.Path, err)
	}
	return nil
}

func normalizeInitPath(path string) string {
	if filepath.Dir(path) == "." {
		return filepath.Join(".", filepath.Base(path))
	}
	return path
}

func inferManifestID() string {
	wd, err := os.Getwd()
	if err != nil {
		return "orch-demo"
	}
	return sanitizeManifestID(filepath.Base(wd))
}

func sanitizeManifestID(value string) string {
	value = strings.ToLower(value)
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteRune('-')
			lastDash = true
		}
	}
	result := strings.Trim(out.String(), "-")
	if result == "" {
		return "orch-demo"
	}
	return result
}

func starterManifest(id string) string {
	return fmt.Sprintf(`version: orch.io/1.0

metadata:
  id: %s
  description: Starter Orch environment
  owner:
    name: Your Name
    email: you@example.com

state:
  backend: local
  config:
    path: .orch

runners:
  local:
    type: local
    config: {}

components:
  setup:
    type: script
    runner: local
    source:
      embedded: |
        echo "message=hello from orch" >> "$ORCH_OUTPUT_ENV"
    outputs:
      - name: message
`, id)
}
