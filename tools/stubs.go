package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// EchoStubFromJSON is a tiny echo tool for Go tests and demos (not a TS built-in).
func EchoStubFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	return in.Message, false, nil
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
	sentAt := time.Now().UTC().Format(time.RFC3339Nano)
	var data map[string]any
	if len(in.Attachments) == 0 {
		data = map[string]any{"message": in.Message, "sentAt": sentAt}
	} else {
		data = map[string]any{"message": in.Message, "attachments": in.Attachments, "sentAt": sentAt}
	}
	out := map[string]any{"data": data}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// ListMcpResourcesFromJSON mirrors ListMcpResourcesTool.call with zero MCP clients (TS returns {data: []}).
// If server is set, TS throws when no client matches — same error text as TS.
func ListMcpResourcesFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		Server string `json:"server"`
	}
	_ = json.Unmarshal(raw, &in)
	target := strings.TrimSpace(in.Server)
	if target != "" {
		return "", true, fmt.Errorf(`Server "%s" not found. Available servers: `, target)
	}
	out := map[string]any{"data": []any{}}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// ReadMcpResourceFromJSON mirrors ReadMcpResourceTool when no MCP client exists for server (TS throws).
func ReadMcpResourceFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		Server string `json:"server"`
		URI    string `json:"uri"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	srv := strings.TrimSpace(in.Server)
	if srv == "" {
		return "", true, fmt.Errorf("server is required")
	}
	if strings.TrimSpace(in.URI) == "" {
		return "", true, fmt.Errorf("uri is required")
	}
	return "", true, fmt.Errorf(`Server "%s" not found. Available servers: `, srv)
}
