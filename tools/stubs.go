package tools

import (
	"encoding/json"
	"fmt"
	"os"
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

// AgentFromJSON mirrors TS-style tool_result content: JSON with {data: {success, message}} (no subagent engine in Go runner).
func AgentFromJSON(raw []byte) (string, bool, error) {
	_ = raw
	msg := "Agent tool is not implemented in the Go ParityToolRunner (use TS socket worker or a future sub-turn engine)."
	out := map[string]any{"data": map[string]any{"success": false, "message": msg}}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// SendMessageFromJSON writes a message to a teammate's mailbox and returns unread responses.
// Supports:
//   - Plain text message to a named teammate (resolved via team roster)
//   - Broadcast (*) to all teammates
//   - Protocol messages: shutdown_request, plan_approval_response
func SendMessageFromJSON(raw []byte) (string, bool, error) {
	if !commands.AgentSwarmsEnabled() {
		msg := "SendMessage requires agent swarms to be enabled"
		out := map[string]any{"data": map[string]any{"success": false, "message": msg}}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	var in struct {
		To      string `json:"to"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}

	to := strings.TrimSpace(in.To)
	message := strings.TrimSpace(in.Message)

	if to == "" {
		return "", true, fmt.Errorf("to is required")
	}
	if message == "" {
		return "", true, fmt.Errorf("message is required")
	}

	senderName := strings.TrimSpace(getenv("CLAUDE_CODE_AGENT_NAME"))
	senderID := strings.TrimSpace(getenv("CLAUDE_CODE_AGENT_ID"))
	if senderName == "" {
		senderName = senderID
	}
	if senderName == "" {
		senderName = "unknown"
	}

	teamName := strings.TrimSpace(getenv("CLAUDE_CODE_TEAM_NAME"))

	// Broadcast: write to all team members
	if to == "*" {
		// Find all members from the team file
		if teamName == "" {
			// Try to find from env
			teamName = strings.TrimSpace(getenv("CLAUDE_CODE_TEAM_NAME"))
		}
		tf, err := readTeamFile(teamName)
		if err != nil || tf == nil {
			out := map[string]any{
				"data": map[string]any{
					"success": false,
					"message": "No team found for broadcast",
					"to":      to,
				},
			}
			b, _ := json.Marshal(out)
			return string(b), false, nil
		}

		sentCount := 0
		for _, m := range tf.Members {
			if m.AgentID == senderID || m.Name == senderName {
				continue // Skip self
			}
			targetName := m.Name
			if targetName == "" {
				targetName = m.AgentID
			}
			if err := writeToMailbox(targetName, teamName, senderName, message); err == nil {
				sentCount++
			}
		}

		out := map[string]any{
			"data": map[string]any{
				"success":   true,
				"message":   fmt.Sprintf("Broadcast sent to %d teammate(s)", sentCount),
				"sent_to":   sentCount,
				"team_name": teamName,
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	// Single target: resolve by name or agent ID
	resolvedTeam := teamName
	targetName := to

	// Try to find target in a team roster
	foundTeam, foundMember, err := findTeamMemberByName(to)
	if err == nil && foundMember != nil {
		targetName = foundMember.Name
		if targetName == "" {
			targetName = foundMember.AgentID
		}
		resolvedTeam = foundTeam
	}

	// Write message to target's mailbox
	if err := writeToMailbox(targetName, resolvedTeam, senderName, message); err != nil {
		out := map[string]any{
			"data": map[string]any{
				"success": false,
				"message": fmt.Sprintf("Failed to deliver message: %v", err),
				"to":      to,
			},
		}
		b, _ := json.Marshal(out)
		return string(b), false, nil
	}

	// Read back any unread messages from the target's mailbox
	msgs, _ := readMailbox(targetName, resolvedTeam)
	var unread []TeammateMessage
	for _, m := range msgs {
		if !m.Read {
			unread = append(unread, m)
		}
	}

	// Format unread messages as XML for LLM context
	var mailboxContext string
	if len(unread) > 0 {
		mailboxContext = formatTeammateMessages(unread)
	}

	out := map[string]any{
		"data": map[string]any{
			"success":           true,
			"message":           "Message delivered",
			"to":                to,
			"resolved_team":     resolvedTeam,
			"unread_count":      len(unread),
			"mailbox_context":   mailboxContext,
			"target_agent_name": targetName,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
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
