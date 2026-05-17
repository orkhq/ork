package adapters

import "testing"

func TestParseOutputEnv(t *testing.T) {
	outputs, err := parseOutputEnv([]byte(`
# ignored
token=abc
api_url=http://localhost:8080
token=latest
`))
	if err != nil {
		t.Fatalf("parseOutputEnv returned error: %v", err)
	}
	if outputs["token"] != "latest" {
		t.Fatalf("expected last token value to win, got %q", outputs["token"])
	}
	if outputs["api_url"] != "http://localhost:8080" {
		t.Fatalf("unexpected api_url: %q", outputs["api_url"])
	}
}

func TestParseOutputJSON(t *testing.T) {
	outputs, err := parseOutputJSON([]byte(`{"token":"abc","enabled":true,"port":8080}`))
	if err != nil {
		t.Fatalf("parseOutputJSON returned error: %v", err)
	}
	if outputs["token"] != "abc" {
		t.Fatalf("unexpected token: %q", outputs["token"])
	}
	if outputs["enabled"] != "true" {
		t.Fatalf("unexpected enabled: %q", outputs["enabled"])
	}
	if outputs["port"] != "8080" {
		t.Fatalf("unexpected port: %q", outputs["port"])
	}
}

func TestParseOutputJSONRejectsNestedValues(t *testing.T) {
	if _, err := parseOutputJSON([]byte(`{"nested":{"token":"abc"}}`)); err == nil {
		t.Fatal("expected nested output value to fail")
	}
}
