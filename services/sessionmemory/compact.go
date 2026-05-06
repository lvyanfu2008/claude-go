// Package sessionmemory provides session memory compaction — a zero-token-cost
// alternative to traditional API-based compaction that reads summary.md (maintained
// by the session memory write-side), truncates oversized sections, and injects it
// as a user-visible summary message.
//
// Mirrors src/services/compact/sessionMemoryCompact.ts.
package sessionmemory

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"goc/ccb-engine/diaglog"
	"goc/compactservice"
	"goc/growthbook"
	"goc/memdir"
	"goc/types"
)

// --- Gate check ----------------------------------------------------------

// shouldUseSessionMemoryCompaction checks whether session memory compaction
// should be used. Mirrors TS shouldUseSessionMemoryCompaction.
func shouldUseSessionMemoryCompaction() bool {
	if compactservice.IsEnvTruthy("ENABLE_CLAUDE_CODE_SM_COMPACT") {
		return true
	}
	if compactservice.IsEnvTruthy("DISABLE_CLAUDE_CODE_SM_COMPACT") {
		return false
	}

	sessionMemoryFlag := growthbook.IsTenguSessionMemory()
	smCompactFlag := growthbook.IsTenguSMCompact()
	shouldUse := sessionMemoryFlag && smCompactFlag

	// Log flag states for debugging (ant-only to avoid noise in external logs).
	// Mirrors TS: if (process.env.USER_TYPE === 'ant') { logEvent('tengu_sm_compact_flag_check', {...}) }
	if os.Getenv("USER_TYPE") == "ant" {
		diaglog.Line("[sessionmemory/sm_compact] flag_check: tengu_session_memory=%v tengu_sm_compact=%v should_use=%v",
			sessionMemoryFlag, smCompactFlag, shouldUse)
	}

	return shouldUse
}

// --- Extraction wait -----------------------------------------------------

// waitForSessionMemoryExtraction polls ExtractionStartedAt until the
// in-progress extraction completes or times out. Mirrors TS
// waitForSessionMemoryExtraction.
func waitForSessionMemoryExtraction(state *State) {
	deadline := nowMS() + extractionWaitTimeoutMs
	for {
		startedAt := func() int64 {
			state.mu.Lock()
			defer state.mu.Unlock()
			return state.ExtractionStartedAt
		}()
		if startedAt == 0 {
			return
		}
		age := nowMS() - startedAt
		if age > extractionStaleThresholdMs {
			return
		}
		if nowMS() > deadline {
			return
		}
		// Spin-wait ~1s
		select {
		case <-time.After(1 * time.Second):
		}
	}
}

// getSessionMemoryContent reads summary.md. Mirrors TS getSessionMemoryContent.
func getSessionMemoryContent(sessionID, cwd string) string {
	memoryPath := memdir.GetSessionMemoryPath(sessionID, cwd)
	data, err := os.ReadFile(memoryPath)
	if err != nil {
		return ""
	}
	return string(data)
}

// --- Message inspection helpers ------------------------------------------

// hasTextBlocks returns true if the message contains text content blocks.
// Mirrors TS hasTextBlocks.
func hasTextBlocks(m types.Message) bool {
	switch m.Type {
	case types.MessageTypeAssistant:
		return messageHasTextContentBlocks(m)
	case types.MessageTypeUser:
		return messageHasTextContentBlocks(m)
	}
	return false
}

func messageHasTextContentBlocks(m types.Message) bool {
	if len(m.Message) == 0 {
		return false
	}
	var payload struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(m.Message, &payload); err != nil {
		return false
	}
	if len(payload.Content) == 0 {
		return false
	}
	// Content may be a plain string.
	var s string
	if err := json.Unmarshal(payload.Content, &s); err == nil {
		return len(s) > 0
	}
	// Content is an array of blocks.
	var blocks []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload.Content, &blocks); err != nil {
		return false
	}
	for _, b := range blocks {
		if b.Type == "text" {
			return true
		}
	}
	return false
}

