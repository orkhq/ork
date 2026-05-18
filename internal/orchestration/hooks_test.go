package orchestration

import (
	"context"
	"strings"
	"testing"

	"orch.io/pkg/events"
	manifestcore "orch.io/pkg/manifest/core"
	"orch.io/pkg/runners"
	"orch.io/pkg/varresolvers"
)

type fakeHookRunner struct {
	commands []runners.ExecCommand
}

func (f *fakeHookRunner) Name() string { return "local" }

func (f *fakeHookRunner) Type() runners.RunnerType { return runners.RunnerTypeLocal }

func (f *fakeHookRunner) ValidateAndInitialize() error { return nil }

func (f *fakeHookRunner) Capabilities() runners.Capabilities {
	return runners.Capabilities{Exec: true}
}

func (f *fakeHookRunner) Exec(ctx context.Context, command runners.ExecCommand) (*runners.ExecResult, error) {
	f.commands = append(f.commands, command)
	return &runners.ExecResult{ExitCode: 0}, nil
}

func (f *fakeHookRunner) CopyFile(ctx context.Context, req runners.FileCopyRequest) (*runners.FileCopyResult, error) {
	return &runners.FileCopyResult{}, nil
}

func (f *fakeHookRunner) UsesNonAmbientCredentials() (bool, []string) { return false, nil }

func (f *fakeHookRunner) Disconnect() error { return nil }

type fakeHookEmitter struct {
	events []events.Event
}

func (f *fakeHookEmitter) Emit(event events.Event) {
	f.events = append(f.events, event)
}

func TestRunLifecycleHooksInterpolatesCommandAndEnv(t *testing.T) {
	componentResolver := varresolvers.NewComponentResolver()
	componentResolver.RegisterPersistedComponentOutput("database", map[string]string{
		"url": "postgres://localhost:5432/app",
	})

	runner := &fakeHookRunner{}
	emitter := &fakeHookEmitter{}
	component := &manifestcore.Component{Name: "api", Type: "docker-compose"}
	err := runLifecycleHooks(context.Background(), runner, []manifestcore.Hook{
		{
			Command: `echo "${database.outputs.url}"`,
			Shell:   []string{"bash", "-c"},
			Env: map[string]string{
				"DATABASE_URL": "${database.outputs.url}",
			},
		},
	}, lifecyclePreApply, hookExecutionContext{
		envID:        "test-env",
		componentRef: component,
		component:    "api",
		runner:       "local",
		workDir:      "/tmp/orch/test-env/api",
		baseEnv:      map[string]string{"BASE": "present"},
		resolver:     componentResolver,
		emitter:      emitter,
	})
	if err != nil {
		t.Fatalf("runLifecycleHooks returned error: %v", err)
	}

	if len(runner.commands) != 2 {
		t.Fatalf("expected mkdir plus hook command, got %d commands", len(runner.commands))
	}

	hookCommand := runner.commands[1]
	if got := hookCommand.Command; len(got) != 3 || got[0] != "bash" || got[1] != "-c" || got[2] != `echo "postgres://localhost:5432/app"` {
		t.Fatalf("unexpected hook command: %#v", got)
	}
	if hookCommand.Env["DATABASE_URL"] != "postgres://localhost:5432/app" {
		t.Fatalf("expected interpolated hook env, got %q", hookCommand.Env["DATABASE_URL"])
	}
	if hookCommand.Env["ORCH_LIFECYCLE"] != string(lifecyclePreApply) {
		t.Fatalf("expected ORCH_LIFECYCLE, got %q", hookCommand.Env["ORCH_LIFECYCLE"])
	}
	if hookCommand.Env["ORCH_WORKDIR"] != "/tmp/orch/test-env/api" {
		t.Fatalf("expected ORCH_WORKDIR, got %q", hookCommand.Env["ORCH_WORKDIR"])
	}
	if hookCommand.Env["BASE"] != "present" {
		t.Fatalf("expected base env to be preserved")
	}
	if len(emitter.events) != 2 {
		t.Fatalf("expected start and success hook events, got %d", len(emitter.events))
	}
	if emitter.events[0].Type != events.EventStart || emitter.events[0].Stage != string(lifecyclePreApply) {
		t.Fatalf("unexpected start event: %#v", emitter.events[0])
	}
	if emitter.events[1].Type != events.EventSuccess || emitter.events[1].Stage != string(lifecyclePreApply) {
		t.Fatalf("unexpected success event: %#v", emitter.events[1])
	}
}

func TestRunLifecycleHooksFailsOnMissingInterpolation(t *testing.T) {
	runner := &fakeHookRunner{}
	err := runLifecycleHooks(context.Background(), runner, []manifestcore.Hook{
		{Command: `echo "${database.outputs.url}"`},
	}, lifecyclePreApply, hookExecutionContext{
		envID:     "test-env",
		component: "api",
		runner:    "local",
		resolver:  varresolvers.NewComponentResolver(),
	})
	if err == nil {
		t.Fatal("expected missing interpolation to fail")
	}
}

func TestRunLifecycleHooksDoesNotUseEnvResolverForCommands(t *testing.T) {
	t.Setenv("ORCH_TEST_COMMAND_SECRET", "secret")

	runner := &fakeHookRunner{}
	err := runLifecycleHooks(context.Background(), runner, []manifestcore.Hook{
		{Command: `echo "${ORCH_TEST_COMMAND_SECRET}"`},
	}, lifecyclePreApply, hookExecutionContext{
		envID:        "test-env",
		componentRef: &manifestcore.Component{Name: "api", Type: "script"},
		component:    "api",
		runner:       "local",
		resolver: &varresolvers.ChainResolver{
			Resolvers: []varresolvers.Resolver{
				varresolvers.NewEnvResolver(),
			},
		},
		commandResolver: varresolvers.NewComponentResolver(),
	})
	if err == nil {
		t.Fatal("expected command interpolation through env resolver to fail")
	}
	if !strings.Contains(err.Error(), "ORCH_TEST_COMMAND_SECRET") {
		t.Fatalf("expected missing command interpolation error to mention env name, got %v", err)
	}
}

func TestRunLifecycleHooksStillUsesEnvResolverForHookEnv(t *testing.T) {
	t.Setenv("ORCH_TEST_HOOK_ENV", "from-os-env")

	runner := &fakeHookRunner{}
	err := runLifecycleHooks(context.Background(), runner, []manifestcore.Hook{
		{
			Command: "true",
			Env: map[string]string{
				"FROM_ENV": "${ORCH_TEST_HOOK_ENV}",
			},
		},
	}, lifecyclePreApply, hookExecutionContext{
		envID:        "test-env",
		componentRef: &manifestcore.Component{Name: "api", Type: "script"},
		component:    "api",
		runner:       "local",
		resolver: &varresolvers.ChainResolver{
			Resolvers: []varresolvers.Resolver{
				varresolvers.NewEnvResolver(),
			},
		},
		commandResolver: varresolvers.NewComponentResolver(),
	})
	if err != nil {
		t.Fatalf("runLifecycleHooks returned error: %v", err)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("expected one hook command, got %d", len(runner.commands))
	}
	if runner.commands[0].Env["FROM_ENV"] != "from-os-env" {
		t.Fatalf("expected hook env to resolve from OS env, got %q", runner.commands[0].Env["FROM_ENV"])
	}
}
