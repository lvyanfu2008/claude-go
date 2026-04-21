package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"goc/diagnostics"
	"goc/growthbook"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/types"
)

const (
	envPuiDebugLog      = "GOC_PROCESS_USER_INPUT_DEBUG_LOG"
	envPuiDebugStderr   = "GOC_PROCESS_USER_INPUT_DEBUG_TO_STDERR"
	envProcessUserInDbg = "CLAUDE_DEBUG_PROCESS_USER_INPUT"
	previewMax          = 400
)

func isEnvTruthy(s string) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func isoTimestampMSUTC() string {
	t := time.Now().UTC()
	return fmt.Sprintf("%s.%03dZ",
		t.Format("2006-01-02T15:04:05"),
		t.Nanosecond()/1e6)
}

func previewForLog(s string) string {
	if len(s) <= previewMax {
		return s
	}
	return fmt.Sprintf("%s…[%d chars]", s[:previewMax], len(s))
}

func processUserInputDebugEnabled() bool {
	return isEnvTruthy(os.Getenv(envProcessUserInDbg))
}

func appendProcessUserInputDebugLineUnconditional(logPath string, toStderr bool, inner string) {
	line := fmt.Sprintf("%s [DEBUG] %s\n", isoTimestampMSUTC(), strings.TrimSpace(inner))
	if toStderr {
		_, _ = os.Stderr.WriteString(line)
		return
	}
	if strings.TrimSpace(logPath) == "" {
		return
	}
	dir := filepath.Dir(logPath)
	_ = os.MkdirAll(dir, 0o755)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

func appendProcessUserInputDebugLine(logPath string, toStderr bool, inner string) {
	if !processUserInputDebugEnabled() {
		return
	}
	appendProcessUserInputDebugLineUnconditional(logPath, toStderr, inner)
}

func logProcessUserInputDebug(logPath string, toStderr bool, stage string, payload map[string]any) {
	b, err := json.Marshal(payload)
	if err != nil {
		b = []byte(`{"error":"json_marshal_failed"}`)
	}
	appendProcessUserInputDebugLine(logPath, toStderr,
		fmt.Sprintf("[processUserInput:%s] %s", stage, string(b)))
}

// logToolUseContextForCLI writes ToolUseContext JSON when CLAUDE_CODE_LOG_TOOL_USE_CONTEXT or
// GOU_DEMO_LOG_TOOL_USE_CONTEXT is 1|summary|full (full = entire serializable context). Payload json field is compact (no indent).
// If CLAUDE_DEBUG_PROCESS_USER_INPUT is off, still writes when GOC_PROCESS_USER_INPUT_DEBUG_LOG is set,
// GOC_PROCESS_USER_INPUT_DEBUG_TO_STDERR=1, or as a last resort to stderr.
func logToolUseContextForCLI(logPath string, toStderr bool, rc *types.ProcessUserInputContextData) {
	raw := strings.TrimSpace(os.Getenv("CLAUDE_CODE_LOG_TOOL_USE_CONTEXT"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("GOU_DEMO_LOG_TOOL_USE_CONTEXT"))
	}
	if raw == "" {
		return
	}
	tv := strings.ToLower(raw)
	full := tv == "full"
	if !full && tv != "1" && tv != "true" && tv != "yes" && tv != "on" && tv != "summary" {
		return
	}
	if rc == nil {
		return
	}
	b, err := types.FormatProcessInputContextForLog(rc, full)
	mode := "summary"
	if full {
		mode = "full"
	}
	payload := map[string]any{"mode": mode}
	if err != nil {
		payload["error"] = err.Error()
	} else {
		payload["json"] = string(b)
	}
	lineB, mErr := json.Marshal(payload)
	if mErr != nil {
		lineB = []byte(`{"error":"marshal_tool_context_payload"}`)
	}
	line := fmt.Sprintf("[processUserInput:TOOL_USE_CONTEXT] %s", string(lineB))
	if processUserInputDebugEnabled() {
		appendProcessUserInputDebugLine(logPath, toStderr, line)
		return
	}
	if toStderr || strings.TrimSpace(logPath) != "" {
		appendProcessUserInputDebugLineUnconditional(logPath, toStderr, line)
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s [DEBUG] %s\n", isoTimestampMSUTC(), strings.TrimSpace(line))
}

func inputKindAndPreview(raw json.RawMessage) (kind string, preview string, blockCount *int) {
	s := strings.TrimSpace(string(raw))
	if len(s) == 0 {
		return "empty", "", nil
	}
	if s[0] == '"' {
		var str string
		if json.Unmarshal(raw, &str) == nil {
			return "string", previewForLog(str), nil
		}
		return "string", previewForLog(s), nil
	}
	var blocks []json.RawMessage
	if json.Unmarshal(raw, &blocks) == nil {
		n := len(blocks)
		types := make([]string, 0, n)
		for _, b := range blocks {
			var head struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(b, &head)
			if head.Type != "" {
				types = append(types, head.Type)
			} else {
				types = append(types, "?")
			}
		}
		return "blocks", previewForLog(strings.Join(types, ",")), &n
	}
	return "unknown", previewForLog(s), nil
}

func buildInPayload(args *processuserinput.ProcessUserInputArgs) map[string]any {
	kind, prev, nBlocks := inputKindAndPreview(args.Input)
	m := map[string]any{
		"inputKind":   kind,
		"inputPreview": prev,
		"mode":        args.Mode,
		"querySource": args.QuerySource,
	}
	if nBlocks != nil {
		m["blockCount"] = *nBlocks
	}
	if args.PreExpansionInput != nil && *args.PreExpansionInput != "" {
		m["preExpansionInput"] = previewForLog(*args.PreExpansionInput)
	}
	if args.SkipSlashCommands != nil {
		m["skipSlashCommands"] = *args.SkipSlashCommands
	}
	if args.BridgeOrigin != nil {
		m["bridgeOrigin"] = *args.BridgeOrigin
	}
	if args.IsMeta != nil {
		m["isMeta"] = *args.IsMeta
	}
	if args.UUID != nil {
		m["uuid"] = *args.UUID
	}
	m["skipAttachments"] = args.SkipAttachments
	m["hasPastedContents"] = len(args.PastedContents) > 0
	m["hasIdeSelection"] = args.IdeSelection != nil
	m["priorMessageCount"] = len(args.Messages)
	if args.IsAlreadyProcessing != nil {
		m["isAlreadyProcessing"] = *args.IsAlreadyProcessing
	}
	return m
}

func extractStringOrBlocksText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	var b strings.Builder
	for _, bl := range blocks {
		if bl.Type == "text" && bl.Text != "" {
			b.WriteString(bl.Text)
		}
	}
	return b.String()
}

func messageTextPreview(m types.Message) string {
	if txt := extractStringOrBlocksText(m.Content); txt != "" {
		return txt
	}
	if m.Type == types.MessageTypeAssistant && len(m.Message) > 0 {
		var inner struct {
			Content json.RawMessage `json:"content"`
		}
		if json.Unmarshal(m.Message, &inner) == nil {
			return extractStringOrBlocksText(inner.Content)
		}
	}
	return ""
}

func summarizeMessagesForLog(msgs []types.Message) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for i, m := range msgs {
		row := map[string]any{"i": i, "type": m.Type}
		switch m.Type {
		case types.MessageTypeUser, types.MessageTypeAssistant, types.MessageTypeSystem:
			if txt := messageTextPreview(m); txt != "" {
				row["textPreview"] = previewForLog(txt)
			}
		case types.MessageTypeAttachment:
			var att struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(m.Attachment, &att) == nil && att.Type != "" {
				row["attachmentType"] = att.Type
			}
		}
		out = append(out, row)
	}
	return out
}

