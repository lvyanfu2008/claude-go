package message

import (
	"strings"

	"goc/gou/messagerow"
)

// toolResultTextPartsFromContent collects text payloads from tool_result.content
// when the API uses either a raw JSON string or a content array of {type:"text",text:"..."} blocks.
func toolResultTextPartsFromContent(content any) []string {
	if content == nil {
		return nil
	}
	switch v := content.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil
		}
		return []string{v}
	case []interface{}:
		var out []string
		for _, item := range v {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if t, _ := m["type"].(string); t != "text" {
				continue
			}
			tx, _ := m["text"].(string)
			if strings.TrimSpace(tx) == "" {
				continue
			}
			out = append(out, tx)
		}
		return out
	default:
		return nil
	}
}

// writeEditDiffLinesFromToolResultBlock returns indented unified-diff lines for FileWrite / FileEdit
// tool_result JSON (structuredPatch or create body), trying each text part until one matches.
// BlockIsToolError reads tool_result is_error from a parsed content block (bool, *bool, or JSON float).
func BlockIsToolError(block map[string]interface{}) bool {
	if block == nil {
		return false
	}
	v, ok := block["is_error"]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case *bool:
		return t != nil && *t
	case float64:
		return t != 0
	default:
		return false
	}
}

func writeEditDiffLinesFromToolResultBlock(block map[string]interface{}) ([]string, bool) {
	if block == nil {
		return nil, false
	}
	for _, p := range toolResultTextPartsFromContent(block["content"]) {
		if lines, ok := messagerow.IndentedWriteEditDiffLinesFromToolResultJSON(p); ok && len(lines) > 0 {
			return lines, true
		}
	}
	return nil, false
}
