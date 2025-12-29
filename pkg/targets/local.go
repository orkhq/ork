package targets

import "context"

type LocalTarget struct {
	name   string
	config SSHTargetConfig
}

func (t *LocalTarget) Name() string {
	return t.name
}

func (t *LocalTarget) Type() TargetType {
	return TargetTypeLocal
}

func (t *LocalTarget) Capabilities() Capabilities {
	return Capabilities{Exec: true, FileCopy: true}
}

func (t *LocalTarget) ValidateAndInitialize() error {
	return nil
}

func (t *LocalTarget) Exec(ctx context.Context, command ExecCommand) (*ExecResult, error) {
	//TODO implement me
	panic("implement me")
}

func (t *LocalTarget) CopyFile(ctx context.Context, req FileCopyRequest) (*FileCopyResult, error) {
	//TODO implement me
	panic("implement me")
}

func (t *LocalTarget) UsesNonAmbientCredentials() (bool, []string) {
	return false, nil
}

func (t *LocalTarget) Disconnect() error {
	return nil
}
