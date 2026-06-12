package localtools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MapExitPlanModeToolResultToAssistantText mirrors ExitPlanModeV2Tool.
// mapToolResultToToolResultBlockParam in TS. Formats the ExitPlanMode JSON
// output as the final tool_result content visible to the model.
func MapExitPlanModeToolResultToAssistantText(toolUseJSON string) (string, error) {
	toolUseJSON = strings.TrimSpace(toolUseJSON)
	if toolUseJSON == "" {
		return "", fmt.Errorf("empty ExitPlanMode result")
	}
	var wrapper struct {
		Data struct {
			Plan     string `json:"plan"`
			IsAgent  bool   `json:"isAgent"`
			FilePath string `json:"filePath"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(toolUseJSON), &wrapper); err != nil {
		return "", fmt.Errorf("parsing ExitPlanMode result: %w", err)
	}

	if wrapper.Data.IsAgent {
		return "User has approved the plan. There is nothing else needed from you now. Please respond with \"ok\"", nil
	}

	plan := strings.TrimSpace(wrapper.Data.Plan)
	if plan == "" {
		return "User has approved exiting plan mode. You can now proceed.", nil
	}

	filePath := wrapper.Data.FilePath
	pathLine := ""
	if filePath != "" {
		pathLine = fmt.Sprintf("Your plan has been saved to: %s\nYou can refer back to it if needed during implementation.\n\n", filePath)
	}

	return fmt.Sprintf("User has approved your plan. You can now start coding. Start with updating your todo list if applicable\n\n%s## Approved Plan:\n%s", pathLine, plan), nil
}