// getToolResultIds returns tool_use_ids from tool_result blocks in a user message.
// Mirrors TS getToolResultIds.
func getToolResultIds(m types.Message) []string {
	if m.Type != types.MessageTypeUser || len(m.Message) == 0 {
		return nil
	}
	var payload struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(m.Message, &payload); err != nil || len(payload.Content) == 0 {
		return nil
	}
	var blocks []struct {
		Type       string `json:"type"`
		ToolUseID  string `json:"tool_use_id"`
	}
	if err := json.Unmarshal(payload.Content, &blocks); err != nil {
		return nil
	}
	var ids []string
	for _, b := range blocks {
		if b.Type == "tool_result" && b.ToolUseID != "" {
			ids = append(ids, b.ToolUseID)
		}
	}
	return ids
}

// hasToolUseWithIds returns true if an assistant message contains tool_use
// blocks matching any of the given ids. Mirrors TS hasToolUseWithIds.
func hasToolUseWithIds(m types.Message, toolUseIds map[string]struct{}) bool {
	if m.Type != types.MessageTypeAssistant || len(m.Message) == 0 || len(toolUseIds) == 0 {
		return false
	}
	var payload struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(m.Message, &payload); err != nil || len(payload.Content) == 0 {
		return false
	}
	var blocks []struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal(payload.Content, &blocks); err != nil {
		return false
	}
	for _, b := range blocks {
		if b.Type == "tool_use" {
			if _, ok := toolUseIds[b.ID]; ok {
				return true
			}
		}
	}
	return false
}

// --- API invariant adjustment --------------------------------------------

// adjustIndexToPreserveAPIInvariants adjusts startIndex so that tool_use /
// tool_result pairs and thinking blocks sharing the same message.id are
// not split. Mirrors TS adjustIndexToPreserveAPIInvariants.
func adjustIndexToPreserveAPIInvariants(messages []types.Message, startIndex int) int {
	if startIndex <= 0 || startIndex >= len(messages) {
		return startIndex
	}

	adjustedIndex := startIndex

	// Step 1: Handle tool_use/tool_result pairs.
	// Collect tool_result IDs from ALL messages in the kept range.
	allToolResultIds := make(map[string]struct{})
	for i := startIndex; i < len(messages); i++ {
		for _, id := range getToolResultIds(messages[i]) {
			allToolResultIds[id] = struct{}{}
		}
	}

	if len(allToolResultIds) > 0 {
		// Collect tool_use IDs already in the kept range.
		toolUseIdsInKeptRange := make(map[string]struct{})
		for i := adjustedIndex; i < len(messages); i++ {
			msg := messages[i]
			if msg.Type != types.MessageTypeAssistant || len(msg.Message) == 0 {
				continue
			}
			var payload struct {
				Content json.RawMessage `json:"content"`
			}
			if err := json.Unmarshal(msg.Message, &payload); err != nil || len(payload.Content) == 0 {
				continue
			}
			var blocks []struct {
				Type string `json:"type"`
				ID   string `json:"id"`
			}
			if err := json.Unmarshal(payload.Content, &blocks); err != nil {
				continue
			}
			for _, b := range blocks {
				if b.Type == "tool_use" {
					toolUseIdsInKeptRange[b.ID] = struct{}{}
				}
			}
		}

		// Only look for tool_uses NOT already in the kept range.
		neededToolUseIds := make(map[string]struct{})
		for id := range allToolResultIds {
			if _, ok := toolUseIdsInKeptRange[id]; !ok {
				neededToolUseIds[id] = struct{}{}
			}
		}

		// Find the assistant message(s) with matching tool_use blocks.
		for i := adjustedIndex - 1; i >= 0 && len(neededToolUseIds) > 0; i-- {
			if hasToolUseWithIds(messages[i], neededToolUseIds) {
				adjustedIndex = i
				// Remove found tool_use IDs from the set.
				if messages[i].Type == types.MessageTypeAssistant && len(messages[i].Message) > 0 {
					var payload struct {
						Content json.RawMessage `json:"content"`
					}
					if err := json.Unmarshal(messages[i].Message, &payload); err == nil && len(payload.Content) > 0 {
						var blocks []struct {
							Type string `json:"type"`
							ID   string `json:"id"`
						}
						if err := json.Unmarshal(payload.Content, &blocks); err == nil {
							for _, b := range blocks {
								if b.Type == "tool_use" {
									delete(neededToolUseIds, b.ID)
								}
							}
						}
					}
				}
			}
		}
	}

	// Step 2: Handle thinking blocks that share message.id with kept assistant messages.
	// Collect all message.ids from assistant messages in the kept range.
	messageIdsInKeptRange := make(map[string]struct{})
	for i := adjustedIndex; i < len(messages); i++ {
		msg := messages[i]
		if msg.Type != types.MessageTypeAssistant || len(msg.Message) == 0 {
			continue
		}
		var payload struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(msg.Message, &payload); err == nil && payload.ID != "" {
			messageIdsInKeptRange[payload.ID] = struct{}{}
		}
	}

	// Look backwards for assistant messages with same message.id not yet in kept range.
	for i := adjustedIndex - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Type != types.MessageTypeAssistant || len(msg.Message) == 0 {
			continue
		}
		var payload struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(msg.Message, &payload); err == nil || payload.ID == "" {
			continue
		}
		if _, ok := messageIdsInKeptRange[payload.ID]; ok {
			adjustedIndex = i
		}
	}

	return adjustedIndex
}

