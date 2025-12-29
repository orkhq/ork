package adapters

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"gopkg.in/yaml.v3"
	"orch.io/pkg/events"
	manifestcore "orch.io/pkg/manifest/core"
	"orch.io/pkg/targets"
	"orch.io/pkg/utils"
)

type DockerComposeAdapter struct{}
type DockerComposeConfig struct {
	WorkingDir string   `mapstructure:"working_dir"`
	Flags      []string `mapstructure:"flags"`

	Services []ComposeServiceMetaData
}

func (d *DockerComposeAdapter) RequiredCapabilities() targets.Capabilities {
	return targets.Capabilities{Exec: true, FileCopy: true}
}

func (d *DockerComposeAdapter) ValidateAndLoadConfig(c *manifestcore.Component) (ComponentConfig, []events.Event, error) {
	var cfg DockerComposeConfig
	var warnings []events.Event

	if err := mapstructure.Decode(c.Config, &cfg); err != nil {
		return nil, warnings, err
	}
	if c.Source.Path == "" {
		return nil, warnings, fmt.Errorf("docker-compose component must have a source path")
	}

	services, err := loadComposeFileAndExtractServices(c.Source.Path)
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to load compose file: %w", err)
	}

	for _, service := range services {
		if service.HasFixedPorts {
			warnings = append(warnings, events.Event{
				Type: events.EventWarning,
				Message: fmt.Sprintf("Compose service %s has fixed port mappings."+
					"This may lead to port conflicts when multiple instances are deployed.", service.Name),
				Hint: "Consider using dynamic port mappings or environment variables to avoid conflicts.\n" +
					"Dynamic port mappings can be specified by omitting the host port (e.g., '8080' instead of '80:8080').\n" +
					"See more info at https://orch.io/docs/guides/docker-compose#handling-port-conflicts",
				Adapter:   c.Type,
				Target:    c.Target,
				Component: c.Name,
			})
		}
	}

	cfg.Services = services
	return &cfg, warnings, nil
}

func (d *DockerComposeAdapter) Apply(ctx context.Context, c *manifestcore.Component, t targets.Target) error {
	cfg, ok := c.LoadedConfig.(*DockerComposeConfig)
	if !ok {
		return fmt.Errorf("invalid config type for DockerComposeAdapter")
	}

	aCtx, ok := AdapterContextFromContext(ctx)
	if !ok {
		return fmt.Errorf("failed to get sandbox ID from context")
	}

	workingDir := path.Join(cfg.WorkingDir, "boxes", aCtx.envID, c.Name)
	composeFilePath := path.Join(workingDir, "docker-compose.yaml")

	copyRes, err := t.CopyFile(ctx, targets.FileCopyRequest{
		Source:      c.Source.Path,
		Destination: composeFilePath,
		ToTarget:    true,
		Overwrite:   true,
		Recursive:   false,
	})

	if err != nil {
		return fmt.Errorf("failed to copy docker-compose file to target: %w", err)
	}
	if copyRes.Error != nil {
		return fmt.Errorf("error during file copy: %w", copyRes.Error)
	}

	aCtx.emitter.Emit(events.Event{
		Type:      events.EventInfo,
		Message:   fmt.Sprintf("Copied compose file to %s", composeFilePath),
		Adapter:   c.Type,
		Target:    c.Target,
		Component: c.Name,
		Duration:  copyRes.Duration,
	})

	execRes, err := t.Exec(ctx, targets.ExecCommand{
		WorkingDir: workingDir,
		Command:    []string{"docker-compose", "-f", composeFilePath, "up", "-d"},
		Env:        buildOrchManagedComposeEnv(c.Env, aCtx.envID, path.Dir(workingDir)),
		Timeout:    0,
		Stderr:     utils.NewPrefixWriter(os.Stderr, fmt.Sprintf("-→] \033[34m[%s > %s]\033[0m ", c.Target, c.Name)),
		Stdout:     utils.NewPrefixWriter(os.Stdout, fmt.Sprintf("-→] \033[34m[%s > %s]\033[0m ", c.Target, c.Name)),
	})

	if err != nil {
		return fmt.Errorf("an error occurred %w", err)
	}

	if execRes.Error != nil {
		return fmt.Errorf("failed to run docker-compose up: %w", execRes.Error)
	}

	return nil
}

func (d *DockerComposeAdapter) Destroy(ctx context.Context, c *manifestcore.Component, t targets.Target) error {
	cmd := exec.Command("docker-compose", "-f", c.Source.Path, "down")
	fmt.Printf("Running docker-compose down for %s...\n", c.Source.Path)
	return cmd.Run()
}

func buildOrchManagedComposeEnv(
	base map[string]string,
	envID string,
	workDir string,
) map[string]string {

	if base == nil {
		base = make(map[string]string)
	}

	base["COMPOSE_PROJECT_NAME"] = fmt.Sprintf("orch_%s", envID)
	base["COMPOSE_FILE"] = path.Join(workDir, "docker-compose.yaml")
	base["ORCH_WORKING_DIR"] = workDir
	base["ORCH_ENV_ID"] = envID
	return base
}

func init() {
	Register("docker-compose", &DockerComposeAdapter{})
}

// Compose Ports Utilities

type ComposeFile struct {
	Services map[string]struct {
		Ports []string `yaml:"ports"`
	} `yaml:"services"`
}

type ComposeServiceMetaData struct {
	Ports          []string
	Name           string
	HasFixedPorts  bool
	PublishesPorts bool
}

func loadComposeFileAndExtractServices(filePath string) ([]ComposeServiceMetaData, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	var composeFile ComposeFile
	if err := yaml.Unmarshal(data, &composeFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal compose file: %w", err)
	}

	services := make([]ComposeServiceMetaData, 0, len(composeFile.Services))
	for name, service := range composeFile.Services {

		var ports []string
		s := ComposeServiceMetaData{
			Name:           name,
			PublishesPorts: len(service.Ports) > 0,
		}

		// Check if any port mapping is fixed (i.e., host port specified)
		for _, port := range service.Ports {
			if len(strings.Split(port, ":")) > 1 {
				s.HasFixedPorts = true
			} else {
				ports = append(ports, port)
			}
		}

		s.Ports = ports

		services = append(services, s)
	}

	return services, nil

}
