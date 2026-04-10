package paritytools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AgentStubFromJSON returns a tool error string (plan P6 path a: explicit unsupported).
func AgentStubFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return "Agent tool is not implemented in the Go ParityToolRunner (use TS socket worker or a future sub-turn engine).", true, nil
}

// SendMessageStubFromJSON — teammate mailbox not available in Go runner.
func SendMessageStubFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	return "SendMessage is not implemented in the Go runner (requires TS teammate / mailbox / bridge).", true, nil
}

// BriefFromJSON records a user-visible message path: returns JSON echo (headless transcript hint).
func BriefFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		Message     string   `json:"message"`
		Attachments []string `json:"attachments"`
		Status      string   `json:"status"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if strings.TrimSpace(in.Message) == "" {
		return "", true, fmt.Errorf("message is required")
	}
	st := strings.TrimSpace(in.Status)
	if st != "normal" && st != "proactive" {
		return "", true, fmt.Errorf("status must be normal or proactive")
	}
	out := map[string]any{
		"message":     in.Message,
		"attachments": in.Attachments,
		"sentAt":      "",
		"note":        "Go runner Brief/SendUserMessage: returned as structured tool_result only (no UI routing).",
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// ListMcpResourcesStub returns structured “no MCP” error.
func ListMcpResourcesStub(raw []byte) (string, bool, error) {
	_ = raw
	return "", true, fmt.Errorf("ListMcpResourcesTool: no MCP client in Go runner (use TS worker or future Go MCP client)")
}

// ReadMcpResourceStub returns structured “no MCP” error.
func ReadMcpResourceStub(raw []byte) (string, bool, error) {
	_ = raw
	return "", true, fmt.Errorf("ReadMcpResourceTool: no MCP client in Go runner (use TS worker or future Go MCP client)")
}
