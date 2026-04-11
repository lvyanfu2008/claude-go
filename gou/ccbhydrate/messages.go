// Package ccbhydrate builds JSON message arrays for ccb-engine SubmitUserTurn / HydrateFromMessages
// (goc/ccb-engine/internal/anthropic.Message: role + content string or block array).
package ccbhydrate

import (
	"encoding/json"
	"slices"
	"strings"

	"goc/gou/messagerow"
	"goc/messagesapi"
	"goc/types"
)

// apiMessage is the on-wire shape json.Unmarshal uses into engine session.
type apiMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// MessagesJSONNormalized runs [messagesapi.NormalizeMessagesForAPI] then the same JSON projection as [MessagesJSON].
func MessagesJSONNormalized(msgs []types.Message, tools []messagesapi.ToolSpec, opts messagesapi.Options) (json.RawMessage, error) {
	norm, err := messagesapi.NormalizeMessagesForAPI(msgs, tools, opts)
	if err != nil {
		return nil, err
	}
	return messagesJSONFromNormalized(norm)
}

// MessagesJSON returns a JSON array suitable for SubmitUserTurn payload.messages.
// Only user and assistant rows with non-empty content (after messagerow.NormalizeMessageJSON) are included, in order.
func MessagesJSON(msgs []types.Message) (json.RawMessage, error) {
	norm := make([]types.Message, len(msgs))
	for i, m := range msgs {
		norm[i] = messagerow.NormalizeMessageJSON(m)
	}
	return messagesJSONFromNormalized(norm)
}

func messagesJSONFromNormalized(msgs []types.Message) (json.RawMessage, error) {
	var out []apiMessage
	for _, m := range msgs {
		m = messagerow.NormalizeMessageJSON(m)
		switch m.Type {
		case types.MessageTypeUser, types.MessageTypeAssistant:
		default:
			continue
		}
		if len(m.Content) == 0 {
			continue
		}
		role := string(m.Type)
		out = append(out, apiMessage{Role: role, Content: m.Content})
	}
	if len(out) == 0 {
		return json.RawMessage("[]"), nil
	}
	return json.Marshal(out)
}

// MessagesJSONWithLeadingMeta runs [messagesapi.NormalizeMessagesForAPI], then injects reminders and
// merges again with [messagesapi.NormalizeMessagesForAPI]:
//   - skill_listing: append a reminder user immediately **after** the last user in the normalized transcript,
//     matching TS [processTextPrompt] order `[userMessage, ...attachmentMessages]` so [normalizeMessagesForAPI]
//     merges attachment-style text **after** the client user blocks (same seam as mergeUserMessagesAndToolResults).
//   - prependUserContext: prepend a meta user (TS [prependUserContext] before normalize).
func MessagesJSONWithLeadingMeta(msgs []types.Message, userContextReminder, skillListingText string, tools []messagesapi.ToolSpec, opts messagesapi.Options) (json.RawMessage, error) {
	if strings.TrimSpace(skillListingText) == "" && strings.TrimSpace(userContextReminder) == "" {
		return MessagesJSONNormalized(msgs, tools, opts)
	}
	norm, err := messagesapi.NormalizeMessagesForAPI(msgs, tools, opts)
	if err != nil {
		return nil, err
	}
	stage := norm
	if s := strings.TrimSpace(skillListingText); s != "" {
		stage = appendReminderUserAfterLastUserOrAppendIfNoUser(stage, s)
	}

	if s := strings.TrimSpace(userContextReminder); s != "" {
		stage = prependReminderUser(stage, s)
	}

	fin, err := messagesapi.NormalizeMessagesForAPI(stage, tools, opts)

	if err != nil {
		return nil, err
	}
	return messagesJSONFromNormalized(fin)
}

func prependReminderUser(msgs []types.Message, text string) []types.Message {
	return append([]types.Message{messagesapi.ReminderUserMessage(text, true)}, msgs...)
}

// appendReminderUserAfterLastUserOrAppendIfNoUser inserts a reminder user immediately after the last
// [types.MessageTypeUser] row (TS attachment order after the turn’s user message). If there is no user
// row, appends the reminder at the end of the slice (assistant-only transcripts).
func appendReminderUserAfterLastUserOrAppendIfNoUser(msgs []types.Message, text string) []types.Message {
	out := slices.Clone(msgs)
	insertAt := len(out)
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].Type == types.MessageTypeUser {
			insertAt = i + 1
			break
		}
	}
	rem := messagesapi.ReminderUserMessage(text, true)
	combined := make([]types.Message, 0, len(out)+1)
	combined = append(combined, out[:insertAt]...)
	combined = append(combined, rem)
	combined = append(combined, out[insertAt:]...)
	return combined
}

