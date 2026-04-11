package messagesapi

import (
	"encoding/json"
	"strings"
	"time"

	"goc/types"
)

func normalizeUserTextContent(m types.Message) ([]map[string]any, error) {
	inner, err := getInner(&m)
	if err != nil {
		return nil, err
	}
	return parseContentArrayOrString(inner.Content)
}

func hoistToolResults(content []map[string]any) []map[string]any {
	var toolResults []map[string]any
	var otherBlocks []map[string]any
	for _, block := range content {
		if t, _ := block["type"].(string); t == "tool_result" {
			toolResults = append(toolResults, block)
		} else {
			otherBlocks = append(otherBlocks, block)
		}
	}
	out := make([]map[string]any, 0, len(content))
	out = append(out, toolResults...)
	out = append(out, otherBlocks...)
	return out
}

func joinTextAtSeam(a, b []map[string]any) []map[string]any {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	lastA := a[len(a)-1]
	firstB := b[0]
	lt, _ := lastA["type"].(string)
	ft, _ := firstB["type"].(string)
	if lt == "text" && ft == "text" {
		la, _ := lastA["text"].(string)
		// Match TS joinTextAtSeam (messages.ts): append "\n" only to a's last text block, then append all of b
		// unchanged — do not merge a's and b's text into one block (preserves startsWith('<system-reminder>') on
		// sibling boundaries; literal user text stays its own text block after prepend meta).
		out := make([]map[string]any, 0, len(a)+len(b))
		out = append(out, a[:len(a)-1]...)
		out = append(out, map[string]any{
			"type": "text",
			"text": la + "\n",
		})
		out = append(out, b...)
		return out
	}
	out := make([]map[string]any, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}

func mergeUserMessages(a, b types.Message, opts Options) (types.Message, error) {
	lastContent, err := normalizeUserTextContent(a)
	if err != nil {
		return types.Message{}, err
	}
	currentContent, err := normalizeUserTextContent(b)
	if err != nil {
		return types.Message{}, err
	}
	joined := joinTextAtSeam(lastContent, currentContent)
	hoisted := hoistToolResults(joined)
	contentRaw, err := marshalContentBlocks(hoisted)
	if err != nil {
		return types.Message{}, err
	}

	out := a
	inner, err := getInner(&out)
	if err != nil {
		return types.Message{}, err
	}
	inner.Role = "user"
	inner.Content = contentRaw
	if err := setInner(&out, inner); err != nil {
		return types.Message{}, err
	}

	if opts.HistorySnip && !opts.TestMode {
		bothMeta := isTruthy(a.IsMeta) && isTruthy(b.IsMeta)
		if bothMeta {
			t := true
			out.IsMeta = &t
		} else {
			out.IsMeta = nil
		}
		if isTruthy(a.IsMeta) {
			out.UUID = b.UUID
		} else {
			out.UUID = a.UUID
		}
		return out, nil
	}

	if isTruthy(a.IsMeta) {
		out.UUID = b.UUID
	} else {
		out.UUID = a.UUID
	}
	return out, nil
}

func mergeUserMessagesAndToolResults(a, b types.Message, opts Options) (types.Message, error) {
	lastContent, err := normalizeUserTextContent(a)
	if err != nil {
		return types.Message{}, err
	}
	currentContent, err := normalizeUserTextContent(b)
	if err != nil {
		return types.Message{}, err
	}
	mergedBlocks, err := mergeUserContentBlocks(lastContent, currentContent, opts)
	if err != nil {
		return types.Message{}, err
	}
	hoisted := hoistToolResults(mergedBlocks)
	contentRaw, err := marshalContentBlocks(hoisted)
	if err != nil {
		return types.Message{}, err
	}
	out := a
	inner, err := getInner(&out)
	if err != nil {
		return types.Message{}, err
	}
	inner.Role = "user"
	inner.Content = contentRaw
	if err := setInner(&out, inner); err != nil {
		return types.Message{}, err
	}
	if isTruthy(a.IsMeta) {
		out.UUID = b.UUID
	} else {
		out.UUID = a.UUID
	}
	return out, nil
}

func mergeAssistantMessages(a, b types.Message) (types.Message, error) {
	out := a
	innerA, err := getInner(&a)
	if err != nil {
		return types.Message{}, err
	}
	innerB, err := getInner(&b)
	if err != nil {
		return types.Message{}, err
	}
	blocksA, err := parseContentArrayOrString(innerA.Content)
	if err != nil {
		return types.Message{}, err
	}
	blocksB, err := parseContentArrayOrString(innerB.Content)
	if err != nil {
		return types.Message{}, err
	}
	merged := append(blocksA, blocksB...)
	raw, err := marshalContentBlocks(merged)
	if err != nil {
		return types.Message{}, err
	}
	innerA.Content = raw
	if err := setInner(&out, innerA); err != nil {
		return types.Message{}, err
	}
	return out, nil
}

func mergeUserContentBlocks(a, b []map[string]any, opts Options) ([]map[string]any, error) {
	if len(a) == 0 {
		return b, nil
	}
	lastBlock := a[len(a)-1]
	lt, _ := lastBlock["type"].(string)
	if lt != "tool_result" {
		out := make([]map[string]any, 0, len(a)+len(b))
		out = append(out, a...)
		out = append(out, b...)
		return out, nil
	}

	if !opts.ChairSermon {
		// Legacy smoosh: string tool_result + all-text siblings
		if _, ok := toolResultContentString(lastBlock); ok {
			allText := true
			for _, x := range b {
				if t, _ := x["type"].(string); t != "text" {
					allText = false
					break
				}
			}
			if allText {
				smooshed, err := smooshIntoToolResult(lastBlock, b)
				if err != nil || smooshed == nil {
					out := make([]map[string]any, 0, len(a)+len(b))
					out = append(out, a...)
					out = append(out, b...)
					return out, nil
				}
				out := make([]map[string]any, 0, len(a))
				out = append(out, a[:len(a)-1]...)
				out = append(out, smooshed)
				return out, nil
			}
		}
		out := make([]map[string]any, 0, len(a)+len(b))
		out = append(out, a...)
		out = append(out, b...)
		return out, nil
	}

	var toSmoosh []map[string]any
	var toolResults []map[string]any
	for _, x := range b {
		if t, _ := x["type"].(string); t == "tool_result" {
			toolResults = append(toolResults, x)
		} else {
			toSmoosh = append(toSmoosh, x)
		}
	}
	if len(toSmoosh) == 0 {
		out := make([]map[string]any, 0, len(a)+len(b))
		out = append(out, a...)
		out = append(out, b...)
		return out, nil
	}
	smooshed, err := smooshIntoToolResult(lastBlock, toSmoosh)
	if err != nil || smooshed == nil {
		out := make([]map[string]any, 0, len(a)+len(b))
		out = append(out, a...)
		out = append(out, b...)
		return out, nil
	}
	out := make([]map[string]any, 0, len(a)+len(toolResults))
	out = append(out, a[:len(a)-1]...)
	out = append(out, smooshed)
	out = append(out, toolResults...)
	return out, nil
}

func toolResultContentString(tr map[string]any) (string, bool) {
	c := tr["content"]
	if s, ok := c.(string); ok {
		return s, true
	}
	return "", false
}

func smooshIntoToolResult(tr map[string]any, blocks []map[string]any) (map[string]any, error) {
	if len(blocks) == 0 {
		return tr, nil
	}
	existing := tr["content"]
	if arr, ok := existing.([]any); ok {
		for _, x := range arr {
			if isToolReferenceBlock(x) {
				return nil, nil
			}
		}
	}
	isErr, _ := tr["is_error"].(bool)
	if isErr {
		var textOnly []map[string]any
		for _, b := range blocks {
			if t, _ := b["type"].(string); t == "text" {
				textOnly = append(textOnly, b)
			}
		}
		if len(textOnly) == 0 {
			return tr, nil
		}
		blocks = textOnly
	}

	allText := true
	for _, b := range blocks {
		if t, _ := b["type"].(string); t != "text" {
			allText = false
			break
		}
	}

	// TS: allText && (existing === undefined || typeof existing === 'string')
	if allText {
		if existing == nil {
			var parts []string
			for _, b := range blocks {
				t, _ := b["text"].(string)
				t = strings.TrimSpace(t)
				if t != "" {
					parts = append(parts, t)
				}
			}
			joined := joinNonEmpty(parts, "\n\n")
			cp := cloneMapShallow(tr)
			cp["content"] = joined
			return cp, nil
		}
		if s, ok := existing.(string); ok {
			var parts []string
			if strings.TrimSpace(s) != "" {
				parts = append(parts, strings.TrimSpace(s))
			}
			for _, b := range blocks {
				t, _ := b["text"].(string)
				t = strings.TrimSpace(t)
				if t != "" {
					parts = append(parts, t)
				}
			}
			joined := joinNonEmpty(parts, "\n\n")
			cp := cloneMapShallow(tr)
			cp["content"] = joined
			return cp, nil
		}
	}

	base, err := toolResultContentToArray(existing)
	if err != nil {
		return nil, err
	}
	merged := mergeAdjacentTextBlocks(append(base, blocksToAny(blocks)...))
	cp := cloneMapShallow(tr)
	cp["content"] = merged
	return cp, nil
}

func joinNonEmpty(parts []string, sep string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return ""
	}
	r := out[0]
	for i := 1; i < len(out); i++ {
		r += sep + out[i]
	}
	return r
}

