package compactservice

import (
	"context"

	"goc/types"
)

// SummaryStreamInput mirrors the relevant inputs to streamCompactSummary in TS.
// We deliberately do NOT port runForkedAgent / cache-sharing here; the default
// streaming path is what ships, and hosts wire a fancier summarizer when needed.
type SummaryStreamInput struct {
	// Messages is the pre-compact conversation slice that should be sent to the
	// summarizer (after stripImagesFromMessages + stripReinjectedAttachments).
	Messages []types.Message
	// SummaryRequest is the synthetic user message carrying the compact prompt.
	SummaryRequest types.Message
	// Model is mainLoopModel — the summarizer runs on this model for cache parity.
	Model string
	// SystemPrompt is the minimal summarizer system prompt (TS hard-codes this).
	SystemPrompt []string
	// MaxOutputTokens is the TS COMPACT_MAX_OUTPUT_TOKENS min'd with the model cap.
	MaxOutputTokens int
	// PreCompactTokenCount is threaded for telemetry.
	PreCompactTokenCount int
}

// SummaryStreamResult mirrors { assistantMessage } — the resulting assistant message
// whose text() we parse for the summary / PTL error prefix.
type SummaryStreamResult struct {
	AssistantMessage types.Message
	Usage            *TokenUsage
}

// SummarizerFn is the injection point for streamCompactSummary.
// Hosts in production wire this to a direct POST to Anthropic/v1/messages with
// tools=[FileReadTool], thinking disabled, max_output_tokens capped per TS. Tests
// provide a synthetic implementation.
type SummarizerFn func(ctx context.Context, in SummaryStreamInput) (SummaryStreamResult, error)
