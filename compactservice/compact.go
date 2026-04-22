package compactservice

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"goc/types"
)

// CompactMaxOutputTokens mirrors COMPACT_MAX_OUTPUT_TOKENS in compact.ts (TS: 20_000).
// Auto-compact reserves the same amount off the model's context window.
const CompactMaxOutputTokens = 20_000

// CompactSystemPrompt mirrors the single-line summarizer system prompt used by
// streamCompactSummary's fallback path in TS.
var CompactSystemPrompt = []string{"You are a helpful AI assistant tasked with summarizing conversations."}

// RecompactionInfo mirrors RecompactionInfo in compact.ts.
type RecompactionInfo struct {
	IsRecompactionInChain     bool
	TurnsSincePreviousCompact int
	PreviousCompactTurnID     string
	AutoCompactThreshold      int
	QuerySource               string
}

// CompactOptions groups the caller-visible knobs to CompactConversation.
// Mirrors the TS function signature's positional args.
type CompactOptions struct {
	// SuppressFollowUpQuestions mirrors the 4th TS arg — true for auto-compact, false for /compact.
	SuppressFollowUpQuestions bool
	// CustomInstructions mirrors the 5th TS arg (user-provided via /compact).
	CustomInstructions string
	// IsAutoCompact mirrors the 6th TS arg.
	IsAutoCompact bool
	// RecompactionInfo mirrors the 7th TS arg; optional.
	RecompactionInfo *RecompactionInfo
	// Model is mainLoopModel — threaded to SummaryStreamInput.Model and SessionStartHookInput.Model.
	Model string
	// AgentID mirrors context.agentId (plan/skill scope).
	AgentID string
	// ToolUseContext is optional runtime context passed to PostCompactAttachmentProvider
	// for richer attachment re-injection (tool definitions, agent definitions, MCP clients).
	ToolUseContext *types.ToolUseContext
	// QuerySource is optional; after a successful compact, main-thread-like sources trigger
	// querycontext.ClearUserContextCache (TS runPostCompactCleanup + getUserContext.cache.clear).
	// Empty uses the same main-thread behavior as TS querySource === undefined.
	QuerySource string
}

