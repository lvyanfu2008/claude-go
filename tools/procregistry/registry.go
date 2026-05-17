package procregistry

import (
	"os/exec"
	"sync"
)

var (
	processMu   sync.Mutex
	processCmds = map[string]*exec.Cmd{}
)

// StoreProcess records a running process under a task ID for later kill.
// Must be called before cmd.Run() / cmd.Start().
func StoreProcess(taskID string, cmd *exec.Cmd) {
	processMu.Lock()
	defer processMu.Unlock()
	processCmds[taskID] = cmd
}

// KillProcess terminates the process identified by taskID.
// Uses process group kill on Unix, taskkill /T on Windows.
// Returns true if a process was found and signaled.
func KillProcess(taskID string) bool {
	processMu.Lock()
	cmd, ok := processCmds[taskID]
	if ok {
		delete(processCmds, taskID)
	}
	processMu.Unlock()
	if !ok || cmd == nil || cmd.Process == nil {
		return false
	}
	return killProcessGroup(cmd)
}

// RemoveProcess deletes a task ID from the registry.
// Called in defer after cmd.Run completes.
func RemoveProcess(taskID string) {
	processMu.Lock()
	defer processMu.Unlock()
	delete(processCmds, taskID)
}

// HasProcess reports whether a task ID is currently registered.
func HasProcess(taskID string) bool {
	processMu.Lock()
	defer processMu.Unlock()
	_, ok := processCmds[taskID]
	return ok
}
