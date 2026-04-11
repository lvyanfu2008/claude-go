package engine

import (
	"context"
	"encoding/json"

	"goc/conversation-runtime/streamingtool"
	"goc/toolexecution"
	"goc/types"
)

// ToolexecutionRunner implements [ToolRunner] by delegating each call to [toolexecution.RunToolUseChan],
// sharing permission and execution behavior with the query streaming parity path (optional A5).
type ToolexecutionRunner struct {
	Deps toolexecution.ExecutionDeps
}

// Run implements [ToolRunner].
func (t ToolexecutionRunner) Run(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
	in := append(json.RawMessage(nil), input...)
	block := streamingtool.ToolUseBlock{ID: toolUseID, Name: name, Input: in}
	asst := types.Message{Type: types.MessageTypeAssistant, UUID: "toolexec-engine-runner"}
	ch := toolexecution.RunToolUseChan(ctx, block, asst, t.Deps, nil)
	var lastContent string
	var lastIsErr bool
	for u := range ch {
		if u.Message == nil {
			continue
		}
		c, isErr := firstToolResultFromUserMessage(u.Message)
		if c != "" {
			lastContent = c
			lastIsErr = isErr
		}
	}
	return lastContent, lastIsErr, nil
}

func firstToolResultFromUserMessage(m *types.Message) (content string, isError bool) {
	if m == nil || m.Type != types.MessageTypeUser {
		return "", false
	}
	var wrap struct {
		Role    string `json:"role"`
		Content []struct {
			Type      string `json:"type"`
			Content   string `json:"content"`
			IsError   bool   `json:"is_error"`
			ToolUseID string `json:"tool_use_id"`
		} `json:"content"`
	}
	if err := json.Unmarshal(m.Message, &wrap); err != nil {
		return "", false
	}
	for _, b := range wrap.Content {
		if b.Type == "tool_result" {
			return b.Content, b.IsError
		}
	}
	return "", false
}
