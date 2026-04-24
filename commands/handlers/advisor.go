package handlers

import (
	"encoding/json"
	"fmt"
	"os"

	"goc/ccb-engine/settingsfile"
)

// AdvisorResult is the JSON payload returned by /advisor.
type AdvisorResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// envReadAdvisorModel checks settings env merge then raw process env.
func envReadAdvisorModel() string {
	if v := os.Getenv("CLAUDE_CODE_ADVISOR_MODEL"); v != "" {
		return v
	}
	env, err := settingsfile.ReadUserSettingsEnv()
	if err == nil && env != nil {
		if v, ok := env["CLAUDE_CODE_ADVISOR_MODEL"]; ok && v != "" {
			return v
		}
	}
	return ""
}

// HandleAdvisorCommand reads the advisor model from settings.
// If args includes a model name, sets CLAUDE_CODE_ADVISOR_MODEL in env for this session.
// Mirrors TS src/commands/advisor.ts.
func HandleAdvisorCommand(args string) ([]byte, error) {
	if args != "" {
		if err := os.Setenv("CLAUDE_CODE_ADVISOR_MODEL", args); err != nil {
			return nil, fmt.Errorf("set advisor model: %w", err)
		}
		return json.Marshal(AdvisorResult{
			Type:  "text",
			Value: fmt.Sprintf("Advisor model set to %s for this session.", args),
		})
	}

	model := envReadAdvisorModel()
	if model == "" {
		return json.Marshal(AdvisorResult{
			Type:  "text",
			Value: "No advisor model configured. Use /advisor <model> to set one, or set CLAUDE_CODE_ADVISOR_MODEL in settings.",
		})
	}
	return json.Marshal(AdvisorResult{
		Type:  "text",
		Value: fmt.Sprintf("Current advisor model: %s", model),
	})
}
