//go:build !windows

package procregistry

import (
	"os/exec"
	"syscall"
	"time"
)

// SetProcessGroup sets Setpgid on the command so we can kill the entire
// process group with a single syscall.Kill(-pgid, ...).
func SetProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func killProcessGroup(cmd *exec.Cmd) bool {
	if cmd.Process == nil {
		return false
	}
	pid := cmd.Process.Pid
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		_ = cmd.Process.Kill()
		return true
	}
	// SIGTERM first, then SIGKILL after a brief grace period.
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	time.Sleep(100 * time.Millisecond)
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	return true
}
