package toolexecution

import (
	"encoding/json"
	"strings"
)

// BashToolName is the primary Bash tool id (src/tools/BashTool/toolName.ts BASH_TOOL_NAME).
const BashToolName = "Bash"

// BashSandboxRule1b carries SandboxManager-style flags for permissions.ts 1b (whole-tool alwaysAsk on Bash
// is skipped when the command runs under sandbox heuristics — see [WholeToolAskSkippedForBash1b]).
type BashSandboxRule1b struct {
	SandboxingEnabled                  bool
	AutoAllowWholeToolAskWhenSandboxed bool
}

// bashSandboxRule1bFromExecutionDeps returns non-nil only when both [ExecutionDeps] flags are true.
func bashSandboxRule1bFromExecutionDeps(d ExecutionDeps) *BashSandboxRule1b {
	if !d.SandboxingEnabled || !d.AutoAllowBashWholeToolAskWhenSandboxed {
		return nil
	}
	return &BashSandboxRule1b{
		SandboxingEnabled:                  true,
		AutoAllowWholeToolAskWhenSandboxed: true,
	}
}

// BashInputUsesSandboxForRule1b is a subset of TS shouldUseSandbox (src/tools/BashTool/shouldUseSandbox.ts):
// sandboxing is assumed enabled by the caller; we require a non-empty command and no dangerously_disable_sandbox.
// Excluded-command patterns and growthbook ants-only paths are not ported (TODO parity).
func BashInputUsesSandboxForRule1b(input json.RawMessage) bool {
	var v struct {
		Command                   string `json:"command"`
		DangerouslyDisableSandbox bool   `json:"dangerously_disable_sandbox"`
	}
	if len(input) == 0 || json.Unmarshal(input, &v) != nil {
		return false
	}
	if strings.TrimSpace(v.Command) == "" {
		return false
	}
	if v.DangerouslyDisableSandbox {
		return false
	}
	return true
}

// WholeToolAskSkippedForBash1b mirrors permissions.ts checkRuleBasedPermissions canSandboxAutoAllow (L1094–1109).
func WholeToolAskSkippedForBash1b(toolName string, input json.RawMessage, b *BashSandboxRule1b) bool {
	if b == nil || !b.SandboxingEnabled || !b.AutoAllowWholeToolAskWhenSandboxed {
		return false
	}
	if toolName != BashToolName {
		return false
	}
	return BashInputUsesSandboxForRule1b(input)
}