// MessagesJSONWithSkillListing inserts listing after the last user message (no user-context reminder).
// With a reminder, use [MessagesJSONWithLeadingMeta].
func MessagesJSONWithSkillListing(msgs []types.Message, listingUserText string, tools []messagesapi.ToolSpec, opts messagesapi.Options) (json.RawMessage, error) {
	return MessagesJSONWithLeadingMeta(msgs, "", listingUserText, tools, opts)
}

// InsertUserMessageAfterLastUserJSON inserts one user message with string content immediately after the last
// entry with role "user" in the messages array. If there is no user message, appends at the end.
// Empty trimmed text returns base unchanged.
func InsertUserMessageAfterLastUserJSON(base json.RawMessage, text string) (json.RawMessage, error) {
	t := strings.TrimSpace(text)
	if t == "" {
		return base, nil
	}
	msgs, err := apiMessagesJSONArrayToMessages(base)
	if err != nil {
		return nil, err
	}
	msgs = appendReminderUserAfterLastUserOrAppendIfNoUser(msgs, t)
	fin, err := messagesapi.NormalizeMessagesForAPI(msgs, nil, messagesapi.DefaultOptions())
	if err != nil {
		return nil, err
	}
	return messagesJSONFromNormalized(fin)
}

// PrependUserMessageJSON prepends one user message with string content (JSON string) to a messages array.
// Empty trimmed text returns base unchanged.
func PrependUserMessageJSON(base json.RawMessage, text string) (json.RawMessage, error) {
	t := strings.TrimSpace(text)
	if t == "" {
		return base, nil
	}
	msgs, err := apiMessagesJSONArrayToMessages(base)
	if err != nil {
		return nil, err
	}
	msgs = prependReminderUser(msgs, t)
	fin, err := messagesapi.NormalizeMessagesForAPI(msgs, nil, messagesapi.DefaultOptions())
	if err != nil {
		return nil, err
	}
	return messagesJSONFromNormalized(fin)
}

// apiMessagesJSONArrayToMessages converts an Anthropic-style [{role,content},...] JSON array into
// typed messages for [messagesapi.NormalizeMessagesForAPI].
func apiMessagesJSONArrayToMessages(base json.RawMessage) ([]types.Message, error) {
	var arr []apiMessage
	if err := json.Unmarshal(base, &arr); err != nil {
		return nil, err
	}
	out := make([]types.Message, 0, len(arr))
	for _, row := range arr {
		if row.Role != "user" && row.Role != "assistant" {
			continue
		}
		if len(row.Content) == 0 {
			continue
		}
		t := types.MessageTypeUser
		if row.Role == "assistant" {
			t = types.MessageTypeAssistant
		}
		inner, err := json.Marshal(struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}{Role: row.Role, Content: row.Content})
		if err != nil {
			return nil, err
		}
		out = append(out, types.Message{Type: t, Message: inner, Content: row.Content})
	}
	return out, nil
}

// MergeConsecutiveUserMessagesJSON merges adjacent entries with role "user" when both contents are
// JSON-encoded strings, joining with a single "\\n" (same seam as joinTextAtSeam for text+text in TS).
// Non-string content (e.g. block arrays) stops the merge run so those pairs stay separate.
func MergeConsecutiveUserMessagesJSON(base json.RawMessage) (json.RawMessage, error) {
	var arr []apiMessage
	if err := json.Unmarshal(base, &arr); err != nil {
		return nil, err
	}
	if len(arr) < 2 {
		return base, nil
	}
	out := make([]apiMessage, 0, len(arr))
	for i := 0; i < len(arr); i++ {
		cur := arr[i]
		if cur.Role != "user" {
			out = append(out, cur)
			continue
		}
		merged := cur.Content
		j := i + 1
		for j < len(arr) && arr[j].Role == "user" {
			next, ok := mergeAdjacentUserJSONStrings(merged, arr[j].Content)
			if !ok {
				break
			}
			merged = next
			j++
		}
		out = append(out, apiMessage{Role: "user", Content: merged})
		i = j - 1
	}
	return json.Marshal(out)
}

func mergeAdjacentUserJSONStrings(a, b json.RawMessage) (json.RawMessage, bool) {
	if !jsonRawIsString(a) || !jsonRawIsString(b) {
		return nil, false
	}
	var sa, sb string
	if err := json.Unmarshal(a, &sa); err != nil {
		return nil, false
	}
	if err := json.Unmarshal(b, &sb); err != nil {
		return nil, false
	}
	joined := sa + "\n" + sb
	raw, err := json.Marshal(joined)
	if err != nil {
		return nil, false
	}
	return raw, true
}

func jsonRawIsString(raw json.RawMessage) bool {
	return len(raw) >= 2 && raw[0] == '"'
}
