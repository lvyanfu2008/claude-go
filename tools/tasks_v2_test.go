package tools

import (
	"testing"
)

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		from, to string
		want     bool
	}{
		// Same status (idempotent).
		{"pending", "pending", true},
		{"in_progress", "in_progress", true},
		{"completed", "completed", true},
		{"failed", "failed", true},
		{"killed", "killed", true},

		// Valid forward transitions.
		{"pending", "in_progress", true},
		{"in_progress", "completed", true},
		{"in_progress", "failed", true},
		{"in_progress", "killed", true},

		// Terminal statuses cannot transition.
		{"completed", "in_progress", false},
		{"completed", "pending", false},
		{"failed", "in_progress", false},
		{"failed", "completed", false},
		{"killed", "pending", false},
		{"killed", "in_progress", false},

		// Invalid backward transitions.
		{"in_progress", "pending", false},

		// Invalid skips.
		{"pending", "completed", false},
		{"pending", "failed", false},
		{"pending", "killed", false},
	}
	for _, tt := range tests {
		got := isValidTransition(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("isValidTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestValidTaskTypes(t *testing.T) {
	validTypes := []string{
		TaskTypeLocalBash,
		TaskTypeLocalAgent,
		TaskTypeRemoteAgent,
		TaskTypeInProcessTeammate,
		TaskTypeLocalWorkflow,
		TaskTypeMonitorMCP,
		TaskTypeDream,
	}
	for _, typ := range validTypes {
		if !validTaskTypes[typ] {
			t.Errorf("expected %q to be valid", typ)
		}
	}
	if validTaskTypes["invalid_type"] {
		t.Error("expected 'invalid_type' to be invalid")
	}
	if validTaskTypes[""] {
		t.Error("expected empty string to be invalid")
	}
}

func TestTaskTypePrefixMapping(t *testing.T) {
	typeToPrefix := map[string]string{
		TaskTypeLocalBash:          "b",
		TaskTypeLocalAgent:         "a",
		TaskTypeRemoteAgent:        "r",
		TaskTypeInProcessTeammate:  "t",
		TaskTypeLocalWorkflow:      "w",
		TaskTypeMonitorMCP:         "m",
		TaskTypeDream:              "d",
	}
	for taskType, expectedPrefix := range typeToPrefix {
		if got := taskTypeToPrefix[taskType]; got != expectedPrefix {
			t.Errorf("taskTypeToPrefix[%q] = %q, want %q", taskType, got, expectedPrefix)
		}
	}
	// Verify reverse mapping.
	for prefix, expectedType := range prefixToTaskType {
		if gotType := taskTypeToPrefix[expectedType]; gotType != prefix {
			t.Errorf("inconsistent mapping: prefix %q -> type %q, but type maps to %q", prefix, expectedType, gotType)
		}
	}
}

func TestGetTaskTypeFromID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"a42", TaskTypeLocalAgent},
		{"b1", TaskTypeLocalBash},
		{"r123", TaskTypeRemoteAgent},
		{"t7", TaskTypeInProcessTeammate},
		{"w99", TaskTypeLocalWorkflow},
		{"m1", TaskTypeMonitorMCP},
		{"d42", TaskTypeDream},
		{"", ""},
		{"x", ""},
		{"z100", ""},
	}
	for _, tt := range tests {
		got := getTaskTypeFromID(tt.id)
		if got != tt.want {
			t.Errorf("getTaskTypeFromID(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestParseTaskNotification(t *testing.T) {
	xml := `<task-notification>
  <event>completed</event>
  <task-id>42</task-id>
  <subject>Fix login bug</subject>
</task-notification>`
	notif, err := parseTaskNotification(xml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notif == nil {
		t.Fatal("expected non-nil notification")
	}
	if notif.TaskID != "42" {
		t.Errorf("TaskID = %q, want %q", notif.TaskID, "42")
	}
	if notif.Action != "completed" {
		t.Errorf("Action = %q, want %q", notif.Action, "completed")
	}
	if notif.Subject != "Fix login bug" {
		t.Errorf("Subject = %q, want %q", notif.Subject, "Fix login bug")
	}
	if !notif.IsTerminalNotification() {
		t.Error("expected IsTerminalNotification() = true for completed")
	}
}

func TestParseTaskNotification_Failed(t *testing.T) {
	xml := `<task-notification>
  <event>failed</event>
  <task-id>10</task-id>
  <subject>Deploy</subject>
</task-notification>`
	notif, err := parseTaskNotification(xml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notif == nil {
		t.Fatal("expected non-nil notification")
	}
	if notif.Action != "failed" {
		t.Errorf("Action = %q, want %q", notif.Action, "failed")
	}
	if !notif.IsTerminalNotification() {
		t.Error("expected IsTerminalNotification() = true for failed")
	}
}

func TestParseTaskNotification_WithTypeAndStatus(t *testing.T) {
	xml := `<task-notification>
  <event>status_change</event>
  <task-id>5</task-id>
  <subject>Refactor</subject>
  <type>local_bash</type>
  <status>in_progress</status>
</task-notification>`
	notif, err := parseTaskNotification(xml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notif == nil {
		t.Fatal("expected non-nil notification")
	}
	if notif.TaskType != "local_bash" {
		t.Errorf("TaskType = %q, want %q", notif.TaskType, "local_bash")
	}
	if notif.Status != "in_progress" {
		t.Errorf("Status = %q, want %q", notif.Status, "in_progress")
	}
	if notif.IsTerminalNotification() {
		t.Error("expected IsTerminalNotification() = false for status_change")
	}
}

func TestParseTaskNotification_Invalid(t *testing.T) {
	notif, err := parseTaskNotification("not xml at all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notif != nil {
		t.Error("expected nil for invalid XML")
	}

	notif, err = parseTaskNotification("<other>stuff</other>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notif != nil {
		t.Error("expected nil for missing task-notification tag")
	}
}

func TestParseTaskNotificationsFromMessages(t *testing.T) {
	msgs := []TeammateMessage{
		{
			MsgType: TeammateMsgTypeTaskNotification,
			Text:    `<task-notification><event>completed</event><task-id>1</task-id><subject>A</subject></task-notification>`,
		},
		{
			MsgType: "other",
			Text:    "plain text",
		},
		{
			MsgType: TeammateMsgTypeTaskNotification,
			Text:    `<task-notification><event>failed</event><task-id>2</task-id><subject>B</subject></task-notification>`,
		},
	}
	notifs := parseTaskNotificationsFromMessages(msgs)
	if len(notifs) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifs))
	}
	if notifs[0].TaskID != "1" {
		t.Errorf("first TaskID = %q, want %q", notifs[0].TaskID, "1")
	}
	if notifs[1].TaskID != "2" {
		t.Errorf("second TaskID = %q, want %q", notifs[1].TaskID, "2")
	}
}

func TestValidateV2Task_Statuses(t *testing.T) {
	validTask := &v2Task{
		ID:      "1",
		Subject: "test",
		Status:  TaskStatusPending,
		Blocks:  []string{},
		BlockedBy: []string{},
	}
	if !validateV2Task(validTask) {
		t.Error("expected valid task with pending status")
	}

	validTask.Status = TaskStatusFailed
	if !validateV2Task(validTask) {
		t.Error("expected valid task with failed status")
	}

	validTask.Status = TaskStatusKilled
	if !validateV2Task(validTask) {
		t.Error("expected valid task with killed status")
	}

	validTask.Status = "invalid_status"
	if validateV2Task(validTask) {
		t.Error("expected invalid task with bad status")
	}
}

func TestValidateV2Task_Type(t *testing.T) {
	validTask := &v2Task{
		ID:      "1",
		Type:    TaskTypeLocalBash,
		Subject: "test",
		Status:  TaskStatusPending,
		Blocks:  []string{},
		BlockedBy: []string{},
	}
	if !validateV2Task(validTask) {
		t.Error("expected valid task with local_bash type")
	}

	validTask.Type = ""
	if !validateV2Task(validTask) {
		t.Error("expected valid task with empty type (legacy compat)")
	}

	validTask.Type = "nonexistent"
	if validateV2Task(validTask) {
		t.Error("expected invalid task with nonexistent type")
	}
}

func TestValidTaskStatus(t *testing.T) {
	expectValid := []string{"pending", "in_progress", "completed", "failed", "killed"}
	for _, s := range expectValid {
		if !validTaskStatus[s] {
			t.Errorf("expected %q to be valid status", s)
		}
	}
	expectInvalid := []string{"unknown", "running", "blocked", ""}
	for _, s := range expectInvalid {
		if validTaskStatus[s] {
			t.Errorf("expected %q to be invalid status", s)
		}
	}
}

func TestTerminalTaskStatus(t *testing.T) {
	expectTerminal := []string{"completed", "failed", "killed"}
	for _, s := range expectTerminal {
		if !terminalTaskStatus[s] {
			t.Errorf("expected %q to be terminal status", s)
		}
	}
	expectNonTerminal := []string{"pending", "in_progress"}
	for _, s := range expectNonTerminal {
		if terminalTaskStatus[s] {
			t.Errorf("expected %q to NOT be terminal status", s)
		}
	}
}
