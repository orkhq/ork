// Package runners defines the Runner interface and supporting types for
// executing commands and copying files on local or remote execution contexts.
// Runners are the execution layer that adapters use to provision components.
package runners

import (
	"context"
	"io"
	"strings"
	"time"
)

// RunnerType identifies the kind of runner (e.g. local process or SSH remote).
type RunnerType string

const (
	// RunnerTypeSSH identifies a runner that executes over SSH.
	RunnerTypeSSH RunnerType = "ssh"
	// RunnerTypeLocal identifies a runner that executes on the local machine.
	RunnerTypeLocal RunnerType = "local"
)

// Capabilities declares what operations a runner supports. Adapters check
// these to ensure the assigned runner can satisfy their requirements.
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

// ExecCommand describes a command to execute on a runner, including optional
// I/O streams, environment variables, and timeout.
type ExecCommand struct {
	Command    []string          // command and args
	WorkingDir string            // optional working directory
	Env        map[string]string // optional environment variables
	Stdin      io.Reader         // optional stdin stream
	Stdout     io.Writer         // optional stdout stream
	Stderr     io.Writer         // optional stderr stream
	Timeout    time.Duration     // optional timeout
}

// ExecResult holds the outcome of a command execution on a runner.
type ExecResult struct {
	ExitCode int           // exit code of the process
	Error    error         // transport or execution error
	Stdout   []byte        // optional captured stdout
	Stderr   []byte        // optional captured stderr
	Duration time.Duration // execution time
}

// FileCopyRequest specifies a file transfer operation between the host and the
// runner (or vice versa).
type FileCopyRequest struct {
	Source      string // local or host-side path
	Destination string // runner-side path
	ToRunner    bool   // true: copy to runner, false: copy from runner
	Recursive   bool   // copy directories recursively
	Overwrite   bool   // overwrite existing files
}

// FileCopyResult reports the outcome of a file copy operation.
type FileCopyResult struct {
	CopiedFiles int           // number of files copied
	Bytes       int64         // total bytes copied
	Duration    time.Duration // execution time
	Error       error         // transport error, not semantic failure
}

// Runner is the interface that all execution backends must implement. It
// provides command execution, file transfer, capability reporting, and
// lifecycle management for a specific execution context.
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
