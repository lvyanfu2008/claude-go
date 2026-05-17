package tools

import (
	"encoding/xml"
	"strings"
)

// TaskNotification mirrors the <task-notification> XML structure sent via teammate mailbox.
// It is parsed from the XML text embedded in TeammateMessage.Text when MsgType is task_notification.
type TaskNotification struct {
	TaskID    string
	TaskType  string
	Action    string // "created" | "assigned" | "status_change" | "completed" | "failed" | "killed"
	Status    string
	AgentName string
	Subject   string
}

// parseTaskNotification parses a <task-notification> XML string into a TaskNotification.
// The XML format:
//
//	<task-notification>
//	  <event>completed</event>
//	  <task-id>42</task-id>
//	  <subject>Fix login bug</subject>
//	</task-notification>
func parseTaskNotification(xmlStr string) (*TaskNotification, error) {
	// Strip any non-XML prefix/suffix (e.g. human-readable text mixed with XML).
	start := strings.Index(xmlStr, "<task-notification>")
	end := strings.Index(xmlStr, "</task-notification>")
	if start < 0 || end < 0 || end <= start {
		return nil, nil
	}
	xmlStr = xmlStr[start : end+len("</task-notification>")]

	type notificationXML struct {
		Event   string `xml:"event"`
		TaskID  string `xml:"task-id"`
		Subject string `xml:"subject"`
		Type    string `xml:"type"`
		Status  string `xml:"status"`
		Agent   string `xml:"agent"`
	}
	var n notificationXML
	if err := xml.Unmarshal([]byte(xmlStr), &n); err != nil {
		return nil, nil
	}
	if n.TaskID == "" {
		return nil, nil
	}
	notif := &TaskNotification{
		TaskID:    n.TaskID,
		TaskType:  n.Type,
		Action:    n.Event,
		Status:    n.Status,
		AgentName: n.Agent,
		Subject:   n.Subject,
	}
	// Infer task type and status from event if not explicitly provided.
	if notif.TaskType == "" {
		notif.TaskType = inferTaskTypeFromAction(notif.Action)
	}
	if notif.Status == "" {
		notif.Status = inferStatusFromAction(notif.Action)
	}
	return notif, nil
}

func inferTaskTypeFromAction(action string) string {
	switch action {
	case "created":
		return TaskTypeLocalAgent
	case "assigned":
		return TaskTypeLocalAgent
	case "completed", "failed", "killed":
		return "" // can't infer without type field
	case "status_change":
		return ""
	}
	return ""
}

func inferStatusFromAction(action string) string {
	switch action {
	case "created":
		return TaskStatusPending
	case "assigned":
		return TaskStatusPending
	case "completed":
		return TaskStatusCompleted
	case "failed":
		return TaskStatusFailed
	case "killed":
		return TaskStatusKilled
	}
	return ""
}

// parseTaskNotificationsFromMessages extracts TaskNotifications from teammate messages.
func parseTaskNotificationsFromMessages(msgs []TeammateMessage) []*TaskNotification {
	var out []*TaskNotification
	for _, m := range msgs {
		if m.MsgType != TeammateMsgTypeTaskNotification && !strings.Contains(m.Text, "<task-notification>") {
			continue
		}
		notif, err := parseTaskNotification(m.Text)
		if err != nil || notif == nil {
			continue
		}
		out = append(out, notif)
	}
	return out
}

// IsTerminalNotification returns true if the notification represents a terminal task event.
func (n *TaskNotification) IsTerminalNotification() bool {
	return n.Action == "completed" || n.Action == "failed" || n.Action == "killed"
}
