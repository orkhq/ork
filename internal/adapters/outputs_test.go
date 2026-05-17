package adapters

import "testing"

func TestParseTerraformOutputsDropsSensitiveOutputs(t *testing.T) {
	outputs, err := parseTerraformOutputs([]byte(`{
		"token": {
			"sensitive": true,
			"type": "string",
			"value": "secret"
		},
		"url": {
			"sensitive": false,
			"type": "string",
			"value": "http://localhost:8080"
		},
		"enabled": {
			"sensitive": false,
			"type": "bool",
			"value": true
		}
	}`))
	if err != nil {
		t.Fatalf("parseTerraformOutputs returned error: %v", err)
	}
	if _, ok := outputs["token"]; ok {
		t.Fatal("expected sensitive terraform output to be dropped")
	}
	if outputs["url"] != "http://localhost:8080" {
		t.Fatalf("unexpected url output: %q", outputs["url"])
	}
	if outputs["enabled"] != "true" {
		t.Fatalf("unexpected enabled output: %q", outputs["enabled"])
	}
}

func TestParseCloudFormationOutputs(t *testing.T) {
	outputs, err := parseCloudFormationOutputs([]byte(`[
		{"OutputKey": "PublicIP", "OutputValue": "127.0.0.1"},
		{"OutputKey": "URL", "OutputValue": "http://localhost:8080"}
	]`))
	if err != nil {
		t.Fatalf("parseCloudFormationOutputs returned error: %v", err)
	}
	if outputs["PublicIP"] != "127.0.0.1" {
		t.Fatalf("unexpected PublicIP output: %q", outputs["PublicIP"])
	}
	if outputs["URL"] != "http://localhost:8080" {
		t.Fatalf("unexpected URL output: %q", outputs["URL"])
	}
}
