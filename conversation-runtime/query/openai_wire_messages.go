package query

import (
	"encoding/json"
	"fmt"
	"strings"
)

// anthropicWireMessagesToOpenAI mirrors src/api-client/openai/convertMessages.ts
// anthropicMessagesToOpenAI for wire JSON from [ccbhydrate.MessagesJSONNormalized].
func anthropicWireMessagesToOpenAI(msgsJSON json.RawMessage, systemPrompt []string) ([]map[string]any, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(msgsJSON, &arr); err != nil {
		return nil, fmt.Errorf("messages json: %w", err)
	}
	var out []map[string]any
	sys := strings.TrimSpace(strings.Join(systemPrompt, "\n\n"))
	if sys != "" {
		out = append(out, map[string]any{"role": "system", "content": sys})
	}
	for _, row := range arr {
		var m struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(row, &m); err != nil {
			continue
		}
		switch m.Role {
		case "user":
			part, err := wireUserToOpenAI(m.Content)
			if err != nil {
				return nil, err
			}
			out = append(out, part...)
		case "assistant":
			part, err := wireAssistantToOpenAI(m.Content)
			if err != nil {
				return nil, err
			}
			out = append(out, part...)
		default:
			continue
		}
	}
	return out, nil
}

func wireUserToOpenAI(content json.RawMessage) ([]map[string]any, error) {
	if len(content) == 0 || string(content) == "null" {
		return nil, nil
	}
	// string content
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		return []map[string]any{{"role": "user", "content": s}}, nil
	}
	var blocks []map[string]any
	if err := json.Unmarshal(content, &blocks); err != nil {
		return nil, fmt.Errorf("user content: %w", err)
	}
	var out []map[string]any
	var textParts []string
	flush := func() {
		if len(textParts) == 0 {
			return
		}
		if len(textParts) == 1 {
			out = append(out, map[string]any{"role": "user", "content": textParts[0]})
		} else {
			parts := make([]map[string]any, len(textParts))
			for i, t := range textParts {
				parts[i] = map[string]any{"type": "text", "text": t}
			}
			out = append(out, map[string]any{"role": "user", "content": parts})
		}
		textParts = nil
	}
	for _, b := range blocks {
		typ, _ := b["type"].(string)
		switch typ {
		case "text":
			if tx, ok := b["text"].(string); ok && tx != "" {
				textParts = append(textParts, tx)
			}
		case "tool_result":
			flush()
			tid, _ := b["tool_use_id"].(string)
			out = append(out, map[string]any{
				"role":         "tool",
				"tool_call_id": tid,
				"content":      toolResultContentToString(b["content"]),
			})
		default:
			if tx, ok := b["text"].(string); ok && tx != "" {
				textParts = append(textParts, tx)
			} else if typ != "" {
				textParts = append(textParts, fallbackBlockSummary(b))
			}
		}
	}
	flush()
	return out, nil
}

func toolResultContentToString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}

func fallbackBlockSummary(b map[string]any) string {
	typ, _ := b["type"].(string)
	line := "[" + typ
	if n, ok := b["name"].(string); ok && n != "" {
		line += " " + n
	}
	if id, ok := b["id"].(string); ok && id != "" {
		line += " id=" + id
	}
	line += "]"
	return line
}

func wireAssistantToOpenAI(content json.RawMessage) ([]map[string]any, error) {
	if len(content) == 0 || string(content) == "null" {
		return []map[string]any{{"role": "assistant", "content": ""}}, nil
	}
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		return []map[string]any{{"role": "assistant", "content": s}}, nil
	}
	var blocks []map[string]any
	if err := json.Unmarshal(content, &blocks); err != nil {
		return nil, fmt.Errorf("assistant content: %w", err)
	}
	var textParts []string
	var toolCalls []map[string]any
	for _, b := range blocks {
		typ, _ := b["type"].(string)
		switch typ {
		case "text":
			if tx, ok := b["text"].(string); ok {
				textParts = append(textParts, tx)
			}
		case "tool_use":
			id, _ := b["id"].(string)
			name, _ := b["name"].(string)
			args := "{}"
			if in := b["input"]; in != nil {
				if s, ok := in.(string); ok {
					args = s
				} else {
					raw, err := json.Marshal(in)
					if err == nil && len(raw) > 0 {
						args = string(raw)
					}
				}
			}
			if strings.TrimSpace(args) == "" {
				args = "{}"
			}
			toolCalls = append(toolCalls, map[string]any{
				"id":   id,
				"type": "function",
				"function": map[string]any{
					"name":      name,
					"arguments": args,
				},
			})
		default:
			if tx, ok := b["text"].(string); ok && tx != "" {
				textParts = append(textParts, tx)
			} else if typ != "" {
				textParts = append(textParts, fallbackBlockSummary(b))
			}
		}
	}
	msg := map[string]any{"role": "assistant"}
	if len(textParts) > 0 {
		msg["content"] = strings.Join(textParts, "\n")
	} else {
		msg["content"] = nil
	}
	if len(toolCalls) > 0 {
		msg["tool_calls"] = toolCalls
	}
	if msg["content"] == nil && len(toolCalls) == 0 {
		msg["content"] = ""
	}
	return []map[string]any{msg}, nil
}
