// Package protocol holds TS↔Go v1 shapes and state revision helpers (see spec/protocol-v1.md).
package protocol

const Version = "ccb-engine-v1"

// StreamEvent is a server→client event (future socket / logging).
type StreamEvent struct {
	Type string `json:"type"`

	Text string `json:"text,omitempty"`

	ID         string         `json:"id,omitempty"`
	Name       string         `json:"name,omitempty"`
	Input      map[string]any `json:"input,omitempty"`
	ToolUseID  string         `json:"tool_use_id,omitempty"`
	CallID     string         `json:"call_id,omitempty"`
	Content    string         `json:"content,omitempty"`
	StateRev   uint64         `json:"state_rev,omitempty"`
	StopReason string         `json:"stop_reason,omitempty"`
	Code       string         `json:"code,omitempty"`
	Message    string         `json:"message,omitempty"`

	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`

	IsError bool `json:"is_error,omitempty"`

	// Policy is optional metadata on execute_tool (e.g. {"decision":"allow","source":"ccb-engine"}).
	Policy map[string]any `json:"policy,omitempty"`
}

func AssistantDelta(text string) StreamEvent {
	return StreamEvent{Type: "assistant_delta", Text: text}
}

func ToolUse(id, name string, input map[string]any) StreamEvent {
	return StreamEvent{Type: "tool_use", ID: id, Name: name, Input: input}
}

func ToolResult(toolUseID, content string, isError bool) StreamEvent {
	return StreamEvent{Type: "tool_result", ToolUseID: toolUseID, Content: content, IsError: isError}
}

func TurnComplete(stateRev uint64, stopReason string) StreamEvent {
	return StreamEvent{Type: "turn_complete", StateRev: stateRev, StopReason: stopReason}
}

func ErrEvent(code, msg string) StreamEvent {
	return StreamEvent{Type: "error", Code: code, Message: msg}
}

func Usage(in, out int) StreamEvent {
	return StreamEvent{Type: "usage", InputTokens: in, OutputTokens: out}
}

// ResponseEnd marks the end of one SubmitUserTurn response (NDJSON stream terminator).
func ResponseEnd(requestID string) StreamEvent {
	return StreamEvent{Type: "response_end", ID: requestID}
}

// ExecuteTool is server→client: ask TS to run a tool and reply with a ToolResult line.
// policy may be nil; when non-empty it is JSON-encoded on the wire (omitempty skips empty maps).
func ExecuteTool(callID, toolUseID, name string, input map[string]any, stateRev uint64, policy map[string]any) StreamEvent {
	ev := StreamEvent{
		Type:      "execute_tool",
		CallID:    callID,
		ToolUseID: toolUseID,
		Name:      name,
		Input:     input,
		StateRev:  stateRev,
	}
	if len(policy) > 0 {
		ev.Policy = policy
	}
	return ev
}

// PromptRequest matches Claude Code hook stdout line protocol (src/types/hooks.ts).
type PromptRequest struct {
	Prompt  string `json:"prompt"`
	Message string `json:"message"`
	Options []struct {
		Key         string `json:"key"`
		Label       string `json:"label"`
		Description string `json:"description,omitempty"`
	} `json:"options"`
}

// PromptResponse is written back to hook stdin (one JSON line).
type PromptResponse struct {
	PromptResponse string `json:"prompt_response"`
	Selected       string `json:"selected"`
}
