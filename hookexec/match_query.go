package hookexec

import (
	"path/filepath"
	"strings"
)

// DeriveMatchQuery mirrors src/utils/hooks.ts getMatchingHooks switch (hookInput.hook_event_name).
// When useFilter is false, TS uses the full matcher list without matchQuery filtering (matchQuery stays undefined).
func DeriveMatchQuery(hookInput map[string]any) (matchQuery string, useFilter bool) {
	ev, _ := hookInput["hook_event_name"].(string)
	switch ev {
	case "PreToolUse", "PostToolUse", "PostToolUseFailure", "PermissionRequest", "PermissionDenied":
		s, _ := hookInput["tool_name"].(string)
		return s, true
	case "SessionStart":
		s, _ := hookInput["source"].(string)
		return s, true
	case "Setup":
		s, _ := hookInput["trigger"].(string)
		return s, true
	case "PreCompact", "PostCompact":
		s, _ := hookInput["trigger"].(string)
		return s, true
	case "Notification":
		s, _ := hookInput["notification_type"].(string)
		return s, true
	case "SessionEnd":
		s, _ := hookInput["reason"].(string)
		return s, true
	case "StopFailure":
		s, _ := hookInput["error"].(string)
		return s, true
	case "SubagentStart", "SubagentStop":
		s, _ := hookInput["agent_type"].(string)
		return s, true
	case "TeammateIdle", "TaskCreated", "TaskCompleted":
		return "", false
	case "Elicitation", "ElicitationResult":
		s, _ := hookInput["mcp_server_name"].(string)
		return s, true
	case "ConfigChange":
		s, _ := hookInput["source"].(string)
		return s, true
	case "InstructionsLoaded":
		s, _ := hookInput["load_reason"].(string)
		return s, true
	case "FileChanged":
		fp, _ := hookInput["file_path"].(string)
		return filepath.Base(strings.TrimSpace(fp)), true
	default:
		return "", false
	}
}
