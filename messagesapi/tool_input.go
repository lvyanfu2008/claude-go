package messagesapi

import (
	"encoding/json"
	"maps"
)

// normalizeToolInputForAPI mirrors src/utils/api.ts normalizeToolInputForAPI.
func normalizeToolInputForAPI(toolName string, input any) any {
	m, ok := input.(map[string]any)
	if !ok || m == nil {
		return input
	}
	switch toolName {
	case exitPlanModeV2ToolName:
		out := maps.Clone(m)
		delete(out, "plan")
		delete(out, "planFilePath")
		return out
	case fileEditToolName:
		if _, hasEdits := m["edits"]; hasEdits {
			out := maps.Clone(m)
			delete(out, "old_string")
			delete(out, "new_string")
			delete(out, "replace_all")
			return out
		}
		return input
	default:
		return input
	}
}

func normalizeToolInputRaw(toolName string, raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	out := normalizeToolInputForAPI(toolName, v)
	b, err := json.Marshal(out)
	if err != nil {
		return raw
	}
	return b
}
