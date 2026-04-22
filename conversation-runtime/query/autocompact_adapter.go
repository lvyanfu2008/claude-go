package query

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"goc/ccb-engine/settingsfile"
	"goc/claudemd"
	"goc/compactquerysource"
	"goc/compactservice"
	"goc/hookexec"
	"goc/querycontext"
	"goc/types"
)

// newCompactAdapter returns a [QueryDeps.Autocompact] function that wires into
// compactservice.AutoCompactIfNeeded. Mirrors the TS productionDeps autocompact slot
// (queryPipeline/deps.ts) where autoCompactIfNeeded is plugged in directly.
//
// The returned function:
//  1. Materializes [compactservice.AutoCompactTrackingState] from the host's
//     [AutocompactInput.Tracking] raw blob (which [queryLoop] round-trips).
//  2. Calls AutoCompactIfNeeded with a Summarizer that mirrors TS [queryModel] provider
//     routing (Gemma → OpenAI chat/completions → Anthropic /v1/messages) for one text-only summary round.
//  3. Serializes [CompactionResult] + tracking back into the [AutocompactResult] wire shape.
//
// Hooks + attachments fall through to the no-op defaults. Hosts that want real
// pre/post-compact hooks + post-compact attachment regeneration supply their
// own adapter that layers a CompactDepsBuilder over this.
func newCompactAdapter() func(ctx context.Context, in *AutocompactInput) (*AutocompactResult, error) {
	return func(ctx context.Context, in *AutocompactInput) (*AutocompactResult, error) {
		if in == nil {
			return nil, nil
		}

		tracking := decodeAutoCompactTracking(in.Tracking)

		model := modelFromToolUseContext(in.ToolUseContext)

		wd, errWd := os.Getwd()
		if errWd != nil {
			wd = "."
		}
		projRoot, errRoot := settingsfile.FindClaudeProjectRoot(wd)
		if errRoot != nil || strings.TrimSpace(projRoot) == "" {
			projRoot = wd
		}
		sid := ""
		if in.ToolUseContext != nil && in.ToolUseContext.ConversationID != nil {
			sid = strings.TrimSpace(*in.ToolUseContext.ConversationID)
		}

		deps := compactservice.Deps{
			Summarize:              defaultSummarizer(model),
			PostCompactAttachments: defaultPostCompactAttachments(),
			PreCompactHooks:        hookexec.PreCompactHookRunner(projRoot, wd, sid, ""),
			PostCompactHooks:       hookexec.PostCompactHookRunner(projRoot, wd, sid, ""),
			SessionStartHooks:      hookexec.SessionStartHookRunner(projRoot, wd, sid, ""),
			AfterSuccessfulCompact: func(qs string) {
				if compactquerysource.MainThreadLike(qs) {
					querycontext.ClearUserContextCache()
					claudemd.ResetMemoryFilesCache("compact")
				}
			},
		}

		snip := 0
		if in.SnipTokensFreed != nil {
			snip = *in.SnipTokensFreed
		}

		res, err := compactservice.AutoCompactIfNeeded(ctx, compactservice.AutoCompactIfNeededInput{
			Messages:        in.Messages,
			Model:           model,
			AgentID:         "",
			QuerySource:     string(in.QuerySource),
			Tracking:        tracking,
			SnipTokensFreed: snip,
			Thresholds:      compactservice.CompactThresholds{},
			Deps:            deps,
			ToolUseContext:  in.ToolUseContext,
		})
		if err != nil {
			// Mirror TS: on failure increment consecutiveFailures, propagate nothing else.
			out := &AutocompactResult{WasCompacted: false}
			if res.ConsecutiveFailures != nil {
				out.ConsecutiveFailures = *res.ConsecutiveFailures
				out.UpdatedTracking = encodeUpdatedTracking(tracking, res)
			}
			return out, nil
		}
		if !res.WasCompacted || res.CompactionResult == nil {
			return &AutocompactResult{WasCompacted: false}, nil
		}

		post := compactservice.BuildPostCompactMessages(*res.CompactionResult)
		blob, _ := json.Marshal(res.CompactionResult)

		return &AutocompactResult{
			WasCompacted:     true,
			PostMessages:     post,
			CompactionResult: blob,
			UpdatedTracking:  encodeUpdatedTracking(tracking, res),
		}, nil
	}
}

// decodeAutoCompactTracking round-trips the raw tracking blob threaded by queryLoop
// (State.AutoCompactTracking is json.RawMessage). Missing fields fall back to zero.
func decodeAutoCompactTracking(raw json.RawMessage) *compactservice.AutoCompactTrackingState {
	if len(raw) == 0 {
		return nil
	}
	var out compactservice.AutoCompactTrackingState
	var aliased struct {
		Compacted           bool   `json:"compacted"`
		TurnCounter         int    `json:"turnCounter"`
		TurnID              string `json:"turnId"`
		ConsecutiveFailures int    `json:"consecutiveFailures"`
	}
	if err := json.Unmarshal(raw, &aliased); err != nil {
		return nil
	}
	out.Compacted = aliased.Compacted
	out.TurnCounter = aliased.TurnCounter
	out.TurnID = aliased.TurnID
	out.ConsecutiveFailures = aliased.ConsecutiveFailures
	return &out
}

