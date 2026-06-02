package adapters

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"ork/pkg/events"
	manifestcore "ork/pkg/manifest/core"
	"ork/pkg/runners"
	"ork/pkg/state"
	"ork/pkg/utils"
)

type ScriptAdapter struct{}

type ScriptConfig struct {
	Shell []string `mapstructure:"shell"`
}

type ScriptState struct {
	Scripts []string `mapstructure:"scripts" json:"scripts"`
	Shell   []string `mapstructure:"shell" json:"shell"`
	WorkDir string   `mapstructure:"workdir" json:"workdir"`
}

func (d *ScriptAdapter) RequiredCapabilities() runners.Capabilities {
	return runners.Capabilities{Exec: true, FileCopy: true}
}

func (d *ScriptAdapter) SupportedSources() ComponentSourceSupport {
	return ComponentSourceSupport{Embedded: true, Files: true}
}

func (d *ScriptAdapter) ValidateAndLoadConfig(ctx context.Context, c *manifestcore.Component) (ComponentConfig, []events.Event, error) {
	var cfg ScriptConfig
	if err := mapstructure.Decode(c.Config, &cfg); err != nil {
		return nil, nil, err
	}
	if len(cfg.Shell) == 0 {
		cfg.Shell = []string{"sh"}
	}
	return &cfg, nil, nil
}

func (d *ScriptAdapter) Apply(ctx context.Context, c *manifestcore.Component, t runners.Runner) (ComponentApplyResult, error) {
	cfg, ok := c.LoadedConfig.(*ScriptConfig)
	if !ok {
		return ComponentApplyResult{}, fmt.Errorf("invalid config type for ScriptAdapter")
	}

	aCtx, ok := AdapterContextFromContext(ctx)
	if !ok {
		return ComponentApplyResult{}, fmt.Errorf("failed to get adapter context")
	}

	workDir := aCtx.BuildRunnerWorkDir(c.WorkDir, c.Name)
	if err := d.ensureWorkDir(ctx, t, workDir, c.Runner, c.Name); err != nil {
		return ComponentApplyResult{}, err
	}
	scripts, err := d.copySource(ctx, c, t, workDir)
	if err != nil {
		return ComponentApplyResult{}, err
	}
	// Copy with-files to runner as supporting context.
	for name, file := range c.WithFiles {
		copyRes, err := t.CopyFile(ctx, runners.FileCopyRequest{
			Source:      file,
			Destination: path.Join(workDir, name),
			ToRunner:    true,
			Overwrite:   true,
			Recursive:   false,
		})

		if err != nil {
			return ComponentApplyResult{}, fmt.Errorf("failed to copy script with-file %q to runner: %w", name, err)
		}
		if copyRes.Error != nil {
			return ComponentApplyResult{}, fmt.Errorf("error during with-file %q copy: %w", name, copyRes.Error)
		}

		aCtx.emitter.Emit(events.Event{
			Type:      events.EventInfo,
			Message:   fmt.Sprintf("Copied with-file %q to %q", name, workDir),
			Adapter:   c.Type,
			Runner:    c.Runner,
			Component: c.Name,
			Duration:  copyRes.Duration,
		})
	}

	outputEnvPath := path.Join(workDir, ".ork-outputs.env")
	outputJSONPath := path.Join(workDir, ".ork-outputs.json")
	env := make(map[string]string, len(c.Env)+2)
	for key, value := range c.Env {
		env[key] = value
	}
	env["ORK_OUTPUT_ENV"] = ".ork-outputs.env"
	env["ORK_OUTPUT_JSON"] = ".ork-outputs.json"

	if err := d.clearOutputFiles(ctx, t, workDir, c.Runner, c.Name); err != nil {
		return ComponentApplyResult{}, err
	}

	for _, script := range scripts {
		if err := d.execScript(ctx, t, cfg, script, workDir, env, c.Runner, c.Name); err != nil {
			return ComponentApplyResult{}, err
		}
	}

	localOutputDir := path.Join(aCtx.GetComponentWorkDirInOrkLocalWorkDir(c.Name), "script", "outputs")
	outputs, err := d.captureOutputs(ctx, t, outputEnvPath, outputJSONPath, localOutputDir)
	if err != nil {
		return ComponentApplyResult{}, err
	}

	scriptState := ScriptState{
		Scripts: scripts,
		Shell:   cfg.Shell,
		WorkDir: workDir,
	}
	stateData, err := state.NewComponentStateData(workDir, scriptState)
	if err != nil {
		return ComponentApplyResult{}, err
	}

	return ComponentApplyResult{
		Outputs: ComponentApplyOutput(outputs),
		State:   stateData,
	}, nil
}