// CompactConversation mirrors compactConversation in services/compact/compact.ts.
//
// Behavioral parity:
//   - Zero-input check → ErrNotEnoughMessages (TS ERROR_MESSAGE_NOT_ENOUGH_MESSAGES).
//   - executePreCompactHooks → merge custom instructions → build compact prompt.
//   - streamCompactSummary with a retry loop for prompt-too-long (truncateHeadForPTLRetry,
//     MAX_PTL_RETRIES times). Emits tengu_compact_ptl_retry and tengu_compact_failed
//     telemetry at matching points.
//   - After the summary succeeds: createPostCompactFileAttachments + plan/plan_mode/skill
//     attachments + agent listing + MCP + deferred tools (delegated to Deps.PostCompactAttachments).
//   - processSessionStartHooks('compact', …) → hookMessages.
//   - createCompactBoundaryMessage(trigger, preCompactTokenCount, lastPreCompactUuid).
//   - getCompactUserSummaryMessage(…) wrapped in a user row with isCompactSummary+isVisibleInTranscriptOnly.
//   - tengu_compact event emission.
//   - executePostCompactHooks(trigger, compactSummary) → combine userDisplayMessages with a newline.
//   - Returns CompactionResult in full TS shape.
//
// Intentional simplifications (clearly TODO'd vs TS):
//   - runForkedAgent / cache-sharing is NOT implemented; SummarizerFn is the only path.
//   - markPostCompaction + reAppendSessionMetadata + prompt-cache-break notifications remain
//     host-side; on success, hosts may set Deps.AfterSuccessfulCompact (default adapter clears
//     user-context memo and resets getMemoryFiles with load_reason=compact, TS runPostCompactCleanup).
//   - analyzeContext-driven stats in the tengu_compact event are elided (logger gets 0s).
func CompactConversation(
	ctx context.Context,
	messages []types.Message,
	deps Deps,
	opts CompactOptions,
) (CompactionResult, error) {
	deps.resolve()

	if len(messages) == 0 {
		return CompactionResult{}, ErrNotEnoughMessages
	}

	preCompactTokenCount := TokenCountWithEstimation(messages)

	// --- Pre-compact hooks ---
	trigger := CompactTriggerManual
	if opts.IsAutoCompact {
		trigger = CompactTriggerAuto
	}
	preHook, err := deps.PreCompactHooks(ctx, PreCompactHookInput{
		Trigger:            trigger,
		CustomInstructions: opts.CustomInstructions,
	})
	if err != nil {
		return CompactionResult{}, fmt.Errorf("pre-compact hooks: %w", err)
	}
	// TS blockingError: hooks can abort compaction without error
	if preHook.Blocked {
		return CompactionResult{}, nil
	}
	customInstructions := MergeHookInstructions(opts.CustomInstructions, preHook.NewCustomInstructions)
	preHookUserDisplay := preHook.UserDisplayMessage

	// --- Build compact prompt ---
	compactPrompt := GetCompactPrompt(customInstructions)
	summaryRequest, err := newUserPromptMessage(compactPrompt, deps)
	if err != nil {
		return CompactionResult{}, fmt.Errorf("build summary request: %w", err)
	}

	// --- streamCompactSummary with PTL retry ---
	if deps.Summarize == nil {
		return CompactionResult{}, fmt.Errorf("compactservice: Deps.Summarize is required")
	}

	// Pre-strip: stripImagesFromMessages + stripReinjectedAttachments.
	// EXPERIMENTAL_SKILL_SEARCH gate mirrors TS behavior.
	summarizeInput := StripImagesFromMessages(StripReinjectedAttachments(
		append([]types.Message{}, GetMessagesAfterCompactBoundary(messages)...),
		IsExperimentalSkillSearchEnabled(),
	))

	messagesToSummarize := summarizeInput
	var summaryResponse types.Message
	var summaryText string
	ptlAttempts := 0
	for {
		if contextIsCanceled(ctx) {
			return CompactionResult{}, ErrUserAbort
		}
		res, err := deps.Summarize(ctx, SummaryStreamInput{
			Messages:             messagesToSummarize,
			SummaryRequest:       summaryRequest,
			Model:                opts.Model,
			SystemPrompt:         CompactSystemPrompt,
			MaxOutputTokens:      CompactMaxOutputTokens,
			PreCompactTokenCount: preCompactTokenCount,
		})
		if err != nil {
			deps.Logger.LogCompactFailed(CompactFailedEvent{
				Reason:               "summarize_error",
				PreCompactTokenCount: preCompactTokenCount,
				PtlAttempts:          ptlAttempts,
			})
			return CompactionResult{}, err
		}
		summaryResponse = res.AssistantMessage
		summaryText = getAssistantMessageText(summaryResponse)
		if summaryText == "" || !startsWithPromptTooLong(summaryText) {
			break
		}
		ptlAttempts++
		var truncated []types.Message
		var ok bool
		if ptlAttempts <= MaxPTLRetries {
			truncated, ok = TruncateHeadForPTLRetry(messagesToSummarize, summaryResponse)
		}
		if !ok {
			deps.Logger.LogCompactFailed(CompactFailedEvent{
				Reason:               "prompt_too_long",
				PreCompactTokenCount: preCompactTokenCount,
				PtlAttempts:          ptlAttempts,
			})
			return CompactionResult{}, ErrPromptTooLongMessage
		}
		deps.Logger.LogCompactPTLRetry(CompactPTLRetryEvent{
			Attempt:           ptlAttempts,
			DroppedMessages:   len(messagesToSummarize) - len(truncated),
			RemainingMessages: len(truncated),
		})
		messagesToSummarize = truncated
	}

	if summaryText == "" {
		deps.Logger.LogCompactFailed(CompactFailedEvent{
			Reason:               "no_summary",
			PreCompactTokenCount: preCompactTokenCount,
		})
		return CompactionResult{}, fmt.Errorf("compactservice: summarizer returned no text: %s", string(summaryResponse.Message))
	}
	if StartsWithApiErrorPrefix(summaryText) {
		deps.Logger.LogCompactFailed(CompactFailedEvent{
			Reason:               "api_error",
			PreCompactTokenCount: preCompactTokenCount,
		})
		return CompactionResult{}, fmt.Errorf("%s", summaryText)
	}

	// --- Post-compact attachments (stubbed default) ---
	beforeMessages := make([]any, 0, len(messages))
	for _, m := range messages {
		beforeMessages = append(beforeMessages, m)
	}
	postAttachments, err := deps.PostCompactAttachments(ctx, PostCompactAttachmentInput{
		Model:                    opts.Model,
		AgentID:                  opts.AgentID,
		MessagesBeforeCompaction: beforeMessages,
		MessagesToKeep:           nil, // full compact: no preserved tail
		ToolUseContext:           opts.ToolUseContext,
	})
	if err != nil {
		return CompactionResult{}, fmt.Errorf("post-compact attachments: %w", err)
	}

	// --- SessionStart hooks (trigger='compact') ---
	hookMessages, err := deps.SessionStartHooks(ctx, SessionStartTriggerCompact, SessionStartHookInput{Model: opts.Model})
	if err != nil {
		return CompactionResult{}, fmt.Errorf("session-start hooks: %w", err)
	}

	// --- Boundary marker ---
	var lastUUID string
	if n := len(messages); n > 0 {
		lastUUID = messages[n-1].UUID
	}
	boundary, err := CreateCompactBoundaryMessage(trigger, preCompactTokenCount, lastUUID, "", nil)
	if err != nil {
		return CompactionResult{}, err
	}

	// --- Summary user message ---
	continuationText := GetCompactUserSummaryMessage(summaryText, CompactUserSummaryOpts{
		SuppressFollowUpQuestions: opts.SuppressFollowUpQuestions,
		TranscriptPath:            deps.TranscriptPath,
		ProactiveActive:           deps.ProactiveActive,
	})
	summaryUserMsg, err := newCompactSummaryUserMessage(continuationText, deps)
	if err != nil {
		return CompactionResult{}, err
	}

	// --- Telemetry ---
	compactionCallTotal := TokenCountFromLastAPIResponse([]types.Message{summaryResponse})
	estimatePool := append([]types.Message{boundary, summaryUserMsg}, postAttachments...)
	estimatePool = append(estimatePool, hookMessages...)
	truePost := RoughTokenCountEstimationForMessages(estimatePool)
	compactionUsage := GetTokenUsage(summaryResponse)

	event := CompactEvent{
		PreCompactTokenCount:      preCompactTokenCount,
		PostCompactTokenCount:     compactionCallTotal,
		TruePostCompactTokenCount: truePost,
		AutoCompactThreshold:      -1,
		IsAutoCompact:             opts.IsAutoCompact,
		IsRecompactionInChain:     false,
		TurnsSincePreviousCompact: -1,
		QueryDepth:                -1,
	}
	if opts.RecompactionInfo != nil {
		event.AutoCompactThreshold = opts.RecompactionInfo.AutoCompactThreshold
		event.WillRetriggerNextTurn = opts.RecompactionInfo.AutoCompactThreshold > 0 &&
			truePost >= opts.RecompactionInfo.AutoCompactThreshold
		event.IsRecompactionInChain = opts.RecompactionInfo.IsRecompactionInChain
		event.TurnsSincePreviousCompact = opts.RecompactionInfo.TurnsSincePreviousCompact
		event.PreviousCompactTurnID = opts.RecompactionInfo.PreviousCompactTurnID
		event.QuerySource = opts.RecompactionInfo.QuerySource
	}
	if compactionUsage != nil {
		event.CompactionInputTokens = compactionUsage.InputTokens
		event.CompactionOutputTokens = compactionUsage.OutputTokens
		event.CompactionCacheReadTokens = compactionUsage.CacheReadInputTokens
		event.CompactionCacheCreationTokens = compactionUsage.CacheCreationInputTokens
		event.CompactionTotalTokens = compactionUsage.InputTokens + compactionUsage.CacheCreationInputTokens + compactionUsage.CacheReadInputTokens + compactionUsage.OutputTokens
	}
	deps.Logger.LogCompact(event)

	// --- PostCompact hooks ---
	postHook, err := deps.PostCompactHooks(ctx, PostCompactHookInput{
		Trigger:        trigger,
		CompactSummary: summaryText,
	})
	if err != nil {
		return CompactionResult{}, fmt.Errorf("post-compact hooks: %w", err)
	}

	combinedDisplay := preHookUserDisplay
	if postHook.UserDisplayMessage != "" {
		if combinedDisplay != "" {
			combinedDisplay += "\n" + postHook.UserDisplayMessage
		} else {
			combinedDisplay = postHook.UserDisplayMessage
		}
	}

	deps.AfterSuccessfulCompact(compactQuerySourceForCleanup(opts))

	return CompactionResult{
		BoundaryMarker:            boundary,
		SummaryMessages:           []types.Message{summaryUserMsg},
		Attachments:               postAttachments,
		HookResults:               hookMessages,
		UserDisplayMessage:        combinedDisplay,
		PreCompactTokenCount:      preCompactTokenCount,
		PostCompactTokenCount:     compactionCallTotal,
		TruePostCompactTokenCount: truePost,
		CompactionUsage:           compactionUsage,
	}, nil
}

