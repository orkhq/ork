package runners

import (
	"context"
	"io"
	"strings"
	"time"
)

// RunnerType identifies an execution transport recorded in component state.
type RunnerType string

const (
	RunnerTypeSSH   RunnerType = "ssh"
	RunnerTypeLocal RunnerType = "local"
)

// Capabilities describes operations a runner can perform. Adapters declare
// their requirements before lifecycle execution begins.
type Capabilities struct {
	Exec     bool // Ability to execute shell-like commands
	FileCopy bool // Ability to copy files to/from the runner
}

func (c Capabilities) String() string {
	var caps []string
	if c.Exec {
		caps = append(caps, "Exec")
	}
	if c.FileCopy {
		caps = append(caps, "FileCopy")
	}

	return "[" + strings.Join(caps, ", ") + "]"
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

// ExecCommand describes a process invocation inside a runner context.
// Command and Env may contain sensitive values and must not be logged by
// runner implementations.
type ExecCommand struct {
	Command    []string          // command and args
	WorkingDir string            // optional working directory
	Env        map[string]string // optional environment variables
	Stdin      io.Reader         // optional stdin stream
	Stdout     io.Writer         // optional stdout stream
	Stderr     io.Writer         // optional stderr stream
	Timeout    time.Duration     // optional timeout
}

// ExecResult captures process outcome separately from transport-level errors.
type ExecResult struct {
	ExitCode int           // exit code of the process
	Error    error         // transport or execution error
	Stdout   []byte        // optional captured stdout
	Stderr   []byte        // optional captured stderr
	Duration time.Duration // execution time
}

// FileCopyRequest describes a transfer between the Ork machine and a runner.
type FileCopyRequest struct {
	Source      string // local or host-side path
	Destination string // runner-side path
	ToRunner    bool   // true: copy to runner, false: copy from runner
	Recursive   bool   // copy directories recursively
	Overwrite   bool   // overwrite existing files
}

// FileCopyResult reports transfer metrics and any runner-reported error.
type FileCopyResult struct {
	CopiedFiles int           // number of files copied
	Bytes       int64         // total bytes copied
	Duration    time.Duration // execution time
	Error       error         // transport error, not semantic failure
}

// Runner is the execution and file-transfer boundary for a component. Provider
// environment and network location are scoped to the runner implementation.
type Runner interface {
	Name() string
	Type() RunnerType
	ValidateAndInitialize() error
	Capabilities() Capabilities
	Exec(ctx context.Context, command ExecCommand) (*ExecResult, error)
	CopyFile(ctx context.Context, req FileCopyRequest) (*FileCopyResult, error)
	UsesNonAmbientCredentials() (bool, []string)
	Disconnect() error
}
