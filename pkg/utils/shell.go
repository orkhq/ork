package utils

// ShellCommand builds a command slice by appending the given command string to
// the provided shell invocation. If shell is empty, it defaults to ["sh", "-c"].
func ShellCommand(shell []string, command string) []string {
	if len(shell) == 0 {
		shell = []string{"sh", "-c"}
	}
	cmd := append([]string{}, shell...)
	return append(cmd, command)
}
