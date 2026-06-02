package version

import "fmt"

var (
	Version   = "v0.1.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func String() string {
	return fmt.Sprintf("Ork %s (commit: %s, built: %s)", Version, Commit, BuildDate)
}
