// Package version exposes build-time version information for the ork binary.
// Values are injected via -ldflags during compilation.
package version

import "fmt"

var (
	// Version is the semantic version of the ork binary.
	Version = "v0.1.0-dev"
	// Commit is the git commit SHA the binary was built from.
	Commit = "unknown"
	// BuildDate is the UTC timestamp when the binary was built.
	BuildDate = "unknown"
)

// String returns a human-readable version string including the version, commit,
// and build date.
func String() string {
	return fmt.Sprintf("ork %s (commit: %s, built: %s)", Version, Commit, BuildDate)
}
