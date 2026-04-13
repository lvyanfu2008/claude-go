package messagesview

import (
	"encoding/json"
	"strings"

	"goc/gou/messagerow"
	"goc/types"
)

// toolUseGroup mirrors claude-code reorderMessagesInUI (messages.ts).
type toolUseGroup struct {
	toolUse    *types.Message
	preHooks   []types.Message
	toolResult *types.Message
	postHooks  []types.Message
}

func reorderContentBlocks(msg types.Message) []types.MessageContentBlock {
	msg = messagerow.NormalizeMessageJSON(msg)
	if len(msg.Content) == 0 {
		return nil
	}
	var blocks []types.MessageContentBlock
	_ = json.Unmarshal(msg.Content, &blocks)
	return blocks
}

func isToolUseRequestMessage(msg types.Message) bool {
	if msg.Type != types.MessageTypeAssistant {
		return false
	}
	for _, b := range reorderContentBlocks(msg) {
		if b.Type == "tool_use" {
			return true
		}
	}
	return false
}

func firstToolUseID(msg types.Message) string {
	for _, b := range reorderContentBlocks(msg) {
		if b.Type == "tool_use" && strings.TrimSpace(b.ID) != "" {
			return strings.TrimSpace(b.ID)
		}
	}
	return ""
}

func isToolResultUserMessage(msg types.Message) bool {
	if msg.Type != types.MessageTypeUser {
		return false
	}
	bl := reorderContentBlocks(msg)
	if len(bl) == 0 {
		return false
	}
	return bl[0].Type == "tool_result"
}

func firstToolResultUseID(msg types.Message) string {
	bl := reorderContentBlocks(msg)
	if len(bl) == 0 || bl[0].Type != "tool_result" {
		return ""
	}
	return strings.TrimSpace(bl[0].ToolUseID)
}

func toolUseIDFromHookAttachment(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	for _, k := range []string{"toolUseId", "tool_use_id", "toolUseID"} {
		v, ok := m[k]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func hookAttachmentMeta(msg types.Message) (toolUseID string, event string, ok bool) {
	if msg.Type != types.MessageTypeAttachment || len(msg.Attachment) == 0 {
		return "", "", false
	}
	var head struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(msg.Attachment, &head)
	tid := toolUseIDFromHookAttachment(msg.Attachment)
	if tid == "" {
		return "", "", false
	}
	var evWrap struct {
		HookEvent string `json:"hookEvent"`
	}
	_ = json.Unmarshal(msg.Attachment, &evWrap)
	ev := strings.TrimSpace(evWrap.HookEvent)
	if ev != "PreToolUse" && ev != "PostToolUse" {
		return "", "", false
	}
	return tid, ev, true
}

// ReorderMessagesInUI ports claude-code/src/utils/messages.ts reorderMessagesInUI.
// Streaming synthetic rows are appended separately by callers (e.g. gou-demo scrollItemKeys), matching TS.
func ReorderMessagesInUI(messages []types.Message) []types.Message {
	if len(messages) == 0 {
		return nil
	}
	groups := make(map[string]*toolUseGroup)

	for i := range messages {
		msg := &messages[i]
		if isToolUseRequestMessage(*msg) {
			id := firstToolUseID(*msg)
			if id == "" {
				continue
			}
			if groups[id] == nil {
				groups[id] = &toolUseGroup{}
			}
			groups[id].toolUse = msg
			continue
		}
		if tid, ev, ok := hookAttachmentMeta(*msg); ok {
			if groups[tid] == nil {
				groups[tid] = &toolUseGroup{}
			}
			switch ev {
			case "PreToolUse":
				groups[tid].preHooks = append(groups[tid].preHooks, *msg)
			case "PostToolUse":
				groups[tid].postHooks = append(groups[tid].postHooks, *msg)
			}
			continue
		}
		if isToolResultUserMessage(*msg) {
			id := firstToolResultUseID(*msg)
			if id == "" {
				continue
			}
			if groups[id] == nil {
				groups[id] = &toolUseGroup{}
			}
			groups[id].toolResult = msg
		}
	}

	var result []types.Message
	processed := make(map[string]struct{})

	for i := range messages {
		msg := messages[i]
		if isToolUseRequestMessage(msg) {
			id := firstToolUseID(msg)
			if id == "" {
				continue
			}
			if _, ok := processed[id]; ok {
				continue
			}
			processed[id] = struct{}{}
			g := groups[id]
			if g != nil && g.toolUse != nil {
				result = append(result, *g.toolUse)
				result = append(result, g.preHooks...)
				if g.toolResult != nil {
					result = append(result, *g.toolResult)
				}
				result = append(result, g.postHooks...)
			}
			continue
		}
		if _, ev, ok := hookAttachmentMeta(msg); ok && (ev == "PreToolUse" || ev == "PostToolUse") {
			continue
		}
		if isToolResultUserMessage(msg) {
			continue
		}
		if msg.Type == types.MessageTypeSystem && msg.Subtype != nil && *msg.Subtype == "api_error" {
			if len(result) > 0 {
				last := result[len(result)-1]
				if last.Type == types.MessageTypeSystem && last.Subtype != nil && *last.Subtype == "api_error" {
					result[len(result)-1] = msg
					continue
				}
			}
			result = append(result, msg)
			continue
		}
		result = append(result, msg)
	}

	if len(result) == 0 {
		return result
	}
	lastIdx := len(result) - 1
	out := make([]types.Message, 0, len(result))
	for i, m := range result {
		if m.Type == types.MessageTypeSystem && m.Subtype != nil && *m.Subtype == "api_error" && i != lastIdx {
			continue
		}
		out = append(out, m)
	}
	return out
}
