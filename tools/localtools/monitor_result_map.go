package localtools

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var errNotMonitorOutput = errors.New("not Monitor structured tool output")

// MapMonitorToolResultToAssistantText mirrors MonitorTool.mapToolResultToToolResultBlockParam
// (MonitorTool.tsx) for the tool_result block's string content.
func MapMonitorToolResultToAssistantText(toolUseJSON string) (string, error) {
	toolUseJSON = strings.TrimSpace(toolUseJSON)
	if toolUseJSON == "" || toolUseJSON[0] != '{' {
		return "", errNotMonitorOutput
	}

	var wrapper struct {
		Data struct {
			TaskID     string `json:"taskId"`
			OutputFile string `json:"outputFile"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(toolUseJSON), &wrapper); err != nil {
		return "", err
	}
	if wrapper.Data.TaskID == "" {
		return "", errNotMonitorOutput
	}

	return fmt.Sprintf("Monitor started (task %s). Output file: %s", wrapper.Data.TaskID, wrapper.Data.OutputFile), nil
}
