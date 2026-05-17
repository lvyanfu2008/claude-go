package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"regexp"

	"github.com/gofrs/flock"
	"goc/commands"
)

var pathSanitize = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// Teammate message type constants (TS structured protocol messages).
const (
	TeammateMsgTypePermissionRequest  = "permission_request"
	TeammateMsgTypePermissionResponse = "permission_response"
	TeammateMsgTypeShutdown           = "shutdown"
	TeammateMsgTypePlanApproval       = "plan_approval"
	TeammateMsgTypeIdleNotification   = "idle_notification"
	TeammateMsgTypeTaskAssignment     = "task_assignment"
	TeammateMsgTypeModeSet            = "mode_set"
	TeammateMsgTypeTaskNotification   = "task_notification"
)

func sanitizePathComponent(s string) string {
	return pathSanitize.ReplaceAllString(s, "-")
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// TeammateMessage mirrors TS TeammateMessage type.
type TeammateMessage struct {
	From      string          `json:"from"`
	Text      string          `json:"text"`
	Timestamp string          `json:"timestamp"`
	Read      bool            `json:"read"`
	Color     string          `json:"color,omitempty"`
	Summary   string          `json:"summary,omitempty"`
	MsgType   string          `json:"msgType,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// getTeamsDir returns the teams directory under Claude config home (TS getTeamsDir).
func getTeamsDir() string {
	return filepath.Join(commands.ClaudeConfigHome(), "teams")
}

// getInboxPath returns the path to a teammate's inbox file (TS getInboxPath).
// Agent and team names are sanitized for filesystem safety.
func getInboxPath(agentName, teamName string) string {
	team := strings.TrimSpace(teamName)
	if team == "" {
		team = getenv("CLAUDE_CODE_TEAM_NAME")
	}
	if team == "" {
		team = "default"
	}
	safeTeam := sanitizePathComponent(team)
	safeAgent := sanitizePathComponent(agentName)
	return filepath.Join(getTeamsDir(), safeTeam, "inboxes", safeAgent+".json")
}

// ensureInboxDir ensures the inbox directory exists for a team (TS ensureInboxDir).
func ensureInboxDir(teamName string) error {
	team := strings.TrimSpace(teamName)
	if team == "" {
		team = getenv("CLAUDE_CODE_TEAM_NAME")
	}
	if team == "" {
		team = "default"
	}
	safeTeam := sanitizePathComponent(team)
	dir := filepath.Join(getTeamsDir(), safeTeam, "inboxes")
	return os.MkdirAll(dir, 0o700)
}

// readMailbox reads all messages from a teammate's inbox (TS readMailbox).
// Returns messages sorted by timestamp ascending.
func readMailbox(agentName, teamName string) ([]TeammateMessage, error) {
	p := getInboxPath(agentName, teamName)

	// Lock the inbox file for reading
	lock := flock.New(p + ".lock")
	if err := lock.Lock(); err != nil {
		return nil, fmt.Errorf("lock inbox: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return []TeammateMessage{}, nil
		}
		return nil, fmt.Errorf("read inbox: %w", err)
	}

	var msgs []TeammateMessage
	if err := json.Unmarshal(b, &msgs); err != nil {
		// Corrupted file — treat as empty
		return []TeammateMessage{}, nil
	}
	if msgs == nil {
		return []TeammateMessage{}, nil
	}
	return msgs, nil
}

// writeToMailbox writes a message to a teammate's inbox with file locking (TS writeToMailbox).
func writeToMailbox(agentName, teamName, from, text string) error {
	if err := ensureInboxDir(teamName); err != nil {
		return fmt.Errorf("ensure inbox dir: %w", err)
	}

	p := getInboxPath(agentName, teamName)

	// Lock the inbox file for writing
	lock := flock.New(p + ".lock")
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("lock inbox: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	// Read existing messages
	var msgs []TeammateMessage
	b, err := os.ReadFile(p)
	if err == nil {
		_ = json.Unmarshal(b, &msgs)
	}
	if msgs == nil {
		msgs = []TeammateMessage{}
	}

	msg := TeammateMessage{
		From:      from,
		Text:      text,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Read:      false,
	}
	if len(text) > 60 {
		msg.Summary = text[:57] + "..."
	}

	msgs = append(msgs, msg)

	out, err := json.MarshalIndent(msgs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}
	return os.WriteFile(p, append(out, '\n'), 0o600)
}

// WriteStructuredMessage writes a typed message to a teammate's inbox (TS writeToMailbox with type).
func WriteStructuredMessage(agentName, teamName, from, msgType string, payload json.RawMessage) error {
	if err := ensureInboxDir(teamName); err != nil {
		return fmt.Errorf("ensure inbox dir: %w", err)
	}
	p := getInboxPath(agentName, teamName)
	lock := flock.New(p + ".lock")
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("lock inbox: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	var msgs []TeammateMessage
	b, err := os.ReadFile(p)
	if err == nil {
		_ = json.Unmarshal(b, &msgs)
	}
	if msgs == nil {
		msgs = []TeammateMessage{}
	}

	msg := TeammateMessage{
		From:      from,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Read:      false,
		MsgType:   msgType,
		Payload:   payload,
	}
	msgs = append(msgs, msg)

	out, err := json.MarshalIndent(msgs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}
	return os.WriteFile(p, append(out, '\n'), 0o600)
}

// FilterMessagesByType returns messages matching the given type discriminator.
func FilterMessagesByType(msgs []TeammateMessage, msgType string) []TeammateMessage {
	var out []TeammateMessage
	for _, m := range msgs {
		if m.MsgType == msgType {
			out = append(out, m)
		}
	}
	return out
}

// clearMailbox clears all messages from a teammate's inbox (TS clearMailbox).
func clearMailbox(agentName, teamName string) error {
	p := getInboxPath(agentName, teamName)
	lock := flock.New(p + ".lock")
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("lock inbox: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	return os.WriteFile(p, []byte("[]\n"), 0o600)
}

// markMessagesAsRead marks all messages in an inbox as read (TS markMessagesAsRead).
func markMessagesAsRead(agentName, teamName string) error {
	p := getInboxPath(agentName, teamName)

	lock := flock.New(p + ".lock")
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("lock inbox: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var msgs []TeammateMessage
	if err := json.Unmarshal(b, &msgs); err != nil {
		return nil
	}
	for i := range msgs {
		msgs[i].Read = true
	}

	out, err := json.MarshalIndent(msgs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(out, '\n'), 0o600)
}

// formatTeammateMessages formats mailbox messages as XML <teammate_message> tags
// for inclusion in the system prompt context (TS formatTeammateMessages).
func formatTeammateMessages(msgs []TeammateMessage) string {
	if len(msgs) == 0 {
		return ""
	}
	var b strings.Builder
	for _, m := range msgs {
		if m.Read {
			continue
		}
		b.WriteString("<teammate_message>\n")
		fmt.Fprintf(&b, "  <from>%s</from>\n", xmlEscape(m.From))
		fmt.Fprintf(&b, "  <text>%s</text>\n", xmlEscape(m.Text))
		b.WriteString("</teammate_message>\n")
	}
	return b.String()
}

// unreadMailboxCount returns the number of unread messages in an inbox.
func unreadMailboxCount(agentName, teamName string) int {
	msgs, err := readMailbox(agentName, teamName)
	if err != nil {
		return 0
	}
	n := 0
	for _, m := range msgs {
		if !m.Read {
			n++
		}
	}
	return n
}

// MailboxEvent is emitted by pollMailbox when new messages arrive.
type MailboxEvent struct {
	Messages []TeammateMessage
	NewCount int
}

// PollMailbox polls an agent's inbox at the given interval, sending events on the
// returned channel when new unread messages are detected. The poller stops when
// ctx is cancelled or the channel is closed by the caller.
//
// Interval defaults to 2s if <= 0. For in_process_teammate tasks, the event
// channel feeds into the task execution loop so the agent can react to
// permission requests, shutdown signals, and task assignments without blocking.
func PollMailbox(ctx context.Context, agentName, teamName string, interval time.Duration) <-chan MailboxEvent {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ch := make(chan MailboxEvent, 8)
	go func() {
		defer close(ch)
		lastCount := 0
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				msgs, err := readMailbox(agentName, teamName)
				if err != nil {
					continue
				}
				unread := 0
				for _, m := range msgs {
					if !m.Read {
						unread++
					}
				}
				if unread > lastCount {
					evt := MailboxEvent{Messages: msgs, NewCount: unread - lastCount}
					select {
					case ch <- evt:
						lastCount = unread
					case <-ctx.Done():
						return
					}
				}
				lastCount = unread
			}
		}
	}()
	return ch
}