// --- Messages-to-keep calculation ----------------------------------------

// calculateMessagesToKeepIndex calculates the starting index for messages
// to keep after SM compaction. Starts from lastSummarizedIndex + 1, expands
// backwards to meet minimum tokens and text block messages, capped at
// maxTokens. Floor is the last compact boundary index + 1.
// Mirrors TS calculateMessagesToKeepIndex.
func calculateMessagesToKeepIndex(messages []types.Message, lastSummarizedIndex int, state *State) int {
	if len(messages) == 0 {
		return 0
	}

	cfg := state.GetSMCompactConfig()

	// Start from the message after lastSummarizedIndex.
	startIndex := len(messages)
	if lastSummarizedIndex >= 0 && lastSummarizedIndex < len(messages)-1 {
		startIndex = lastSummarizedIndex + 1
	}

	// Calculate current tokens and text-block count from startIndex to end.
	totalTokens := 0
	textBlockCount := 0
	for i := startIndex; i < len(messages); i++ {
		totalTokens += compactservice.RoughTokenCountEstimationForMessages([]types.Message{messages[i]})
		if hasTextBlocks(messages[i]) {
			textBlockCount++
		}
	}

	// Already hit max cap?
	if totalTokens >= cfg.MaxTokens {
		return adjustIndexToPreserveAPIInvariants(messages, startIndex)
	}

	// Already meet both minimums?
	if totalTokens >= cfg.MinTokens && textBlockCount >= cfg.MinTextBlockMessages {
		return adjustIndexToPreserveAPIInvariants(messages, startIndex)
	}

	// Floor: last compact boundary index + 1.
	floor := 0
	if idx := compactservice.FindLastCompactBoundaryIndex(messages); idx >= 0 {
		floor = idx + 1
	}

	// Expand backwards.
	for i := startIndex - 1; i >= floor; i-- {
		msgTokens := compactservice.RoughTokenCountEstimationForMessages([]types.Message{messages[i]})
		totalTokens += msgTokens
		if hasTextBlocks(messages[i]) {
			textBlockCount++
		}
		startIndex = i

		if totalTokens >= cfg.MaxTokens {
			break
		}
		if totalTokens >= cfg.MinTokens && textBlockCount >= cfg.MinTextBlockMessages {
			break
		}
	}

	return adjustIndexToPreserveAPIInvariants(messages, startIndex)
}

// --- extractDiscoveredToolNames for types.Message -----------------------

// extractDiscoveredToolNamesFromMessages scans messages for tool_reference
// blocks and compact_boundary preCompactDiscoveredTools.
// Mirrors TS extractDiscoveredToolNames.
func extractDiscoveredToolNamesFromMessages(messages []types.Message) map[string]struct{} {
	out := make(map[string]struct{})
	for _, m := range messages {
		// Check compact boundary messages for preCompactDiscoveredTools.
		if compactservice.IsCompactBoundaryMessage(m) {
			var meta struct {
				PreCompactDiscoveredTools []string `json:"preCompactDiscoveredTools"`
			}
			if len(m.CompactMetadata) > 0 && json.Unmarshal(m.CompactMetadata, &meta) == nil {
				for _, n := range meta.PreCompactDiscoveredTools {
					n = strings.TrimSpace(n)
					if n != "" {
						out[n] = struct{}{}
					}
				}
			}
		}
		// Walk user messages for tool_reference blocks in tool_result content.
		if m.Type == types.MessageTypeUser && len(m.Message) > 0 {
			walkUserMsgForToolRefs(m, out)
		}
	}
	return out
}

