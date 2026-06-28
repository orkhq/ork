package version

import "fmt"

var (
	Version   = "v0.1.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// String returns a human-readable build version.
func String() string {
	return fmt.Sprintf("ork %s (commit: %s, built: %s)", Version, Commit, BuildDate)
}
