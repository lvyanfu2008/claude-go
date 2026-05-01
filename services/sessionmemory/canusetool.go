package sessionmemory

import (
	"context"
	"encoding/json"
	"fmt"

	"goc/tools/toolexecution"
)

const fileEditToolName = "Edit"

// CreateMemoryFileCanUseTool returns a QueryCanUseToolFn that only allows
// the Edit tool on the exact memoryPath. All other tools and paths are denied.
// Mirrors TS createMemoryFileCanUseTool.
func CreateMemoryFileCanUseTool(memoryPath string) toolexecution.QueryCanUseToolFn {
	return func(ctx context.Context, toolName, _ string, input json.RawMessage) (toolexecution.PermissionDecision, error) {
		if toolName == fileEditToolName {
			var v struct {
				FilePath string `json:"file_path"`
			}
			if err := json.Unmarshal(input, &v); err == nil && v.FilePath == memoryPath {
				return toolexecution.AllowDecision(), nil
			}
		}
		return toolexecution.DenyDecision(fmt.Sprintf("only Edit on %s is allowed", memoryPath)), nil
	}
}
