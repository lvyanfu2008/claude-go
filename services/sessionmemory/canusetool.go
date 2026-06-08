package sessionmemory

import (
	"context"
	"encoding/json"
	"fmt"

	"goc/tools/toolexecution"
)

const (
	fileEditToolName = "Edit"
	fileReadToolName = "Read"
)

// CreateMemoryFileCanUseTool returns a QueryCanUseToolFn that allows
// Read (any path) and Edit (only the exact memoryPath). All other tools are denied.
// Mirrors TS createMemoryFileCanUseTool, extended with Read for non-Anthropic models.
func CreateMemoryFileCanUseTool(memoryPath string) toolexecution.QueryCanUseToolFn {
	return func(ctx context.Context, toolName, _ string, input json.RawMessage) (toolexecution.PermissionDecision, error) {
		switch toolName {
		case fileReadToolName:
			return toolexecution.AllowDecision(), nil
		case fileEditToolName:
			var v struct {
				FilePath string `json:"file_path"`
			}
			if err := json.Unmarshal(input, &v); err == nil && v.FilePath == memoryPath {
				return toolexecution.AllowDecision(), nil
			}
		}
		return toolexecution.DenyDecision(fmt.Sprintf("only Read or Edit on %s is allowed", memoryPath)), nil
	}
}
