package query

import (
	"encoding/json"
	"sort"
	"strings"

	"goc/anthropicmessages"
	"goc/conversation-runtime/streamingtool"
)

// assistantStreamAccumulator turns Anthropic Messages SSE events into one assistant API message (content blocks).
// Mirrors the assembly path in query.ts before normalizeMessagesForAPI.
type assistantStreamAccumulator struct {
	blocks       map[int]*accBlock
	stopReason   string
	inputTokens  int
	outputTokens int
	// apiMessageID is message.message.id from message_start (Anthropic API message id).
	apiMessageID string
}

type accBlock struct {
	typ         string
	id          string
	name        string
	text        strings.Builder
	toolInput   strings.Builder
	inputParsed json.RawMessage
}

func newAssistantStreamAccumulator() *assistantStreamAccumulator {
	return &assistantStreamAccumulator{blocks: make(map[int]*accBlock)}
}

func (a *assistantStreamAccumulator) OnEvent(ev anthropicmessages.MessageStreamEvent) error {
	switch ev.Type {
	case "message_start":
		var wrap struct {
			Message struct {
				ID string `json:"id"`
			} `json:"message"`
		}
		_ = json.Unmarshal(ev.Raw, &wrap)
		if id := strings.TrimSpace(wrap.Message.ID); id != "" {
			a.apiMessageID = id
		}
		return nil
	case "content_block_start":
		var wrap struct {
			Index        int             `json:"index"`
			ContentBlock json.RawMessage `json:"content_block"`
		}
		_ = json.Unmarshal(ev.Raw, &wrap)
		var cb struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		_ = json.Unmarshal(wrap.ContentBlock, &cb)
		a.blocks[wrap.Index] = &accBlock{typ: cb.Type, id: cb.ID, name: cb.Name}
		return nil
	case "content_block_delta":
		var wrap struct {
			Index int             `json:"index"`
			Delta json.RawMessage `json:"delta"`
		}
		if err := json.Unmarshal(ev.Raw, &wrap); err != nil {
			return err
		}
		b := a.blocks[wrap.Index]
		if b == nil {
			return nil
		}
		var d struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			PartialJSON string `json:"partial_json"`
		}
		_ = json.Unmarshal(wrap.Delta, &d)
		switch d.Type {
		case "text_delta":
			b.text.WriteString(d.Text)
		case "input_json_delta":
			b.toolInput.WriteString(d.PartialJSON)
		}
		return nil
	case "content_block_stop":
		var wrap struct {
			Index int `json:"index"`
		}
		_ = json.Unmarshal(ev.Raw, &wrap)
		b := a.blocks[wrap.Index]
		if b != nil && b.typ == "tool_use" {
			raw := strings.TrimSpace(b.toolInput.String())
			if raw == "" {
				raw = "{}"
			}
			b.inputParsed = json.RawMessage(raw)
		}
		return nil
	case "message_delta":
		var wrap struct {
			Delta json.RawMessage `json:"delta"`
		}
		_ = json.Unmarshal(ev.Raw, &wrap)
		var d struct {
			StopReason *string `json:"stop_reason"`
			Usage      *struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		_ = json.Unmarshal(wrap.Delta, &d)
		if d.StopReason != nil {
			a.stopReason = strings.TrimSpace(*d.StopReason)
		}
		if d.Usage != nil {
			if d.Usage.InputTokens > 0 {
				a.inputTokens = d.Usage.InputTokens
			}
			if d.Usage.OutputTokens > 0 {
				a.outputTokens = d.Usage.OutputTokens
			}
		}
		return nil
	case "message_stop", "ping":
		return nil
	default:
		return nil
	}
}

// AssistantWire returns role+content JSON for types.Message.Message (assistant row).
func (a *assistantStreamAccumulator) AssistantWire(uuid string) (inner json.RawMessage, err error) {
	keys := make([]int, 0, len(a.blocks))
	for k := range a.blocks {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var content []any
	for _, idx := range keys {
		b := a.blocks[idx]
		switch b.typ {
		case "text":
			if b.text.Len() > 0 {
				content = append(content, map[string]any{"type": "text", "text": b.text.String()})
			}
		case "tool_use":
			var inputObj any
			raw := b.inputParsed
			if len(raw) == 0 {
				raw = json.RawMessage(`{}`)
			}
			if err := json.Unmarshal(raw, &inputObj); err != nil {
				inputObj = map[string]any{}
			}
			content = append(content, map[string]any{
				"type":  "tool_use",
				"id":    b.id,
				"name":  b.name,
				"input": inputObj,
			})
		}
	}
	innerObj := map[string]any{
		"role":    "assistant",
		"content": content,
	}
	if id := strings.TrimSpace(a.apiMessageID); id != "" {
		innerObj["id"] = id
	}
	inner, err = json.Marshal(innerObj)
	if err != nil {
		return nil, err
	}
	_ = uuid
	return inner, nil
}

func (a *assistantStreamAccumulator) StopReason() string { return a.stopReason }

func (a *assistantStreamAccumulator) Usage() (in, out int) {
	return a.inputTokens, a.outputTokens
}

func (a *assistantStreamAccumulator) HasToolUse() bool {
	for _, b := range a.blocks {
		if b.typ == "tool_use" {
			return true
		}
	}
	return false
}

// ToolUseBlocks returns tool_use blocks in stream order for [streamingtool.StreamingToolExecutor.AddTool].
func (a *assistantStreamAccumulator) ToolUseBlocks() []streamingtool.ToolUseBlock {
	keys := make([]int, 0, len(a.blocks))
	for k := range a.blocks {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var out []streamingtool.ToolUseBlock
	for _, idx := range keys {
		b := a.blocks[idx]
		if b == nil || b.typ != "tool_use" {
			continue
		}
		raw := b.inputParsed
		if len(raw) == 0 {
			raw = json.RawMessage(`{}`)
		}
		out = append(out, streamingtool.ToolUseBlock{
			ID:    b.id,
			Name:  b.name,
			Input: append(json.RawMessage(nil), raw...),
		})
	}
	return out
}

// StreamingToolUsesLive returns a snapshot of tool_use blocks still being assembled (TS streamingToolUses).
func (a *assistantStreamAccumulator) StreamingToolUsesLive() []StreamingToolUseLive {
	if a == nil {
		return nil
	}
	keys := make([]int, 0, len(a.blocks))
	for k := range a.blocks {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var out []StreamingToolUseLive
	for _, idx := range keys {
		b := a.blocks[idx]
		if b == nil || b.typ != "tool_use" {
			continue
		}
		out = append(out, StreamingToolUseLive{
			Index:         idx,
			ToolUseID:     b.id,
			Name:          b.name,
			UnparsedInput: b.toolInput.String(),
		})
	}
	return out
}
