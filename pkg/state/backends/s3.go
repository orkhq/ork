package statebackends

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/go-viper/mapstructure/v2"
	"ork/pkg/logging"
	"ork/pkg/state"
)

// S3Config configures AWS S3 or an S3-compatible object store.
type S3Config struct {
	Bucket               string `mapstructure:"bucket"`
	Prefix               string `mapstructure:"prefix"`
	Region               string `mapstructure:"region"`
	Endpoint             string `mapstructure:"endpoint"`
	ForcePathStyle       bool   `mapstructure:"force_path_style"`
	ServerSideEncryption string `mapstructure:"server_side_encryption"`
	KMSKeyID             string `mapstructure:"kms_key_id"`
}

// S3AuthConfig selects either a shared profile or explicit SigV4 credentials.
// Explicit values should normally arrive through manifest interpolation.
type S3AuthConfig struct {
	Profile         string `mapstructure:"profile"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	SessionToken    string `mapstructure:"session_token"`
}

type s3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
}

// S3 stores state and artifacts under a bucket prefix using the S3 API.
type S3 struct {
	client               s3Client
	bucket               string
	prefix               string
	serverSideEncryption string
	kmsKeyID             string
	logger               logging.DebugLogger
}

// NewS3FromConfig strictly decodes backend and authentication configuration,
// then constructs an SDK client whose credentials are scoped to state access.
func NewS3FromConfig(ctx context.Context, rawConfig map[string]interface{}, rawAuth map[string]interface{}, logger logging.DebugLogger) (*S3, error) {
	cfg, err := decodeS3Config(rawConfig)
	if err != nil {
		return nil, err
	}
	authCfg, err := decodeS3AuthConfig(rawAuth)
	if err != nil {
		return nil, err
	}

	var loadOptions []func(*config.LoadOptions) error
	if cfg.Region != "" {
		loadOptions = append(loadOptions, config.WithRegion(cfg.Region))
	}
	if authCfg.Profile != "" {
		loadOptions = append(loadOptions, config.WithSharedConfigProfile(authCfg.Profile))
	} else if authCfg.AccessKeyID != "" {
		loadOptions = append(loadOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(authCfg.AccessKeyID, authCfg.SecretAccessKey, authCfg.SessionToken),
		))
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for S3 state backend: %w", err)
	}

	return NewS3(newS3Client(awsCfg, cfg), cfg, logger), nil
}

func decodeS3AuthConfig(rawAuth map[string]interface{}) (S3AuthConfig, error) {
	if len(rawAuth) == 0 {
		return S3AuthConfig{}, nil
	}

	var cfg S3AuthConfig
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      &cfg,
		TagName:     "mapstructure",
	})
	if err != nil {
		return S3AuthConfig{}, fmt.Errorf("failed to initialize S3 state backend auth decoder: %w", err)
	}
	if err := decoder.Decode(rawAuth); err != nil {
		return S3AuthConfig{}, fmt.Errorf("invalid S3 state backend auth config: %w", err)
	}
	cfg.Profile = strings.TrimSpace(cfg.Profile)
	cfg.AccessKeyID = strings.TrimSpace(cfg.AccessKeyID)
	if cfg.Profile != "" {
		if cfg.AccessKeyID != "" || cfg.SecretAccessKey != "" || cfg.SessionToken != "" {
			return S3AuthConfig{}, fmt.Errorf("invalid S3 state backend auth config: profile cannot be combined with access key credentials")
		}
		return cfg, nil
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return S3AuthConfig{}, fmt.Errorf("invalid S3 state backend auth config: access_key_id and secret_access_key are both required")
	}
	return cfg, nil
}

func newS3Client(awsCfg aws.Config, cfg S3Config) *s3.Client {
	return s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		if cfg.Endpoint != "" {
			options.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		options.UsePathStyle = cfg.ForcePathStyle
	})
}

// NewS3 creates a backend around an existing client. It is primarily useful for
// tests and alternate client construction within this package.
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
	if cfg.Endpoint != "" {
		cfg.Endpoint = strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
		parsedEndpoint, err := url.Parse(cfg.Endpoint)
		if err != nil || parsedEndpoint.Host == "" || (parsedEndpoint.Scheme != "http" && parsedEndpoint.Scheme != "https") {
			return S3Config{}, fmt.Errorf("invalid S3 state backend config: endpoint must be an absolute http or https URL")
		}
		if parsedEndpoint.User != nil || parsedEndpoint.RawQuery != "" || parsedEndpoint.Fragment != "" {
			return S3Config{}, fmt.Errorf("invalid S3 state backend config: endpoint must not include credentials, a query, or a fragment")
		}
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

func (b *S3) Load(ctx context.Context, envID string) (*state.OrkState, error) {
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

	var orkState state.OrkState
	if err := json.Unmarshal(data, &orkState); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	return &orkState, nil
}

func (b *S3) Save(ctx context.Context, envID string, orkState *state.OrkState) error {
	data, err := json.MarshalIndent(orkState, "", "  ")
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
	prefix := b.envKeyPrefix(envID)
	b.logger.Debug("deleting S3 state environment", logging.Field{Key: "bucket", Value: b.bucket}, logging.Field{Key: "prefix", Value: prefix})

	var continuationToken *string
	for {
		listOut, err := b.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(b.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return fmt.Errorf("failed to list S3 state objects: %w", err)
		}

		objects := make([]types.ObjectIdentifier, 0, len(listOut.Contents))
		for _, object := range listOut.Contents {
			if object.Key == nil {
				continue
			}
			objects = append(objects, types.ObjectIdentifier{Key: object.Key})
		}
		if len(objects) > 0 {
			_, err := b.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(b.bucket),
				Delete: &types.Delete{
					Objects: objects,
					Quiet:   aws.Bool(true),
				},
			})
			if err != nil {
				return fmt.Errorf("failed to delete S3 state objects: %w", err)
			}
		}

		if listOut.IsTruncated == nil || !*listOut.IsTruncated {
			break
		}
		continuationToken = listOut.NextContinuationToken
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

func (b *S3) envKeyPrefix(envID string) string {
	return joinS3Key(b.prefix, envID) + "/"
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
