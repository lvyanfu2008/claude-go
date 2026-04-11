package query

import (
	"encoding/json"
	"iter"
	"strings"

	"goc/types"
)

// MaxOutputTokensRecoveryLimit mirrors query.ts MAX_OUTPUT_TOKENS_RECOVERY_LIMIT.
const MaxOutputTokensRecoveryLimit = 3

// IsWithheldMaxOutputTokens mirrors query.ts isWithheldMaxOutputTokens.
func IsWithheldMaxOutputTokens(msg *types.Message) bool {
	if msg == nil || msg.Type != types.MessageTypeAssistant {
		return false
	}
	if len(msg.Message) == 0 {
		return false
	}
	var inner struct {
		APIError string `json:"apiError"`
	}
	if err := json.Unmarshal(msg.Message, &inner); err != nil {
		return false
	}
	return inner.APIError == "max_output_tokens"
}

// MissingToolResultUserMessages mirrors yieldMissingToolResultBlocks in query.ts
// (sync iterator over synthetic user tool_result messages).
func MissingToolResultUserMessages(assistants []types.Message, errorMessage string) iter.Seq[types.Message] {
	return func(yield func(types.Message) bool) {
		for _, am := range assistants {
			if am.Type != types.MessageTypeAssistant {
				continue
			}
			ids := toolUseIDsFromAssistant(am)
			for _, id := range ids {
				if strings.TrimSpace(id) == "" {
					continue
				}
				um, err := toolResultUserMessage(id, errorMessage, am.UUID)
				if err != nil {
					return
				}
				if !yield(um) {
					return
				}
			}
		}
	}
}

func toolUseIDsFromAssistant(am types.Message) []string {
	var env struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(am.Message, &env); err != nil {
		return nil
	}
	if len(env.Content) == 0 {
		return nil
	}
	// content may be string (no tool uses) or array of blocks
	if env.Content[0] == '"' {
		return nil
	}
	var blocks []struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal(env.Content, &blocks); err != nil {
		return nil
	}
	var ids []string
	for _, b := range blocks {
		if b.Type == "tool_use" && b.ID != "" {
			ids = append(ids, b.ID)
		}
	}
	return ids
}

func toolResultUserMessage(toolUseID, errText, assistantUUID string) (types.Message, error) {
	content := []any{
		map[string]any{
			"type":        "tool_result",
			"content":     errText,
			"is_error":    true,
			"tool_use_id": toolUseID,
		},
	}
	inner, err := json.Marshal(map[string]any{"role": "user", "content": content})
	if err != nil {
		return types.Message{}, err
	}
	src := assistantUUID
	return types.Message{
		Type:                    types.MessageTypeUser,
		UUID:                    randomUUID(),
		Message:                 inner,
		ToolUseResult:           json.RawMessage(mustStringJSON(errText)),
		SourceToolAssistantUUID: &src,
	}, nil
}

func mustStringJSON(s string) []byte {
	b, err := json.Marshal(s)
	if err != nil {
		return []byte(`""`)
	}
	return b
}
