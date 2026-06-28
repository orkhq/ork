package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"ork/pkg/events"
	manifestcore "ork/pkg/manifest/core"
	"ork/pkg/runners"
	"ork/pkg/state"
	"ork/pkg/utils"
)

// CloudFormationAdapter is the early AWS CloudFormation component adapter.
// Its contract should not be treated as feature-complete yet.
type CloudFormationAdapter struct{}

// CloudFormationConfig configures stack identity and template parameters.
type CloudFormationConfig struct {
	StackName    string            `mapstructure:"stack_name"`
	Region       string            `mapstructure:"region"`
	Parameters   map[string]string `mapstructure:"parameters"`
	Capabilities []string          `mapstructure:"capabilities"`
	Tags         map[string]string `mapstructure:"tags"`
	RoleARN      string            `mapstructure:"role_arn"`

	TemplatePath string
}

// CloudFormationState preserves the stack identity required for teardown.
type CloudFormationState struct {
	Region       string `mapstructure:"region" json:"region,omitempty"`
	StackName    string `mapstructure:"stack_name" json:"stack_name"`
	TemplateFile string `mapstructure:"template_file" json:"template_file"`
	WorkDir      string `mapstructure:"workdir" json:"workdir"`
}

func (d *CloudFormationAdapter) RequiredCapabilities() runners.Capabilities {
	return runners.Capabilities{Exec: true, FileCopy: true}
}

func (d *CloudFormationAdapter) SupportedSources() ComponentSourceSupport {
	return ComponentSourceSupport{
		Embedded: true,
		Files:    true,
	}
}

func (d *CloudFormationAdapter) ValidateAndLoadConfig(ctx context.Context, c *manifestcore.Component) (ComponentConfig, []events.Event, error) {
	var cfg CloudFormationConfig
	var warnings []events.Event
	if err := mapstructure.Decode(c.Config, &cfg); err != nil {
		return nil, warnings, err
	}

	aCtx, ok := AdapterContextFromContext(ctx)
	if !ok {
		return nil, warnings, fmt.Errorf("failed to get adapter context")
	}

	if cfg.StackName == "" {
		cfg.StackName = fmt.Sprintf("ork-%s-%s", aCtx.EnvID(), c.Name)
	}

	templatePath, err := d.localTemplatePath(aCtx, c)
	if err != nil {
		return nil, warnings, err
	}
	cfg.TemplatePath = templatePath

	return &cfg, warnings, nil
}

