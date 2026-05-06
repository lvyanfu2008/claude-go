package toolresultpersist

import (
	"encoding/json"

	"goc/ccb-engine/diaglog"
	"goc/types"
)

// Candidate partitions — see partitionByPriorDecision in TS.
type candidatePartition struct {
	mustReapply []toolResultCandidate
	frozen      []toolResultCandidate
	fresh       []toolResultCandidate
}

// toolResultCandidate represents a tool_result block eligible for budget enforcement.
type toolResultCandidate struct {
	toolUseID  string
	content    any
	size       int
	reapply    string // non-empty only for mustReapply entries
	isSeen     bool
	isReplaced bool
}

// ContentReplacementState mirrors TS ContentReplacementState.
// Tracks prior decisions so enforcement is stable across turns (prompt cache safety).
// MUTATED by EnforceToolResultBudget — the caller holds a stable reference.
type ContentReplacementState struct {
	SeenIDs      map[string]bool   `json:"seenIds"`      // set semantics
	Replacements map[string]string  `json:"replacements"` // toolUseID → preview string
}

// NewContentReplacementState creates a fresh state.
func NewContentReplacementState() *ContentReplacementState {
	return &ContentReplacementState{
		SeenIDs:      make(map[string]bool),
		Replacements: make(map[string]string),
	}
}

// Clone shallow-copies the state (for cache-sharing forks like agentSummary).
func (s *ContentReplacementState) Clone() *ContentReplacementState {
	if s == nil {
		return nil
	}
	seen := make(map[string]bool, len(s.SeenIDs))
	for k, v := range s.SeenIDs {
		seen[k] = v
	}
	repl := make(map[string]string, len(s.Replacements))
	for k, v := range s.Replacements {
		repl[k] = v
	}
	return &ContentReplacementState{SeenIDs: seen, Replacements: repl}
}

// ToJSON marshals the state for storage (e.g. in ToolUseContext.ContentReplacementState).
func (s *ContentReplacementState) ToJSON() json.RawMessage {
	if s == nil {
		return nil
	}
	seenIDs := make([]string, 0, len(s.SeenIDs))
	for id := range s.SeenIDs {
		seenIDs = append(seenIDs, id)
	}
	wire := contentReplacementWire{
		Replacements: s.Replacements,
		SeenIDs:      seenIDs,
	}
	b, _ := json.Marshal(wire)
	return b
}

// contentReplacementWire is the JSON shape matching conversation-runtime/query/tool_result_reapply.go.
type contentReplacementWire struct {
	Replacements map[string]string `json:"replacements"`
	SeenIDs      []string          `json:"seenIds,omitempty"`
}

// ContentReplacementRecord mirrors TS ContentReplacementRecord for transcript persistence.
type ContentReplacementRecord struct {
	Kind        string `json:"kind"`        // "tool-result"
	ToolUseID   string `json:"toolUseId"`
	Replacement string `json:"replacement"`
}

// EnforceResult contains the output of EnforceToolResultBudget.
type EnforceResult struct {
	Messages       []types.Message
	NewlyReplaced  []ContentReplacementRecord
}

