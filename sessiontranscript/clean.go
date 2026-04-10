package sessiontranscript

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"goc/types"
)

// REPLToolName matches src/tools/REPLTool/constants.ts REPL_TOOL_NAME.
const REPLToolName = "REPL"

func envTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// IsLoggableMessage mirrors sessionStorage.ts isLoggableMessage.
func IsLoggableMessage(m types.Message, userType string) bool {
	if m.Type == types.MessageTypeProgress {
		return false
	}
	if m.Type == types.MessageTypeAttachment {
		if strings.TrimSpace(userType) == "ant" {
			return true
		}
		var att struct {
			Type string `json:"type"`
		}
		if len(m.Attachment) > 0 {
			_ = json.Unmarshal(m.Attachment, &att)
		}
		if att.Type == "hook_additional_context" && envTruthy("CLAUDE_CODE_SAVE_HOOK_ADDITIONAL_CONTEXT") {
			return true
		}
		return false
	}
	return true
}

// assistantContentBlocks returns API content blocks from Message.message (TS message.content).
func assistantContentBlocks(msg json.RawMessage) []struct {
	Type string `json:"type"`
	Name string `json:"name"`
	ID   string `json:"id"`
} {
	if len(msg) == 0 {
		return nil
	}
	var wrap struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(msg, &wrap); err != nil || len(wrap.Content) == 0 {
		return nil
	}
	var blocks []struct {
		Type string `json:"type"`
		Name string `json:"name"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal(wrap.Content, &blocks); err != nil {
		return nil
	}
	return blocks
}

// CollectReplIDs mirrors collectReplIds (from full session array).
func CollectReplIDs(messages []types.Message) map[string]struct{} {
	ids := make(map[string]struct{})
	for _, m := range messages {
		if m.Type != types.MessageTypeAssistant {
			continue
		}
		for _, b := range assistantContentBlocks(m.Message) {
			if b.Type == "tool_use" && b.Name == REPLToolName && b.ID != "" {
				ids[b.ID] = struct{}{}
			}
		}
	}
	return ids
}

// filterMessageContent strips REPL tool_use / tool_result pairs for external transcripts.
func filterAssistantReplContent(msg json.RawMessage, stripIDs map[string]struct{}) (json.RawMessage, bool, error) {
	if len(msg) == 0 {
		return msg, false, nil
	}
	var wrap struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(msg, &wrap); err != nil {
		return msg, false, err
	}
	if len(wrap.Content) == 0 {
		return msg, false, nil
	}
	var blocks []map[string]any
	if err := json.Unmarshal(wrap.Content, &blocks); err != nil {
		return msg, false, nil
	}
	var hasRepl bool
	for _, b := range blocks {
		if typ, _ := b["type"].(string); typ == "tool_use" {
			if name, _ := b["name"].(string); name == REPLToolName {
				hasRepl = true
				break
			}
		}
	}
	if !hasRepl {
		return msg, false, nil
	}
	var out []map[string]any
	for _, b := range blocks {
		typ, _ := b["type"].(string)
		if typ == "tool_use" {
			if name, _ := b["name"].(string); name == REPLToolName {
				continue
			}
		}
		out = append(out, b)
	}
	if len(out) == 0 {
		return nil, true, nil
	}
	newContent, err := json.Marshal(out)
	if err != nil {
		return msg, false, err
	}
	// Re-encode full message wrapper
	var full map[string]any
	_ = json.Unmarshal(msg, &full)
	full["content"] = json.RawMessage(newContent)
	raw, err := json.Marshal(full)
	// Second bool: omit entire message (only when out blocks empty above).
	return raw, false, err
}

func filterUserReplResults(msg json.RawMessage, replIDs map[string]struct{}) (json.RawMessage, bool, error) {
	if len(replIDs) == 0 || len(msg) == 0 {
		return msg, false, nil
	}
	var wrap struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(msg, &wrap); err != nil {
		return msg, false, err
	}
	var blocks []map[string]any
	if err := json.Unmarshal(wrap.Content, &blocks); err != nil {
		return msg, false, nil
	}
	var hasRepl bool
	for _, b := range blocks {
		if typ, _ := b["type"].(string); typ == "tool_result" {
			if id, _ := b["tool_use_id"].(string); id != "" {
				if _, ok := replIDs[id]; ok {
					hasRepl = true
					break
				}
			}
		}
	}
	if !hasRepl {
		return msg, false, nil
	}
	var out []map[string]any
	for _, b := range blocks {
		if typ, _ := b["type"].(string); typ == "tool_result" {
			if id, _ := b["tool_use_id"].(string); id != "" {
				if _, ok := replIDs[id]; ok {
					continue
				}
			}
		}
		out = append(out, b)
	}
	if len(out) == 0 {
		return nil, true, nil
	}
	newContent, err := json.Marshal(out)
	if err != nil {
		return msg, false, err
	}
	var full map[string]any
	_ = json.Unmarshal(msg, &full)
	full["content"] = json.RawMessage(newContent)
	raw, err := json.Marshal(full)
	return raw, false, err
}

// TransformMessagesForExternalTranscript mirrors transformMessagesForExternalTranscript.
func TransformMessagesForExternalTranscript(messages []types.Message, replIDs map[string]struct{}) []types.Message {
	out := make([]types.Message, 0, len(messages))
	for _, m := range messages {
		mm := m
		switch m.Type {
		case types.MessageTypeAssistant:
			if len(m.Message) == 0 {
				out = append(out, mm)
				continue
			}
			newMsg, drop, err := filterAssistantReplContent(m.Message, replIDs)
			if err != nil || drop {
				continue
			}
			if !bytes.Equal(newMsg, m.Message) {
				mm.Message = newMsg
				if m.IsVirtual != nil && *m.IsVirtual {
					mm.IsVirtual = boolPtr(false)
				}
			}
			out = append(out, mm)
		case types.MessageTypeUser:
			if len(m.Message) == 0 {
				out = append(out, mm)
				continue
			}
			newMsg, drop, err := filterUserReplResults(m.Message, replIDs)
			if err != nil || drop {
				continue
			}
			if !bytes.Equal(newMsg, m.Message) {
				mm.Message = newMsg
				if m.IsVirtual != nil && *m.IsVirtual {
					mm.IsVirtual = boolPtr(false)
				}
			}
			out = append(out, mm)
		default:
			out = append(out, mm)
		}
	}
	return out
}

func boolPtr(b bool) *bool { return &b }

// CleanMessagesForLogging mirrors cleanMessagesForLogging (subset: loggable filter + external REPL strip).
func CleanMessagesForLogging(messages, allMessages []types.Message, userType string) []types.Message {
	if len(allMessages) == 0 {
		allMessages = messages
	}
	var filtered []types.Message
	for _, m := range messages {
		if IsLoggableMessage(m, userType) {
			filtered = append(filtered, m)
		}
	}
	if strings.TrimSpace(userType) == "ant" {
		return filtered
	}
	repl := CollectReplIDs(allMessages)
	return TransformMessagesForExternalTranscript(filtered, repl)
}