func (d *CloudFormationAdapter) Apply(ctx context.Context, c *manifestcore.Component, t runners.Runner) (ComponentApplyResult, error) {
	cfg, ok := c.LoadedConfig.(*CloudFormationConfig)
	if !ok {
		return ComponentApplyResult{}, fmt.Errorf("invalid config type for CloudFormationAdapter")
	}

	aCtx, ok := AdapterContextFromContext(ctx)
	if !ok {
		return ComponentApplyResult{}, fmt.Errorf("failed to get adapter context")
	}

	workDir := aCtx.BuildRunnerWorkDir(c.WorkDir, c.Name)
	// Copy with-files to runner
	for name, file := range c.WithFiles {
		copyRes, err := t.CopyFile(ctx, runners.FileCopyRequest{
			Source:      file,
			Destination: path.Join(workDir, name),
			ToRunner:    true,
			Overwrite:   true,
			Recursive:   false,
		})

		if err != nil {
			return ComponentApplyResult{}, fmt.Errorf("failed to copy CloudFormation with-file %q to runner: %w", name, err)
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

	templateFile := path.Base(cfg.TemplatePath)
	if err := d.copyTemplate(ctx, c, t, cfg.TemplatePath, path.Join(workDir, templateFile)); err != nil {
		return ComponentApplyResult{}, err
	}

	cmd := d.deployCommand(cfg, templateFile)
	if err := d.execAWS(ctx, t, workDir, cmd, c.Env, c.Runner, c.Name, "deploy"); err != nil {
		return ComponentApplyResult{}, err
	}

	outputs, err := d.outputs(ctx, t, cfg, workDir, c.Env, c.Runner, c.Name)
	if err != nil {
		return ComponentApplyResult{}, err
	}

	cfState := CloudFormationState{
		Region:       cfg.Region,
		StackName:    cfg.StackName,
		TemplateFile: templateFile,
		WorkDir:      workDir,
	}
	stateData, err := state.NewComponentStateData(workDir, cfState)
	if err != nil {
		return ComponentApplyResult{}, err
	}

	return ComponentApplyResult{
		Outputs: outputs,
		State:   stateData,
	}, nil
}

func (d *CloudFormationAdapter) Destroy(ctx context.Context, componentState state.ComponentState, t runners.Runner) error {
	var s CloudFormationState
	if err := mapstructure.Decode(componentState.Payload, &s); err != nil {
		return fmt.Errorf("failed to decode CloudFormation state: %w", err)
	}
	if s.StackName == "" {
		return fmt.Errorf("CloudFormation state for component %q has no stack_name", componentState.Name)
	}
	if s.WorkDir == "" {
		return fmt.Errorf("CloudFormation state for component %q has no workdir", componentState.Name)
	}

	return d.deleteStack(ctx, s, componentState.Env, componentState.Runner.Name, componentState.Name, t)
}

func (d *CloudFormationAdapter) localTemplatePath(aCtx AdapterContext, c *manifestcore.Component) (string, error) {
	if c.Source.Embedded != "" {
		dir := path.Join(aCtx.GetComponentWorkDirInOrkLocalWorkDir(c.Name), "cloudformation")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create local CloudFormation template directory: %w", err)
		}

		templatePath := path.Join(dir, "template.yml")
		if err := os.WriteFile(templatePath, []byte(c.Source.Embedded), 0644); err != nil {
			return "", fmt.Errorf("failed to write embedded CloudFormation template: %w", err)
		}
		return templatePath, nil
	}

	if len(c.Source.Files) != 1 {
		return "", fmt.Errorf("CloudFormation file source requires exactly one template file, got %d", len(c.Source.Files))
	}

	for _, file := range c.Source.Files {
		return file, nil
	}
	return "", fmt.Errorf("CloudFormation file source requires exactly one template file")
}

func (d *CloudFormationAdapter) copyTemplate(ctx context.Context, c *manifestcore.Component, t runners.Runner, source, destination string) error {
	copyRes, err := t.CopyFile(ctx, runners.FileCopyRequest{
		Source:      source,
		Destination: destination,
		ToRunner:    true,
		Overwrite:   true,
		Recursive:   false,
	})
	if err != nil {
		return fmt.Errorf("failed to copy CloudFormation template to runner: %w", err)
	}
	if copyRes.Error != nil {
		return fmt.Errorf("error during CloudFormation template copy: %w", copyRes.Error)
	}

	if aCtx, ok := AdapterContextFromContext(ctx); ok {
		aCtx.emitter.Emit(events.Event{
			Type:      events.EventInfo,
			Message:   fmt.Sprintf("Copied CloudFormation template to %q", destination),
			Adapter:   c.Type,
			Runner:    c.Runner,
			Component: c.Name,
			Duration:  copyRes.Duration,
		})
	}

	return nil
}

func (d *CloudFormationAdapter) deployCommand(cfg *CloudFormationConfig, templateFile string) []string {
	cmd := []string{
		"aws", "cloudformation", "deploy",
		"--stack-name", cfg.StackName,
		"--template-file", templateFile,
		"--no-fail-on-empty-changeset",
	}

	if cfg.Region != "" {
		cmd = append(cmd, "--region", cfg.Region)
	}
	if cfg.RoleARN != "" {
		cmd = append(cmd, "--role-arn", cfg.RoleARN)
	}
	if len(cfg.Capabilities) > 0 {
		cmd = append(cmd, "--capabilities")
		cmd = append(cmd, cfg.Capabilities...)
	}
	if len(cfg.Parameters) > 0 {
		cmd = append(cmd, "--parameter-overrides")
		for _, key := range sortedKeys(cfg.Parameters) {
			cmd = append(cmd, fmt.Sprintf("%s=%s", key, cfg.Parameters[key]))
		}
	}
	if len(cfg.Tags) > 0 {
		cmd = append(cmd, "--tags")
		for _, key := range sortedKeys(cfg.Tags) {
			cmd = append(cmd, fmt.Sprintf("%s=%s", key, cfg.Tags[key]))
		}
	}

	return cmd
}

func (d *CloudFormationAdapter) deleteStack(ctx context.Context, s CloudFormationState, env map[string]string, runnerName, componentName string, t runners.Runner) error {
	deleteCmd := []string{"aws", "cloudformation", "delete-stack", "--stack-name", s.StackName}
	waitCmd := []string{"aws", "cloudformation", "wait", "stack-delete-complete", "--stack-name", s.StackName}
	if s.Region != "" {
		deleteCmd = append(deleteCmd, "--region", s.Region)
		waitCmd = append(waitCmd, "--region", s.Region)
	}

	if err := d.execAWS(ctx, t, s.WorkDir, deleteCmd, env, runnerName, componentName, "delete-stack"); err != nil {
		return err
	}
	if err := d.execAWS(ctx, t, s.WorkDir, waitCmd, env, runnerName, componentName, "wait stack-delete-complete"); err != nil {
		return err
	}

	return nil
}

func (d *CloudFormationAdapter) execAWS(ctx context.Context, t runners.Runner, workDir string, cmd []string, env map[string]string, runnerName, componentName, action string) error {
	execRes, err := t.Exec(ctx, runners.ExecCommand{
		WorkingDir: workDir,
		Command:    cmd,
		Env:        env,
		Timeout:    0,
		Stderr:     utils.NewPrefixWriter(os.Stderr, utils.RunnerComponentPrefix(runnerName, componentName)),
		Stdout:     utils.NewPrefixWriter(os.Stdout, utils.RunnerComponentPrefix(runnerName, componentName)),
	})
	if err != nil {
		return fmt.Errorf("failed to execute CloudFormation %s: %w", action, err)
	}
	if execRes.Error != nil || execRes.ExitCode != 0 {
		return fmt.Errorf("CloudFormation %s failed with exit code %d: %v", action, execRes.ExitCode, execRes.Error)
	}
	return nil
}

type cloudFormationOutput struct {
	Key   string `json:"OutputKey"`
	Value string `json:"OutputValue"`
}

func (d *CloudFormationAdapter) outputs(ctx context.Context, t runners.Runner, cfg *CloudFormationConfig, workDir string, env map[string]string, runnerName string, componentName string) (ComponentApplyOutput, error) {
	cmd := []string{
		"aws", "cloudformation", "describe-stacks",
		"--stack-name", cfg.StackName,
		"--query", "Stacks[0].Outputs",
		"--output", "json",
	}
	if cfg.Region != "" {
		cmd = append(cmd, "--region", cfg.Region)
	}

	outputRes, err := t.Exec(ctx, runners.ExecCommand{
		WorkingDir: workDir,
		Command:    cmd,
		Env:        env,
		Timeout:    0,
		Stderr:     utils.NewPrefixWriter(os.Stderr, utils.RunnerComponentPrefix(runnerName, componentName)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute CloudFormation describe-stacks: %w", err)
	}
	if outputRes.Error != nil || outputRes.ExitCode != 0 {
		return nil, fmt.Errorf("CloudFormation describe-stacks failed with exit code %d: %v", outputRes.ExitCode, outputRes.Error)
	}

	trimmed := strings.TrimSpace(string(outputRes.Stdout))
	if trimmed == "" || trimmed == "null" {
		return make(ComponentApplyOutput), nil
	}

	outputs, err := parseCloudFormationOutputs(outputRes.Stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CloudFormation outputs JSON: %w", err)
	}
	return outputs, nil
}

func parseCloudFormationOutputs(data []byte) (ComponentApplyOutput, error) {
	var raw []cloudFormationOutput
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	outputs := make(ComponentApplyOutput, len(raw))
	for _, output := range raw {
		if output.Key == "" {
			continue
		}
		outputs[output.Key] = output.Value
	}
	return outputs, nil
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func init() {
	Register("cloudformation", &CloudFormationAdapter{})
}
