package tools

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"goc/appstate"
	"goc/types"
)

type planModeFile struct {
	Active         bool   `json:"active"`
	EnteredAt      string `json:"enteredAt,omitempty"`
	ExitedAt       string `json:"exitedAt,omitempty"`
	AllowedPrompts any    `json:"allowedPrompts,omitempty"`
}

// EnterPlanModeFromJSON transitions the session into plan mode.
// Mirrors TS EnterPlanModeTool.call().
func EnterPlanModeFromJSON(_ []byte, c Config) (string, bool, error) {
	// 1. Reject agent contexts
	if c.AgentID != "" {
		return "", true, errors.New("EnterPlanMode tool cannot be used in agent contexts")
	}

	store := c.AppStateStore
	if store == nil {
		return "", true, errors.New("EnterPlanMode: AppStateStore not available")
	}

	// 2. Get current mode
	state := store.GetState()

	// 3. Prepare context for plan mode
	newCtx := prepareContextForPlanMode(&state.ToolPermissionContext)

	// 4. Set mode to plan
	newCtx.Mode = types.PermissionPlan

	// 5. Update store atomically
	store.Update(func(prev appstate.AppState) appstate.AppState {
		prev.ToolPermissionContext = *newCtx
		prev.NeedsPlanModeExitAttachment = false
		return prev
	})

	// 6. Write plan mode file (backward compat)
	path := c.PlanModePath()
	rec := planModeFile{Active: true, EnteredAt: time.Now().UTC().Format(time.RFC3339)}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return "", true, err
	}
	if err := ensureDirFromFile(path); err != nil {
		return "", true, err
	}
	if err := writeFileAtomic(path, append(data, '\n'), 0o644); err != nil {
		return "", true, err
	}

	msg := "Entered plan mode. You should now focus on exploring the codebase and designing an implementation approach."
	out := map[string]any{"data": map[string]any{"message": msg}}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

func ensureDirFromFile(path string) error {
	return ensureDir(filepath.Dir(path))
}

// ExitPlanModeFromJSON restores the pre-plan permission mode and cleans up plan state.
// Mirrors TS ExitPlanModeV2Tool.call().
func ExitPlanModeFromJSON(raw []byte, c Config) (string, bool, error) {
	store := c.AppStateStore
	if store == nil {
		return "", true, errors.New("ExitPlanMode: AppStateStore not available")
	}

	state := store.GetState()
	if state.ToolPermissionContext.Mode != types.PermissionPlan {
		return "", true, errors.New("ExitPlanMode: not currently in plan mode")
	}

	var in struct {
		AllowedPrompts []struct {
			Tool   string `json:"tool"`
			Prompt string `json:"prompt"`
		} `json:"allowedPrompts"`
	}
	_ = json.Unmarshal(raw, &in)

	// Determine restore mode
	prePlanMode := state.ToolPermissionContext.PrePlanMode
	restoreMode := types.PermissionDefault
	if prePlanMode != nil {
		restoreMode = *prePlanMode
	}

	// Circuit breaker: if restoring to auto but gate is off, fall back to default
	restoringAuto := restoreMode == types.PermissionAuto
	if restoringAuto && !isAutoModeGateEnabled() {
		restoreMode = types.PermissionDefault
		restoringAuto = false
	}

	// Handle auto mode state reconciliation
	autoWasActive := appstate.GlobalAutoModeState.IsActive()
	needsAutoExit := false
	if restoringAuto {
		appstate.GlobalAutoModeState.SetActive(true)
	} else {
		appstate.GlobalAutoModeState.SetActive(false)
		if autoWasActive {
			needsAutoExit = true
		}
	}

	// Restore or re-strip dangerous permissions
	newCtx := state.ToolPermissionContext
	if restoringAuto {
		stripResult := stripDangerousPermissionsForAutoMode(&newCtx)
		newCtx = *stripResult
	} else if len(newCtx.StrippedDangerousRules) > 0 {
		restored := restoreDangerousPermissions(&newCtx)
		newCtx = *restored
	}

	newCtx.Mode = restoreMode
	newCtx.PrePlanMode = nil

	// Update store
	store.Update(func(prev appstate.AppState) appstate.AppState {
		prev.ToolPermissionContext = newCtx
		prev.HasExitedPlanMode = true
		prev.NeedsPlanModeExitAttachment = true
		prev.NeedsAutoModeExitAttachment = needsAutoExit
		return prev
	})

	// Read plan file content
	planPath := c.PlanFilePath()
	planContent := ""
	if data, err := os.ReadFile(planPath); err == nil {
		planContent = string(data)
	}

	// Write plan mode file
	path := c.PlanModePath()
	rec := map[string]any{
		"active":         false,
		"exitedAt":       time.Now().UTC().Format(time.RFC3339),
		"allowedPrompts": in.AllowedPrompts,
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return "", true, err
	}
	if err := ensureDirFromFile(path); err != nil {
		return "", true, err
	}
	if err := writeFileAtomic(path, append(data, '\n'), 0o644); err != nil {
		return "", true, err
	}

	out := map[string]any{
		"data": map[string]any{
			"plan":     planContent,
			"isAgent":  false,
			"filePath": planPath,
		},
	}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}
