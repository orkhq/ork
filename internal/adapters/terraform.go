package adapters

import (
	"fmt"
	"os/exec"

	manifestcore "orch.io/pkg/manifest/core"
)

type TerraformAdapter struct{}

func (d *TerraformAdapter) ValidateComponent(c manifestcore.Component) error {
	// Add validation logic if needed
	return nil
}

func (d *TerraformAdapter) Apply(c manifestcore.Component) error {
	cmd := exec.Command("docker-compose", "-f", c.Source.Path, "up", "-d")
	cmd.Stdout, cmd.Stderr = nil, nil
	fmt.Printf("Running docker-compose up for %s...\n", c.Source.Path)
	return cmd.Run()
}

func (d *TerraformAdapter) Destroy(c manifestcore.Component) error {
	cmd := exec.Command("docker-compose", "-f", c.Source.Path, "down")
	fmt.Printf("Running docker-compose down for %s...\n", c.Source.Path)
	return cmd.Run()
}

func init() {
	Register("terraform", &TerraformAdapter{})
}
