package targets

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// AWSContext is implemented by targets that can provide AWS configuration.
type AWSContext interface {
	AWSConfig(ctx context.Context) (*aws.Config, error)
}
