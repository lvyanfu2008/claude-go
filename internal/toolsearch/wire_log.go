package toolsearch

import (
	"strings"

	"goc/ccb-engine/diaglog"
	"goc/internal/anthropic"
	"goc/tstenv"
)

func wireDiagEnabled() bool {
	if envTruthy("CLAUDE_CODE_GO_TOOL_SEARCH_DIAG") {
		return true
	}
	if envTruthy("CLAUDE_CODE_LOG_API_REQUEST_BODY") {
		return true
	}
	if envTruthy("GOU_DEMO_LOG") {
		return true
	}
	return false
}

// LogWireRound appends one diagnostic line for the tools[] actually sent on this API round
// (after ApplyWire). Gated by CLAUDE_CODE_GO_TOOL_SEARCH_DIAG, CLAUDE_CODE_LOG_API_REQUEST_BODY,
// or GOU_DEMO_LOG. Uses [diaglog.Line] (debug log file).
func LogWireRound(round int, resolvedModel string, msgs []anthropic.Message, cfg WireConfig, openAICompat bool, fullTools, wired []anthropic.ToolDefinition) {
	if !wireDiagEnabled() {
		return
	}
	names := make([]string, 0, len(wired))
	for _, t := range wired {
		n := strings.TrimSpace(t.Name)
		if n == "" {
			n = "(unnamed)"
		}
		if t.DeferLoading != nil && *t.DeferLoading {
			n += "(defer_loading)"
		}
		names = append(names, n)
	}
	discovered := ExtractDiscoveredToolNames(msgs)
	discN := len(discovered)
	reason := wireReason(openAICompat, cfg, fullTools)
	diaglog.Line("[ccb-engine toolsearch-wire] round=%d model=%q wire_reason=%q openai_compat=%v ENABLE_TOOL_SEARCH=%q tstenv_mode=%q wire_mode_effective=%q CLAUDE_CODE_GO_TOOL_SEARCH=%q CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=%q use_dynamic=%v model_toolref=%v prepend_deferred_block=%v pending_mcp=%v full_tools=%d wired_tools=%d discovered_names=%d wired_names=[%s]",
		round,
		resolvedModel,
		reason,
		openAICompat,
		envTrim("ENABLE_TOOL_SEARCH"),
		tstenv.GetToolSearchMode(),
		effectiveToolSearchModeForWire(),
		envTrim("CLAUDE_CODE_GO_TOOL_SEARCH"),
		envTrim("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"),
		cfg.UseDynamicToolLoading,
		cfg.ModelSupportsToolReference,
		cfg.PrependAvailableDeferredBlock,
		cfg.HasPendingMcpServers,
		len(fullTools),
		len(wired),
		discN,
		strings.Join(names, ", "),
	)
}

func wireReason(openAICompat bool, cfg WireConfig, fullTools []anthropic.ToolDefinition) string {
	if openAICompat {
		if cfg.UseDynamicToolLoading {
			return "openai_compat_client_side_toolsearch_protocol"
		}
		return "openai_compat_dynamic_off_strip_toolsearch_only"
	}
	if !cfg.ModelSupportsToolReference {
		return "model_no_tool_reference"
	}
	if !isToolSearchToolAvailable(fullTools) {
		return "no_toolsearch_in_registry"
	}
	if !cfg.UseDynamicToolLoading {
		return "dynamic_loading_off_tst_standard_or_tst_auto_below_threshold_or_kill_switch"
	}
	return "anthropic_dynamic_defer"
}
