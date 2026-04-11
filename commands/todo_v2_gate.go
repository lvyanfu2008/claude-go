package commands

// TodoV2Enabled mirrors isTodoV2Enabled in claude-code/src/utils/tasks.ts:
// CLAUDE_CODE_ENABLE_TASKS forces task tools on; otherwise they follow "interactive"
// (off when the host signals non-interactive via env).
//
// Go uses CLAUDE_CODE_NON_INTERACTIVE or GOU_DEMO_NON_INTERACTIVE as non-interactive
// signals (TS uses in-app session state). Set CLAUDE_CODE_ENABLE_TASKS=1 for Task*
// tools in non-interactive runs (e.g. SDK / CI).
func TodoV2Enabled() bool {
	if IsEnvTruthy("CLAUDE_CODE_ENABLE_TASKS") {
		return true
	}
	if IsEnvTruthy("CLAUDE_CODE_NON_INTERACTIVE") || IsEnvTruthy("GOU_DEMO_NON_INTERACTIVE") {
		return false
	}
	return true
}
