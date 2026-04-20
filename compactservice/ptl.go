package compactservice

import (
	"encoding/json"
	"math"
	"regexp"
	"strconv"

	"goc/types"
)

// PromptTooLongErrorMessage mirrors PROMPT_TOO_LONG_ERROR_MESSAGE in services/api/errors.ts.
const PromptTooLongErrorMessage = "Prompt is too long"

// PtlRetryMarker is the meta-user preamble we prepend after truncating the head,
// so the API sees a role=user first message. Mirrors PTL_RETRY_MARKER in compact.ts.
const PtlRetryMarker = "[earlier conversation truncated for compaction retry]"

// MaxPTLRetries mirrors MAX_PTL_RETRIES in compact.ts.
const MaxPTLRetries = 3

// ptlGapRE mirrors parsePromptTooLongTokenCounts' regex in services/api/errors.ts.
// Example raw: "prompt is too long: 137500 tokens > 135000 maximum" (case-insensitive,
// with possible SDK envelope text around it). We look for "<N> tokens > <M>".
var ptlGapRE = regexp.MustCompile(`(?i)prompt is too long[^0-9]*(\d+)\s*tokens?\s*>\s*(\d+)`)

// ParsePromptTooLongTokenCounts mirrors parsePromptTooLongTokenCounts in TS.
// Returns (actual, limit, ok) — ok is false when the message doesn't match.
func ParsePromptTooLongTokenCounts(raw string) (int, int, bool) {
	m := ptlGapRE.FindStringSubmatch(raw)
	if m == nil {
		return 0, 0, false
	}
	actual, err1 := strconv.Atoi(m[1])
	limit, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return actual, limit, true
}

// IsPromptTooLongMessage mirrors isPromptTooLongMessage in services/api/errors.ts.
// Treats an assistant message as PTL iff isApiErrorMessage=true and its content array
// contains a text block starting with "Prompt is too long".
func IsPromptTooLongMessage(m types.Message) bool {
	if m.Type != types.MessageTypeAssistant {
		return false
	}
	if m.IsApiErrorMessage == nil || !*m.IsApiErrorMessage {
		return false
	}
	if len(m.Message) == 0 {
		return false
	}
	var probe struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(m.Message, &probe); err != nil {
		return false
	}
	for _, b := range probe.Content {
		if b.Type == "text" && len(b.Text) >= len(PromptTooLongErrorMessage) && b.Text[:len(PromptTooLongErrorMessage)] == PromptTooLongErrorMessage {
			return true
		}
	}
	return false
}

// GetPromptTooLongTokenGap mirrors getPromptTooLongTokenGap in TS.
// Needs access to the raw errorDetails; we probe the outer Message for
// errorDetails field (Go Message does not currently have a typed errorDetails;
// sync via json.RawMessage probe — hosts populate it with a string value).
func GetPromptTooLongTokenGap(m types.Message) (int, bool) {
	if !IsPromptTooLongMessage(m) {
		return 0, false
	}
	raw := extractErrorDetailsString(m)
	if raw == "" {
		return 0, false
	}
	actual, limit, ok := ParsePromptTooLongTokenCounts(raw)
	if !ok {
		return 0, false
	}
	gap := actual - limit
	if gap <= 0 {
		return 0, false
	}
	return gap, true
}

// extractErrorDetailsString pulls the errorDetails field from a message JSON without
// coupling us to a typed Go field (TS message has optional errorDetails: unknown).
// We check both the outer msg struct (via raw marshal) and the inner message envelope
// for an errorDetails string.
func extractErrorDetailsString(m types.Message) string {
	// Outer: Message struct has no ErrorDetails field; try marshalling and re-probing.
	raw, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	var probe struct {
		ErrorDetails json.RawMessage `json:"errorDetails"`
	}
	if err := json.Unmarshal(raw, &probe); err == nil && len(probe.ErrorDetails) > 0 {
		var s string
		if err := json.Unmarshal(probe.ErrorDetails, &s); err == nil {
			return s
		}
	}
	// Inner message envelope: some assistant error surfaces embed errorDetails under message.errorDetails.
	if len(m.Message) > 0 {
		var inner struct {
			ErrorDetails json.RawMessage `json:"errorDetails"`
		}
		if err := json.Unmarshal(m.Message, &inner); err == nil && len(inner.ErrorDetails) > 0 {
			var s string
			if err := json.Unmarshal(inner.ErrorDetails, &s); err == nil {
				return s
			}
		}
	}
	return ""
}

