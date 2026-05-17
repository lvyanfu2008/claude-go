package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PlanResult is the JSON payload for /plan.
type PlanResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandlePlanCommand handles /plan [on|off].
func HandlePlanCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)
	enabled := os.Getenv("CLAUDE_CODE_PLAN_MODE") == "1"

	if args == "" {
		status := "disabled"
		if enabled {
			status = "enabled"
		}
		return json.Marshal(PlanResult{
			Type: "text",
			Value: fmt.Sprintf("Plan mode is %s.\n"+
				"Use /plan on to enter plan mode (design before implementing).\n"+
				"Use /plan off to exit plan mode.", status),
		})
	}

	switch strings.ToLower(args) {
	case "on", "enable", "true", "1":
		os.Setenv("CLAUDE_CODE_PLAN_MODE", "1")
		return json.Marshal(PlanResult{
			Type: "text",
			Value: "Plan mode enabled. Claude will design solutions before implementing.\nUse /plan off to exit plan mode.",
		})
	case "off", "disable", "false", "0":
		os.Unsetenv("CLAUDE_CODE_PLAN_MODE")
		return json.Marshal(PlanResult{Type: "text", Value: "Plan mode disabled."})
	default:
		return json.Marshal(PlanResult{Type: "text", Value: "Usage: /plan [on|off]"})
	}
}
