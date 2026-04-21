package toolsearch

import (
	"encoding/json"
	"strconv"
	"strings"

	"goc/ccb-engine/internal/anthropic"
	"goc/commands/featuregates"
	"goc/tstenv"
)

// WireConfig controls API tools[] shaping and companion message transforms for one engine turn.
type WireConfig struct {
	UseDynamicToolLoading      bool
	ModelSupportsToolReference bool
	HasPendingMcpServers       bool
	OpenAICompat               bool
	// PrependAvailableDeferredBlock mirrors claude.ts: useToolSearch && !isDeferredToolsDeltaEnabled().
	PrependAvailableDeferredBlock bool
}

// deferredToolsDeltaEnabled mirrors isDeferredToolsDeltaEnabled (src/utils/toolSearch.ts).
func deferredToolsDeltaEnabled() bool {
	if featuregates.UserTypeAnt() {
		return true
	}
	return envTruthy("CLAUDE_CODE_GO_DEFERRED_TOOLS_DELTA")
}

func isToolSearchToolAvailable(tools []anthropic.ToolDefinition) bool {
	if len(tools) == 0 {
		return false
	}
	for i := range tools {
		if tools[i].Name == ToolSearchToolName {
			return true
		}
	}
	return false
}

func deferredToolDescriptionChars(tools []anthropic.ToolDefinition) int {
	total := 0
	for _, t := range tools {
		if !IsDeferredToolName(t.Name) {
			continue
		}
		schema, _ := json.Marshal(t.InputSchema)
		total += len(t.Name) + len(t.Description) + len(schema)
	}
	return total
}

func contextWindowTokens(modelID string) int {
	m := strings.ToLower(strings.TrimSpace(modelID))
	if strings.Contains(m, "1m") || strings.Contains(m, "[1m]") {
		return 1_000_000
	}
	// Subset of src/utils/context.ts; unknown models default like TS generic window.
	if strings.Contains(m, "opus-4") || strings.Contains(m, "sonnet-4") || strings.Contains(m, "haiku-4") {
		return 100_000
	}
	return 100_000
}

func autoCharThreshold(modelID string) int {
	pct := float64(tstenv.AutoToolSearchPercentage()) / 100
	tw := contextWindowTokens(modelID)
	tokenThreshold := int(float64(tw) * pct)
	return int(float64(tokenThreshold) * 2.5)
}

// tstAutoDeferredCharScale scales measured deferred description size before comparing to
// the tst-auto char threshold. TS calculateDeferredToolDescriptionChars uses live
// tool.prompt() text, which is usually larger than static tools_api.json descriptions;
// without scaling, ENABLE_TOOL_SEARCH=auto often stays below threshold in Go while TS enables defer.
func tstAutoDeferredCharScale() float64 {
	if s := envTrim("CLAUDE_CODE_GO_TST_AUTO_CHAR_SCALE"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			return v
		}
	}
	return 1.65
}

func checkAutoThresholdSync(modelID string, tools []anthropic.ToolDefinition) bool {
	raw := float64(deferredToolDescriptionChars(tools))
	ch := int(raw * tstAutoDeferredCharScale())
	th := autoCharThreshold(modelID)
	return ch >= th
}

// BuildWireConfig mirrors isToolSearchEnabled + claude.ts gates (model, ToolSearch presence, tst-auto threshold, empty deferred).
func BuildWireConfig(modelID string, tools []anthropic.ToolDefinition, hasPendingMcpServers bool, openAICompat bool) WireConfig {
	cfg := WireConfig{
		ModelSupportsToolReference: modelSupportsToolReference(modelID),
		HasPendingMcpServers:       hasPendingMcpServers,
		OpenAICompat:               openAICompat,
	}
	if envTruthy("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS") {
		return cfg
	}
	if openAICompat {
		// Chat/completions has no Anthropic defer_loading/tool_reference on wire; still shrink tools[]
		// the same way and discover deferred tools via local ToolSearch JSON (see exec_toolsearch.go).
		cfg.ModelSupportsToolReference = true
	}
	if s := envTrim("CLAUDE_CODE_GO_TOOL_SEARCH"); s != "" && envDefinedFalsy("CLAUDE_CODE_GO_TOOL_SEARCH") {
		return cfg
	}
	if envTruthy("CLAUDE_CODE_GO_TOOL_SEARCH") {
		cfg.applyDynamicDecision(true, tools, modelID)
		return cfg
	}
	if !cfg.ModelSupportsToolReference {
		return cfg
	}
	if !isToolSearchToolAvailable(tools) {
		return cfg
	}
	mode := effectiveToolSearchModeForWire()
	switch mode {
	case "standard":
		return cfg
	case "tst":
		cfg.applyDynamicDecision(true, tools, modelID)
		return cfg
	case "tst-auto":
		cfg.applyDynamicDecision(checkAutoThresholdSync(modelID, tools), tools, modelID)
		return cfg
	default:
		cfg.applyDynamicDecision(true, tools, modelID)
		return cfg
	}
}

func (cfg *WireConfig) applyDynamicDecision(enabled bool, tools []anthropic.ToolDefinition, modelID string) {
	if !enabled {
		return
	}
	deferredCount := 0
	for _, t := range tools {
		if IsDeferredToolName(t.Name) {
			deferredCount++
		}
	}
	if deferredCount == 0 && !cfg.HasPendingMcpServers {
		return
	}
	cfg.UseDynamicToolLoading = true
	// Prepend <available-deferred-tools> unless delta-attachments mode (TS) or user disables message-side context (Go).
	cfg.PrependAvailableDeferredBlock = !deferredToolsDeltaEnabled() && toolSearchContextInMessagesEnabled()
	_ = modelID // reserved for future token-API parity
}

// effectiveToolSearchModeForWire maps tst-auto → tst when using embedded tools_api (gou-demo): char-only tst-auto
// can stay "below threshold" while TS product path still defers (token path / defaults). Matches parity reports for tools[].
func effectiveToolSearchModeForWire() string {
	m := tstenv.GetToolSearchMode()
	if m != "tst-auto" {
		return m
	}
	if envTruthy("GOU_DEMO_USE_EMBEDDED_TOOLS_API") {
		return "tst"
	}
	return m
}

// toolSearchContextInMessagesEnabled is false when CLAUDE_CODE_GO_TOOL_SEARCH_CONTEXT is explicitly falsy (e.g. "0"):
// still filter tools[] for the API, but omit the ephemeral <available-deferred-tools> user prepend (TS delta path uses attachments instead).
func toolSearchContextInMessagesEnabled() bool {
	if s := envTrim("CLAUDE_CODE_GO_TOOL_SEARCH_CONTEXT"); s != "" && envDefinedFalsy("CLAUDE_CODE_GO_TOOL_SEARCH_CONTEXT") {
		return false
	}
	return true
}
