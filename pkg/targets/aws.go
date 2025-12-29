package targets

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type AWSTargetConfig struct {
	Region      string `yaml:"region"`
	Credentials struct {
		Type    string `yaml:"type"`
		RoleARN string `yaml:"role_arn"`
	} `yaml:"credentials"`
}

type AWSTarget struct {
	name   string
	config AWSTargetConfig
}

func (t *AWSTarget) Name() string {
	return t.name
}

func (t *AWSTarget) Type() TargetType {
	return TargetTypeAWS
}

func (t *AWSTarget) Capabilities() Capabilities {
	return Capabilities{Exec: false, FileCopy: false, Cloud: true}
}

func (t *AWSTarget) ValidateAndInitialize() error {
	if t.config.Region == "" {
		return errors.New("region required")
	}

	return nil
}

func (t *AWSTarget) Exec(ctx context.Context, command ExecCommand) (*ExecResult, error) {
	//TODO implement me
	panic("implement me")
}

func (t *AWSTarget) CopyFile(ctx context.Context, req FileCopyRequest) (*FileCopyResult, error) {
	//TODO implement me
	panic("implement me")
}

func (t *AWSTarget) UsesNonAmbientCredentials() (bool, []string) {
	if t.config.Credentials.Type != "" {
		return true, []string{"credentials"}
	}
	return false, nil
}

func (t *AWSTarget) Disconnect() error {
	// No persistent connections to close for AWS target
	return nil
}

// AWSConfig loads the AWS configuration for the target.
// It returns an aws.Config object that can be used to create AWS service clients.
func (t *AWSTarget) AWSConfig(ctx context.Context) (*aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(t.config.Region),
	)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
