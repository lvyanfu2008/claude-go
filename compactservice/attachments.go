package compactservice

import "context"

// PostCompactAttachmentInput mirrors the inputs collected by the TS
// compactConversation right after clearing readFileState and before issuing
// createPostCompactFileAttachments + related calls.
type PostCompactAttachmentInput struct {
	// Model is mainLoopModel — passed through to per-provider routines (e.g. get tools delta).
	Model string
	// AgentID mirrors context.agentId in TS (plan / skill attachment scope).
	AgentID string
	// MessagesBeforeCompaction is the full pre-compact conversation (for discovering
	// previously-read files, invoked skills, deferred tools, agents, MCP tools).
	MessagesBeforeCompaction []any // typed as any because callers use domain-specific shapes; see host wiring
}

// PostCompactAttachmentProvider is the injection point that collectively produces the
// attachment messages TS appends after the summary. Returning an empty slice is the
// documented no-op default; hosts wire concrete providers (file re-read, plan, plan_mode,
// skills, agent listing, MCP, deferred tools) as those subsystems land Go parity.
type PostCompactAttachmentProvider func(ctx context.Context, in PostCompactAttachmentInput) ([]HookResultMessage, error)

// NoopPostCompactAttachmentProvider is the safe default.
func NoopPostCompactAttachmentProvider(_ context.Context, _ PostCompactAttachmentInput) ([]HookResultMessage, error) {
	return nil, nil
}
