// Package toolsearchwire is a thin facade over goc/internal/toolsearch for callers that want a stable tools/* import (e.g. query streaming).
package toolsearchwire

import (
	"encoding/json"

	"goc/ccb-engine/diaglog"
	"goc/deferredtoolsdelta"
	"goc/internal/anthropic"
	"goc/internal/toolsearch"
	"goc/types"
)

// WireToolsJSON delegates to package toolsearch under goc/internal (same BuildWireConfig + ApplyWire as streaming parity HTTP payloads).
func WireToolsJSON(toolsJSON json.RawMessage, modelID string, hasPendingMcp, openAICompat bool, discoveryMsgsJSON json.RawMessage) (json.RawMessage, error) {
	return toolsearch.WireToolsJSON(toolsJSON, modelID, hasPendingMcp, openAICompat, discoveryMsgsJSON)
}

// PrepareMessagesForWire unmarshals messages and tools, builds the wire config, and prepends
// <available-deferred-tools> when dynamic tool loading is active.
//
// storeMsgs is the full conversation store messages (including attachments). When non-empty,
// delta-based injection is used: the function scans store messages for prior announcements,
// diffs against current deferred tools, and only prepends when the pool changed.
// When storeMsgs is nil (streaming paths without store access), always prepends.
func PrepareMessagesForWire(messagesJSON, toolsJSON json.RawMessage, modelID string, hasPendingMcp, openAICompat bool, storeMsgs []types.Message) json.RawMessage {
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

	deferredCount := countDeferredTools(tools)
	diaglog.Line("[deferred-tools] model=%s UseDynamic=%v SupportsToolRef=%v PrependBlock=%v deferredCount=%d msgs=%d storeMsgs=%d",
		modelID, cfg.UseDynamicToolLoading, cfg.ModelSupportsToolReference, cfg.PrependAvailableDeferredBlock,
		deferredCount, len(msgs), len(storeMsgs))

	if !cfg.UseDynamicToolLoading || !cfg.ModelSupportsToolReference || deferredCount == 0 {
		prepared := toolsearch.PrepareAnthropicMessages(msgs, tools, cfg)
		out, _ := json.Marshal(prepared)
		return json.RawMessage(out)
	}

	var currentDeferred []string
	for _, t := range tools {
		if toolsearch.IsDeferredToolName(t.Name) {
			currentDeferred = append(currentDeferred, t.Name)
		}
	}

	// Delta mode: only prepend when the deferred pool changed since last announcement.
	if len(storeMsgs) > 0 {
		delta := deferredtoolsdelta.GetDeferredToolsDelta(currentDeferred, storeMsgs)
		if delta == nil {
			diaglog.Line("[deferred-tools] delta unchanged, skip prepend")
			prepared := toolsearch.PrepareAnthropicMessages(msgs, tools, cfg)
			out, _ := json.Marshal(prepared)
			return json.RawMessage(out)
		}
		diaglog.Line("[deferred-tools] delta: added=%d removed=%d", len(delta.AddedNames), len(delta.RemovedNames))
	}

	// Prepend system reminder with full deferred tools list.
	reminder := deferredtoolsdelta.BuildDeferredToolsSystemReminder(currentDeferred)
	prepend := anthropic.Message{Role: "user", Content: reminder}
	prepared := append([]anthropic.Message{prepend}, msgs...)

	out, err := json.Marshal(prepared)
	if err != nil {
		return messagesJSON
	}
	return json.RawMessage(out)
}

// PersistDeferredAnnouncement appends an isMeta user message containing
// <available-deferred-tools> to the conversation store so the delta scanner
// can detect prior announcements on subsequent turns.
func PersistDeferredAnnouncement(store deferredAnnouncementStore, toolsJSON json.RawMessage) {
	if store == nil || len(toolsJSON) == 0 {
		return
	}
	var tools []anthropic.ToolDefinition
	_ = json.Unmarshal(toolsJSON, &tools)
	var names []string
	for _, t := range tools {
		if toolsearch.IsDeferredToolName(t.Name) {
			names = append(names, t.Name)
		}
	}
	if len(names) == 0 {
		return
	}
	reminder := deferredtoolsdelta.BuildDeferredToolsSystemReminder(names)
	// Use pui.SystemNoticeMessage to create an isMeta user message.
	msgBytes, _ := json.Marshal(map[string]any{
		"role":    "user",
		"content": reminder,
	})
	contentBytes, _ := json.Marshal(reminder)
	trueVal := true
	store.AppendMessage(types.Message{
		Type:    types.MessageTypeUser,
		Message: json.RawMessage(msgBytes),
		Content: json.RawMessage(contentBytes),
		IsMeta:  &trueVal,
	})
}

// deferredAnnouncementStore is the subset of *conversation.Store needed.
type deferredAnnouncementStore interface {
	AppendMessage(m types.Message)
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
