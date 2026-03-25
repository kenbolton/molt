package driver

import (
	"os/exec"
	"strings"
)

// execDriver returns a configured exec.Cmd for a driver with the given input.
func execDriver(path, input string) *exec.Cmd {
	cmd := exec.Command(path)
	cmd.Stdin = strings.NewReader(input + "\n")
	return cmd
}