// EnforceToolResultBudget mirrors TS enforceToolResultBudget.
//
// For each user message whose tool_result blocks together exceed the per-message limit,
// the largest FRESH (never-before-seen) results are persisted to disk and replaced with
// previews. Previously-replaced results get the same replacement re-applied (zero I/O).
//
// State is MUTATED in place — seenIds and replacements are updated to record choices.
//
// Parameters:
//   - messages: current conversation messages
//   - state: MUTATED state tracking prior decisions
//   - info: session identity for path resolution
//   - limit: per-message aggregate budget (0 uses default MaxToolResultsPerMessageChars)
//   - skipToolNames: set of tool names to never persist (e.g., "Read")
//   - nameByToolUseID: optional map from tool_use_id → tool_name (built from assistant messages)
func EnforceToolResultBudget(
	messages []types.Message,
	state *ContentReplacementState,
	info SessionInfo,
	limit int,
	skipToolNames map[string]bool,
	nameByToolUseID map[string]string,
) EnforceResult {
	if state == nil {
		return EnforceResult{Messages: messages}
	}
	if limit <= 0 {
		limit = MaxToolResultsPerMessageChars
	}
	// Resolve once per call. A mid-session limit change only affects FRESH
	// messages (prior decisions are frozen via SeenIDs/Replacements).
	threshold := limit

	candidatesByMessage := collectCandidatesByMessage(messages)

	replacementMap := make(map[string]string)
	var toPersist []toolResultCandidate
	reappliedCount := 0

	for _, candidates := range candidatesByMessage {
		partition := partitionByPriorDecision(candidates, state)

		// Re-apply: pure Map lookups. No file I/O, byte-identical.
		for _, c := range partition.mustReapply {
			replacementMap[c.toolUseID] = c.reapply
		}
		reappliedCount += len(partition.mustReapply)

		if len(partition.fresh) == 0 {
			for _, c := range candidates {
				state.SeenIDs[c.toolUseID] = true
			}
			continue
		}

		// Skip tools that should never be persisted (e.g. Read with MaxInt64 threshold)
		shouldSkip := func(id string) bool {
			if skipToolNames == nil {
				return false
			}
			name, ok := nameByToolUseID[id]
			if !ok {
				return false
			}
			return skipToolNames[name]
		}

		skipped := filterCandidates(partition.fresh, shouldSkip)
		for _, c := range skipped {
			state.SeenIDs[c.toolUseID] = true
		}
		eligible := filterCandidatesNot(partition.fresh, shouldSkip)

		frozenSize := sumCandidateSizes(partition.frozen)
		freshSize := sumCandidateSizes(eligible)

		var selected []toolResultCandidate
		if frozenSize+freshSize > threshold {
			selected = selectFreshToReplace(eligible, frozenSize, threshold)
		}

		selectedIDs := make(map[string]bool)
		for _, c := range selected {
			selectedIDs[c.toolUseID] = true
		}
		for _, c := range candidates {
			if !selectedIDs[c.toolUseID] {
				state.SeenIDs[c.toolUseID] = true
			}
		}

		toPersist = append(toPersist, selected...)
	}

	if len(replacementMap) == 0 && len(toPersist) == 0 {
		return EnforceResult{Messages: messages}
	}

	// Fresh: concurrent persist for all selected candidates.
	newlyReplaced := make([]ContentReplacementRecord, 0)
	for _, c := range toPersist {
		state.SeenIDs[c.toolUseID] = true
		persisted, persistErr := PersistToolResult(info, c.content, c.toolUseID)
		if persistErr != nil {
			diaglog.Line("[toolresultpersist] budget persist failed for toolUseID=%s: %s", c.toolUseID, persistErr.Error)
			continue
		}
		replacement := BuildLargeToolResultMessage(persisted)
		replacementMap[c.toolUseID] = replacement
		state.Replacements[c.toolUseID] = replacement
		newlyReplaced = append(newlyReplaced, ContentReplacementRecord{
			Kind:        "tool-result",
			ToolUseID:   c.toolUseID,
			Replacement: replacement,
		})
	}

	if len(replacementMap) == 0 {
		return EnforceResult{Messages: messages}
	}

	if len(newlyReplaced) > 0 {
		diaglog.Line("[toolresultpersist] budget: persisted %d results, %d re-applied",
			len(newlyReplaced), reappliedCount)
	}

	return EnforceResult{
		Messages:      replaceToolResultContents(messages, replacementMap),
		NewlyReplaced: newlyReplaced,
	}
}

// ApplyToolResultBudget mirrors TS applyToolResultBudget — query-loop integration.
// Gates on state (nil means feature disabled → messages returned as-is).
func ApplyToolResultBudget(
	messages []types.Message,
	state *ContentReplacementState,
	info SessionInfo,
	limit int,
	skipToolNames map[string]bool,
) []types.Message {
	if state == nil {
		return messages
	}
	if len(messages) == 0 {
		return messages
	}
	nameByToolUseID := buildToolNameMap(messages)
	result := EnforceToolResultBudget(messages, state, info, limit, skipToolNames, nameByToolUseID)
	return result.Messages
}

// --- internal helpers ---

// isContentAlreadyCompacted checks if content starts with PersistedOutputTag.
func isContentAlreadyCompacted(content any) bool {
	s, ok := content.(string)
	return ok && len(s) >= len(PersistedOutputTag) && s[:len(PersistedOutputTag)] == PersistedOutputTag
}

// collectCandidatesFromMessage extracts tool_result blocks from a single user message.
func collectCandidatesFromMessage(msg types.Message) []toolResultCandidate {
	if msg.Type != types.MessageTypeUser || len(msg.Content) == 0 {
		return nil
	}
	var blocks []map[string]any
	if json.Unmarshal(msg.Content, &blocks) != nil {
		return nil
	}
	var out []toolResultCandidate
	for _, b := range blocks {
		typ, _ := b["type"].(string)
		if typ != "tool_result" {
			continue
		}
		content := b["content"]
		if content == nil {
			continue
		}
		if isContentAlreadyCompacted(content) {
			continue
		}
		if hasImageBlock(content) {
			continue
		}
		toolUseID, _ := b["tool_use_id"].(string)
		out = append(out, toolResultCandidate{
			toolUseID: toolUseID,
			content:   content,
			size:      contentSize(content),
		})
	}
	return out
}

