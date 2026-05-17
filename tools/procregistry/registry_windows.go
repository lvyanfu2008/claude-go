//go:build windows

package procregistry

import (
	"fmt"
	"os/exec"
)

// SetProcessGroup is a no-op on Windows (process tree killing is handled
// natively by TerminateProcess or taskkill /T).
func SetProcessGroup(cmd *exec.Cmd) {}

func killProcessGroup(cmd *exec.Cmd) bool {
	if cmd.Process == nil {
		return false
	}
	// On Windows, cmd.Process.Kill() calls TerminateProcess which does not
	// kill child processes. Use taskkill /T for full process tree termination.
	pid := cmd.Process.Pid
	killCmd := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", pid))
	_ = killCmd.Run()
	// Fallback: direct kill if taskkill fails (e.g. not in PATH).
	_ = cmd.Process.Kill()
	return true
}
