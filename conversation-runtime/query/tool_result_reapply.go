package query

import (
	"encoding/json"
	"strings"

	"goc/types"
)

// contentReplacementWire is the JSON shape we re-apply from (TS ContentReplacementState serializes replacements).
type contentReplacementWire struct {
	Replacements map[string]string `json:"replacements"`
	SeenIds      []string          `json:"seenIds,omitempty"`
}

// ReapplyToolResultReplacementsFromState replaces tool_result block contents when
// [stateJSON] contains a non-empty "replacements" map (TS replaceToolResultContents read path).
// Does not enforce budgets or persist — parity with re-applying cached previews on resume.
func ReapplyToolResultReplacementsFromState(messages []types.Message, stateJSON json.RawMessage) []types.Message {
	if len(stateJSON) == 0 || !json.Valid(stateJSON) {
		out := make([]types.Message, len(messages))
		copy(out, messages)
		return out
	}
	var w contentReplacementWire
	if err := json.Unmarshal(stateJSON, &w); err != nil || len(w.Replacements) == 0 {
		out := make([]types.Message, len(messages))
		copy(out, messages)
		return out
	}
	return replaceToolResultsInUserMessages(messages, w.Replacements)
}

func replaceToolResultsInUserMessages(msgs []types.Message, rep map[string]string) []types.Message {
	out := make([]types.Message, len(msgs))
	copy(out, msgs)
	for i, m := range out {
		if m.Type != types.MessageTypeUser || len(m.Message) == 0 {
			continue
		}
		var env struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if json.Unmarshal(m.Message, &env) != nil || env.Role != "user" {
			continue
		}
		if len(env.Content) == 0 || env.Content[0] != '[' {
			continue
		}
		var blocks []map[string]any
		if json.Unmarshal(env.Content, &blocks) != nil {
			continue
		}
		changed := false
		for j, b := range blocks {
			typ, _ := b["type"].(string)
			if typ != "tool_result" {
				continue
			}
			id, _ := b["tool_use_id"].(string)
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			repl, ok := rep[id]
			if !ok {
				continue
			}
			b["content"] = repl
			blocks[j] = b
			changed = true
		}
		if !changed {
			continue
		}
		newContent, err := json.Marshal(blocks)
		if err != nil {
			continue
		}
		inner, err := json.Marshal(struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}{Role: "user", Content: newContent})
		if err != nil {
			continue
		}
		nm := m
		nm.Message = inner
		nm.Content = newContent
		out[i] = nm
	}
	return out
}
