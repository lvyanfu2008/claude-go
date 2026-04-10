package claudeinit

import (
	"sync"
)

var (
	cleanupMu sync.Mutex
	cleanups  []func()
)

// RegisterCleanup appends a callback (TS registerCleanup). No global signal handler is
// installed here — Bubble Tea and other binaries own SIGINT; call RunCleanups manually if needed.
func RegisterCleanup(fn func()) {
	if fn == nil {
		return
	}
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	cleanups = append(cleanups, fn)
}

// RunCleanups executes registered callbacks in reverse order (for tests or explicit shutdown).
func RunCleanups() {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}
	cleanups = nil
}

func registerGracefulShutdown() {
	// P1d explicit_gap: TS setupGracefulShutdown registers process handlers; Go avoids competing with TUI.
}
