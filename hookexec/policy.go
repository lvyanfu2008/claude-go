package hookexec

import "os"

// ShouldDisableAllHooksIncludingManaged mirrors hooksConfigSnapshot.shouldDisableAllHooksIncludingManaged:
// policySettings.disableAllHooks === true. Go does not yet merge policy JSON here; gate via env until managed policy port lands.
func ShouldDisableAllHooksIncludingManaged() bool {
	return envTruthy(os.Getenv("CLAUDE_CODE_POLICY_DISABLE_ALL_HOOKS"))
}

// ShouldSkipHookDueToTrust mirrors hooks.ts shouldSkipHookDueToTrust for interactive sessions without accepted workspace trust.
// Go headless hosts default to implicit trust (never skip). Set CLAUDE_CODE_INTERACTIVE=1 and omit CLAUDE_CODE_WORKSPACE_TRUST_ACCEPTED to skip hooks.
func ShouldSkipHookDueToTrust() bool {
	if !envTruthy(os.Getenv("CLAUDE_CODE_INTERACTIVE")) {
		return false
	}
	return !envTruthy(os.Getenv("CLAUDE_CODE_WORKSPACE_TRUST_ACCEPTED"))
}
