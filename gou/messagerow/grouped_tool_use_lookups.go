package messagerow

import (
	"encoding/json"
	"strings"

	"goc/types"
)

// GroupedAgentLookups mirrors a minimal subset of TS buildMessageLookups
// specifically for resolving state of tool uses in grouped_tool_use blocks.
type GroupedAgentLookups struct {
	ResolvedToolUseIDs          map[string]bool
	ErroredToolUseIDs           map[string]bool
	InProgressToolUseIDs        map[string]bool
	ProgressMessagesByToolUseID map[string][]types.Message
}

// BuildGroupedAgentLookups creates lookup maps for resolved, errored, and in-progress tool uses,
// along with a mapping from tool_use_id to its progress messages.
func BuildGroupedAgentLookups(messages []types.Message) *GroupedAgentLookups {
	lookups := &GroupedAgentLookups{
		ResolvedToolUseIDs:          make(map[string]bool),
		ErroredToolUseIDs:           make(map[string]bool),
		InProgressToolUseIDs:        make(map[string]bool),
		ProgressMessagesByToolUseID: make(map[string][]types.Message),
	}

	for _, msg := range messages {
		// Build progress messages lookup
		if msg.Type == types.MessageTypeProgress {
			var toolUseID string
			if msg.ParentToolUseID != nil && *msg.ParentToolUseID != "" {
				toolUseID = *msg.ParentToolUseID
			} else if len(msg.Data) > 0 {
				var data struct {
					ParentToolUseID string `json:"parentToolUseID"`
				}
				if json.Unmarshal(msg.Data, &data) == nil && data.ParentToolUseID != "" {
					toolUseID = data.ParentToolUseID
				}
			}
			if toolUseID != "" {
				lookups.ProgressMessagesByToolUseID[toolUseID] = append(lookups.ProgressMessagesByToolUseID[toolUseID], msg)
			}
			continue
		}

		// Build tool result lookup and resolved/errored sets
		if msg.Type == types.MessageTypeUser && len(msg.Content) > 0 {
			var blocks []struct {
				Type      string `json:"type"`
				ToolUseID string `json:"tool_use_id"`
				IsError   *bool  `json:"is_error"`
			}
			if json.Unmarshal(msg.Content, &blocks) == nil {
				for _, block := range blocks {
					if block.Type == "tool_result" && strings.TrimSpace(block.ToolUseID) != "" {
						lookups.ResolvedToolUseIDs[block.ToolUseID] = true
						if block.IsError != nil && *block.IsError {
							lookups.ErroredToolUseIDs[block.ToolUseID] = true
						}
					}
				}
			}
		}
	}

	// Calculate InProgressToolUseIDs: tool uses that have been seen but not resolved
	for _, msg := range messages {
		if msg.Type == types.MessageTypeAssistant && len(msg.Content) > 0 {
			var blocks []struct {
				Type string `json:"type"`
				ID   string `json:"id"`
			}
			if json.Unmarshal(msg.Content, &blocks) == nil {
				for _, block := range blocks {
					if block.Type == "tool_use" && strings.TrimSpace(block.ID) != "" {
						if !lookups.ResolvedToolUseIDs[block.ID] {
							lookups.InProgressToolUseIDs[block.ID] = true
						}
					}
				}
			}
		} else if msg.Type == types.MessageTypeGroupedToolUse {
			for _, m := range msg.Messages {
				if len(m.Content) > 0 {
					var blocks []struct {
						Type string `json:"type"`
						ID   string `json:"id"`
					}
					if json.Unmarshal(m.Content, &blocks) == nil {
						for _, block := range blocks {
							if block.Type == "tool_use" && strings.TrimSpace(block.ID) != "" {
								if !lookups.ResolvedToolUseIDs[block.ID] {
									lookups.InProgressToolUseIDs[block.ID] = true
								}
							}
						}
					}
				}
			}
		}
	}

	return lookups
}
