package targets

import (
	"context"
	"io"
	"time"
)

type TargetType string

const (
	TargetTypeSSH   TargetType = "ssh"
	TargetTypeAWS   TargetType = "aws"
	TargetTypeLocal TargetType = "local"
)

type Capabilities struct {
	Exec     bool // Ability to execute shell-like commands
	FileCopy bool // Ability to copy files to/from the target
	Cloud    bool // Ability to interact with cloud services. Usually accompanied by a specific cloud config in the target.
}

func (c Capabilities) SatisfiedBy(r Capabilities) bool {
	if c.Exec && !r.Exec {
		return false
	}

	if c.FileCopy && !r.FileCopy {
		return false
	}

	return true
}

type ExecCommand struct {
	Command    []string          // command and args
	WorkingDir string            // optional working directory
	Env        map[string]string // optional environment variables
	Stdin      io.Reader         // optional stdin stream
	Stdout     io.Writer         // optional stdout stream
	Stderr     io.Writer         // optional stderr stream
	Timeout    time.Duration     // optional timeout
}

type ExecResult struct {
	ExitCode int           // exit code of the process
	Error    error         // transport or execution error
	Stdout   []byte        // optional captured stdout
	Stderr   []byte        // optional captured stderr
	Duration time.Duration // execution time
}

type FileCopyRequest struct {
	Source      string // local or host-side path
	Destination string // target-side path
	ToTarget    bool   // true: copy to target, false: copy from target
	Recursive   bool   // copy directories recursively
	Overwrite   bool   // overwrite existing files
}

type FileCopyResult struct {
	CopiedFiles int           // number of files copied
	Bytes       int64         // total bytes copied
	Duration    time.Duration // execution time
	Error       error         // transport error, not semantic failure
}

type Target interface {
	Name() string
	Type() TargetType
	ValidateAndInitialize() error
	Capabilities() Capabilities
	Exec(ctx context.Context, command ExecCommand) (*ExecResult, error)
	CopyFile(ctx context.Context, req FileCopyRequest) (*FileCopyResult, error)
	UsesNonAmbientCredentials() (bool, []string)
	Disconnect() error
}
