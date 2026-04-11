// Package ccbhydrate builds JSON message arrays for ccb-engine SubmitUserTurn / HydrateFromMessages
// (goc/ccb-engine/internal/anthropic.Message: role + content string or block array).
package ccbhydrate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goc/ccb-engine/diaglog"
	"slices"

	"goc/commands"
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
		content := m.Content
		if m.Type == types.MessageTypeUser {
			if c, ok := messagesapi.UserAllTextContentAsJSONString(m.Content); ok {
				content = c
			}
		}
		out = append(out, apiMessage{Role: role, Content: content})
	}
	if len(out) == 0 {
		return json.RawMessage("[]"), nil
	}
	return marshalJSONNoEscapeHTML(out)
}

// marshalJSONNoEscapeHTML matches JSON.stringify: do not escape < > & in strings (Go's json.Marshal does by default).
func marshalJSONNoEscapeHTML(v any) (json.RawMessage, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// MessagesJSONWithLeadingMeta mirrors TS [query.ts] prependUserContext + [processTextPrompt] attachment
// order, then [messagesapi.NormalizeMessagesForAPI]: skill_listing is appended after the **last** transcript
// user (same turn as TS processTextPrompt: [userMessage, ...attachmentMessages]), then prependUserContext
// prepends the meta user at the front (TS query callModel). [messagesapi.NormalizeMessagesForAPI] merges
// consecutive users with joinTextAtSeam and attachments with mergeUserContentBlocks (sibling text blocks
// preserved like TS). [messagesJSONFromNormalized] then encodes all-text-only user `content` as a JSON
// string (concatenated text blocks) when building the API array, matching common Claude Code / Anthropic
// wire dumps. Set [messagesapi.Options.CompactAllTextUserContent] true to collapse each all-text user row
// before projection.
//
// Diagnostic: set CLAUDE_CODE_GO_MESSAGESJSON_STAGE_LOG=1 to append the pre-normalize [stage] JSON to the diag log
// (see [diaglog.Line]; same path as other Claude debug lines, truncated at 32KiB).
func MessagesJSONWithLeadingMeta(msgs []types.Message, userContextReminder, skillListingText string, tools []messagesapi.ToolSpec, opts messagesapi.Options) (json.RawMessage, error) {
	skill := strings.TrimSpace(skillListingText)
	ctx := strings.TrimSpace(userContextReminder)
	if skill == "" && ctx == "" {
		return MessagesJSONNormalized(msgs, tools, opts)
	}
	stage := slices.Clone(msgs)
	switch {
	case skill != "" && ctx != "":
		stage = appendSkillListingAttachmentAfterLastUser(stage, skillListingText)
		stage = prependReminderUser(stage, userContextReminder)
	case skill != "":
		stage = appendSkillListingAttachmentAfterLastUser(stage, skillListingText)
	default:
		stage = prependReminderUser(stage, userContextReminder)
	}

	fin, err := messagesapi.NormalizeMessagesForAPI(stage, tools, opts)
	if err != nil {
		return nil, err
	}
	return messagesJSONFromNormalized(fin)
}

const messagesJSONStageLogMaxBytes = 32 * 1024

// logMessagesJSONWithLeadingMetaStage appends the pre-normalize [stage] slice to the diag log when
// CLAUDE_CODE_GO_MESSAGESJSON_STAGE_LOG is truthy (1/true/yes/on). JSON is truncated after 32KiB.
func logMessagesJSONWithLeadingMetaStage(stage []types.Message) {
	raw, err := json.Marshal(stage)
	if err != nil {
		diaglog.Line("[goc/ccbhydrate] MessagesJSONWithLeadingMeta stage: json.Marshal err=%v len=%d", err, len(stage))
		return
	}
	s := string(raw)
	if len(s) > messagesJSONStageLogMaxBytes {
		s = s[:messagesJSONStageLogMaxBytes] + "…(truncated)"
	}
	diaglog.Line("[goc/ccbhydrate] MessagesJSONWithLeadingMeta stage len=%d json=%s", len(stage), s)
}

func prependReminderUser(msgs []types.Message, text string) []types.Message {
	return append([]types.Message{messagesapi.ReminderUserMessage(text, true)}, msgs...)
}

func skillListingAttachmentMessage(listingAPIUserText string) (types.Message, bool) {
	inner := skillListingAttachmentInner(listingAPIUserText)
	if strings.TrimSpace(inner) == "" {
		return types.Message{}, false
	}
	att, err := json.Marshal(map[string]string{
		"type":    "skill_listing",
		"content": inner,
	})
	if err != nil {
		return types.Message{}, false
	}
	return types.Message{Type: types.MessageTypeAttachment, Attachment: att}, true
}

// SkillListingStoreMessage builds an attachment row for [conversation.Store].Messages, matching TS
// QueryEngine push of type attachment + skill_listing before the assistant reply for that turn.
// Use the same string as [commands.AppendSkillListingForAPI]. Persist only after [MessagesJSONWithLeadingMeta]
// succeeds so hydrate does not see a duplicate listing in the same stage.
func SkillListingStoreMessage(listingAPIUserText string) (types.Message, bool) {
	m, ok := skillListingAttachmentMessage(listingAPIUserText)
	if !ok {
		return types.Message{}, false
	}
	if strings.TrimSpace(m.UUID) == "" {
		m.UUID = fmt.Sprintf("sl-%d", time.Now().UnixNano())
	}
	return m, true
}

// skillListingAttachmentInner strips the API reminder wrapper from [commands.AppendSkillListingForAPI] output
// (or equivalent) to recover the `content` field for a `skill_listing` attachment.
func skillListingAttachmentInner(apiUserText string) string {
	s := strings.TrimSpace(apiUserText)
	const open = "<system-reminder>\n"
	const close = "\n</system-reminder>"
	if strings.HasPrefix(s, open) && strings.HasSuffix(s, close) {
		s = strings.TrimSpace(s[len(open) : len(s)-len(close)])
	}
	if strings.HasPrefix(s, commands.SkillListingBodyPrefix) {
		return strings.TrimPrefix(s, commands.SkillListingBodyPrefix)
	}
	return s
}

// appendSkillListingAttachmentAfterLastUser appends a skill_listing attachment row immediately after the last
// [types.MessageTypeUser] row (TS processTextPrompt: attachments after the current user message).
func appendSkillListingAttachmentAfterLastUser(msgs []types.Message, listingAPIUserText string) []types.Message {
	attMsg, ok := skillListingAttachmentMessage(listingAPIUserText)
	if !ok {
		return msgs
	}
	i := lastUserMessageIndex(msgs)
	if i < 0 {
		return append(slices.Clone(msgs), attMsg)
	}
	out := slices.Clone(msgs[:i+1])
	out = append(out, attMsg)
	out = append(out, slices.Clone(msgs[i+1:])...)
	return out
}

// lastUserMessageIndex returns the index of the last [types.MessageTypeUser] row, or -1 if none.
func lastUserMessageIndex(msgs []types.Message) int {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type == types.MessageTypeUser {
			return i
		}
	}
	return -1
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
