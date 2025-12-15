package adapters

import manifestcore "orch.io/pkg/manifest/core"

type CloudFormationAdapter struct{}

func (d *CloudFormationAdapter) ValidateComponent(c manifestcore.Component) error {
	return nil
}

func (d *CloudFormationAdapter) Apply(c manifestcore.Component) error {
	// Implement CloudFormation stack creation logic here
	return nil
}

func (d *CloudFormationAdapter) Destroy(c manifestcore.Component) error {
	// Implement CloudFormation stack deletion logic here
	return nil
}

func init() {
	Register("cloudformation", &CloudFormationAdapter{})
}
