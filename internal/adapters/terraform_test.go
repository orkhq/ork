package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectTerraformBackend(t *testing.T) {
	t.Run("detects backend block", func(t *testing.T) {
		modulePath := t.TempDir()
		writeTestFile(t, filepath.Join(modulePath, "main.tf"), `
terraform {
  backend "s3" {}
}
`)

		hasBackend, backendType, err := detectTerraformBackend(modulePath)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if !hasBackend {
			t.Fatal("expected backend to be detected")
		}
		if backendType != "s3" {
			t.Fatalf("expected backend type s3, got %q", backendType)
		}
	})

	t.Run("ignores modules without backend block", func(t *testing.T) {
		modulePath := t.TempDir()
		writeTestFile(t, filepath.Join(modulePath, "main.tf"), `
resource "terraform_data" "smoke" {
  input = "orch"
}
`)

		hasBackend, backendType, err := detectTerraformBackend(modulePath)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if hasBackend {
			t.Fatalf("expected no backend, got %q", backendType)
		}
	})
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
}
