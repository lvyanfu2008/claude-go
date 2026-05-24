// Package toolsearchwire is a thin facade over goc/internal/toolsearch for callers that want a stable tools/* import (e.g. query streaming).
package toolsearchwire

import (
	"encoding/json"

	"goc/internal/anthropic"
	"goc/internal/toolsearch"
)

// WireToolsJSON delegates to package toolsearch under goc/internal (same BuildWireConfig + ApplyWire as streaming parity HTTP payloads).
func WireToolsJSON(toolsJSON json.RawMessage, modelID string, hasPendingMcp, openAICompat bool, discoveryMsgsJSON json.RawMessage) (json.RawMessage, error) {
	return toolsearch.WireToolsJSON(toolsJSON, modelID, hasPendingMcp, openAICompat, discoveryMsgsJSON)
}

// PrepareMessagesForWire unmarshals messages and tools, builds the wire config, and applies
// [toolsearch.PrepareAnthropicMessages] so <available-deferred-tools> is prepended when dynamic
// tool loading is active. Returns the modified messages JSON (unchanged if wiring is not active).
func PrepareMessagesForWire(messagesJSON, toolsJSON json.RawMessage, modelID string, hasPendingMcp, openAICompat bool) json.RawMessage {
	if len(messagesJSON) == 0 {
		return messagesJSON
	}
	var msgs []anthropic.Message
	if err := json.Unmarshal(messagesJSON, &msgs); err != nil {
		return messagesJSON
	}
	if len(msgs) == 0 {
		return messagesJSON
	}
	var tools []anthropic.ToolDefinition
	_ = json.Unmarshal(toolsJSON, &tools)
	cfg := toolsearch.BuildWireConfig(modelID, tools, hasPendingMcp, openAICompat)
	prepared := toolsearch.PrepareAnthropicMessages(msgs, tools, cfg)
	out, err := json.Marshal(prepared)
	if err != nil {
		return messagesJSON
	}
	return json.RawMessage(out)
}