func compactQuerySourceForCleanup(opts CompactOptions) string {
	if s := strings.TrimSpace(opts.QuerySource); s != "" {
		return opts.QuerySource
	}
	if opts.RecompactionInfo != nil {
		if s := strings.TrimSpace(opts.RecompactionInfo.QuerySource); s != "" {
			return opts.RecompactionInfo.QuerySource
		}
	}
	return ""
}

// newUserPromptMessage builds a user Message {role:user, content:prompt}. Mirrors
// createUserMessage in TS for plain-text user rows.
func newUserPromptMessage(prompt string, deps Deps) (types.Message, error) {
	inner := map[string]any{"role": "user", "content": prompt}
	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return types.Message{}, err
	}
	return types.Message{
		Type:    types.MessageTypeUser,
		UUID:    deps.NewUUID(),
		Message: json.RawMessage(innerJSON),
	}, nil
}

// newCompactSummaryUserMessage mirrors the summary user row in TS:
//
//	createUserMessage({ content: …, isCompactSummary: true, isVisibleInTranscriptOnly: true })
func newCompactSummaryUserMessage(content string, deps Deps) (types.Message, error) {
	inner := map[string]any{"role": "user", "content": content}
	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return types.Message{}, err
	}
	trueP := true
	return types.Message{
		Type:                      types.MessageTypeUser,
		UUID:                      deps.NewUUID(),
		Message:                   json.RawMessage(innerJSON),
		IsCompactSummary:          &trueP,
		IsVisibleInTranscriptOnly: &trueP,
	}, nil
}

// getAssistantMessageText mirrors getAssistantMessageText — joins all text blocks in the
// assistant's inner message.content with no separator. Strings pass through untouched.
func getAssistantMessageText(m types.Message) string {
	if m.Type != types.MessageTypeAssistant || len(m.Message) == 0 {
		return ""
	}
	var probe struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(m.Message, &probe); err != nil {
		return ""
	}
	if len(probe.Content) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(probe.Content, &s); err == nil {
		return s
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(probe.Content, &arr); err != nil {
		return ""
	}
	out := ""
	for _, blk := range arr {
		var b struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(blk, &b); err != nil {
			continue
		}
		if b.Type == "text" {
			out += b.Text
		}
	}
	return out
}

// startsWithPromptTooLong is a thin alias for the PTL prefix check used in the retry loop.
func startsWithPromptTooLong(s string) bool {
	if len(s) < len(PromptTooLongErrorMessage) {
		return false
	}
	return s[:len(PromptTooLongErrorMessage)] == PromptTooLongErrorMessage
}
