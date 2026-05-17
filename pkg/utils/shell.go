package utils

func ShellCommand(shell []string, command string) []string {
	if len(shell) == 0 {
		shell = []string{"sh", "-c"}
	}
	cmd := append([]string{}, shell...)
	return append(cmd, command)
}
