package tools

import (
	"context"
	"encoding/json"
	"time"
)

func dataUnavailable(tool, message string) (string, bool, error) {
	out := map[string]any{
		"data": map[string]any{
			"success": false,
			"tool":    tool,
			"message": message,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// TestingPermissionFromJSON matches TS TestingPermissionTool.call output shape.
func TestingPermissionFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	out := map[string]any{"data": "TestingPermission executed successfully"}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// SleepFromJSON waits up to duration_seconds (default 1, max 60). Kairos SleepTool is not in-tree; schema matches typical duration payloads.
func SleepFromJSON(ctx context.Context, raw []byte) (string, bool, error) {
	var in struct {
		DurationSeconds float64 `json:"duration_seconds"`
		Seconds         float64 `json:"seconds"`
	}
	_ = json.Unmarshal(raw, &in)
	sec := in.DurationSeconds
	if sec <= 0 {
		sec = in.Seconds
	}
	if sec <= 0 {
		sec = 1
	}
	if sec > 60 {
		sec = 60
	}
	d := time.Duration(sec * float64(time.Second))
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return "", true, ctx.Err()
	case <-timer.C:
	}
	out := map[string]any{
		"data": map[string]any{
			"slept_seconds": sec,
			"message":       "Sleep completed",
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// ListPeersFromJSON returns an empty peer list (no UDS inbox / bridge in Go runner).
func ListPeersFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	out := map[string]any{"data": []any{}}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// VerifyPlanExecutionFromJSON matches disabled VerifyPlanExecutionTool.js in this tree.
func VerifyPlanExecutionFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("VerifyPlanExecution", "VerifyPlanExecution is disabled in this build (use TS worker when enabled).")
}

// OverflowTestFromJSON feature tool not wired in Go runner.
func OverflowTestFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("OverflowTest", "OverflowTest is not implemented in the Go parity runner.")
}

// CtxInspectFromJSON feature tool not wired in Go runner.
func CtxInspectFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("CtxInspect", "CtxInspect is not implemented in the Go parity runner.")
}

// TerminalCaptureFromJSON feature tool not wired in Go runner.
func TerminalCaptureFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("TerminalCapture", "TerminalCapture is not implemented in the Go parity runner.")
}

// LSPFromJSON is not available without an LSP client in Go.
func LSPFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("LSP", "LSP tool is not implemented in the Go parity runner.")
}

// EnterWorktreeFromJSON — git worktree / session integration is TS-only here.
func EnterWorktreeFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("EnterWorktree", "EnterWorktree requires git worktree integration (use TS worker).")
}

// ExitWorktreeFromJSON — git worktree / session integration is TS-only here.
func ExitWorktreeFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("ExitWorktree", "ExitWorktree requires git worktree integration (use TS worker).")
}

// TeamCreateFromJSON — agent swarms not in Go runner.
func TeamCreateFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("TeamCreate", "TeamCreate requires TS agent swarms / team context.")
}

// TeamDeleteFromJSON — agent swarms not in Go runner.
func TeamDeleteFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("TeamDelete", "TeamDelete requires TS agent swarms / team context.")
}

// ConfigFromJSON — ant-only tool.
func ConfigFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("Config", "Config tool is not available in the Go parity runner (TS ant build).")
}

// TungstenFromJSON — ant-only tool.
func TungstenFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("Tungsten", "Tungsten tool is not available in the Go parity runner (TS ant build).")
}

// SuggestBackgroundPRFromJSON — ant feature tool.
func SuggestBackgroundPRFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("SuggestBackgroundPR", "SuggestBackgroundPR is not available in the Go parity runner.")
}

// WebBrowserFromJSON — browser automation not in Go runner.
func WebBrowserFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("WebBrowser", "WebBrowser is not available in the Go parity runner.")
}

// RemoteTriggerFromJSON — remote triggers not in Go runner.
func RemoteTriggerFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("RemoteTrigger", "RemoteTrigger is not available in the Go parity runner.")
}

// MonitorFromJSON — monitor tool not in Go runner.
func MonitorFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("Monitor", "Monitor is not available in the Go parity runner.")
}

// WorkflowFromJSON — workflow imports not in Go runner.
func WorkflowFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("Workflow", "Workflow is not available in the Go parity runner.")
}

// SnipFromJSON — feature import not in Go runner.
func SnipFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("Snip", "Snip is not available in the Go parity runner.")
}

// SendUserFileFromJSON — kairos path not in Go runner.
func SendUserFileFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("SendUserFile", "SendUserFile requires Kairos / TS upload path.")
}

// PushNotificationFromJSON — kairos path not in Go runner.
func PushNotificationFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("PushNotification", "PushNotification requires Kairos (TS worker).")
}

// SubscribePRFromJSON — kairos github path not in Go runner.
func SubscribePRFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return dataUnavailable("SubscribePR", "SubscribePR requires Kairos GitHub integration (TS worker).")
}