// collectCandidatesByMessage groups candidates by API-level user message.
// Mirrors TS collectCandidatesByMessage — consecutive user messages (not separated
// by an assistant) form one group, because normalizeMessagesForAPI merges them.
func collectCandidatesByMessage(messages []types.Message) [][]toolResultCandidate {
	var groups [][]toolResultCandidate
	var current []toolResultCandidate

	flush := func() {
		if len(current) > 0 {
			groups = append(groups, current)
		}
		current = nil
	}

	seenAsstIDs := make(map[string]bool)
	for _, msg := range messages {
		if msg.Type == types.MessageTypeUser {
			current = append(current, collectCandidatesFromMessage(msg)...)
		} else if msg.Type == types.MessageTypeAssistant {
			// Get assistant message ID
			var env struct {
				ID string `json:"id"`
			}
			id := ""
			if json.Unmarshal(msg.Message, &env) == nil {
				id = env.ID
			}
			if id != "" && !seenAsstIDs[id] {
				flush()
				seenAsstIDs[id] = true
			}
		}
	}
	flush()
	return groups
}

// partitionByPriorDecision splits candidates by prior decision state.
func partitionByPriorDecision(candidates []toolResultCandidate, state *ContentReplacementState) candidatePartition {
	var p candidatePartition
	for _, c := range candidates {
		if repl, ok := state.Replacements[c.toolUseID]; ok {
			c.reapply = repl
			p.mustReapply = append(p.mustReapply, c)
		} else if state.SeenIDs[c.toolUseID] {
			p.frozen = append(p.frozen, c)
		} else {
			p.fresh = append(p.fresh, c)
		}
	}
	return p
}

func sumCandidateSizes(candidates []toolResultCandidate) int {
	sum := 0
	for _, c := range candidates {
		sum += c.size
	}
	return sum
}

func filterCandidates(candidates []toolResultCandidate, fn func(id string) bool) []toolResultCandidate {
	var out []toolResultCandidate
	for _, c := range candidates {
		if fn(c.toolUseID) {
			out = append(out, c)
		}
	}
	return out
}

func filterCandidatesNot(candidates []toolResultCandidate, fn func(id string) bool) []toolResultCandidate {
	var out []toolResultCandidate
	for _, c := range candidates {
		if !fn(c.toolUseID) {
			out = append(out, c)
		}
	}
	return out
}

// selectFreshToReplace picks the largest fresh results until model-visible total
// (frozen + remaining fresh) is at or under threshold, or fresh is exhausted.
func selectFreshToReplace(fresh []toolResultCandidate, frozenSize, threshold int) []toolResultCandidate {
	if threshold <= 0 {
		return nil
	}
	sorted := make([]toolResultCandidate, len(fresh))
	copy(sorted, fresh)
	// Sort descending by size
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].size > sorted[i].size {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	remaining := frozenSize + sumCandidateSizes(sorted)
	var selected []toolResultCandidate
	for _, c := range sorted {
		if remaining <= threshold {
			break
		}
		selected = append(selected, c)
		remaining -= c.size
	}
	return selected
}

// replaceToolResultContents replaces tool_result block contents when their ID
// is in replacementMap. Messages with no replacements are returned by reference.
func replaceToolResultContents(messages []types.Message, replacementMap map[string]string) []types.Message {
	out := make([]types.Message, len(messages))
	copy(out, messages)
	for i, m := range out {
		if m.Type != types.MessageTypeUser || len(m.Content) == 0 {
			continue
		}
		var blocks []map[string]any
		if json.Unmarshal(m.Content, &blocks) != nil {
			continue
		}
		changed := false
		for j, b := range blocks {
			if typ, _ := b["type"].(string); typ != "tool_result" {
				continue
			}
			if id, _ := b["tool_use_id"].(string); id != "" {
				if repl, ok := replacementMap[id]; ok {
					b["content"] = repl
					blocks[j] = b
					changed = true
				}
			}
		}
		if !changed {
			continue
		}
		newContent, _ := json.Marshal(blocks)
		inner, _ := json.Marshal(map[string]any{"role": "user", "content": blocks})
		nm := m
		nm.Message = inner
		nm.Content = newContent
		out[i] = nm
	}
	return out
}

// buildToolNameMap walks messages and builds tool_use_id → tool_name from assistant tool_use blocks.
func buildToolNameMap(messages []types.Message) map[string]string {
	m := make(map[string]string)
	for _, msg := range messages {
		if msg.Type != types.MessageTypeAssistant || len(msg.Content) == 0 {
			continue
		}
		var blocks []map[string]any
		if json.Unmarshal(msg.Content, &blocks) != nil {
			continue
		}
		for _, b := range blocks {
			if typ, _ := b["type"].(string); typ == "tool_use" {
				id, _ := b["id"].(string)
				name, _ := b["name"].(string)
				if id != "" {
					m[id] = name
				}
			}
		}
	}
	return m
}
