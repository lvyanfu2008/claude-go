package claudeinit

import (
	"fmt"

	"goc/ccb-engine/settingsfile"
)

// phase1ConfigAndEnv: TS enableConfigs + applySafeConfigEnvironmentVariables + applyExtraCACertsFromConfig (subset).
func phase1ConfigAndEnv(opts Options) error {
	// P1c before P1a merge: user-controlled CA path only (TS applyExtraCACertsFromConfig).
	if err := applyExtraCACertsUserControlled(); err != nil {
		if opts.NonInteractive {
			return fmt.Errorf("claudeinit: extra ca certs: %w (non-interactive)", err)
		}
		return fmt.Errorf("claudeinit: extra ca certs: %w", err)
	}
	// P1a: merged project/user settings → env (closest Go equivalent to TS config + safe env).
	if err := settingsfile.EnsureProjectClaudeEnvOnce(); err != nil {
		if opts.NonInteractive {
			return fmt.Errorf("claudeinit: settings/env: %w (non-interactive)", err)
		}
		return fmt.Errorf("claudeinit: settings/env: %w", err)
	}
	// P1b applySafeConfigEnvironmentVariables: explicit_gap — additional TS-only keys.
	return nil
}
