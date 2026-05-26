package messagerow

import (
	"encoding/json"
	"fmt"

	"goc/types"
)

const maxProgressMessagesToShow = 3

// AgentProgressSummary is the processed representation of an agent's progress
// messages during execution, matching TS renderToolUseProgressMessage output.
type AgentProgressSummary struct {
	ToolUseCount int
	Tokens       int
	IsEmpty      bool
}

// FormatAgentProgressSegments is a no-op now that progress is shown in AgentFooter.
func FormatAgentProgressSegments(progressMessages []types.Message) []Segment {
	return nil
}

func computeAgentProgressStats(progressMessages []types.Message) AgentProgressSummary {
	var stats AgentProgressSummary
	for _, pm := range progressMessages {
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
		if json.Unmarshal(pm.Data, &data) != nil {
			continue
		}
		if data.Message.Type == "user" {
			for _, c := range data.Message.Message.Content {
				if c.Type == "tool_result" {
					stats.ToolUseCount++
				}
			}
		} else if data.Message.Type == "assistant" && data.Message.Message.Usage != nil {
			stats.Tokens = data.Message.Message.Usage.InputTokens + data.Message.Message.Usage.OutputTokens
		}
	}
	stats.IsEmpty = stats.ToolUseCount == 0 && stats.Tokens == 0
	return stats
}

func extractRecentActivities(progressMessages []types.Message, maxShow int) []string {
	var activities []string
	// Collect from the end (most recent first), then reverse
	for i := len(progressMessages) - 1; i >= 0; i-- {
		pm := progressMessages[i]
		if pm.Type != types.MessageTypeProgress || len(pm.Data) == 0 {
			continue
		}
		var data struct {
			Message struct {
				Type    string `json:"type"`
				Message struct {
					Content []struct {
						Type  string          `json:"type"`
						Text  string          `json:"text"`
						Name  string          `json:"name"`
						ID    string          `json:"id"`
						Input json.RawMessage `json:"input"`
					} `json:"content"`
				} `json:"message"`
			} `json:"message"`
		}
		if json.Unmarshal(pm.Data, &data) != nil {
			continue
		}
		if data.Message.Type != "assistant" {
			continue
		}
		for _, c := range data.Message.Message.Content {
			if c.Type == "tool_use" {
				activity := ActivityLineForToolUse(c.Name, c.Input)
				if activity != "" {
					activities = append(activities, activity)
					if len(activities) >= maxShow {
						// Reverse to restore chronological order
						for left, right := 0, len(activities)-1; left < right; left, right = left+1, right-1 {
							activities[left], activities[right] = activities[right], activities[left]
						}
						return activities
					}
				}
			}
		}
	}
	// Reverse to restore chronological order
	for left, right := 0, len(activities)-1; left < right; left, right = left+1, right-1 {
		activities[left], activities[right] = activities[right], activities[left]
	}
	return activities
}

func formatTokenCount(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%.1fk", float64(n)/1000)
}
