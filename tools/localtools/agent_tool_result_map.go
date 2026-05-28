package localtools

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var errNotAgentToolOutput = errors.New("not Agent tool structured output")

// oneShotBuiltinAgentTypes mirrors ONE_SHOT_BUILTIN_AGENT_TYPES in TS AgentTool/constants.ts.
// Results from these agent types omit the agentId/usage trailer to save tokens.
var oneShotBuiltinAgentTypes = map[string]bool{
	"Explore": true,
	"Plan":    true,
}

// MapAgentToolResultToAssistantText mirrors AgentTool.mapToolResultToToolResultBlockParam
// (AgentTool.tsx). For one-shot builtin agents (Explore, Plan) without a worktree, it strips
// the agentId/usage trailer. For regular agents, it appends the agentId hint line.
func MapAgentToolResultToAssistantText(toolUseJSON string) (string, error) {
	toolUseJSON = strings.TrimSpace(toolUseJSON)
	if toolUseJSON == "" || toolUseJSON[0] != '{' {
		return "", errNotAgentToolOutput
	}

	var wrapper struct {
		Data struct {
			Output       string `json:"output"`
			AgentID      string `json:"agent_id"`
			AgentType    string `json:"agent_type"`
			WorktreePath string `json:"worktree_path"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(toolUseJSON), &wrapper); err != nil {
		return "", err
	}

	output := strings.TrimSpace(wrapper.Data.Output)
	if output == "" {
		return output, nil
	}

	if oneShotBuiltinAgentTypes[wrapper.Data.AgentType] && wrapper.Data.WorktreePath == "" {
		return output, nil
	}

	trailer := fmt.Sprintf("\n\nagentId: %s (use SendMessage with to: '%s' to continue this agent)",
		wrapper.Data.AgentID, wrapper.Data.AgentID)
	return output + trailer, nil
}
