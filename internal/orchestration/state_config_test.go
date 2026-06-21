package orchestration

import (
	"context"
	"testing"

	manifestcore "ork/pkg/manifest/core"
	"ork/pkg/varresolvers"
)

func TestResolveStateConfigInterpolatesInputsAndEnvironment(t *testing.T) {
	t.Setenv("ORK_STATE_PROFILE", "ork-minio")
	t.Setenv("MINIO_SECRET_ACCESS_KEY", "minio-secret")
	inputsResolver, err := varresolvers.NewInputsResolver(
		map[string]string{"state_bucket": "ork-state", "access_key": "minio-access"},
		map[string]manifestcore.Input{
			"state_bucket": {Required: true},
			"access_key":   {Required: true, Sensitive: true},
		},
	)
	if err != nil {
		t.Fatalf("create inputs resolver: %v", err)
	}
	original := &manifestcore.StateConfig{
		Backend: "s3",
		Auth: map[string]interface{}{
			"profile": "${ORK_STATE_PROFILE}",
		},
		Config: map[string]interface{}{
			"bucket": "${inputs.state_bucket}",
		},
	}

	resolved, err := resolveStateConfig(context.Background(), original, stateConfigResolver(inputsResolver))
	if err != nil {
		t.Fatalf("resolve state config: %v", err)
	}
	if resolved.Auth["profile"] != "ork-minio" {
		t.Fatalf("profile = %v, want ork-minio", resolved.Auth["profile"])
	}
	if resolved.Config["bucket"] != "ork-state" {
		t.Fatalf("bucket = %v, want ork-state", resolved.Config["bucket"])
	}
	if original.Auth["profile"] != "${ORK_STATE_PROFILE}" {
		t.Fatal("state config interpolation mutated the manifest")
	}
}

func TestResolveStateConfigInterpolatesAccessKeys(t *testing.T) {
	t.Setenv("MINIO_ACCESS_KEY_ID", "minio-access")
	t.Setenv("MINIO_SECRET_ACCESS_KEY", "minio-secret")
	inputsResolver, err := varresolvers.NewInputsResolver(map[string]string{}, nil)
	if err != nil {
		t.Fatalf("create inputs resolver: %v", err)
	}

	resolved, err := resolveStateConfig(context.Background(), &manifestcore.StateConfig{
		Backend: "s3",
		Auth: map[string]interface{}{
			"access_key_id":     "${MINIO_ACCESS_KEY_ID}",
			"secret_access_key": "${MINIO_SECRET_ACCESS_KEY}",
		},
	}, stateConfigResolver(inputsResolver))
	if err != nil {
		t.Fatalf("resolve state config: %v", err)
	}
	if resolved.Auth["access_key_id"] != "minio-access" || resolved.Auth["secret_access_key"] != "minio-secret" {
		t.Fatal("state access key credentials were not interpolated")
	}
}
