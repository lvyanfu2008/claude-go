package compactservice

import (
	"encoding/json"

	"goc/types"
)

// StripImagesFromMessages mirrors stripImagesFromMessages in services/compact/compact.ts.
// User messages whose inner message.content is an array have image/document blocks
// replaced by a text "[image]" / "[document]" block (including blocks nested inside
// tool_result.content arrays). Other message shapes are returned unchanged.
func StripImagesFromMessages(messages []types.Message) []types.Message {
	out := make([]types.Message, len(messages))
	for i, m := range messages {
		if m.Type != types.MessageTypeUser || len(m.Message) == 0 {
			out[i] = m
			continue
		}
		replaced, ok := stripImagesInUserMessageJSON(m.Message)
		if !ok {
			out[i] = m
			continue
		}
		m.Message = replaced
		out[i] = m
	}
	return out
}

// stripImagesInUserMessageJSON walks the inner {role, content:[…]} shape and rewrites media
// blocks. Returns (possibly rewritten JSON, true) on success or (nil, false) when the payload
// is not an array-content user message (we return the original JSON unchanged).
func stripImagesInUserMessageJSON(innerJSON json.RawMessage) (json.RawMessage, bool) {
	var inner struct {
		Role    string            `json:"role"`
		Content json.RawMessage   `json:"content"`
		Extra   map[string]any    `json:"-"`
	}
	// Capture everything else so we can roundtrip it.
	var probe map[string]any
	if err := json.Unmarshal(innerJSON, &probe); err != nil {
		return nil, false
	}
	contentVal, ok := probe["content"]
	if !ok {
		return nil, false
	}
	arr, ok := contentVal.([]any)
	if !ok {
		// String content has no media blocks.
		return nil, false
	}

	newContent := make([]any, 0, len(arr))
	mutated := false
	for _, block := range arr {
		b, ok := block.(map[string]any)
		if !ok {
			newContent = append(newContent, block)
			continue
		}
		t, _ := b["type"].(string)
		switch t {
		case "image":
			newContent = append(newContent, map[string]any{"type": "text", "text": "[image]"})
			mutated = true
			continue
		case "document":
			newContent = append(newContent, map[string]any{"type": "text", "text": "[document]"})
			mutated = true
			continue
		case "tool_result":
			if inner, ok := b["content"].([]any); ok {
				newInner := make([]any, 0, len(inner))
				toolMutated := false
				for _, item := range inner {
					im, ok := item.(map[string]any)
					if !ok {
						newInner = append(newInner, item)
						continue
					}
					it, _ := im["type"].(string)
					switch it {
					case "image":
						newInner = append(newInner, map[string]any{"type": "text", "text": "[image]"})
						toolMutated = true
					case "document":
						newInner = append(newInner, map[string]any{"type": "text", "text": "[document]"})
						toolMutated = true
					default:
						newInner = append(newInner, item)
					}
				}
				if toolMutated {
					nb := make(map[string]any, len(b))
					for k, v := range b {
						nb[k] = v
					}
					nb["content"] = newInner
					newContent = append(newContent, nb)
					mutated = true
					continue
				}
			}
			newContent = append(newContent, b)
		default:
			newContent = append(newContent, b)
		}
	}

	if !mutated {
		return nil, false
	}
	probe["content"] = newContent
	raw, err := json.Marshal(probe)
	if err != nil {
		return nil, false
	}
	_ = inner
	return raw, true
}

// StripReinjectedAttachments mirrors stripReinjectedAttachments in TS. Outside the
// EXPERIMENTAL_SKILL_SEARCH feature gate (which Go does not yet ship), this is a pass-through.
// The optional FeatureExperimentalSkillSearch argument lets hosts enable the TS-parity filter.
func StripReinjectedAttachments(messages []types.Message, experimentalSkillSearch bool) []types.Message {
	if !experimentalSkillSearch {
		return messages
	}
	out := make([]types.Message, 0, len(messages))
	for _, m := range messages {
		if m.Type == types.MessageTypeAttachment && len(m.Attachment) > 0 {
			var probe struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(m.Attachment, &probe); err == nil {
				if probe.Type == "skill_discovery" || probe.Type == "skill_listing" {
					continue
				}
			}
		}
		out = append(out, m)
	}
	return out
}

// GetMessagesAfterCompactBoundary returns the suffix of messages after the last compact
// boundary (mirrors getMessagesAfterCompactBoundary in utils/messages.ts). Returns the full
// slice when no boundary is present.
func GetMessagesAfterCompactBoundary(messages []types.Message) []types.Message {
	idx := FindLastCompactBoundaryIndex(messages)
	if idx < 0 {
		return messages
	}
	return messages[idx+1:]
}
