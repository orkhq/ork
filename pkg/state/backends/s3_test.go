package statebackends

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"ork/pkg/logging"
	"ork/pkg/state"
)

func TestDecodeS3Config(t *testing.T) {
	t.Run("requires bucket", func(t *testing.T) {
		_, err := decodeS3Config(map[string]interface{}{"prefix": "ork"})
		if err == nil {
			t.Fatal("expected error for missing bucket")
		}
	})

	t.Run("rejects unknown config", func(t *testing.T) {
		_, err := decodeS3Config(map[string]interface{}{
			"bucket": "ork-state",
			"path":   "not-for-s3",
		})
		if err == nil {
			t.Fatal("expected error for unknown config")
		}
	})

	t.Run("accepts encryption config", func(t *testing.T) {
		cfg, err := decodeS3Config(map[string]interface{}{
			"bucket":                 "ork-state",
			"prefix":                 "previews/",
			"server_side_encryption": string(types.ServerSideEncryptionAwsKms),
			"kms_key_id":             "alias/ork",
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if cfg.Prefix != "previews/" {
			t.Fatalf("expected raw prefix to be preserved before backend construction, got %q", cfg.Prefix)
		}
	})

	t.Run("kms key requires kms encryption", func(t *testing.T) {
		_, err := decodeS3Config(map[string]interface{}{
			"bucket":                 "ork-state",
			"server_side_encryption": string(types.ServerSideEncryptionAes256),
			"kms_key_id":             "alias/ork",
		})
		if err == nil {
			t.Fatal("expected error for kms key without aws:kms encryption")
		}
	})
}

func TestS3KeyLayout(t *testing.T) {
	backend := NewS3(nil, S3Config{
		Bucket: "ork-state",
		Prefix: "/previews/",
	}, &logging.NoopDebugLogger{})

	if got := backend.stateKey("pr-123"); got != "previews/pr-123/state.json" {
		t.Fatalf("unexpected state key %q", got)
	}

	key, err := backend.artifactKey("pr-123", "tf", state.Artifact{Path: "nested/terraform.tfstate"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if key != "previews/pr-123/artifacts/tf/nested/terraform.tfstate" {
		t.Fatalf("unexpected artifact key %q", key)
	}
}

func TestS3DeleteRemovesEnvironmentPrefix(t *testing.T) {
	client := &deleteRecordingS3Client{
		listPages: []*s3.ListObjectsV2Output{
			{
				Contents: []types.Object{
					{Key: aws.String("previews/pr-123/state.json")},
					{Key: aws.String("previews/pr-123/artifacts/tf/terraform.tfstate")},
				},
			},
		},
	}
	backend := NewS3(client, S3Config{
		Bucket: "ork-state",
		Prefix: "previews",
	}, &logging.NoopDebugLogger{})

	if err := backend.Delete(context.Background(), "pr-123"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if client.listPrefix != "previews/pr-123/" {
		t.Fatalf("list prefix = %q, want %q", client.listPrefix, "previews/pr-123/")
	}
	if len(client.deletedKeys) != 2 {
		t.Fatalf("deleted keys = %v, want 2 keys", client.deletedKeys)
	}
	want := map[string]bool{
		"previews/pr-123/state.json":                     true,
		"previews/pr-123/artifacts/tf/terraform.tfstate": true,
	}
	for _, key := range client.deletedKeys {
		if !want[key] {
			t.Fatalf("unexpected deleted key %q", key)
		}
	}
}

type deleteRecordingS3Client struct {
	listPages   []*s3.ListObjectsV2Output
	listPrefix  string
	deletedKeys []string
}

func (c *deleteRecordingS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	panic("not implemented")
}

func (c *deleteRecordingS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	panic("not implemented")
}

func (c *deleteRecordingS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	panic("not implemented")
}

func (c *deleteRecordingS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	c.listPrefix = aws.ToString(params.Prefix)
	if len(c.listPages) == 0 {
		return &s3.ListObjectsV2Output{}, nil
	}
	page := c.listPages[0]
	c.listPages = c.listPages[1:]
	return page, nil
}

func (c *deleteRecordingS3Client) DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	for _, object := range params.Delete.Objects {
		c.deletedKeys = append(c.deletedKeys, aws.ToString(object.Key))
	}
	return &s3.DeleteObjectsOutput{}, nil
}
