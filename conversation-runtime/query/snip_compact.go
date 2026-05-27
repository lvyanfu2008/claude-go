package query

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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

// SnipCompactFn returns a QueryDeps.SnipCompact function that scans messages
// for Snip tool results and filters the referenced messages out of the working set.
func SnipCompactFn(newUUID func() string) func(ctx context.Context, in *SnipCompactInput) (*SnipCompactResult, error) {
	return func(_ context.Context, in *SnipCompactInput) (*SnipCompactResult, error) {
		return snipCompact(in.Messages, newUUID)
	}
}

// snipCompact is the core implementation, extracted for testability.
func snipCompact(messages []types.Message, newUUID func() string) (*SnipCompactResult, error) {
	shortIDs, summary := findSnipResults(messages)
	if len(shortIDs) == 0 {
		return nil, nil
	}

	// Build short-ID → full UUID map from all messages.
	shortToUUID := make(map[string]string, len(messages))
	for _, m := range messages {
		sid := deriveShortMessageID(m.UUID)
		shortToUUID[sid] = m.UUID
	}

	removedSet := make(map[string]bool)
	var removedUUIDs []string
	for _, sid := range shortIDs {
		if uuid, ok := shortToUUID[sid]; ok {
			if !removedSet[uuid] {
				removedSet[uuid] = true
				removedUUIDs = append(removedUUIDs, uuid)
			}
		}
	}
	if len(removedSet) == 0 {
		return nil, nil
	}

	// Filter messages.
	filtered := make([]types.Message, 0, len(messages))
	var tokensFreed int
	for _, m := range messages {
		if removedSet[m.UUID] {
			tokensFreed += estimateMsgTokens(m)
			continue
		}
		filtered = append(filtered, m)
	}

	boundary, err := createSnipBoundaryMessage(summary, removedUUIDs, len(removedSet), newUUID)
	if err != nil {
		return nil, fmt.Errorf("snipCompact: create boundary: %w", err)
	}

	return &SnipCompactResult{
		Messages:        filtered,
		TokensFreed:     tokensFreed,
		BoundaryMessage: &boundary,
	}, nil
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