func walkUserMsgForToolRefs(m types.Message, out map[string]struct{}) {
	var payload struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(m.Message, &payload); err != nil || len(payload.Content) == 0 {
		return
	}
	var blocks []struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(payload.Content, &blocks); err != nil {
		return
	}
	for _, b := range blocks {
		if b.Type != "tool_result" || len(b.Content) == 0 {
			continue
		}
		// Parse discovery JSON from tool_result content.
		scanToolResultForRefs(b.Content, out)
	}
}

func scanToolResultForRefs(content json.RawMessage, out map[string]struct{}) {
	// Try as a string first (JSON-encoded discovery).
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		scanDiscoveryJSONString(s, out)
		return
	}
	// Try as array.
	var arr []map[string]any
	if err := json.Unmarshal(content, &arr); err == nil {
		for _, m := range arr {
			scanToolRefMap(m, out)
		}
	}
}

func scanDiscoveryJSONString(s string, out map[string]struct{}) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	var wrap struct {
		Discovery []map[string]any `json:"discovery"`
	}
	if json.Unmarshal([]byte(s), &wrap) == nil && len(wrap.Discovery) > 0 {
		for _, m := range wrap.Discovery {
			scanToolRefMap(m, out)
		}
		return
	}
	var arr []map[string]any
	if json.Unmarshal([]byte(s), &arr) == nil {
		for _, m := range arr {
			scanToolRefMap(m, out)
		}
	}
}

func scanToolRefMap(m map[string]any, out map[string]struct{}) {
	typ, _ := m["type"].(string)
	if typ != "tool_reference" {
		return
	}
	name, _ := m["tool_name"].(string)
	if name != "" {
		out[name] = struct{}{}
	}
}

// --- CompactionResult builder --------------------------------------------

