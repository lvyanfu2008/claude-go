package claudeinit

import (
	"runtime"
)

// phase4FSAndPlatform: TS setShellIfWindows, registerCleanup stubs, scratchpad.
func phase4FSAndPlatform() error {
	_ = runtime.GOOS
	// P4a setShellIfWindows: explicit_gap on Windows.
	// P4c scratchpad: explicit_gap (Statsig-gated in TS).
	return nil
}
