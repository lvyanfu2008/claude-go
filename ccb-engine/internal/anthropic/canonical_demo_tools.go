package anthropic

import (
	"os"
	"strings"
)

// GouParityToolList is gou-demo tools[] aligned with the embedded tools_api.json export (plus echo_stub).
//
// Tool descriptions here are short stubs for tests and ParityToolRunner; API-facing tools[] for gou-demo
// uses [GouParityToolsJSON] (embedded commands/data/tools_api.json + echo_stub). See TestGouParityToolsIntersectToolsAPIExport.
// AskUserQuestion appears in the export; toolpool.GetTools omits it when TS AskUserQuestionTool.isEnabled is false (Kairos + channels).
// Glob/Grep are always included here even when TS omits them under hasEmbeddedSearchTools.
func GouParityToolList() []ToolDefinition {
	out := make([]ToolDefinition, 0, 40)
	if name := discoverSkillsToolName(); name != "" {
		out = append(out, DiscoverSkillsToolDefinition(name))
	}
	out = append(out,
		agentToolDefinition(),
		askUserQuestionToolDefinition(),
		taskOutputToolDefinition(),
		bashToolDefinition(),
		globToolDefinition(),
		grepToolDefinition(),
		exitPlanModeToolDefinition(),
		readToolDefinition(),
		writeToolDefinition(),
		editToolDefinition(),
		notebookEditToolDefinition(),
		webFetchToolDefinition(),
		todoWriteToolDefinition(),
		webSearchToolDefinition(),
		taskStopToolDefinition(),
		SkillToolDefinition(),
		enterPlanModeToolDefinition(),
		sendMessageToolDefinition(),
		cronCreateToolDefinition(),
		cronDeleteToolDefinition(),
		cronListToolDefinition(),
		sendUserMessageToolDefinition(),
		briefAliasToolDefinition(),
		listMcpResourcesToolDefinition(),
		readMcpResourceToolDefinition(),
	)
	out = append(out, DefaultStubTools()...)
	return out
}

// GouParityToolNames returns tool names from [GouParityToolList] (stable order) for drift checks vs TS.
func GouParityToolNames() []string {
	tools := GouParityToolList()
	names := make([]string, len(tools))
	for i := range tools {
		names[i] = tools[i].Name
	}
	return names
}

func discoverSkillsToolName() string {
	return strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME"))
}

// DiscoverSkillsToolDefinition returns the optional DiscoverSkills tool row when env enables it.
func DiscoverSkillsToolDefinition(name string) ToolDefinition {
	return ToolDefinition{
		Name:        name,
		Description: "Search and discover skills relevant to the current task when surfaced skills are insufficient.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"description": map[string]any{
					"type":        "string",
					"description": "Specific description of what you are doing or need skills for",
				},
			},
			"required": []string{"description"},
		},
	}
}

func readToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Read",
		Description: "Read a file from the local filesystem.",
		InputSchema: mustExportInputSchema("Read"),
	}
}

func writeToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Write",
		Description: "Writes a file to the local filesystem.",
		InputSchema: mustExportInputSchema("Write"),
	}
}

func editToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Edit",
		Description: "A tool for editing files",
		InputSchema: mustExportInputSchema("Edit"),
	}
}

func bashToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Bash",
		Description: "Execute a shell command in the user's environment.",
		InputSchema: mustExportInputSchema("Bash"),
	}
}

func globToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Glob",
		Description: "Find files matching a glob pattern.",
		InputSchema: mustExportInputSchema("Glob"),
	}
}

func grepToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Grep",
		Description: "Search file contents using ripgrep.",
		InputSchema: mustExportInputSchema("Grep"),
	}
}