func buildResultPayload(path string, r *processuserinput.ProcessUserInputBaseResult) map[string]any {
	if r == nil {
		return map[string]any{"error": "nil_result"}
	}
	p := map[string]any{
		"shouldQuery":     r.ShouldQuery,
		"messageCount":    len(r.Messages),
		"messagesSummary": summarizeMessagesForLog(r.Messages),
	}
	if path != "" {
		p["path"] = path
	}
	if r.ResultText != "" {
		p["resultText"] = previewForLog(r.ResultText)
	}
	if r.NextInput != "" {
		p["nextInput"] = previewForLog(r.NextInput)
	}
	if r.SubmitNextInput {
		p["submitNextInput"] = true
	}
	if r.Model != "" {
		p["model"] = r.Model
	}
	if r.Effort != nil {
		p["effort"] = r.Effort
	}
	if len(r.AllowedTools) > 0 {
		p["allowedTools"] = r.AllowedTools
	}
	if r.Execution != nil {
		p["executionKind"] = r.Execution.Kind
		if r.Execution.Input != "" {
			p["executionInput"] = previewForLog(r.Execution.Input)
		}
		if r.Execution.Command != "" {
			p["executionCommand"] = previewForLog(r.Execution.Command)
		}
		if r.Execution.CommandName != "" {
			p["executionCommandName"] = r.Execution.CommandName
		}
		if r.Execution.Args != "" {
			p["executionArgs"] = previewForLog(r.Execution.Args)
		}
		if r.Execution.RejectReason != "" {
			p["executionRejectReason"] = previewForLog(r.Execution.RejectReason)
		}
	}
	return p
}

// AnalyticsStderrPrefix prefixes JSON analytics lines on stderr (optional host consumption).
const AnalyticsStderrPrefix = "GOC_ANALYTICS_EVENT:"

func emitAnalyticsEventToStderr(name string, payload map[string]any) {
	b, err := json.Marshal(map[string]any{"name": name, "payload": payload})
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s%s\n", AnalyticsStderrPrefix, string(b))
}

// wireProcessUserInputCallbacks sets LogEvent (stderr) and optional debug checkpoint / event mirroring.
func wireProcessUserInputCallbacks(
	p *processuserinput.ProcessUserInputParams,
	logPath string,
	toStderr bool,
) {
	// Initialize analytics system
	diagnostics.InitAnalytics()

	// Initialize GrowthBook feature flags
	growthbook.Init()

	p.LogEvent = func(name string, payload map[string]any) {
		// Use new diagnostics package for analytics events
		diagnostics.EmitAnalyticsEvent(name, payload)

		// Keep backward compatibility with stderr output
		emitAnalyticsEventToStderr(name, payload)

		if processUserInputDebugEnabled() {
			wrapped := map[string]any{"name": name, "payload": payload}
			logProcessUserInputDebug(logPath, toStderr, "event", wrapped)
		}
	}
	if !processUserInputDebugEnabled() {
		return
	}
	p.QueryCheckpoint = func(label string) {
		logProcessUserInputDebug(logPath, toStderr, label, map[string]any{})
	}
}
