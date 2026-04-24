package toolsearch

import (
	"encoding/json"
	"sort"
	"strings"

	"goc/internal/anthropic"
)

// PrepareAnthropicMessages mirrors claude.ts post-normalize steps for tool search:
// API-only user+assistant rows, optional <available-deferred-tools> prepend, or strip tool_reference / caller when standard.
func PrepareAnthropicMessages(msgs []anthropic.Message, allTools []anthropic.ToolDefinition, cfg WireConfig) []anthropic.Message {
	api := filterAPIRoleMessages(msgs)
	if cfg.UseDynamicToolLoading && cfg.ModelSupportsToolReference {
		if cfg.PrependAvailableDeferredBlock {
			list := deferredToolNamesSorted(allTools)
			if list != "" {
				pre := "<available-deferred-tools>\n" + list + "\n</available-deferred-tools>"
				api = append([]anthropic.Message{{Role: "user", Content: pre}}, api...)
			}
		}
		return api
	}
	out := make([]anthropic.Message, 0, len(api))
	for _, m := range api {
		out = append(out, stripToolReferencesFromUser(stripCallerFromAssistant(m)))
	}
	return out
}

func filterAPIRoleMessages(msgs []anthropic.Message) []anthropic.Message {
	out := make([]anthropic.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		out = append(out, anthropic.Message{
			Role:            m.Role,
			Content:         m.Content,
			Type:            m.Type,
			Subtype:         m.Subtype,
			CompactMetadata: m.CompactMetadata,
		})
	}
	return out
}

func deferredToolNamesSorted(tools []anthropic.ToolDefinition) string {
	var names []string
	for _, t := range tools {
		if IsDeferredToolName(t.Name) {
			names = append(names, t.Name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return strings.Join(names, "\n")
}

func stripToolReferencesFromUser(m anthropic.Message) anthropic.Message {
	if m.Role != "user" {
		return m
	}
	switch c := m.Content.(type) {
	case string:
		return m
	case []anthropic.ContentBlock:
		blocks := make([]anthropic.ContentBlock, len(c))
		copy(blocks, c)
		for i := range blocks {
			if blocks[i].Type != "tool_result" {
				continue
			}
			blocks[i].Content = stripToolReferenceFromToolResultBody(blocks[i].Content)
		}
		m.Content = blocks
		return m
	default:
		raw, err := json.Marshal(c)
		if err != nil {
			return m
		}
		var blocks []anthropic.ContentBlock
		if json.Unmarshal(raw, &blocks) != nil {
			return m
		}
		m.Content = blocks
		return stripToolReferencesFromUser(m)
	}
}

func stripToolReferenceFromToolResultBody(body any) any {
	if body == nil {
		return body
	}
	switch v := body.(type) {
	case string:
		return body
	case []any:
		var kept []any
		for _, el := range v {
			if m, ok := el.(map[string]any); ok {
				if typ, _ := m["type"].(string); typ == "tool_reference" {
					continue
				}
			}
			kept = append(kept, el)
		}
		if len(kept) == 0 {
			return []any{map[string]any{"type": "text", "text": "[Tool references removed - tool search not enabled]"}}
		}
		return kept
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return body
		}
		var arr []any
		if json.Unmarshal(raw, &arr) != nil {
			return body
		}
		return stripToolReferenceFromToolResultBody(arr)
	}
}

func stripCallerFromAssistant(m anthropic.Message) anthropic.Message {
	if m.Role != "assistant" {
		return m
	}
	switch c := m.Content.(type) {
	case string:
		return m
	case []anthropic.ContentBlock:
		blocks := make([]anthropic.ContentBlock, len(c))
		copy(blocks, c)
		for i := range blocks {
			if blocks[i].Type != "tool_use" || len(blocks[i].Input) == 0 {
				continue
			}
			var obj map[string]any
			if json.Unmarshal(blocks[i].Input, &obj) != nil {
				continue
			}
			delete(obj, "caller")
			blocks[i].Input, _ = json.Marshal(obj)
		}
		m.Content = blocks
		return m
	default:
		raw, err := json.Marshal(c)
		if err != nil {
			return m
		}
		var blocks []anthropic.ContentBlock
		if json.Unmarshal(raw, &blocks) != nil {
			return m
		}
		m.Content = blocks
		return stripCallerFromAssistant(m)
	}
}
