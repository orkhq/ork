package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

type CmdResult struct {
	Stdout string
	Stderr string
	Code   int
}

// RunCommand runs a shell command with arguments and prints output to console.
func RunCommand(name string, args ...string) (*CmdResult, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("→ Running: %s %v\n", name, args)
	err := cmd.Run()

	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	}

	if err != nil {
		return &CmdResult{Code: code}, fmt.Errorf("%s failed: %w", name, err)
	}

	return &CmdResult{Code: code}, nil
}

// RunCommandCapture runs a command and returns captured stdout/stderr (no live print)
func RunCommandCapture(name string, args ...string) (*CmdResult, error) {
	var outBuf, errBuf bytes.Buffer

	cmd := exec.Command(name, args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	}

	return &CmdResult{
		Stdout: outBuf.String(),
		Stderr: errBuf.String(),
		Code:   code,
	}, err
}
