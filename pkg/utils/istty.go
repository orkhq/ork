package utils

import (
	"os"

	"golang.org/x/term"
)

// IsTTY reports whether stdout is connected to a terminal, used to select
// between human-readable and machine-readable output formats.
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
