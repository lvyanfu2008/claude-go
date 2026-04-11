package streamingtool

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	"goc/types"
)

// Mirrors src/utils/messagesLiteralConstants.ts REJECT_MESSAGE + MEMORY_CORRECTION_HINT (withMemoryCorrectionHint path).
const (
	rejectMessage = "The user doesn't want to proceed with this tool use. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). STOP what you are doing and wait for the user to tell you how to proceed."

	memoryCorrectionHint = "\n\nNote: The user's next message may contain a correction or preference. Pay close attention — if they explain what went wrong or how they'd prefer you to work, consider saving that to memory for future sessions."
)

func withMemoryCorrectionHint(s string) string {
	return s + memoryCorrectionHint
}

func randomUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// createUserMessage mirrors createUserMessage from src/utils/messages.ts for synthetic tool_result rows.
func createUserMessage(content []map[string]any, toolUseResult string, sourceAssistantUUID string) types.Message {
	inner, err := json.Marshal(map[string]any{"role": "user", "content": content})
	if err != nil {
		inner = []byte(`{"role":"user","content":[]}`)
	}
	src := sourceAssistantUUID
	return types.Message{
		Type:                    types.MessageTypeUser,
		UUID:                    randomUUID(),
		Message:                 inner,
		ToolUseResult:           json.RawMessage(mustJSONString(toolUseResult)),
		SourceToolAssistantUUID: &src,
	}
}

func mustJSONString(s string) []byte {
	b, err := json.Marshal(s)
	if err != nil {
		return []byte(`""`)
	}
	return b
}
