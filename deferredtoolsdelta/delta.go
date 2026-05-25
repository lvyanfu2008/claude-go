package deferredtoolsdelta

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"goc/types"
)

// DeferredToolsDelta mirrors TS DeferredToolsDelta in searchExtraTools.ts.
type DeferredToolsDelta struct {
	AddedNames   []string `json:"addedNames"`
	AddedLines   []string `json:"addedLines"`
	RemovedNames []string `json:"removedNames"`
}

// GetDeferredToolsDelta scans messages for existing deferred_tools_delta attachments
// AND <available-deferred-tools> blocks in user messages, reconstructs the announced
// tool name set, and diffs against the current deferred tool names.
// Returns nil when nothing changed.
//
// TS parity: getDeferredToolsDelta in src/utils/searchExtraTools.ts.
func GetDeferredToolsDelta(currentDeferredNames []string, messages []types.Message) *DeferredToolsDelta {
	announced := make(map[string]struct{})
	for _, msg := range messages {
		// Scan deferred_tools_delta attachments (TS delta path).
		if msg.Type == types.MessageTypeAttachment && len(msg.Attachment) > 0 {
			var att struct {
				Type         string   `json:"type"`
				AddedNames   []string `json:"addedNames"`
				RemovedNames []string `json:"removedNames"`
			}
			if json.Unmarshal(msg.Attachment, &att) == nil && att.Type == "deferred_tools_delta" {
				for _, n := range att.AddedNames {
					announced[n] = struct{}{}
				}
				for _, n := range att.RemovedNames {
					delete(announced, n)
				}
			}
		}
		// Also scan <available-deferred-tools> blocks in user messages (non-delta path
		// and compaction recovery). Handles both string content and array-of-blocks formats.
		if msg.Type == types.MessageTypeUser && len(msg.Message) > 0 {
			// Try string content first (simple user messages).
			var inner struct {
				Content string `json:"content"`
			}
			if json.Unmarshal(msg.Message, &inner) == nil && inner.Content != "" {
				announcedFromBlock(inner.Content, announced)
			}
			// Also try array-of-blocks content (API-normalized format).
			var innerArr struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			}
			if json.Unmarshal(msg.Message, &innerArr) == nil {
				for _, blk := range innerArr.Content {
					if blk.Type == "text" && blk.Text != "" {
						announcedFromBlock(blk.Text, announced)
					}
				}
			}
		}
	}

	currentSet := make(map[string]struct{}, len(currentDeferredNames))
	for _, n := range currentDeferredNames {
		currentSet[n] = struct{}{}
	}

	var added []string
	for _, n := range currentDeferredNames {
		if _, ok := announced[n]; !ok {
			added = append(added, n)
		}
	}
	sort.Strings(added)

	var removed []string
	for n := range announced {
		if _, ok := currentSet[n]; ok {
			continue
		}
		removed = append(removed, n)
	}
	sort.Strings(removed)

	if len(added) == 0 && len(removed) == 0 {
		return nil
	}

	// Build addedLines: " - <name>: <description>" like TS formatDeferredToolLine
	addedLines := make([]string, len(added))
	for i, n := range added {
		addedLines[i] = " - " + n + ": deferred (use ToolSearch to discover schema)"
	}

	return &DeferredToolsDelta{
		AddedNames:   added,
		AddedLines:   addedLines,
		RemovedNames: removed,
	}
}

// BuildDeltaAttachmentJSON returns a json.RawMessage for a deferred_tools_delta attachment,
// or nil if the delta is nil.
func BuildDeltaAttachmentJSON(delta *DeferredToolsDelta) json.RawMessage {
	if delta == nil {
		return nil
	}
	b, _ := json.Marshal(delta)
	return json.RawMessage(b)
}

// BuildDeferredToolsSystemReminder returns a <system-reminder> user message content
// announcing available deferred tools and how to discover them. Matches TS claude.ts format.
func BuildDeferredToolsSystemReminder(names []string) string {
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	lines := make([]string, len(names))
	for i, n := range names {
		lines[i] = n
	}
	list := strings.Join(lines, "\n")
	return fmt.Sprintf(`<system-reminder>
<available-deferred-tools>
%s
</available-deferred-tools>
IMPORTANT: These tools are deferred-loading. You MUST first discover a tool via ToolSearch before invoking it. Do NOT call a deferred tool directly — it will fail if the tool has not been discovered.

Steps:
1. ToolSearch("select:<tool_name>") — discover the tool and its schema
2. Call the tool with correct parameters
</system-reminder>`, list)
}

// announcedFromBlock extracts deferred tool names from a <available-deferred-tools> block.
func announcedFromBlock(content string, announced map[string]struct{}) {
	start := strings.Index(content, "<available-deferred-tools>")
	if start < 0 {
		return
	}
	end := strings.Index(content[start:], "</available-deferred-tools>")
	if end < 0 {
		return
	}
	block := content[start+len("<available-deferred-tools>") : start+end]
	for _, line := range strings.Split(block, "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			announced[name] = struct{}{}
		}
	}
}

// BuildDeltaSystemReminder returns a system-reminder for a delta (added + removed), or "" if empty.
func BuildDeltaSystemReminder(delta *DeferredToolsDelta) string {
	if delta == nil {
		return ""
	}
	var parts []string
	if len(delta.AddedLines) > 0 {
		parts = append(parts, "The following deferred tools are now available via ToolSearch:\n"+strings.Join(delta.AddedLines, "\n"))
	}
	if len(delta.RemovedNames) > 0 {
		parts = append(parts, "The following deferred tools are no longer available. Do not search for them — ToolSearch will return no match:\n"+strings.Join(delta.RemovedNames, "\n"))
	}
	if len(parts) == 0 {
		return ""
	}
	return "<system-reminder>\n" + strings.Join(parts, "\n\n") + "\n</system-reminder>"
}
