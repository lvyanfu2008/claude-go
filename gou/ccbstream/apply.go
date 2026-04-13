package ccbstream

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goc/gou/conversation"
	"goc/gou/messagerow"
	"goc/types"
)

// Apply updates store from one stream event (assistant_delta, tool_use, tool_result, turn_complete, error).
func Apply(store *conversation.Store, ev StreamEvent) {
	switch ev.Type {
	case "assistant_delta":
		store.AppendStreamingChunk(ev.Text)
	case "tool_use":
		flushStreamingAssistant(store)
		raw, err := json.Marshal([]map[string]any{{
			"type":  "tool_use",
			"id":    ev.ID,
			"name":  ev.Name,
			"input": ev.Input,
		}})
		if err != nil {
			return
		}
		store.AppendMessage(types.Message{
			Type:    types.MessageTypeAssistant,
			UUID:    fmt.Sprintf("tu-%s", ev.ID),
			Content: raw,
		})
	case "tool_result":
		raw, err := json.Marshal([]map[string]any{{
			"type":        "tool_result",
			"tool_use_id": ev.ToolUseID,
			"content":     ev.Content,
			"is_error":    ev.IsError,
		}})
		if err != nil {
			return
		}
		store.AppendMessage(types.Message{
			Type:    types.MessageTypeUser,
			UUID:    fmt.Sprintf("tr-%d", time.Now().UnixNano()),
			Content: raw,
		})
		messagerow.CollapseReadSearchTail(&store.Messages)
	case "turn_complete":
		flushStreamingAssistant(store)
		// Always clear buffer after a turn (flush may no-op on empty trim but buffer had whitespace-only).
		store.ClearStreaming()
	case "response_end":
		// Safety net if turn_complete was not received; avoids a stuck streaming buffer / empty UI.
		flushStreamingAssistant(store)
		store.ClearStreaming()
	case "error":
		ts := time.Now().UTC().Format(time.RFC3339Nano)
		txt := ev.Message
		if txt == "" {
			txt = ev.Code
		}
		raw, _ := json.Marshal([]map[string]string{{"type": "text", "text": txt}})
		store.AppendMessage(types.Message{
			Type:      types.MessageTypeSystem,
			UUID:      fmt.Sprintf("err-%d", time.Now().UnixNano()),
			Subtype:   strPtr("error"),
			Content:   raw,
			Timestamp: &ts,
		})
	case "usage":
		store.AddUsage(ev.InputTokens, ev.OutputTokens)
	case "execute_tool":
		// Client must run the tool and write tool_result; we only surface a placeholder for replay visibility.
		flushStreamingAssistant(store)
		name := strings.TrimSpace(ev.Name)
		tuid := strings.TrimSpace(ev.ToolUseID)
		cid := strings.TrimSpace(ev.CallID)
		txt := fmt.Sprintf("[ccbstream] execute_tool pending (no client execution): name=%q tool_use_id=%q call_id=%q", name, tuid, cid)
		rawJSON, err := json.Marshal([]map[string]string{{"type": "text", "text": txt}})
		if err != nil {
			return
		}
		store.AppendMessage(types.Message{
			Type:    types.MessageTypeSystem,
			UUID:    fmt.Sprintf("et-%d", time.Now().UnixNano()),
			Content: rawJSON,
		})
	default:
	}
}

func flushStreamingAssistant(store *conversation.Store) {
	raw := store.StreamingText
	if raw == "" {
		return
	}
	t := strings.TrimSpace(raw)
	// Persist trimmed text; if the model only sent whitespace, keep raw so we do not drop the turn.
	textOut := t
	if textOut == "" {
		textOut = raw
	}
	rawJSON, err := json.Marshal([]map[string]string{{"type": "text", "text": textOut}})
	if err != nil {
		return
	}
	store.AppendMessage(types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    fmt.Sprintf("a-%d", time.Now().UnixNano()),
		Content: rawJSON,
	})
	store.ClearStreaming()
}

func strPtr(s string) *string { return &s }
