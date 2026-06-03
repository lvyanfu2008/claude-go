package tools

import (
	"encoding/json"
	"os"
	"testing"

	"goc/appstate"
	"goc/types"
)

func newTestConfig(t *testing.T, store *appstate.Store) Config {
	t.Helper()
	dir := t.TempDir()
	return Config{
		WorkDir:       dir,
		ProjectRoot:   dir,
		AppStateStore: store,
	}
}

func TestEnterPlanModeFromJSON_AgentRejection(t *testing.T) {
	store := appstate.NewStore(appstate.DefaultAppState())
	cfg := newTestConfig(t, store)
	cfg.AgentID = "agent-123"

	_, isErr, err := EnterPlanModeFromJSON(nil, cfg)
	if !isErr {
		t.Error("should return error for agent context")
	}
	if err == nil {
		t.Error("should have error message")
	}
}

func TestEnterPlanModeFromJSON_NoStore(t *testing.T) {
	cfg := Config{}
	_, isErr, err := EnterPlanModeFromJSON(nil, cfg)
	if !isErr {
		t.Error("should return error when AppStateStore is nil")
	}
	if err == nil {
		t.Error("should have error message")
	}
}

func TestEnterPlanModeFromJSON_Success(t *testing.T) {
	store := appstate.NewStore(appstate.DefaultAppState())
	cfg := newTestConfig(t, store)

	result, isErr, err := EnterPlanModeFromJSON(nil, cfg)
	if isErr {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check result message
	var out struct {
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid result JSON: %v", err)
	}
	if out.Data.Message == "" {
		t.Error("result should contain a message")
	}

	// Check state was updated
	state := store.GetState()
	if state.ToolPermissionContext.Mode != types.PermissionPlan {
		t.Errorf("mode = %s, want plan", state.ToolPermissionContext.Mode)
	}
	if state.ToolPermissionContext.PrePlanMode == nil {
		t.Fatal("prePlanMode should be set")
	}
	if *state.ToolPermissionContext.PrePlanMode != types.PermissionDefault {
		t.Errorf("prePlanMode = %s, want default", *state.ToolPermissionContext.PrePlanMode)
	}

	// Check file was written
	planPath := cfg.PlanModePath()
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		t.Error("plan mode file should exist")
	}
}

func TestExitPlanModeFromJSON_NotInPlan(t *testing.T) {
	store := appstate.NewStore(appstate.DefaultAppState())
	cfg := newTestConfig(t, store)

	_, isErr, err := ExitPlanModeFromJSON(nil, cfg)
	if !isErr {
		t.Error("should return error when not in plan mode")
	}
	if err == nil {
		t.Error("should have error message")
	}
}

func TestExitPlanModeFromJSON_Success(t *testing.T) {
	// First enter plan mode
	store := appstate.NewStore(appstate.DefaultAppState())
	cfg := newTestConfig(t, store)

	_, isErr, _ := EnterPlanModeFromJSON(nil, cfg)
	if isErr {
		t.Fatal("enter plan mode failed")
	}

	// Then exit
	result, isErr, err := ExitPlanModeFromJSON(nil, cfg)
	if isErr {
		t.Fatalf("unexpected error: %v", err)
	}

	var out struct {
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(result), &out)
	if out.Data.Message == "" {
		t.Error("result should contain a message")
	}

	// Check state was restored
	state := store.GetState()
	if state.ToolPermissionContext.Mode != types.PermissionDefault {
		t.Errorf("mode = %s, want default", state.ToolPermissionContext.Mode)
	}
	if state.ToolPermissionContext.PrePlanMode != nil {
		t.Error("prePlanMode should be cleared")
	}
	if !state.HasExitedPlanMode {
		t.Error("HasExitedPlanMode should be true")
	}
	if !state.NeedsPlanModeExitAttachment {
		t.Error("NeedsPlanModeExitAttachment should be true")
	}
}

func TestEnterThenExitPlanMode_PrePlanModeRoundTrip(t *testing.T) {
	store := appstate.NewStore(appstate.DefaultAppState())

	// Set initial mode to acceptEdits (non-default)
	store.Update(func(prev appstate.AppState) appstate.AppState {
		prev.ToolPermissionContext.Mode = types.PermissionAcceptEdits
		return prev
	})

	cfg := newTestConfig(t, store)

	// Enter
	_, isErr, _ := EnterPlanModeFromJSON(nil, cfg)
	if isErr {
		t.Fatal("enter failed")
	}

	state := store.GetState()
	if *state.ToolPermissionContext.PrePlanMode != types.PermissionAcceptEdits {
		t.Errorf("prePlanMode = %s, want acceptEdits", *state.ToolPermissionContext.PrePlanMode)
	}

	// Exit
	_, isErr, _ = ExitPlanModeFromJSON(nil, cfg)
	if isErr {
		t.Fatal("exit failed")
	}

	state = store.GetState()
	if state.ToolPermissionContext.Mode != types.PermissionAcceptEdits {
		t.Errorf("restored mode = %s, want acceptEdits", state.ToolPermissionContext.Mode)
	}
}