// TruncateHeadForPTLRetry mirrors truncateHeadForPTLRetry in compact.ts.
// Drops the oldest API-round groups from messages until tokenGap is covered
// (or a 20% fallback), always keeping at least one group so the summarize
// request has context. Returns (nil, false) when nothing can be dropped
// without emptying the set.
func TruncateHeadForPTLRetry(messages []types.Message, ptlResponse types.Message) ([]types.Message, bool) {
	// Strip a prior synthetic marker to avoid stalling on repeat retries.
	input := messages
	if len(input) > 0 && input[0].Type == types.MessageTypeUser && input[0].IsMeta != nil && *input[0].IsMeta {
		if userMessageContentEquals(input[0], PtlRetryMarker) {
			input = input[1:]
		}
	}

	groups := GroupMessagesByApiRound(input)
	if len(groups) < 2 {
		return nil, false
	}

	var dropCount int
	if gap, ok := GetPromptTooLongTokenGap(ptlResponse); ok {
		acc := 0
		for _, g := range groups {
			acc += RoughTokenCountEstimationForMessages(g)
			dropCount++
			if acc >= gap {
				break
			}
		}
	} else {
		dropCount = int(math.Max(1, math.Floor(float64(len(groups))*0.2)))
	}

	// Keep at least one group.
	if dropCount > len(groups)-1 {
		dropCount = len(groups) - 1
	}
	if dropCount < 1 {
		return nil, false
	}

	sliced := flattenGroups(groups[dropCount:])
	// If first message is assistant, prepend meta user marker so API gets a user-first sequence.
	if len(sliced) > 0 && sliced[0].Type == types.MessageTypeAssistant {
		marker, err := createMetaUserMarker(PtlRetryMarker)
		if err == nil {
			sliced = append([]types.Message{marker}, sliced...)
		}
	}
	return sliced, true
}

func flattenGroups(groups [][]types.Message) []types.Message {
	n := 0
	for _, g := range groups {
		n += len(g)
	}
	out := make([]types.Message, 0, n)
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// userMessageContentEquals returns true when m is a user message whose content
// string equals target. Used for detecting a prior PTL retry marker.
func userMessageContentEquals(m types.Message, target string) bool {
	if m.Type != types.MessageTypeUser || len(m.Message) == 0 {
		return false
	}
	var probe struct {
		Content any `json:"content"`
	}
	if err := json.Unmarshal(m.Message, &probe); err != nil {
		return false
	}
	switch v := probe.Content.(type) {
	case string:
		return v == target
	case []any:
		for _, b := range v {
			bm, ok := b.(map[string]any)
			if !ok {
				continue
			}
			if t, _ := bm["type"].(string); t == "text" {
				if s, _ := bm["text"].(string); s == target {
					return true
				}
			}
		}
	}
	return false
}

// createMetaUserMarker builds a minimal user message {role:user, content:target} with isMeta=true.
func createMetaUserMarker(target string) (types.Message, error) {
	inner := map[string]any{
		"role":    "user",
		"content": target,
	}
	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return types.Message{}, err
	}
	isMeta := true
	return types.Message{
		Type:    types.MessageTypeUser,
		UUID:    newUUID(),
		IsMeta:  &isMeta,
		Message: json.RawMessage(innerJSON),
	}, nil
}
