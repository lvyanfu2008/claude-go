package processuserinput

import (
	"encoding/json"
	"fmt"
	"os"

	"goc/types"
)

func debugProcessUserInputEnabled() bool {
	return envTruthy("CLAUDE_DEBUG_PROCESS_USER_INPUT")
}

// debugProcessUserInput mirrors logProcessUserInputDebug in processUserInput.ts (stderr JSON line).
func debugProcessUserInput(stage string, payload map[string]any) {
	if !debugProcessUserInputEnabled() {
		return
	}
	b, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[processUserInput:%s] <marshal error:%v>\n", stage, err)
		return
	}
	fmt.Fprintf(os.Stderr, "[processUserInput:%s] %s\n", stage, string(b))
}

func previewForDebugLog(s string, max int) string {
	if max <= 0 {
		max = 400
	}
	if len(s) <= max {
		return s
	}
	return fmt.Sprintf("%s…[%d chars]", s[:max], len(s))
}

func withDebugPath(path string, base map[string]any) map[string]any {
	out := make(map[string]any, len(base)+1)
	out["path"] = path
	for k, v := range base {
		out[k] = v
	}
	return out
}

func buildProcessUserInputDebugInPayload(
	p *ProcessUserInputParams,
	isString bool,
	blockCount int,
	text string,
) map[string]any {
	m := map[string]any{
		"inputKind": func() string {
			if isString {
				return "string"
			}
			return "blocks"
		}(),
		"mode": p.Mode,
	}
	if isString {
		m["inputPreview"] = previewForDebugLog(text, 400)
	} else {
		m["inputPreview"] = previewForDebugLog("(blocks)", 400)
		m["blockCount"] = blockCount
	}
	if p.PreExpansionInput != nil {
		m["preExpansionInput"] = previewForDebugLog(*p.PreExpansionInput, 400)
	}
	m["skipSlashCommands"] = boolVal(p.SkipSlashCommands)
	m["bridgeOrigin"] = boolVal(p.BridgeOrigin)
	m["isMeta"] = boolVal(p.IsMeta)
	m["uuid"] = p.UUID
	m["querySource"] = p.QuerySource
	m["skipAttachments"] = boolVal(p.SkipAttachments)
	m["hasPastedContents"] = p.PastedContents != nil && len(p.PastedContents) > 0
	m["hasIdeSelection"] = p.IdeSelection != nil
	if p.Messages != nil {
		m["priorMessageCount"] = len(p.Messages)
	}
	m["isAlreadyProcessing"] = boolVal(p.IsAlreadyProcessing)
	return m
}

func buildProcessUserInputDebugResultPayload(r *ProcessUserInputBaseResult) map[string]any {
	if r == nil {
		return map[string]any{}
	}
	out := map[string]any{
		"shouldQuery":     r.ShouldQuery,
		"submitNextInput": r.SubmitNextInput,
		"model":           r.Model,
		"effort":          r.Effort,
		"allowedTools":    r.AllowedTools,
		"messageCount":    len(r.Messages),
	}
	if r.ResultText != "" {
		out["resultText"] = previewForDebugLog(r.ResultText, 400)
	}
	if r.NextInput != "" {
		out["nextInput"] = previewForDebugLog(r.NextInput, 400)
	}
	var summaries []map[string]any
	for i, msg := range r.Messages {
		summaries = append(summaries, summarizeMessageForDebug(i, msg))
	}
	out["messagesSummary"] = summaries
	return out
}

func summarizeMessageForDebug(i int, m types.Message) map[string]any {
	base := map[string]any{"i": i, "type": m.Type}
	switch m.Type {
	case types.MessageTypeUser:
		if txt := userMessageTextPreview(m); txt != "" {
			base["textPreview"] = previewForDebugLog(txt, 400)
		}
	case types.MessageTypeAssistant:
		// optional: decode message.content text
	case types.MessageTypeSystem:
		var s string
		_ = json.Unmarshal(m.Content, &s)
		if s != "" {
			base["textPreview"] = previewForDebugLog(s, 400)
		}
	case types.MessageTypeAttachment:
		var att struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(m.Attachment, &att)
		base["attachmentType"] = att.Type
	}
	return base
}

func userMessageTextPreview(m types.Message) string {
	if len(m.Message) > 0 {
		var inner struct {
			Content any `json:"content"`
		}
		if json.Unmarshal(m.Message, &inner) == nil {
			switch c := inner.Content.(type) {
			case string:
				return c
			}
		}
	}
	if len(m.Content) > 0 {
		var s string
		if json.Unmarshal(m.Content, &s) == nil {
			return s
		}
	}
	return ""
}