// encodeUpdatedTracking returns the JSON-encoded next tracking state to thread back
// into [State.AutoCompactTracking]. Mirrors the TS path in query.ts that seeds
// tracking.compacted=true + tracking.turnId=newUuid on success.
func encodeUpdatedTracking(prev *compactservice.AutoCompactTrackingState, res compactservice.AutoCompactIfNeededResult) json.RawMessage {
	next := compactservice.AutoCompactTrackingState{}
	if prev != nil {
		next = *prev
	}
	if res.WasCompacted {
		next.Compacted = true
		next.ConsecutiveFailures = 0
	} else if res.ConsecutiveFailures != nil {
		next.ConsecutiveFailures = *res.ConsecutiveFailures
	}
	out := map[string]any{
		"compacted":           next.Compacted,
		"turnCounter":         next.TurnCounter,
		"turnId":              next.TurnID,
		"consecutiveFailures": next.ConsecutiveFailures,
	}
	raw, _ := json.Marshal(out)
	return raw
}

// modelFromToolUseContext plucks mainLoopModel from the context. Falls back to env default.
func modelFromToolUseContext(tcx *types.ToolUseContext) string {
	if tcx != nil {
		if m := strings.TrimSpace(tcx.Options.MainLoopModel); m != "" {
			return m
		}
	}
	if m := strings.TrimSpace(os.Getenv("CLAUDE_MODEL")); m != "" {
		return m
	}
	return ""
}

// defaultSummarizer returns a [compactservice.SummarizerFn] that mirrors TS compact's
// [queryModelWithStreaming] provider selection (see [summarizeAutocompact]).
func defaultSummarizer(_ string) compactservice.SummarizerFn {
	return func(ctx context.Context, in compactservice.SummaryStreamInput) (compactservice.SummaryStreamResult, error) {
		return summarizeAutocompact(ctx, in)
	}
}

// wireShapeFromMessages converts []types.Message into the API-wire shape used by
// the POST body (role/content pairs extracted from the inner message envelope).
// Mirrors normalizeMessagesForAPI's output — we do a lightweight projection so the
// default summarizer can run without dragging the full message-normalization pipeline.
func wireShapeFromMessages(messages []types.Message) ([]any, error) {
	out := make([]any, 0, len(messages))
	for _, m := range messages {
		if m.Type != types.MessageTypeUser && m.Type != types.MessageTypeAssistant {
			// Skip system/attachment/progress — they don't go on the wire.
			continue
		}
		if len(m.Message) == 0 {
			continue
		}
		var envelope map[string]any
		if err := json.Unmarshal(m.Message, &envelope); err != nil {
			return nil, err
		}
		out = append(out, envelope)
	}
	return out, nil
}

// defaultPostCompactAttachments returns a PostCompactAttachmentProvider that
// extracts information from the pre-compact messages and generates attachment
// messages for post-compact re-injection. This is a fallback implementation
// that works without access to runtime state (ReadFileState, skill store, etc.).
//
// What it supports:
//   - invoked_skills: extracts from prior invoked_skills attachments in messages
//
// What it does NOT support (requires host-side state):
//   - file attachments (would need ReadFileState + actual file re-read)
//   - plan / plan_mode (would need plan store)
//   - deferred_tools_delta (would need live tool pool)
//   - agent_listing_delta (would need live agent definitions)
//   - mcp_instructions_delta (would need live MCP clients)
//
// Hosts with richer state should provide their own PostCompactAttachmentProvider.
func defaultPostCompactAttachments() compactservice.PostCompactAttachmentProvider {
	return func(_ context.Context, in compactservice.PostCompactAttachmentInput) ([]compactservice.HookResultMessage, error) {
		// Convert []any to []types.Message for extraction
		messages := make([]types.Message, 0, len(in.MessagesBeforeCompaction))
		for _, m := range in.MessagesBeforeCompaction {
			if tm, ok := m.(types.Message); ok {
				messages = append(messages, tm)
			}
		}

		var out []compactservice.HookResultMessage

		// Extract and re-inject invoked_skills from prior messages
		skills := compactservice.ExtractInvokedSkills(messages)
		if len(skills) > 0 {
			skillAtt := buildInvokedSkillsAttachment(skills)
			if skillAtt != nil {
				out = append(out, *skillAtt)
			}
		}

		// TODO: When host provides ReadFileState, add file re-injection here.
		// TODO: When host provides plan store, add plan/plan_mode attachments.
		// TODO: When host provides tool/agent/MCP definitions, add delta attachments.

		return out, nil
	}
}

// buildInvokedSkillsAttachment creates an attachment message for invoked_skills.
func buildInvokedSkillsAttachment(skills []compactservice.ExtractedSkill) *types.Message {
	if len(skills) == 0 {
		return nil
	}

	type skillEntry struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	skillsData := make([]skillEntry, len(skills))
	for i, s := range skills {
		skillsData[i] = skillEntry{Name: s.Name, Path: s.Path, Content: s.Content}
	}

	att := map[string]any{
		"type":   "invoked_skills",
		"skills": skillsData,
	}
	attJSON, err := json.Marshal(att)
	if err != nil {
		return nil
	}

	msg := types.Message{
		Type:       types.MessageTypeAttachment,
		UUID:       randomUUID(),
		Attachment: attJSON,
	}
	return &msg
}
