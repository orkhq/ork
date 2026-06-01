package adapters

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"orch/pkg/runners"
)

type fakeComposePortRunner struct {
	stdoutByServicePort map[string][]byte
}

func (f fakeComposePortRunner) Name() string { return "local" }

func (f fakeComposePortRunner) Type() runners.RunnerType { return runners.RunnerTypeLocal }

func (f fakeComposePortRunner) ValidateAndInitialize() error { return nil }

func (f fakeComposePortRunner) Capabilities() runners.Capabilities {
	return runners.Capabilities{Exec: true, FileCopy: true}
}

func (f fakeComposePortRunner) Exec(ctx context.Context, command runners.ExecCommand) (*runners.ExecResult, error) {
	service := command.Command[len(command.Command)-2]
	port := command.Command[len(command.Command)-1]
	return &runners.ExecResult{
		Stdout: f.stdoutByServicePort[service+":"+port],
	}, nil
}

func (f fakeComposePortRunner) CopyFile(ctx context.Context, req runners.FileCopyRequest) (*runners.FileCopyResult, error) {
	return &runners.FileCopyResult{}, nil
}

func (f fakeComposePortRunner) UsesNonAmbientCredentials() (bool, []string) { return false, nil }

func (f fakeComposePortRunner) Disconnect() error { return nil }

func TestLoadComposeFileAndExtractServicesNormalizesPortSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yml")
	if err := os.WriteFile(composeFile, []byte(`
services:
  web:
    ports:
      - "80"
      - "8080:8080"
      - target: 9090
        published: "19090"
`), 0644); err != nil {
		t.Fatal(err)
	}

	services, err := loadComposeFileAndExtractServices(composeFile)
	if err != nil {
		t.Fatalf("expected compose file to parse: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected one service, got %d", len(services))
	}
	if services[0].Name != "web" {
		t.Fatalf("unexpected service name: %q", services[0].Name)
	}
	if !services[0].HasFixedPorts {
		t.Fatal("expected fixed port mapping to be detected")
	}
	wantPorts := []string{"80", "8080", "9090"}
	if !reflect.DeepEqual(services[0].Ports, wantPorts) {
		t.Fatalf("ports mismatch\nwant: %#v\n got: %#v", wantPorts, services[0].Ports)
	}
}

func TestDockerComposeCapturePortOutputs(t *testing.T) {
	adapter := &DockerComposeAdapter{}
	outputs, err := adapter.capturePortOutputs(
		context.Background(),
		fakeComposePortRunner{stdoutByServicePort: map[string][]byte{
			"web:80": []byte("0.0.0.0:49153\n"),
		}},
		"/work",
		[]string{"docker", "compose", "-f", "compose.yml"},
		"orch_test_web",
		nil,
		map[string][]ComposeServiceMetaData{
			"compose.yml": {
				{Name: "web", Ports: []string{"80"}},
			},
		},
	)
	if err != nil {
		t.Fatalf("expected port outputs to capture: %v", err)
	}
	if outputs["_meta.ports.services.web.80"] != "49153" {
		t.Fatalf("unexpected port output: %q", outputs["_meta.ports.services.web.80"])
	}
	if outputs["_meta.bindings.services.web.80"] != "0.0.0.0:49153" {
		t.Fatalf("unexpected binding output: %q", outputs["_meta.bindings.services.web.80"])
	}
}
