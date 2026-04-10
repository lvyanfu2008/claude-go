package commands

import (
	"os"
	"strings"

	"goc/types"
)

// FeatureMcpSkills mirrors TS feature('MCP_SKILLS') via FEATURE_MCP_SKILLS=1 (see docs/features/mcp-skills.md).
func FeatureMcpSkills() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("FEATURE_MCP_SKILLS")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// MergeCommandsUniqByName mirrors lodash uniqBy([...first, ...second], 'name'): first occurrence wins (TS spread order).
func MergeCommandsUniqByName(first, second []types.Command) []types.Command {
	seen := make(map[string]struct{}, len(first)+len(second))
	out := make([]types.Command, 0, len(first)+len(second))
	for _, c := range first {
		n := strings.TrimSpace(c.Name)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue // lodash uniqBy: first occurrence wins within first as well
		}
		seen[n] = struct{}{}
		out = append(out, c)
	}
	for _, c := range second {
		n := strings.TrimSpace(c.Name)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, c)
	}
	return out
}

// FilterMcpPromptCommands returns commands with type prompt and loadedFrom mcp (TS getAllCommands filter on appState.mcp.commands).
func FilterMcpPromptCommands(mcp []types.Command) []types.Command {
	if len(mcp) == 0 {
		return nil
	}
	out := make([]types.Command, 0)
	for _, cmd := range mcp {
		if cmd.Type != "prompt" {
			continue
		}
		if cmd.LoadedFrom == nil || *cmd.LoadedFrom != "mcp" {
			continue
		}
		out = append(out, cmd)
	}
	return out
}

// MergeCommandsForSkillTool mirrors SkillTool.ts getAllCommands merge: local getCommands + MCP prompt commands, uniqBy name.
func MergeCommandsForSkillTool(local []types.Command, mcp []types.Command) []types.Command {
	mcpPrompt := FilterMcpPromptCommands(mcp)
	if len(mcpPrompt) == 0 {
		return local
	}
	return MergeCommandsUniqByName(local, mcpPrompt)
}

// SkillListingCommandsForAPI mirrors getSkillListingAttachments command list:
// getSkillToolCommands slice merged with getMcpSkillCommands when MCP_SKILLS and mcp non-empty.
func SkillListingCommandsForAPI(localGetCommands []types.Command, mcp []types.Command, mcpSkillsFeature bool) []types.Command {
	localSkill := SkillToolCommands(localGetCommands)
	mcpSkills := GetMcpSkillCommands(mcp, mcpSkillsFeature)
	if len(mcpSkills) == 0 {
		return localSkill
	}
	return MergeCommandsUniqByName(localSkill, mcpSkills)
}

// SkillListingFromTSPresliced merges TS getSkillToolCommands output (already filtered) with MCP skills.
// Do not pass a full getCommands list — use [SkillListingCommandsForAPI] instead.
func SkillListingFromTSPresliced(tsSkillToolCommands []types.Command, mcp []types.Command, mcpSkillsFeature bool) []types.Command {
	mcpSkills := GetMcpSkillCommands(mcp, mcpSkillsFeature)
	if len(mcpSkills) == 0 {
		out := make([]types.Command, len(tsSkillToolCommands))
		copy(out, tsSkillToolCommands)
		return out
	}
	return MergeCommandsUniqByName(tsSkillToolCommands, mcpSkills)
}