// createCompactionResultFromSessionMemory builds a CompactionResult from
// session memory. Mirrors TS createCompactionResultFromSessionMemory.
//
// TS signature:
//
//	function createCompactionResultFromSessionMemory(
//	  messages: Message[],
//	  sessionMemory: string,
//	  messagesToKeep: Message[],
//	  hookResults: HookResultMessage[],
//	  transcriptPath: string,
//	  agentId?: AgentId,
//	): CompactionResult
func createCompactionResultFromSessionMemory(
	messages []types.Message,
	sessionMemoryContent string,
	messagesToKeep []types.Message,
	hookResults []types.Message,
	transcriptPath string,
	agentID string,
	sessionID, cwd string,
	planAttachmentProvider func(agentID string) *types.Message,
) compactservice.CompactionResult {
	preCompactTokenCount := compactservice.TokenCountFromLastAPIResponse(messages)

	boundaryMarker, err := compactservice.CreateCompactBoundaryMessage(
		compactservice.CompactTriggerAuto,
		preCompactTokenCount,
		lastMessageUUID(messages),
		"",  // userContext
		nil, // messagesSummarized
	)
	if err != nil {
		// Fallback: create a minimal boundary.
		boundaryMarker = types.Message{}
	}

	// Stamp preCompactDiscoveredTools onto the boundary.
	preCompactDiscovered := extractDiscoveredToolNamesFromMessages(messages)
	if len(preCompactDiscovered) > 0 {
		sorted := sortedKeys(preCompactDiscovered)
		var meta compactservice.CompactMetadata
		if len(boundaryMarker.CompactMetadata) > 0 {
			json.Unmarshal(boundaryMarker.CompactMetadata, &meta)
		}
		meta.PreCompactDiscoveredTools = sorted
		if raw, err := json.Marshal(meta); err == nil {
			boundaryMarker.CompactMetadata = raw
		}
	}

	// Truncate oversized sections.
	truncatedContent, wasTruncated := TruncateSessionMemoryForCompact(sessionMemoryContent)

	summaryContent := compactservice.GetCompactUserSummaryMessage(truncatedContent, compactservice.CompactUserSummaryOpts{
		SuppressFollowUpQuestions: true,
		TranscriptPath:            transcriptPath,
		RecentMessagesPreserved:   true,
	})

	if wasTruncated {
		memoryPath := memdir.GetSessionMemoryPath(sessionID, cwd)
		summaryContent += "\n\nSome session memory sections were truncated for length. The full session memory can be viewed at: " + memoryPath
	}

	// Build summary user message.
	summaryUserMsg := newCompactSummaryUserMessage(summaryContent)

	// Annotate boundary with preservedSegment — mirrors TS:
	//   boundaryMarker: annotateBoundaryWithPreservedSegment(
	//     boundaryMarker,
	//     summaryMessages[summaryMessages.length - 1].uuid,
	//     messagesToKeep,
	//   )
	boundaryMarker = compactservice.AnnotateBoundaryWithPreservedSegment(
		boundaryMarker,
		summaryUserMsg.UUID,
		messagesToKeep,
	)

	summaryMessages := []types.Message{summaryUserMsg}

	// Build plan attachment if applicable, mirroring TS createPlanAttachmentIfNeeded(agentId).
	var attachments []types.Message
	if planAttachmentProvider != nil && agentID != "" {
		if pa := planAttachmentProvider(agentID); pa != nil {
			attachments = append(attachments, *pa)
		}
	}

	// SM-compact has no compact-API-call, so postCompactTokenCount and
	// truePostCompactTokenCount converge to the same value.
	summaryTokens := compactservice.RoughTokenCountEstimationForMessages(summaryMessages)

	return compactservice.CompactionResult{
		BoundaryMarker:           boundaryMarker,
		SummaryMessages:          summaryMessages,
		Attachments:              attachments,
		HookResults:              hookResults,
		MessagesToKeep:           messagesToKeep,
		PreCompactTokenCount:     preCompactTokenCount,
		PostCompactTokenCount:    summaryTokens,
		TruePostCompactTokenCount: summaryTokens,
	}
}

// --- Main entry point ----------------------------------------------------

