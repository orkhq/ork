package statebackends

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"orch.io/pkg/logging"
	"orch.io/pkg/state"
)

func TestDecodeS3Config(t *testing.T) {
	t.Run("requires bucket", func(t *testing.T) {
		_, err := decodeS3Config(map[string]interface{}{"prefix": "orch"})
		if err == nil {
			t.Fatal("expected error for missing bucket")
		}
	})

	t.Run("rejects unknown config", func(t *testing.T) {
		_, err := decodeS3Config(map[string]interface{}{
			"bucket": "orch-state",
			"path":   "not-for-s3",
		})
		if err == nil {
			t.Fatal("expected error for unknown config")
		}
	})

	t.Run("accepts encryption config", func(t *testing.T) {
		cfg, err := decodeS3Config(map[string]interface{}{
			"bucket":                 "orch-state",
			"prefix":                 "previews/",
			"server_side_encryption": string(types.ServerSideEncryptionAwsKms),
			"kms_key_id":             "alias/orch",
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
			"bucket":                 "orch-state",
			"server_side_encryption": string(types.ServerSideEncryptionAes256),
			"kms_key_id":             "alias/orch",
		})
		if err == nil {
			t.Fatal("expected error for kms key without aws:kms encryption")
		}
	})
}

func TestS3KeyLayout(t *testing.T) {
	backend := NewS3(nil, S3Config{
		Bucket: "orch-state",
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
