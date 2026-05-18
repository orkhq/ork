package adapters

import (
	"strings"
	"testing"

	manifestcore "orch.io/pkg/manifest/core"
)

func TestDetectExplicitProviderCredentialEnv(t *testing.T) {
	got := detectExplicitProviderCredentialEnv(map[string]string{
		"AWS_ACCESS_KEY_ID":     "key",
		"aws_secret_access_key": "secret",
		"AWS_PROFILE":           "profile",
		"APP_TOKEN":             "app-token",
	})

	want := []string{"env.AWS_ACCESS_KEY_ID", "env.aws_secret_access_key"}
	if len(got) != len(want) {
		t.Fatalf("credential refs = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("credential refs = %#v, want %#v", got, want)
		}
	}
}

func TestDetectExplicitTerraformCredentialVars(t *testing.T) {
	got := detectExplicitTerraformCredentialVars(map[string]string{
		"access_key":        "key",
		"aws_session_token": "token",
		"db_password":       "app-secret",
	})

	want := []string{"vars.access_key", "vars.aws_session_token"}
	if len(got) != len(want) {
		t.Fatalf("credential refs = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("credential refs = %#v, want %#v", got, want)
		}
	}
}

func TestProviderCredentialWarningDoesNotIncludeValues(t *testing.T) {
	event := providerCredentialWarning(&manifestcore.Component{
		Name:   "infra",
		Type:   "terraform",
		Runner: "local",
	}, []string{"env.AWS_SECRET_ACCESS_KEY"})

	if !strings.Contains(event.Message, "env.AWS_SECRET_ACCESS_KEY") {
		t.Fatalf("warning message does not include credential ref: %q", event.Message)
	}
	if strings.Contains(event.Message, "secret-value") {
		t.Fatalf("warning message should not include credential value: %q", event.Message)
	}
	if event.Component != "infra" || event.Adapter != "terraform" || event.Runner != "local" {
		t.Fatalf("unexpected event metadata: %#v", event)
	}
}
