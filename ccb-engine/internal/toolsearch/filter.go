package toolsearch

import (
	"goc/ccb-engine/internal/anthropic"
)

// ApplyWire returns the tools[] slice for the Messages API (mirrors filteredTools + defer_loading in src/services/api/claude.ts).
// fullTools is the host registry (unchanged for validation); the return value is a shallow copy with optional DeferLoading set.
func ApplyWire(fullTools []anthropic.ToolDefinition, msgs []anthropic.Message, cfg WireConfig) []anthropic.ToolDefinition {
	if !cfg.UseDynamicToolLoading || !cfg.ModelSupportsToolReference {
		return stripToolSearchOnly(fullTools)
	}

	deferredNames := make(map[string]struct{})
	for _, t := range fullTools {
		if IsDeferredToolName(t.Name) {
			deferredNames[t.Name] = struct{}{}
		}
	}
	if len(deferredNames) == 0 && !cfg.HasPendingMcpServers {
		return stripToolSearchOnly(fullTools)
	}

	discovered := ExtractDiscoveredToolNames(msgs)

	out := make([]anthropic.ToolDefinition, 0, len(fullTools))
	for _, t := range fullTools {
		name := t.Name
		if _, def := deferredNames[name]; !def {
			out = append(out, cloneTool(t))
			continue
		}
		if name == ToolSearchToolName {
			out = append(out, cloneTool(t))
			continue
		}
		if _, ok := discovered[name]; ok {
			u := cloneTool(t)
			v := true
			u.DeferLoading = &v
			out = append(out, u)
		}
	}
	return out
}

func stripToolSearchOnly(tools []anthropic.ToolDefinition) []anthropic.ToolDefinition {
	out := make([]anthropic.ToolDefinition, 0, len(tools))
	for _, t := range tools {
		if t.Name == ToolSearchToolName {
			continue
		}
		out = append(out, cloneTool(t))
	}
	return out
}

func cloneTool(t anthropic.ToolDefinition) anthropic.ToolDefinition {
	u := t
	u.DeferLoading = nil
	return u
}

// BetasForWiredTools returns anthropic-beta header values when the wired tools[] still includes ToolSearch
// (dynamic loading path). Mirrors the presence-based beta gate in src/services/api/claude.ts.
func BetasForWiredTools(tools []anthropic.ToolDefinition) []string {
	if envTruthy("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS") {
		return nil
	}
	for _, t := range tools {
		if t.Name == ToolSearchToolName {
			return []string{AnthropicBetaHeader()}
		}
	}
	return nil
}
