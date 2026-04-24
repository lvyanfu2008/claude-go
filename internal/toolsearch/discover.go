package toolsearch

import (
	"encoding/json"
	"strings"

	"goc/internal/anthropic"
)

// ExtractDiscoveredToolNames scans messages for tool_reference blocks and compact_boundary
// preCompactDiscoveredTools (mirrors extractDiscoveredToolNames in src/utils/toolSearch.ts).
func ExtractDiscoveredToolNames(msgs []anthropic.Message) map[string]struct{} {
	out := make(map[string]struct{})
	for _, m := range msgs {
		if isCompactBoundaryMessage(m) {
			var meta struct {
				PreCompactDiscoveredTools []string `json:"preCompactDiscoveredTools"`
			}
			if len(m.CompactMetadata) > 0 && json.Unmarshal(m.CompactMetadata, &meta) == nil {
				for _, n := range meta.PreCompactDiscoveredTools {
					n = strings.TrimSpace(n)
					if n != "" {
						out[n] = struct{}{}
					}
				}
			}
		}
		if m.Role == "user" {
			walkUserContentForToolReferences(m.Content, out)
		}
	}
	return out
}

func isCompactBoundaryMessage(m anthropic.Message) bool {
	if !strings.EqualFold(m.Subtype, "compact_boundary") {
		return false
	}
	return strings.EqualFold(m.Type, "system") || m.Role == "system"
}

func walkUserContentForToolReferences(content any, out map[string]struct{}) {
	if content == nil {
		return
	}
	switch v := content.(type) {
	case string:
		return
	case []anthropic.ContentBlock:
		for _, b := range v {
			if b.Type == "tool_result" {
				walkToolResultBody(b.Content, out)
			}
		}
	default:
		raw, err := json.Marshal(content)
		if err != nil {
			return
		}
		var blocks []anthropic.ContentBlock
		if err := json.Unmarshal(raw, &blocks); err != nil {
			return
		}
		walkUserContentForToolReferences(blocks, out)
	}
}

func walkToolResultBody(body any, out map[string]struct{}) {
	if body == nil {
		return
	}
	switch v := body.(type) {
	case string:
		walkToolResultDiscoveryJSON(v, out)
		return
	case []any:
		for _, el := range v {
			scanToolReferenceItem(el, out)
		}
	case []map[string]any:
		for _, m := range v {
			scanToolReferenceMap(m, out)
		}
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return
		}
		var arr []map[string]any
		if json.Unmarshal(raw, &arr) == nil {
			for _, m := range arr {
				scanToolReferenceMap(m, out)
			}
		}
	}
}

// walkToolResultDiscoveryJSON parses string tool_result bodies: OpenAI-compat ToolSearch returns
// {"discovery":[{"type":"tool_reference","tool_name":"..."}]} or a raw JSON array of those objects.
func walkToolResultDiscoveryJSON(s string, out map[string]struct{}) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	var wrap struct {
		Discovery []any `json:"discovery"`
	}
	if json.Unmarshal([]byte(s), &wrap) == nil && len(wrap.Discovery) > 0 {
		for _, el := range wrap.Discovery {
			scanToolReferenceItem(el, out)
		}
		return
	}
	var arr []any
	if json.Unmarshal([]byte(s), &arr) == nil {
		for _, el := range arr {
			scanToolReferenceItem(el, out)
		}
	}
}

func scanToolReferenceItem(el any, out map[string]struct{}) {
	m, ok := el.(map[string]any)
	if !ok {
		return
	}
	scanToolReferenceMap(m, out)
}

func scanToolReferenceMap(m map[string]any, out map[string]struct{}) {
	typ, _ := m["type"].(string)
	if typ != "tool_reference" {
		return
	}
	name, _ := m["tool_name"].(string)
	if name == "" {
		return
	}
	out[name] = struct{}{}
}
