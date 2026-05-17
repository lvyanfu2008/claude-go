package tools

import (
	"context"
	"encoding/json"

	"goc/types"
)

// HandoffClassifierResult is the result of reviewing a sub-agent's transcript
// before handing control back to the parent agent.
// Mirrors TS classifyYoloAction return in src/tools/AgentTool/agentToolUtils.ts.
type HandoffClassifierResult struct {
	ShouldBlock bool
	Reason      string
	Unavailable bool
}

// HandoffParams groups the inputs needed for [ClassifyHandoffIfNeeded].
type HandoffParams struct {
	AgentMessages       []types.Message
	Tools               json.RawMessage
	ToolPermissionContext *types.ToolPermissionContextData
	SubagentType        string
	TotalToolUseCount   int
	PermissionMode      string
}

// TRANSCRIPT_CLASSIFIER feature flag. When true, sub-agent transcripts are
// reviewed before handoff to the parent agent.
// Mirrors TS TRANSCRIPT_CLASSIFIER in GrowthBook feature flags.
func transcriptClassifierEnabled() bool {
	return envTruthy("CLAUDE_CODE_TRANSCRIPT_CLASSIFIER")
}

// ClassifyHandoffIfNeeded reviews a sub-agent's transcript to decide whether
// its output is safe to hand back to the parent agent.
//
// Mirrors TS classifyHandoffIfNeeded in src/tools/AgentTool/agentToolUtils.ts.
//
// When the TRANSCRIPT_CLASSIFIER feature flag is disabled or permission mode
// is not "auto", returns nil immediately (no block).
//
// Returns nil to allow handoff, or a HandoffClassifierResult with ShouldBlock=true
// (and a SECURITY WARNING reason) to block.
func ClassifyHandoffIfNeeded(ctx context.Context, params HandoffParams) (*HandoffClassifierResult, error) {
	if !transcriptClassifierEnabled() {
		return nil, nil
	}
	if params.PermissionMode != "auto" {
		return nil, nil
	}
	// stub: classifier model call not yet implemented
	// TODO: wire classifyYoloAction equivalent when LLM call infrastructure is ready
	_ = ctx
	_ = params.AgentMessages
	_ = params.Tools
	_ = params.ToolPermissionContext
	_ = params.SubagentType
	_ = params.TotalToolUseCount
	return nil, nil
}
