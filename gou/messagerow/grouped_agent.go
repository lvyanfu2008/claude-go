package messagerow

import (
	"encoding/json"
	"fmt"
	"strings"

	"goc/types"
)

// AgentStat tracks state and token usage for a single agent tool use.
type AgentStat struct {
	ID               string
	AgentType        string
	Description      string
	ToolUseCount     int
	Tokens           int
	IsResolved       bool
	IsError          bool
	IsAsync          bool
	TaskDescription  string
	Name             string
}

// FormatGroupedAgentToolUse generates segments for a grouped_tool_use message (TS AgentTool/UI.tsx renderGroupedAgentToolUse).
func FormatGroupedAgentToolUse(msg types.Message, lookups *GroupedAgentLookups) []Segment {
	if len(msg.Messages) == 0 {
		return nil
	}

	var stats []AgentStat
	for _, m := range msg.Messages {
		if len(m.Content) == 0 {
			continue
		}
		var blocks []struct {
			Type  string          `json:"type"`
			ID    string          `json:"id"`
			Input json.RawMessage `json:"input"`
		}
		if json.Unmarshal(m.Content, &blocks) != nil {
			continue
		}

		for _, block := range blocks {
			if block.Type != "tool_use" {
				continue
			}

			stat := AgentStat{
				ID:        block.ID,
				AgentType: "Agent",
			}

			// Parse input
			var input struct {
				SubagentType    string `json:"subagent_type"`
				Name            string `json:"name"`
				Description     string `json:"description"`
				RunInBackground bool   `json:"run_in_background"`
			}
			json.Unmarshal(block.Input, &input)

			isTeammateSpawn := false // teammate_spawned is not in output struct directly, but we guess if it has a custom subagent_type and name
			if input.SubagentType != "" && input.SubagentType != "generalPurpose" && input.Name != "" {
				isTeammateSpawn = true
			}

			if isTeammateSpawn {
				stat.AgentType = "@" + input.Name
				stat.TaskDescription = input.Description
				if input.SubagentType != "worker" {
					stat.Description = input.SubagentType
				}
			} else {
				if input.SubagentType != "" {
					if input.SubagentType == "worker" {
						stat.AgentType = "Agent"
					} else {
						stat.AgentType = input.SubagentType
					}
				}
				stat.Description = input.Description
			}
			stat.Name = input.Name

			// Set resolved/error status
			if lookups != nil {
				stat.IsResolved = lookups.ResolvedToolUseIDs[block.ID]
				stat.IsError = lookups.ErroredToolUseIDs[block.ID]
			}

			// In Go, since we don't have deep progress integration yet, we just check background flag
			// Async if requested in background or is teammate
			stat.IsAsync = input.RunInBackground || isTeammateSpawn

			// Calculate progress stats (tool uses and tokens)
			if lookups != nil && lookups.ProgressMessagesByToolUseID != nil {
				progMsgs := lookups.ProgressMessagesByToolUseID[block.ID]
				for _, pm := range progMsgs {
					if pm.Type != types.MessageTypeProgress || len(pm.Data) == 0 {
						continue
					}
					var data struct {
						Message struct {
							Type    string `json:"type"`
							Message struct {
								Content []struct {
									Type string `json:"type"`
								} `json:"content"`
								Usage *struct {
									InputTokens  int `json:"input_tokens"`
									OutputTokens int `json:"output_tokens"`
								} `json:"usage"`
							} `json:"message"`
						} `json:"message"`
					}
					if json.Unmarshal(pm.Data, &data) == nil {
						if data.Message.Type == "user" {
							for _, c := range data.Message.Message.Content {
								if c.Type == "tool_result" {
									stat.ToolUseCount++
								}
							}
						} else if data.Message.Type == "assistant" && data.Message.Message.Usage != nil {
							stat.Tokens = data.Message.Message.Usage.InputTokens + data.Message.Message.Usage.OutputTokens
						}
					}
				}
			}

			stats = append(stats, stat)
		}
	}

	if len(stats) == 0 {
		return nil
	}

	// Calculate top-line summary
	anyUnresolved := false
	allAsync := true
	commonType := stats[0].AgentType
	allSameType := true

	for _, s := range stats {
		if !s.IsResolved {
			anyUnresolved = true
		}
		if !s.IsAsync {
			allAsync = false
		}
		if s.AgentType != commonType {
			allSameType = false
		}
	}
	if !allSameType || commonType == "Agent" {
		commonType = ""
	}

	var out []Segment

	// Top line
	agentNoun := "agents"
	if commonType != "" {
		agentNoun = commonType + " agents"
	}

	var title string
	if !anyUnresolved {
		if allAsync {
			title = fmt.Sprintf("%d background agents launched", len(stats))
		} else {
			title = fmt.Sprintf("%d %s finished", len(stats), agentNoun)
		}
	} else {
		title = fmt.Sprintf("Running %d %s…", len(stats), agentNoun)
	}

	if !allAsync {
		title += CtrlOToExpandHint
	}
	out = append(out, Segment{Kind: SegGroupedToolUse, Text: title})

	// Sub lines: simplified to name + status only (progress moved to AgentFooter)
	for _, s := range stats {
		var b strings.Builder
		b.WriteString("  ⎿  ")

		nameDisplay := s.AgentType
		if !allSameType && s.AgentType == "Agent" && s.Name != "" {
			nameDisplay = s.Name
		}
		b.WriteString(nameDisplay)

		if s.IsError {
			b.WriteString(" · error")
		} else if s.IsResolved {
			b.WriteString(" · done")
		} else if s.IsAsync {
			b.WriteString(" · background")
		}

		out = append(out, Segment{Kind: SegDisplayHint, Text: b.String()})
	}

	return out
}
