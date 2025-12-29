package adapters

import (
	"context"
	"fmt"

	"orch.io/pkg/events"
	manifestcore "orch.io/pkg/manifest/core"
	"orch.io/pkg/targets"
)

type CloudFormationAdapter struct{}

func (d *CloudFormationAdapter) RequiredCapabilities() targets.Capabilities {
	return targets.Capabilities{Cloud: true}
}

func (d *CloudFormationAdapter) ValidateAndLoadConfig(c *manifestcore.Component) (ComponentConfig, []events.Event, error) {
	return nil, make([]events.Event, 0), nil
}

func (d *CloudFormationAdapter) Apply(ctx context.Context, c *manifestcore.Component, t targets.Target) error {
	awsCtx, ok := t.(targets.AWSContext)
	if !ok {
		return fmt.Errorf("invalid target type for CloudFormationAdapter")
	}

	_, err := awsCtx.AWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AWS config: %w", err)
	}

	// Implement CloudFormation stack creation logic here
	return nil
}

func (d *CloudFormationAdapter) Destroy(ctx context.Context, c *manifestcore.Component, t targets.Target) error {
	// Implement CloudFormation stack deletion logic here
	return nil
}

func init() {
	Register("cloudformation", &CloudFormationAdapter{})
}
