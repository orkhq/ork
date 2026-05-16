package statebackends

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/go-viper/mapstructure/v2"
	"orch.io/pkg/logging"
	"orch.io/pkg/state"
)

type S3Config struct {
	Bucket               string `mapstructure:"bucket"`
	Prefix               string `mapstructure:"prefix"`
	Region               string `mapstructure:"region"`
	ServerSideEncryption string `mapstructure:"server_side_encryption"`
	KMSKeyID             string `mapstructure:"kms_key_id"`
}

type s3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type S3 struct {
	client               s3Client
	bucket               string
	prefix               string
	serverSideEncryption string
	kmsKeyID             string
	logger               logging.DebugLogger
}

func NewS3FromConfig(ctx context.Context, rawConfig map[string]interface{}, logger logging.DebugLogger) (*S3, error) {
	cfg, err := decodeS3Config(rawConfig)
	if err != nil {
		return nil, err
	}

	var loadOptions []func(*config.LoadOptions) error
	if cfg.Region != "" {
		loadOptions = append(loadOptions, config.WithRegion(cfg.Region))
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for S3 state backend: %w", err)
	}

	return NewS3(s3.NewFromConfig(awsCfg), cfg, logger), nil
}

func NewS3(client s3Client, cfg S3Config, logger logging.DebugLogger) *S3 {
	return &S3{
		client:               client,
		bucket:               cfg.Bucket,
		prefix:               cleanS3Prefix(cfg.Prefix),
		serverSideEncryption: cfg.ServerSideEncryption,
		kmsKeyID:             cfg.KMSKeyID,
		logger:               normalizeLogger(logger),
	}
}

func decodeS3Config(rawConfig map[string]interface{}) (S3Config, error) {
	var cfg S3Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      &cfg,
		TagName:     "mapstructure",
	})
	if err != nil {
		return S3Config{}, fmt.Errorf("failed to initialize S3 state backend decoder: %w", err)
	}
	if err := decoder.Decode(rawConfig); err != nil {
		return S3Config{}, fmt.Errorf("invalid S3 state backend config: %w", err)
	}
	if cfg.Bucket == "" {
		return S3Config{}, fmt.Errorf("invalid S3 state backend config: bucket is required")
	}
	switch cfg.ServerSideEncryption {
	case "", string(types.ServerSideEncryptionAes256), string(types.ServerSideEncryptionAwsKms):
	default:
		return S3Config{}, fmt.Errorf("invalid S3 state backend config: server_side_encryption must be %q or %q", types.ServerSideEncryptionAes256, types.ServerSideEncryptionAwsKms)
	}
	if cfg.KMSKeyID != "" && cfg.ServerSideEncryption != string(types.ServerSideEncryptionAwsKms) {
		return S3Config{}, fmt.Errorf("invalid S3 state backend config: kms_key_id requires server_side_encryption=%q", types.ServerSideEncryptionAwsKms)
	}
	return cfg, nil
}

func (b *S3) Load(ctx context.Context, envID string) (*state.OrchState, error) {
	key := b.stateKey(envID)
	b.logger.Debug("loading S3 state", logging.Field{Key: "bucket", Value: b.bucket}, logging.Field{Key: "key", Value: key})
	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil, fmt.Errorf("state file not found: s3://%s/%s (environment may not exist or was never created)", b.bucket, key)
		}
		return nil, fmt.Errorf("failed to read S3 state object: %w", err)
	}
	defer b.closeBody(out.Body, "S3 state object", key)

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 state body: %w", err)
	}

	var orchState state.OrchState
	if err := json.Unmarshal(data, &orchState); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	return &orchState, nil
}

