package terminal

import (
	"os/exec"
)

// buildCmd constructs an *exec.Cmd ready for PTY spawning.
// The working directory is set if cwd is non-empty.
func buildCmd(file string, args []string, cwd string) *exec.Cmd {
	cmd := exec.Command(file, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	return cmd
}
