package utils

import (
	"fmt"
	"regexp"
)

var envIDRegex = regexp.MustCompile(`^[a-z][a-z0-9-]{1,61}[a-z0-9]$`)

// ValidateEnvID checks that the environment ID matches the required format:
// lowercase alphanumeric with hyphens, 3-63 characters (e.g. "pr-123",
// "dev-alice").
func ValidateEnvID(id string) error {
	if id == "" {
		return fmt.Errorf("environment ID is required")
	}

	if !envIDRegex.MatchString(id) {
		return fmt.Errorf(
			"invalid environment ID %q: must match %s (e.g. pr-123, dev-alice)",
			id,
			envIDRegex.String(),
		)
	}

	return nil
}
