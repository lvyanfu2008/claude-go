package querycontext

import (
	"os"
	"sort"
	"strings"
)

// workerToolsAll mirrors the async-agent-allowed tools (minus internal worker tools)
// from TS ASYNC_AGENT_ALLOWED_TOOLS \ INTERNAL_WORKER_TOOLS.
var workerToolsAll = []string{
	"Bash", "CronCreate", "CronDelete", "CronList", "Edit",
	"EnterWorktree", "ExitWorktree", "Glob", "Grep", "NotebookEdit",
	"Read", "Skill", "StructuredOutput", "TaskCreate", "TaskGet",
	"TaskList", "TaskUpdate", "TodoWrite", "WebFetch", "WebSearch", "Write",
}

// GetCoordinatorUserContext mirrors TS getCoordinatorUserContext in src/coordinator/coordinatorMode.ts.
// Returns a map with "workerToolsContext" when coordinator mode is active, otherwise nil.
func GetCoordinatorUserContext(mcpClientNames []string, scratchpadDir string) map[string]string {
	if !isCoordinatorMode() {
		return nil
	}

	var toolNames []string
	if IsEnvTruthy(os.Getenv("CLAUDE_CODE_SIMPLE")) {
		toolNames = []string{"Bash", "Edit", "Read"}
	} else {
		toolNames = make([]string, len(workerToolsAll))
		copy(toolNames, workerToolsAll)
	}

	content := "Workers spawned via the Agent tool have access to these tools: " + strings.Join(toolNames, ", ")

	if len(mcpClientNames) > 0 {
		sort.Strings(mcpClientNames)
		content += "\n\nWorkers also have access to MCP tools from connected MCP servers: " + strings.Join(mcpClientNames, ", ")
	}

	if scratchpadDir != "" {
		content += "\n\nScratchpad directory: " + scratchpadDir + "\nWorkers can read and write here without permission prompts. Use this for durable cross-worker knowledge — structure files however fits the work."
	}

	return map[string]string{"workerToolsContext": content}
}

// isCoordinatorMode mirrors toolpool.IsCoordinatorMode but avoids the import cycle
// through bashzog → querycontext → toolpool.
func isCoordinatorMode() bool {
	if !IsEnvTruthy(os.Getenv("FEATURE_COORDINATOR_MODE")) {
		return false
	}
	return IsEnvTruthy(os.Getenv("CLAUDE_CODE_COORDINATOR_MODE"))
}
