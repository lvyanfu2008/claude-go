package query

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"goc/diagnostics"
	"goc/types"
)

// charsPerToken mirrors TS CHARS_PER_TOKEN = 4 in snipCompact.ts estimateTokens.
const charsPerToken = 4

// snipToolOutputData mirrors the "data" envelope of SnipFromJSON output.
type snipToolOutputData struct {
	SnippedCount int      `json:"snipped_count"`
	Summary      string   `json:"summary"`
	MessageIDs   []string `json:"message_ids"`
}

// snipBoundaryMetadata mirrors the TS snipMetadata shape on snip_boundary system messages.
type snipBoundaryMetadata struct {
	Trigger      string   `json:"trigger"`
	Summary      string   `json:"summary,omitempty"`
	RemovedUuids []string `json:"removedUuids"`
	RemovedCount int      `json:"removedCount"`
}

// SnipCompactFn returns a QueryDeps.SnipCompact function.
func SnipCompactFn(newUUID func() string) func(ctx context.Context, in *SnipCompactInput) (*SnipCompactResult, error) {
	return func(_ context.Context, in *SnipCompactInput) (*SnipCompactResult, error) {
		return snipCompact(in.Messages, newUUID)
	}
}

// snipCompact mirrors TS snipCompactIfNeeded + projectSnippedView.
//
// Phase 1: collect already-removed UUIDs from existing snip_boundary messages
// (handles subsequent turns where the boundary already exists).
//
// Phase 2: find unprocessed Snip tool results, match short IDs to full UUIDs,
// create a new boundary for any newly-removed messages.
func snipCompact(messages []types.Message, newUUID func() string) (*SnipCompactResult, error) {
	// Phase 1: collect already-removed UUIDs from existing snip_boundary messages.
	alreadyRemoved := make(map[string]bool)
	for _, m := range messages {
		if m.Type != types.MessageTypeSystem || m.Subtype == nil || *m.Subtype != "snip_boundary" {
			continue
		}
		for _, uuid := range parseSnipRemovedUuids(m.CompactMetadata) {
			alreadyRemoved[uuid] = true
		}
	}

	// Phase 2: find new Snip tool results and match short IDs to full UUIDs.
	shortIDs, summary := findSnipResults(messages)
	shortToUUID := buildShortToUUID(messages)

	var newRemovedUUIDs []string
	for _, sid := range shortIDs {
		uuid, ok := shortToUUID[sid]
		if !ok {
			continue
		}
		if alreadyRemoved[uuid] {
			continue
		}
		alreadyRemoved[uuid] = true
		newRemovedUUIDs = append(newRemovedUUIDs, uuid)
	}

	// Phase 3: filter messages.
	if len(alreadyRemoved) == 0 {
		return nil, nil
	}
	diagnostics.LogForDiagnosticsNoPII("snipCompact: already-removed=%d new-removed=%d", len(alreadyRemoved)-len(newRemovedUUIDs), len(newRemovedUUIDs))
	filtered := make([]types.Message, 0, len(messages))
	var tokensFreed int
	for _, m := range messages {
		if alreadyRemoved[m.UUID] {
			tokensFreed += estimateMsgTokens(m)
			continue
		}
		filtered = append(filtered, m)
	}

	// Phase 4: create boundary only if new messages were removed this turn.
	if len(newRemovedUUIDs) == 0 {
		return &SnipCompactResult{
			Messages:    filtered,
			TokensFreed: tokensFreed,
		}, nil
	}

	diagnostics.LogForDiagnosticsNoPII("snipCompact: creating boundary removed=%d uuids=%v summary=%q", len(newRemovedUUIDs), newRemovedUUIDs, summary)
	boundary, err := createSnipBoundaryMessage(summary, newRemovedUUIDs, len(newRemovedUUIDs), newUUID)
	if err != nil {
		return nil, fmt.Errorf("snipCompact: create boundary: %w", err)
	}

	return &SnipCompactResult{
		Messages:        filtered,
		TokensFreed:     tokensFreed,
		BoundaryMessage: &boundary,
	}, nil
}

// parseSnipRemovedUuids extracts removedUuids from a snip_boundary's CompactMetadata.
func parseSnipRemovedUuids(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var meta snipBoundaryMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil
	}
	return meta.RemovedUuids
}

// findSnipResults scans messages for user messages whose ToolUseResult contains
// a Snip tool output (with snipped_count > 0). Returns all collected short message IDs
// and a concatenated summary.
func findSnipResults(messages []types.Message) (shortIDs []string, summary string) {
	var summaries []string
	for _, m := range messages {
		if m.Type != types.MessageTypeUser || len(m.ToolUseResult) == 0 {
			continue
		}
		var env struct {
			Data snipToolOutputData `json:"data"`
		}
		if err := json.Unmarshal(m.ToolUseResult, &env); err != nil {
			continue
		}
		if env.Data.SnippedCount <= 0 || len(env.Data.MessageIDs) == 0 {
			continue
		}
		shortIDs = append(shortIDs, env.Data.MessageIDs...)
		if s := strings.TrimSpace(env.Data.Summary); s != "" {
			summaries = append(summaries, s)
		}
	}
	if len(summaries) > 0 {
		summary = strings.Join(summaries, "; ")
	}
	return
}

// buildShortToUUID builds a map from short base-36 message IDs to full UUIDs.
func buildShortToUUID(messages []types.Message) map[string]string {
	m := make(map[string]string, len(messages))
	for _, msg := range messages {
		sid := deriveShortMessageID(msg.UUID)
		m[sid] = msg.UUID
	}
	return m
}

// deriveShortMessageID mirrors messagesapi/format_helpers.go deriveShortMessageId.
func deriveShortMessageID(uuidStr string) string {
	hex := strings.ReplaceAll(uuidStr, "-", "")
	if len(hex) < 10 {
		hex = hex + strings.Repeat("0", 10-len(hex))
	}
	hex = hex[:10]
	n, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		return "0"
	}
	s := strconv.FormatUint(n, 36)
	if len(s) > 6 {
		return s[:6]
	}
	return s
}

// estimateMsgTokens estimates the token count of a message using ~4 chars/token heuristic.
func estimateMsgTokens(msg types.Message) int {
	b, err := json.Marshal(msg)
	if err != nil {
		return 0
	}
	n := len(b) / charsPerToken
	if n < 1 {
		return 1
	}
	return n
}

// createSnipBoundaryMessage creates a system message with subtype "snip_boundary"
// and CompactMetadata containing snip-specific fields.
func createSnipBoundaryMessage(summary string, removedUUIDs []string, removedCount int, newUUID func() string) (types.Message, error) {
	meta := snipBoundaryMetadata{
		Trigger:      "snip",
		Summary:      summary,
		RemovedUuids: removedUUIDs,
		RemovedCount: removedCount,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return types.Message{}, err
	}

	contentStr := fmt.Sprintf("Snipped %d messages", removedCount)
	if summary != "" {
		contentStr += ": " + summary
	}
	content, err := json.Marshal(contentStr)
	if err != nil {
		return types.Message{}, err
	}

	subtype := "snip_boundary"
	level := "info"
	isMeta := false

	return types.Message{
		Type:            types.MessageTypeSystem,
		UUID:            newUUID(),
		Subtype:         &subtype,
		Level:           &level,
		IsMeta:          &isMeta,
		Content:         json.RawMessage(content),
		CompactMetadata: json.RawMessage(metaJSON),
	}, nil
}
