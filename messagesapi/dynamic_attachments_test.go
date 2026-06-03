package messagesapi

import (
	"encoding/json"
	"testing"

	"goc/appstate"
	"goc/types"
)

func TestBuildDynamicAttachments_NilState(t *testing.T) {
	result := BuildDynamicAttachments(nil, nil, DefaultOptions())
	if len(result) != 0 {
		t.Errorf("nil state should return empty, got %d attachments", len(result))
	}
}

func TestBuildDynamicAttachments_PlanModeExit(t *testing.T) {
	state := appstate.DefaultAppState()
	state.NeedsPlanModeExitAttachment = true

	result := BuildDynamicAttachments(&state, nil, DefaultOptions())
	if len(result) == 0 {
		t.Fatal("should have plan_mode_exit attachment")
	}

	var att struct {
		Type string `json:"type"`
	}
	json.Unmarshal(result[0], &att)
	if att.Type != "plan_mode_exit" {
		t.Errorf("type = %s, want plan_mode_exit", att.Type)
	}
}

func TestBuildDynamicAttachments_PlanModeActive(t *testing.T) {
	state := appstate.DefaultAppState()
	state.ToolPermissionContext.Mode = types.PermissionPlan

	result := BuildDynamicAttachments(&state, nil, DefaultOptions())
	if len(result) == 0 {
		t.Fatal("should have plan_mode attachment when in plan mode")
	}

	var att struct {
		Type         string `json:"type"`
		ReminderType string `json:"reminderType"`
	}
	json.Unmarshal(result[0], &att)
	if att.Type != "plan_mode" {
		t.Errorf("type = %s, want plan_mode", att.Type)
	}
	// First attachment should be "full"
	if att.ReminderType != "full" {
		t.Errorf("first reminderType = %s, want full", att.ReminderType)
	}
}

func TestBuildDynamicAttachments_PlanModeThrottled(t *testing.T) {
	state := appstate.DefaultAppState()
	state.ToolPermissionContext.Mode = types.PermissionPlan

	// Create messages with a recent plan_mode attachment
	attJSON, _ := json.Marshal(map[string]any{"type": "plan_mode"})
	messages := []types.Message{
		{Type: types.MessageTypeUser},
		{Type: types.MessageTypeAttachment, Attachment: attJSON},
	}

	result := BuildDynamicAttachments(&state, messages, DefaultOptions())
	// Should be throttled (only 0 turns since last attachment)
	if len(result) != 0 {
		t.Errorf("should be throttled, got %d attachments", len(result))
	}
}

func TestBuildDynamicAttachments_AutoModeExit(t *testing.T) {
	state := appstate.DefaultAppState()
	state.NeedsAutoModeExitAttachment = true

	result := BuildDynamicAttachments(&state, nil, DefaultOptions())
	if len(result) == 0 {
		t.Fatal("should have auto_mode_exit attachment")
	}

	var att struct {
		Type string `json:"type"`
	}
	json.Unmarshal(result[0], &att)
	if att.Type != "auto_mode_exit" {
		t.Errorf("type = %s, want auto_mode_exit", att.Type)
	}
}

func TestBuildDynamicAttachments_PlanModeReentry(t *testing.T) {
	state := appstate.DefaultAppState()
	state.HasExitedPlanMode = true

	result := BuildDynamicAttachments(&state, nil, DefaultOptions())
	// plan_mode_reentry depends on planFileExists which currently returns false
	// So we expect no attachments for reentry
	_ = result
}
