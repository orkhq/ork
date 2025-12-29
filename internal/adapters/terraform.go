package adapters

import (
	"context"
	"fmt"
	"os/exec"

	"orch.io/pkg/events"
	manifestcore "orch.io/pkg/manifest/core"
	"orch.io/pkg/targets"
)

type TerraformAdapter struct{}

func (d *TerraformAdapter) RequiredCapabilities() targets.Capabilities {
	return targets.Capabilities{Exec: true, FileCopy: true}
}

func (d *TerraformAdapter) ValidateAndLoadConfig(c *manifestcore.Component) (ComponentConfig, []events.Event, error) {
	// Add validation logic if needed
	return nil, make([]events.Event, 0), nil
}

func (d *TerraformAdapter) Apply(ctx context.Context, c *manifestcore.Component, t targets.Target) error {
	cmd := exec.Command("docker-compose", "-f", c.Source.Path, "up", "-d")
	cmd.Stdout, cmd.Stderr = nil, nil
	fmt.Printf("Running docker-compose up for %s...\n", c.Source.Path)
	return cmd.Run()
}

func (d *TerraformAdapter) Destroy(ctx context.Context, c *manifestcore.Component, t targets.Target) error {
	cmd := exec.Command("docker-compose", "-f", c.Source.Path, "down")
	fmt.Printf("Running docker-compose down for %s...\n", c.Source.Path)
	return cmd.Run()
}

func init() {
	Register("terraform", &TerraformAdapter{})
}
