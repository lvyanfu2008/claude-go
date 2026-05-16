package tools

import (
	"os"
	"strings"

	"goc/modelenv"
)

// GetAgentModel resolves the effective model for a subagent, mirroring TS
// src/utils/model/agent.ts getAgentModel().
//
// Priority order:
//  1. CLAUDE_CODE_SUBAGENT_MODEL env var (highest)
//  2. toolSpecifiedModel (from Agent tool's "model" parameter)
//  3. agentDefinitionModel (from agent frontmatter; defaults to "inherit")
//  4. parentModel fallback (for "inherit" and alias-matches-parent-tier cases)
func GetAgentModel(
	agentDefinitionModel string,
	parentModel string,
	toolSpecifiedModel string,
) string {
	if subagentEnv := strings.TrimSpace(os.Getenv("CLAUDE_CODE_SUBAGENT_MODEL")); subagentEnv != "" {
		return resolveAliasToModel(subagentEnv)
	}

	if toolSpecified := strings.TrimSpace(toolSpecifiedModel); toolSpecified != "" {
		if aliasMatchesParentTier(toolSpecified, parentModel) {
			return parentModel
		}
		return resolveAliasToModel(toolSpecified)
	}

	agentModel := strings.TrimSpace(agentDefinitionModel)
	if agentModel == "" {
		agentModel = getDefaultSubagentModel()
	}

	if strings.EqualFold(agentModel, "inherit") {
		return parentModel
	}

	if aliasMatchesParentTier(agentModel, parentModel) {
		return parentModel
	}

	return resolveAliasToModel(agentModel)
}

// getDefaultSubagentModel returns "inherit", matching TS getDefaultSubagentModel().
func getDefaultSubagentModel() string {
	return "inherit"
}

// aliasMatchesParentTier checks whether a model alias (sonnet/opus/haiku)
// matches the parent model's tier. Mirrors TS aliasMatchesParentTier().
func aliasMatchesParentTier(alias string, parentModel string) bool {
	canonical := strings.ToLower(strings.TrimSpace(parentModel))
	switch strings.ToLower(strings.TrimSpace(alias)) {
	case "opus":
		return strings.Contains(canonical, "opus")
	case "sonnet":
		return strings.Contains(canonical, "sonnet")
	case "haiku":
		return strings.Contains(canonical, "haiku")
	default:
		return false
	}
}

// resolveAliasToModel maps a model alias to a concrete model ID, or passes
// through a concrete ID unchanged. For aliases it checks tier-specific
// environment variables first, falling back to EffectiveMainLoopModel().
func resolveAliasToModel(model string) string {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return modelenv.DefaultMainLoopModelID
	}

	switch strings.ToLower(trimmed) {
	case "sonnet":
		if v := strings.TrimSpace(os.Getenv("ANTHROPIC_DEFAULT_SONNET_MODEL")); v != "" {
			return v
		}
		return modelenv.EffectiveMainLoopModel()
	case "opus":
		if v := strings.TrimSpace(os.Getenv("ANTHROPIC_DEFAULT_OPUS_MODEL")); v != "" {
			return v
		}
		return modelenv.EffectiveMainLoopModel()
	case "haiku":
		if v := strings.TrimSpace(os.Getenv("ANTHROPIC_DEFAULT_HAIKU_MODEL")); v != "" {
			return v
		}
		return modelenv.EffectiveMainLoopModel()
	default:
		return trimmed
	}
}
