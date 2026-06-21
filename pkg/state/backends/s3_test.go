package statebackends

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
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

	t.Run("accepts S3-compatible endpoint config", func(t *testing.T) {
		cfg, err := decodeS3Config(map[string]interface{}{
			"bucket":           "ork-state",
			"region":           "us-east-1",
			"endpoint":         "https://objects.example.com/",
			"force_path_style": true,
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if cfg.Endpoint != "https://objects.example.com" {
			t.Fatalf("endpoint = %q, want normalized endpoint", cfg.Endpoint)
		}
		if !cfg.ForcePathStyle {
			t.Fatal("expected force_path_style to be enabled")
		}
	})

	t.Run("rejects invalid endpoint", func(t *testing.T) {
		_, err := decodeS3Config(map[string]interface{}{
			"bucket":   "ork-state",
			"endpoint": "objects.example.com",
		})
		if err == nil {
			t.Fatal("expected error for endpoint without scheme")
		}
	})

	t.Run("rejects credentials in endpoint", func(t *testing.T) {
		_, err := decodeS3Config(map[string]interface{}{
			"bucket":   "ork-state",
			"endpoint": "https://access:secret@objects.example.com",
		})
		if err == nil {
			t.Fatal("expected error for credentials in endpoint")
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

func TestDecodeS3AuthConfig(t *testing.T) {
	t.Run("defaults to ambient credentials", func(t *testing.T) {
		cfg, err := decodeS3AuthConfig(nil)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if cfg.Profile != "" {
			t.Fatalf("profile = %q, want empty ambient profile", cfg.Profile)
		}
	})

	t.Run("accepts profile", func(t *testing.T) {
		cfg, err := decodeS3AuthConfig(map[string]interface{}{"profile": " ork-minio "})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if cfg.Profile != "ork-minio" {
			t.Fatalf("profile = %q, want %q", cfg.Profile, "ork-minio")
		}
	})

	t.Run("accepts access key credentials", func(t *testing.T) {
		cfg, err := decodeS3AuthConfig(map[string]interface{}{
			"access_key_id":     "minio-access",
			"secret_access_key": "minio-secret",
			"session_token":     "temporary-token",
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if cfg.AccessKeyID != "minio-access" || cfg.SecretAccessKey != "minio-secret" || cfg.SessionToken != "temporary-token" {
			t.Fatal("decoded access key credentials do not match")
		}
	})

	t.Run("requires complete access key credentials", func(t *testing.T) {
		_, err := decodeS3AuthConfig(map[string]interface{}{"access_key_id": "minio-access"})
		if err == nil {
			t.Fatal("expected incomplete access key credentials to be rejected")
		}
	})

	t.Run("rejects profile combined with access keys", func(t *testing.T) {
		_, err := decodeS3AuthConfig(map[string]interface{}{
			"profile":           "ork-minio",
			"access_key_id":     "minio-access",
			"secret_access_key": "minio-secret",
		})
		if err == nil {
			t.Fatal("expected mixed auth sources to be rejected")
		}
	})

	t.Run("rejects unknown auth config", func(t *testing.T) {
		_, err := decodeS3AuthConfig(map[string]interface{}{"access_key": "do-not-allow-secrets"})
		if err == nil {
			t.Fatal("expected unknown auth config to be rejected")
		}
	})
}

func TestS3AccessKeysOverrideAmbientCredentials(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "real-aws-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "real-aws-secret")

	backend, err := NewS3FromConfig(
		context.Background(),
		map[string]interface{}{"bucket": "ork-state", "region": "us-east-1"},
		map[string]interface{}{
			"access_key_id":     "minio-access",
			"secret_access_key": "minio-secret",
		},
		&logging.NoopDebugLogger{},
	)
	if err != nil {
		t.Fatalf("configure S3 backend: %v", err)
	}
	client := backend.client.(*s3.Client)
	resolved, err := client.Options().Credentials.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("retrieve access key credentials: %v", err)
	}
	if resolved.AccessKeyID != "minio-access" {
		t.Fatalf("access key = %q, want explicit state credentials", resolved.AccessKeyID)
	}
}

func TestS3ProfileOverridesAmbientCredentials(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "credentials")
	credentialsFile := "[ork-minio]\naws_access_key_id = minio-access\naws_secret_access_key = minio-secret\n"
	if err := os.WriteFile(credentialsPath, []byte(credentialsFile), 0600); err != nil {
		t.Fatalf("write credentials file: %v", err)
	}
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credentialsPath)
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "missing-config"))
	t.Setenv("AWS_ACCESS_KEY_ID", "real-aws-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "real-aws-secret")

	backend, err := NewS3FromConfig(
		context.Background(),
		map[string]interface{}{"bucket": "ork-state", "region": "us-east-1"},
		map[string]interface{}{"profile": "ork-minio"},
		&logging.NoopDebugLogger{},
	)
	if err != nil {
		t.Fatalf("configure S3 backend: %v", err)
	}
	client, ok := backend.client.(*s3.Client)
	if !ok {
		t.Fatalf("client = %T, want *s3.Client", backend.client)
	}
	credentials, err := client.Options().Credentials.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("retrieve profile credentials: %v", err)
	}
	if credentials.AccessKeyID != "minio-access" {
		t.Fatalf("access key = %q, want selected profile credentials", credentials.AccessKeyID)
	}
}

func TestNewS3ClientConfiguresCompatibleStorage(t *testing.T) {
	client := newS3Client(aws.Config{Region: "us-east-1"}, S3Config{
		Endpoint:       "https://objects.example.com",
		ForcePathStyle: true,
	})
	options := client.Options()

	if got := aws.ToString(options.BaseEndpoint); got != "https://objects.example.com" {
		t.Fatalf("base endpoint = %q, want %q", got, "https://objects.example.com")
	}
	if !options.UsePathStyle {
		t.Fatal("expected path-style addressing to be enabled")
	}
}

func TestS3CompatibleEndpointReceivesPathStyleRequest(t *testing.T) {
	var gotPath string
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := S3Config{
		Bucket:         "ork-state",
		Prefix:         "previews",
		Endpoint:       server.URL,
		ForcePathStyle: true,
	}
	client := newS3Client(aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("access", "secret", ""),
	}, cfg)
	backend := NewS3(client, cfg, &logging.NoopDebugLogger{})

	if err := backend.Save(context.Background(), "pr-123", &state.OrkState{}); err != nil {
		t.Fatalf("save through compatible endpoint failed: %v", err)
	}
	if gotPath != "/ork-state/previews/pr-123/state.json" {
		t.Fatalf("request path = %q, want path-style bucket and state key", gotPath)
	}
	if gotAuthorization == "" {
		t.Fatal("expected request to use AWS signature authentication")
	}
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