// TrySessionMemoryCompaction attempts to use session memory for compaction.
// Returns nil when SM compaction cannot be used (caller falls back to API compaction).
// Mirrors TS trySessionMemoryCompaction.
//
// TS signature:
//
//	export async function trySessionMemoryCompaction(
//	  messages: Message[],
//	  agentId?: AgentId,
//	  autoCompactThreshold?: number,
//	): Promise<CompactionResult | null>
//
// planAttachmentProvider mirrors createPlanAttachmentIfNeeded(agentId) from compact.ts.
// When nil, no plan attachment is created.
func TrySessionMemoryCompaction(
	ctx context.Context,
	state *State,
	sessionID, cwd string,
	messages []types.Message,
	transcriptPath string,
	autoCompactThreshold *int,
	runSessionStartHooks func(ctx context.Context, trigger string, model string) ([]types.Message, error),
	agentID string,
	model string,
	planAttachmentProvider func(agentID string) *types.Message,
) (result *compactservice.CompactionResult, err error) {
	_ = ctx

	// TS wraps everything in try/catch with logEvent('tengu_sm_compact_error').
	defer func() {
		if r := recover(); r != nil {
			diaglog.Line("[sessionmemory/sm_compact] error (panic): %v", r)
			result = nil
			err = nil
		}
	}()

	if !shouldUseSessionMemoryCompaction() {
		return nil, nil
	}

	// Initialize config from remote (only fetches once).
	state.InitSMCompactConfig()

	// Wait for any in-progress extraction to complete.
	waitForSessionMemoryExtraction(state)

	sessionMemoryContent := getSessionMemoryContent(sessionID, cwd)

	// No session memory file exists at all.
	if sessionMemoryContent == "" {
		diaglog.Line("[sessionmemory/sm_compact] no_session_memory")
		return nil, nil
	}

	// Session memory exists but matches the template (no actual content).
	if IsSessionMemoryEmpty(sessionMemoryContent) {
		diaglog.Line("[sessionmemory/sm_compact] empty_template")
		return nil, nil
	}

	lastSummarizedMessageID := state.GetLastSummarizedMessageID()

	var lastSummarizedIndex int

	if lastSummarizedMessageID != "" {
		// Normal case: find the index of the last summarized message.
		lastSummarizedIndex = -1
		for i, m := range messages {
			if m.UUID == lastSummarizedMessageID {
				lastSummarizedIndex = i
				break
			}
		}

		if lastSummarizedIndex == -1 {
			// The summarized message ID doesn't exist in current messages.
			diaglog.Line("[sessionmemory/sm_compact] summarized_id_not_found: lastSummarizedMessageId=%s", lastSummarizedMessageID)
			return nil, nil
		}
	} else {
		// Resumed session: session memory has content but we don't know the boundary.
		// Set lastSummarizedIndex to last message so startIndex becomes messages.length.
		lastSummarizedIndex = len(messages) - 1
		diaglog.Line("[sessionmemory/sm_compact] resumed_session: no lastSummarizedMessageId, using lastSummarizedIndex=%d", lastSummarizedIndex)
	}

	// Calculate the starting index for messages to keep.
	startIndex := calculateMessagesToKeepIndex(messages, lastSummarizedIndex, state)

	// Filter out old compact boundary messages from messagesToKeep.
	messagesToKeep := filterWithoutCompactBoundaries(messages[startIndex:])

	// Run session start hooks, mirroring TS:
	//   const hookResults = await processSessionStartHooks('compact', { model: getMainLoopModel() })
	var hookResults []types.Message
	if runSessionStartHooks != nil {
		var hookErr error
		hookResults, hookErr = runSessionStartHooks(ctx, "compact", model)
		if hookErr != nil {
			diaglog.Line("[sessionmemory/sm_compact] session_start_hooks error: %v", hookErr)
			return nil, nil
		}
	}

	compactionResult := createCompactionResultFromSessionMemory(
		messages,
		sessionMemoryContent,
		messagesToKeep,
		hookResults,
		transcriptPath,
		agentID,
		sessionID, cwd,
		planAttachmentProvider,
	)

	postCompactMessages := compactservice.BuildPostCompactMessages(compactionResult)
	postCompactTokenCount := compactservice.RoughTokenCountEstimationForMessages(postCompactMessages)

	// Only check threshold if one was provided (for autocompact).
	if autoCompactThreshold != nil && postCompactTokenCount >= *autoCompactThreshold {
		diaglog.Line("[sessionmemory/sm_compact] threshold_exceeded: postCompactTokenCount=%d autoCompactThreshold=%d",
			postCompactTokenCount, *autoCompactThreshold)
		return nil, nil
	}

	compactionResult.PostCompactTokenCount = postCompactTokenCount
	compactionResult.TruePostCompactTokenCount = postCompactTokenCount

	return &compactionResult, nil
}

// --- Helpers -------------------------------------------------------------

// lastMessageUUID returns the UUID of the last message, or empty string.
func lastMessageUUID(messages []types.Message) string {
	if n := len(messages); n > 0 {
		return messages[n-1].UUID
	}
	return ""
}

// filterWithoutCompactBoundaries filters out compact_boundary messages.
func filterWithoutCompactBoundaries(messages []types.Message) []types.Message {
	out := make([]types.Message, 0, len(messages))
	for _, m := range messages {
		if !compactservice.IsCompactBoundaryMessage(m) {
			out = append(out, m)
		}
	}
	return out
}

// sortedKeys returns a sorted slice of map keys.
func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Bubble sort (small sets, performance not critical).
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// newCompactSummaryUserMessage creates a user message with
// isCompactSummary=true and isVisibleInTranscriptOnly=true.
func newCompactSummaryUserMessage(content string) types.Message {
	isCompact := true
	isVisible := true
	inner := map[string]any{"role": "user", "content": content}
	innerJSON, _ := json.Marshal(inner)
	return types.Message{
		Type:                      types.MessageTypeUser,
		UUID:                      compactservice.NewUUID(),
		Message:                   json.RawMessage(innerJSON),
		Content: func() json.RawMessage { b, _ := json.Marshal(content); return b }(),
		IsCompactSummary:          &isCompact,
		IsVisibleInTranscriptOnly: &isVisible,
	}
}