func (d *ScriptAdapter) Destroy(ctx context.Context, c state.ComponentState, t runners.Runner) error {
	return nil
}

func (d *ScriptAdapter) ensureWorkDir(ctx context.Context, t runners.Runner, workDir string, runnerName string, componentName string) error {
	res, err := t.Exec(ctx, runners.ExecCommand{
		Command: []string{"mkdir", "-p", workDir},
		Timeout: 0,
		Stderr:  utils.NewPrefixWriter(os.Stderr, utils.RunnerComponentPrefix(runnerName, componentName)),
		Stdout:  utils.NewPrefixWriter(os.Stdout, utils.RunnerComponentPrefix(runnerName, componentName)),
	})
	if err != nil {
		return fmt.Errorf("failed to create script workdir %q: %w", workDir, err)
	}
	if res.Error != nil || res.ExitCode != 0 {
		return fmt.Errorf("failed to create script workdir %q with exit code %d: %v", workDir, res.ExitCode, res.Error)
	}
	return nil
}

func (d *ScriptAdapter) copySource(ctx context.Context, c *manifestcore.Component, t runners.Runner, workDir string) ([]string, error) {
	switch c.Source.Type() {
	case manifestcore.ComponentSourceTypeEmbedded:
		return d.copyEmbeddedSource(ctx, c, t, workDir)
	case manifestcore.ComponentSourceTypeFiles:
		scripts := make([]string, 0, len(c.Source.Files))
		for script, file := range c.Source.Files {
			if err := copyScriptFile(ctx, t, file, path.Join(workDir, script)); err != nil {
				return nil, err
			}
			scripts = append(scripts, script)
		}
		return scripts, nil
	default:
		return nil, fmt.Errorf("unsupported script source type %q", c.Source.Type())
	}
}

func copyScriptFile(ctx context.Context, t runners.Runner, source string, destination string) error {
	copyRes, err := t.CopyFile(ctx, runners.FileCopyRequest{
		Source:      source,
		Destination: destination,
		ToRunner:    true,
		Overwrite:   true,
		Recursive:   false,
	})
	if err != nil {
		return fmt.Errorf("failed to copy script source to runner: %w", err)
	}
	if copyRes.Error != nil {
		return fmt.Errorf("error during script source copy: %w", copyRes.Error)
	}
	return nil
}

func (d *ScriptAdapter) execScript(ctx context.Context, t runners.Runner, cfg *ScriptConfig, script string, workDir string, env map[string]string, runnerName string, componentName string) error {
	execRes, err := t.Exec(ctx, runners.ExecCommand{
		WorkingDir: workDir,
		Command:    utils.ShellCommand(cfg.Shell, "./"+script),
		Env:        env,
		Timeout:    0,
		Stderr:     utils.NewPrefixWriter(os.Stderr, utils.RunnerComponentPrefix(runnerName, componentName)),
		Stdout:     utils.NewPrefixWriter(os.Stdout, utils.RunnerComponentPrefix(runnerName, componentName)),
	})
	if err != nil {
		return fmt.Errorf("failed to execute script %q: %w", script, err)
	}
	if execRes.Error != nil || execRes.ExitCode != 0 {
		return fmt.Errorf("script %q failed with exit code %d: %v", script, execRes.ExitCode, execRes.Error)
	}
	return nil
}

