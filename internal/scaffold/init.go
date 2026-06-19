// Package scaffold provides manifest initialization and scaffolding for new ork
// projects. It generates a starter ork.yaml manifest from a built-in template.
package scaffold

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

//go:embed templates/starter.ork.yaml
var templatesFS embed.FS

// InitOptions configures manifest initialization behavior.
type InitOptions struct {
	// Path is the file path to write the generated manifest (default: "ork.yaml").
	Path string
	// ID is the manifest metadata ID; defaults to the sanitized current directory name.
	ID string
	// Force allows overwriting an existing manifest file.
	Force bool
}

// RunInit creates a starter ork manifest file at the configured path. It
// infers a manifest ID from the working directory if none is provided and
// refuses to overwrite an existing file unless Force is set.
func RunInit(options InitOptions) error {
	if options.Path == "" {
		options.Path = "ork.yaml"
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

	manifest, err := starterManifest(options.ID)
	if err != nil {
		return err
	}
	if err := os.WriteFile(options.Path, manifest, 0644); err != nil {
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
		return "ork-demo"
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
		return "ork-demo"
	}
	return result
}

func starterManifest(id string) ([]byte, error) {
	data, err := templatesFS.ReadFile("templates/starter.ork.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read starter manifest template: %w", err)
	}

	tmpl, err := template.New("starter.ork.yaml").Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse starter manifest template: %w", err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, map[string]string{"ID": id}); err != nil {
		return nil, fmt.Errorf("failed to render starter manifest template: %w", err)
	}
	return out.Bytes(), nil
}
