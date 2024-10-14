package utils

import (
	"os/exec"
)

// RunCommand runs an executable (name)
func RunCommand(args []string) (string, error) {

	executable := args[0]
	args = args[1:]

	cmd := exec.Command(executable, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