func (b *S3) Save(ctx context.Context, envID string, orchState *state.OrchState) error {
	data, err := json.MarshalIndent(orchState, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	key := b.stateKey(envID)
	b.logger.Debug("saving S3 state", logging.Field{Key: "bucket", Value: b.bucket}, logging.Field{Key: "key", Value: key})
	return b.putObject(ctx, key, bytes.NewReader(data), "application/json")
}

func (b *S3) Exists(ctx context.Context, envID string) (bool, error) {
	return b.objectExists(ctx, b.stateKey(envID))
}

func (b *S3) Delete(ctx context.Context, envID string) error {
	key := b.stateKey(envID)
	b.logger.Debug("deleting S3 state", logging.Field{Key: "bucket", Value: b.bucket}, logging.Field{Key: "key", Value: key})
	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete S3 state object: %w", err)
	}
	return nil
}

func (b *S3) SaveArtifact(ctx context.Context, envID string, componentName string, artifact state.Artifact, sourcePath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	key, err := b.artifactKey(envID, componentName, artifact)
	if err != nil {
		return err
	}
	b.logger.Debug("saving S3 state artifact", logging.Field{Key: "bucket", Value: b.bucket}, logging.Field{Key: "key", Value: key})
	return b.putObject(ctx, key, bytes.NewReader(data), "application/octet-stream")
}

func (b *S3) RestoreArtifact(ctx context.Context, envID string, componentName string, artifact state.Artifact, destinationPath string) error {
	key, err := b.artifactKey(envID, componentName, artifact)
	if err != nil {
		return err
	}
	b.logger.Debug("restoring S3 state artifact", logging.Field{Key: "bucket", Value: b.bucket}, logging.Field{Key: "key", Value: key})
	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to read S3 artifact object %q: %w", key, err)
	}
	defer b.closeBody(out.Body, "S3 artifact object", key)

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return fmt.Errorf("failed to read S3 artifact body %q: %w", key, err)
	}
	return os.WriteFile(destinationPath, data, 0600)
}

func (b *S3) ArtifactExists(ctx context.Context, envID string, componentName string, artifact state.Artifact) (bool, error) {
	key, err := b.artifactKey(envID, componentName, artifact)
	if err != nil {
		return false, err
	}
	return b.objectExists(ctx, key)
}

func (b *S3) objectExists(ctx context.Context, key string) (bool, error) {
	b.logger.Debug("checking S3 object", logging.Field{Key: "bucket", Value: b.bucket}, logging.Field{Key: "key", Value: key})
	_, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}
	if isS3NotFound(err) {
		return false, nil
	}
	return false, err
}

func (b *S3) putObject(ctx context.Context, key string, body io.Reader, contentType string) error {
	b.logger.Debug("writing S3 object", logging.Field{Key: "bucket", Value: b.bucket}, logging.Field{Key: "key", Value: key})
	input := &s3.PutObjectInput{
		Bucket:      aws.String(b.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}
	if b.serverSideEncryption != "" {
		input.ServerSideEncryption = types.ServerSideEncryption(b.serverSideEncryption)
	}
	if b.kmsKeyID != "" {
		input.SSEKMSKeyId = aws.String(b.kmsKeyID)
	}
	if _, err := b.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("failed to write S3 object %q: %w", key, err)
	}
	return nil
}

func (b *S3) closeBody(body io.Closer, label string, key string) {
	if err := body.Close(); err != nil {
		b.logger.Debug(
			"failed to close S3 response body",
			logging.Field{Key: "object", Value: label},
			logging.Field{Key: "key", Value: key},
			logging.Field{Key: "error", Value: err},
		)
	}
}

func (b *S3) stateKey(envID string) string {
	return joinS3Key(b.prefix, envID, "state.json")
}

func (b *S3) artifactKey(envID string, componentName string, artifact state.Artifact) (string, error) {
	artifactPath, err := state.CleanArtifactPath(artifact.Path)
	if err != nil {
		return "", err
	}
	return joinS3Key(b.prefix, envID, "artifacts", componentName, artifactPath), nil
}

func cleanS3Prefix(prefix string) string {
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return ""
	}
	return path.Clean(prefix)
}

func joinS3Key(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, "/")
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return strings.Join(cleaned, "/")
}

func isS3NotFound(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	code := apiErr.ErrorCode()
	return code == "NotFound" || code == "NoSuchKey" || code == "404"
}
