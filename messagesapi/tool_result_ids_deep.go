package messagesapi

import (
	"encoding/json"

	"goc/types"
)

// toolResultUseIDsFromMessageDeep returns tool_use_id / tool_call_id from every
// JSON object with "type":"tool_result" in the full serialized message, regardless of
// where it sits (transcript importers can nest blocks outside message.content).
func toolResultUseIDsFromMessageDeep(m *types.Message) []string {
	if m == nil {
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil
	}
	seen := make(map[string]struct{})
	scanValueForToolResult(v, &seen)
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out
}

func scanValueForToolResult(v any, out *map[string]struct{}) {
	switch t := v.(type) {
	case map[string]any:
		typ, _ := t["type"].(string)
		if typ == "tool_result" {
			if id, ok := t["tool_use_id"].(string); ok && id != "" {
				(*out)[id] = struct{}{}
			}
			if id, ok := t["tool_call_id"].(string); ok && id != "" {
				(*out)[id] = struct{}{}
			}
		}
		for _, vv := range t {
			scanValueForToolResult(vv, out)
		}
	case []any:
		for _, el := range t {
			scanValueForToolResult(el, out)
		}
	}
}

// messageDeepCoversToolUseIDs reports whether every id in required appears on at least one
// tool_result object in the deep scan of the message.
func messageDeepCoversToolUseIDs(m types.Message, required []string) bool {
	if len(required) == 0 {
		return true
	}
	have := make(map[string]struct{})
	ids := toolResultUseIDsFromMessageDeep(&m)
	for _, id := range ids {
		have[id] = struct{}{}
	}
	for _, r := range required {
		if _, ok := have[r]; !ok {
			return false
		}
	}
	return true
}