func (d *ScriptAdapter) clearOutputFiles(ctx context.Context, t runners.Runner, workDir string, runnerName string, componentName string) error {
	res, err := t.Exec(ctx, runners.ExecCommand{
		WorkingDir: workDir,
		Command:    []string{"rm", "-f", ".ork-outputs.env", ".ork-outputs.json"},
		Timeout:    0,
		Stderr:     utils.NewPrefixWriter(os.Stderr, utils.RunnerComponentPrefix(runnerName, componentName)),
		Stdout:     utils.NewPrefixWriter(os.Stdout, utils.RunnerComponentPrefix(runnerName, componentName)),
	})
	if err != nil {
		return fmt.Errorf("failed to clear script output files: %w", err)
	}
	if res.Error != nil || res.ExitCode != 0 {
		return fmt.Errorf("failed to clear script output files with exit code %d: %v", res.ExitCode, res.Error)
	}
	return nil
}

func (d *ScriptAdapter) copyEmbeddedSource(ctx context.Context, c *manifestcore.Component, t runners.Runner, workDir string) ([]string, error) {
	aCtx, ok := AdapterContextFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get adapter context")
	}

	dir := path.Join(aCtx.GetComponentWorkDirInOrkLocalWorkDir(c.Name), "script")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create local script source directory: %w", err)
	}

	scriptPath := path.Join(dir, "script.sh")
	if err := os.WriteFile(scriptPath, []byte(c.Source.Embedded), 0755); err != nil {
		return nil, fmt.Errorf("failed to write embedded script: %w", err)
	}

	if err := copyScriptFile(ctx, t, scriptPath, path.Join(workDir, "script.sh")); err != nil {
		return nil, err
	}
	return []string{"script.sh"}, nil
}

func (d *ScriptAdapter) captureOutputs(ctx context.Context, t runners.Runner, envPath string, jsonPath string, localOutputDir string) (map[string]string, error) {
	outputs := make(map[string]string)

	// TODO: Distinguish "output file was not created" from real copy failures
	// such as permission errors or runner transport failures. Missing output
	// files are valid because scripts may choose only one output format, but
	// other copy failures should not be silently treated as empty outputs.
	envData, err := copyRunnerFileToBytes(ctx, t, envPath, path.Join(localOutputDir, ".ork-outputs.env"))
	if err == nil && len(bytes.TrimSpace(envData)) > 0 {
		parsed, err := parseOutputEnv(envData)
		if err != nil {
			return nil, err
		}
		for key, value := range parsed {
			outputs[key] = value
		}
	}

	jsonData, err := copyRunnerFileToBytes(ctx, t, jsonPath, path.Join(localOutputDir, ".ork-outputs.json"))
	if err == nil && len(bytes.TrimSpace(jsonData)) > 0 {
		parsed, err := parseOutputJSON(jsonData)
		if err != nil {
			return nil, err
		}
		for key, value := range parsed {
			outputs[key] = value
		}
	}

	return outputs, nil
}

func copyRunnerFileToBytes(ctx context.Context, t runners.Runner, runnerPath string, localPath string) ([]byte, error) {
	if err := os.MkdirAll(path.Dir(localPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create local output directory: %w", err)
	}
	copyRes, err := t.CopyFile(ctx, runners.FileCopyRequest{
		Source:      runnerPath,
		Destination: localPath,
		ToRunner:    false,
		Overwrite:   true,
		Recursive:   false,
	})
	if err != nil {
		return nil, err
	}
	if copyRes.Error != nil {
		return nil, copyRes.Error
	}
	return os.ReadFile(localPath)
}

func parseOutputEnv(data []byte) (map[string]string, error) {
	outputs := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		// escape comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid output env line %d: expected KEY=VALUE", lineNumber)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid output env line %d: key is empty", lineNumber)
		}
		outputs[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse output env: %w", err)
	}
	return outputs, nil
}

func parseOutputJSON(data []byte) (map[string]string, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse output JSON: %w", err)
	}

	outputs := make(map[string]string, len(raw))
	for key, value := range raw {
		switch typed := value.(type) {
		case string:
			outputs[key] = typed
		case bool:
			outputs[key] = strconv.FormatBool(typed)
		case float64:
			outputs[key] = strconv.FormatFloat(typed, 'f', -1, 64)
		case nil:
			outputs[key] = ""
		default:
			return nil, fmt.Errorf("output JSON key %q must be a scalar value", key)
		}
	}
	return outputs, nil
}

func init() {
	Register("script", &ScriptAdapter{})
}
