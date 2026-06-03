package tools

import (
	"encoding/json"
	"os"
	"testing"

	"goc/appstate"
	"goc/types"
)

func TestPlanModeFullLifecycle(t *testing.T) {
	store := appstate.NewStore(appstate.DefaultAppState())
	cfg := newTestConfig(t, store)

	// Phase 1: Enter plan mode
	_, isErr, err := EnterPlanModeFromJSON(nil, cfg)
	if isErr {
		t.Fatalf("enter: %v", err)
	}

	state := store.GetState()
	if state.ToolPermissionContext.Mode != types.PermissionPlan {
		t.Fatalf("not in plan mode after enter, mode=%s", state.ToolPermissionContext.Mode)
	}
	if state.NeedsPlanModeExitAttachment {
		t.Error("NeedsPlanModeExitAttachment should be false after entering")
	}

	// Phase 2: Exit plan mode
	_, isErr, err = ExitPlanModeFromJSON(nil, cfg)
	if isErr {
		t.Fatalf("exit: %v", err)
	}

	state = store.GetState()
	if state.ToolPermissionContext.Mode != types.PermissionDefault {
		t.Errorf("mode should be restored to default, got %s", state.ToolPermissionContext.Mode)
	}
	if !state.HasExitedPlanMode {
		t.Error("HasExitedPlanMode should be true after exit")
	}
	if !state.NeedsPlanModeExitAttachment {
		t.Error("NeedsPlanModeExitAttachment should be true after exit")
	}

	// Phase 3: Can re-enter
	_, isErr, err = EnterPlanModeFromJSON(nil, cfg)
	if isErr {
		t.Fatalf("re-enter: %v", err)
	}

	state = store.GetState()
	if state.ToolPermissionContext.Mode != types.PermissionPlan {
		t.Fatalf("not in plan mode after re-enter, mode=%s", state.ToolPermissionContext.Mode)
	}
}

func TestPlanModeFilePersistence(t *testing.T) {
	store := appstate.NewStore(appstate.DefaultAppState())
	cfg := newTestConfig(t, store)

	// Enter
	EnterPlanModeFromJSON(nil, cfg)

	// Read file
	data, err := os.ReadFile(cfg.PlanModePath())
	if err != nil {
		t.Fatalf("read plan file: %v", err)
	}
	var pf planModeFile
	if err := json.Unmarshal(data, &pf); err != nil {
		t.Fatalf("parse plan file: %v", err)
	}
	if !pf.Active {
		t.Error("plan file should be active")
	}
	if pf.EnteredAt == "" {
		t.Error("plan file should have enteredAt timestamp")
	}

	// Exit
	ExitPlanModeFromJSON(nil, cfg)

	// Read file again
	data, err = os.ReadFile(cfg.PlanModePath())
	if err != nil {
		t.Fatalf("read plan file after exit: %v", err)
	}
	var pf2 planModeFile
	json.Unmarshal(data, &pf2)
	if pf2.Active {
		t.Error("plan file should be inactive after exit")
	}
}

func TestEnterPlanModeFromJSON_Result(t *testing.T) {
	store := appstate.NewStore(appstate.DefaultAppState())
	cfg := newTestConfig(t, store)

	result, isErr, err := EnterPlanModeFromJSON(nil, cfg)
	if isErr {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	data := out["data"].(map[string]any)
	if data["message"].(string) == "" {
		t.Error("message should not be empty")
	}
}
