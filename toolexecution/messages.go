package toolexecution

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"

	"goc/types"
)

func randomUUID(deps ExecutionDeps) string {
	if deps.RandomUUID != nil {
		return deps.RandomUUID()
	}
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func mustJSONString(s string) []byte {
	b, err := json.Marshal(s)
	if err != nil {
		return []byte(`""`)
	}
	return b
}

// CreateUserMessage mirrors createUserMessage from src/utils/messages.ts for synthetic tool_result rows.
func CreateUserMessage(deps ExecutionDeps, content []map[string]any, toolUseResult, sourceAssistantUUID string) types.Message {
	inner, err := json.Marshal(map[string]any{"role": "user", "content": content})
	if err != nil {
		inner = []byte(`{"role":"user","content":[]}`)
	}
	src := sourceAssistantUUID
	return types.Message{
		Type:                    types.MessageTypeUser,
		UUID:                    randomUUID(deps),
		Message:                 inner,
		ToolUseResult:           json.RawMessage(mustJSONString(toolUseResult)),
		SourceToolAssistantUUID: &src,
	}
}

func syntheticUnknownTool(deps ExecutionDeps, toolName, toolUseID, assistantUUID string) types.Message {
	return CreateUserMessage(deps, []map[string]any{{
		"type":        "tool_result",
		"content":     `<tool_use_error>Error: No such tool available: ` + toolName + `</tool_use_error>`,
		"is_error":    true,
		"tool_use_id": toolUseID,
	}}, "Error: No such tool available: "+toolName, assistantUUID)
}

// syntheticAborted mirrors toolExecution.ts runToolUse L418–454 (cancel + memory hint on tool_result).
func syntheticAborted(deps ExecutionDeps, toolUseID, assistantUUID string) types.Message {
	content := withMemoryCorrectionHint(CANCEL_MESSAGE)
	return CreateUserMessage(deps, []map[string]any{{
		"type":        "tool_result",
		"content":     content,
		"is_error":    true,
		"tool_use_id": toolUseID,
	}}, CANCEL_MESSAGE, assistantUUID)
}

func syntheticPipelineTODO(deps ExecutionDeps, toolUseID, assistantUUID string) types.Message {
	const msg = "toolexecution: CheckPermissionsAndCallTool pipeline skeleton (see toolexecution/check_permissions_and_call.go)"
	return CreateUserMessage(deps, []map[string]any{{
		"type":        "tool_result",
		"content":     `<tool_use_error>` + msg + `</tool_use_error>`,
		"is_error":    true,
		"tool_use_id": toolUseID,
	}}, msg, assistantUUID)
}

func syntheticPreToolHookDenied(deps ExecutionDeps, toolUseID, assistantUUID, reason string) types.Message {
	msg := strings.TrimSpace(reason)
	if msg == "" {
		msg = "blocked by pre-tool hook"
	}
	return CreateUserMessage(deps, []map[string]any{{
		"type":        "tool_result",
		"content":     `<tool_use_error>` + msg + `</tool_use_error>`,
		"is_error":    true,
		"tool_use_id": toolUseID,
	}}, msg, assistantUUID)
}
