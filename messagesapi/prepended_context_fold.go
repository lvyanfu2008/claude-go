package messagesapi

import (
	"encoding/json"
	"strings"

	"goc/types"
)

// userMessageStartsWithSystemReminder is true when the user row's primary text begins with
// "<system-reminder>", matching prependUserContext / api.ts user-context injection.
func userMessageStartsWithSystemReminder(m *types.Message) (bool, error) {
	inner, err := getInner(m)
	if err != nil {
		return false, err
	}
	var str string
	if err := json.Unmarshal(inner.Content, &str); err == nil {
		return strings.HasPrefix(str, "<system-reminder>"), nil
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil || len(blocks) == 0 {
		return false, err
	}
	if t, _ := blocks[0]["type"].(string); t != "text" {
		return false, nil
	}
	tx, _ := blocks[0]["text"].(string)
	return strings.HasPrefix(tx, "<system-reminder>"), nil
}

// foldLeadingPrependedUserContextMetaIntoTrailingPlainUser mirrors TS mergeUserMessages semantics
// (joinTextAtSeam + hoistToolResults) for the case TS prependUserContext creates:
//
//	[ meta user (isMeta, <system-reminder>…), …, assistant(s) …, trailing plain user ]
//
// The main normalize loop only merges consecutive users, so meta stays separated from the
// current-turn user until this pass. Adjacent [meta, user] is already merged in that loop — this
// only runs when there is at least one assistant between the leading meta prefix and the tail user.
func foldLeadingPrependedUserContextMetaIntoTrailingPlainUser(messages []types.Message, opts Options) ([]types.Message, error) {
	if len(messages) < 3 {
		return messages, nil
	}
	last := len(messages) - 1
	if messages[last].Type != types.MessageTypeUser || isTruthy(messages[last].IsMeta) {
		return messages, nil
	}

	var fold []int
	for i := 0; i < len(messages); i++ {
		if messages[i].Type != types.MessageTypeUser {
			break
		}
		if !isTruthy(messages[i].IsMeta) {
			break
		}
		ok, err := userMessageStartsWithSystemReminder(&messages[i])
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		fold = append(fold, i)
	}
	if len(fold) == 0 {
		return messages, nil
	}
	maxFold := fold[len(fold)-1]
	if maxFold >= last {
		return messages, nil
	}
	hasAssistant := false
	for i := maxFold + 1; i < last; i++ {
		if messages[i].Type == types.MessageTypeAssistant {
			hasAssistant = true
			break
		}
	}
	if !hasAssistant {
		return messages, nil
	}

	merged := messages[last]
	var err error
	for i := len(fold) - 1; i >= 0; i-- {
		merged, err = mergeUserMessages(messages[fold[i]], merged, opts)
		if err != nil {
			return nil, err
		}
	}
	merged.IsMeta = messages[last].IsMeta
	syncTopLevelContent(&merged)

	skip := make(map[int]struct{}, len(fold))
	for _, i := range fold {
		skip[i] = struct{}{}
	}
	out := make([]types.Message, 0, len(messages)-len(fold))
	for i := 0; i < len(messages); i++ {
		if _, drop := skip[i]; drop {
			continue
		}
		if i == last {
			out = append(out, merged)
			continue
		}
		out = append(out, messages[i])
	}
	return out, nil
}