func cloneMapShallow(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func blocksToAny(blocks []map[string]any) []any {
	out := make([]any, len(blocks))
	for i := range blocks {
		out[i] = blocks[i]
	}
	return out
}

func toolResultContentToArray(existing any) ([]any, error) {
	if existing == nil {
		return nil, nil
	}
	if s, ok := existing.(string); ok {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		return []any{map[string]any{"type": "text", "text": s}}, nil
	}
	if arr, ok := existing.([]any); ok {
		return append([]any(nil), arr...), nil
	}
	return nil, nil
}

func mergeAdjacentTextBlocks(blocks []any) []any {
	var merged []any
	for _, b := range blocks {
		m, ok := b.(map[string]any)
		if !ok {
			merged = append(merged, b)
			continue
		}
		if t, _ := m["type"].(string); t != "text" {
			merged = append(merged, b)
			continue
		}
		ts, _ := m["text"].(string)
		txt := strings.TrimSpace(ts)
		if txt == "" {
			continue
		}
		if len(merged) > 0 {
			if pm, ok := merged[len(merged)-1].(map[string]any); ok {
				if pt, _ := pm["type"].(string); pt == "text" {
					prev, _ := pm["text"].(string)
					prev = strings.TrimSpace(prev)
					pm["text"] = prev + "\n\n" + txt
					continue
				}
			}
		}
		merged = append(merged, map[string]any{"type": "text", "text": txt})
	}
	return merged
}

func mergeAdjacentUserMessages(msgs []types.Message, opts Options) ([]types.Message, error) {
	if len(msgs) == 0 {
		return msgs, nil
	}
	out := []types.Message{msgs[0]}
	for i := 1; i < len(msgs); i++ {
		m := msgs[i]
		prev := &out[len(out)-1]
		if m.Type == types.MessageTypeUser && prev.Type == types.MessageTypeUser {
			merged, err := mergeUserMessages(*prev, m, opts)
			if err != nil {
				return nil, err
			}
			out[len(out)-1] = merged
		} else {
			out = append(out, m)
		}
	}
	return out, nil
}

func createUserMessageFromContent(content json.RawMessage, uuid string, timestamp string, isMeta bool) types.Message {
	if timestamp == "" {
		timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	inner := userOrAssistantInner{
		Role:    "user",
		Content: content,
	}
	msgJSON, _ := json.Marshal(inner)
	m := types.Message{
		Type:      types.MessageTypeUser,
		UUID:      uuid,
		Message:   msgJSON,
		Content:   content,
		Timestamp: &timestamp,
	}
	if isMeta {
		t := true
		m.IsMeta = &t
	}
	return m
}

func createUserMessageString(text string, uuid string, timestamp string, isMeta bool) types.Message {
	if text == "" {
		text = noContentMessage
	}
	raw, _ := json.Marshal(text)
	return createUserMessageFromContent(raw, uuid, timestamp, isMeta)
}

// ReminderUserMessage returns a user message with JSON-string content (plain text) for prependUserContext
// or skill_listing reminders. Pass non-empty text (callers should trim); isMeta true matches TS meta rows.
func ReminderUserMessage(text string, isMeta bool) types.Message {
	return createUserMessageString(text, randomUUID(), "", isMeta)
}
