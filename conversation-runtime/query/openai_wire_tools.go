package query

import (
	"encoding/json"
)

// anthropicToolsWireToOpenAI mirrors src/api-client/openai/convertTools.ts anthropicToolsToOpenAI
// plus the advisor/computer filter from src/api-client/openai/index.ts standardTools.
func anthropicToolsWireToOpenAI(toolsJSON json.RawMessage) ([]map[string]any, error) {
	if len(toolsJSON) == 0 || string(toolsJSON) == "null" {
		return nil, nil
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(toolsJSON, &arr); err != nil {
		return nil, err
	}
	var out []map[string]any
	for _, raw := range arr {
		var t map[string]any
		if err := json.Unmarshal(raw, &t); err != nil {
			continue
		}
		typ, _ := t["type"].(string)
		if typ == "advisor_20260301" || typ == "computer_20250124" {
			continue
		}
		if typ == "server" {
			continue
		}
		name, _ := t["name"].(string)
		desc, _ := t["description"].(string)
		params := t["input_schema"]
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        name,
				"description": desc,
				"parameters":  params,
			},
		})
	}
	return out, nil
}
