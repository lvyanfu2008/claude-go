// Package toolsearchwire is a thin facade over goc/internal/toolsearch for callers that want a stable tools/* import (e.g. query streaming).
package toolsearchwire

import (
	"encoding/json"

	"goc/ccb-engine/diaglog"
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
	// When delta mode is on, TS uses <system-reminder> attachments instead of
	// <available-deferred-tools> prepend. Go does not implement the delta attachment
	// path, so always force prepend when dynamic loading is active.
	if cfg.UseDynamicToolLoading && cfg.ModelSupportsToolReference && !cfg.PrependAvailableDeferredBlock {
		cfg.PrependAvailableDeferredBlock = true
	}
	prepared := toolsearch.PrepareAnthropicMessages(msgs, tools, cfg)
	changed := len(prepared) != len(msgs) || (len(prepared) > 0 && len(msgs) > 0 && prepared[0].Role != msgs[0].Role)
	diaglog.Line("[deferred-tools] model=%s UseDynamic=%v SupportsToolRef=%v PrependBlock=%v deferredCount=%d msgsBefore=%d msgsAfter=%d changed=%v",
		modelID, cfg.UseDynamicToolLoading, cfg.ModelSupportsToolReference, cfg.PrependAvailableDeferredBlock,
		countDeferredTools(tools), len(msgs), len(prepared), changed)
	out, err := json.Marshal(prepared)
	if err != nil {
		return messagesJSON
	}
	return json.RawMessage(out)
}

func countDeferredTools(tools []anthropic.ToolDefinition) int {
	n := 0
	for _, t := range tools {
		if toolsearch.IsDeferredToolName(t.Name) {
			n++
		}
	}
	return n
}
