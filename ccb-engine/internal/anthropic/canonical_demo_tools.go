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
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file_path": map[string]any{"type": "string", "description": "The absolute path to the file to read"},
				"offset":    map[string]any{"type": "integer", "description": "Line number to start reading from"},
				"limit":     map[string]any{"type": "integer", "description": "Number of lines to read"},
				"pages":     map[string]any{"type": "string", "description": "Page range for PDF files (e.g. \"1-5\")"},
			},
			"required": []string{"file_path"},
		},
	}
}

func writeToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Write",
		Description: "Writes a file to the local filesystem.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file_path": map[string]any{"type": "string", "description": "The absolute path to the file to write"},
				"content":   map[string]any{"type": "string", "description": "The content to write to the file"},
			},
			"required": []string{"file_path", "content"},
		},
	}
}

func editToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Edit",
		Description: "A tool for editing files",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file_path":   map[string]any{"type": "string", "description": "The absolute path to the file to modify"},
				"old_string":  map[string]any{"type": "string", "description": "The text to replace"},
				"new_string":  map[string]any{"type": "string", "description": "The text to replace it with"},
				"replace_all": map[string]any{"type": "boolean", "description": "Replace all occurrences of old_string"},
			},
			"required": []string{"file_path", "old_string", "new_string"},
		},
	}
}

func bashToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Bash",
		Description: "Execute a shell command in the user's environment.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command":     map[string]any{"type": "string", "description": "The command to execute"},
				"timeout":     map[string]any{"type": "number", "description": "Optional timeout in milliseconds"},
				"description": map[string]any{"type": "string", "description": "Clear description of what the command does"},
			},
			"required": []string{"command"},
		},
	}
}

func globToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Glob",
		Description: "Find files matching a glob pattern.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string", "description": "The glob pattern to match files against"},
				"path":    map[string]any{"type": "string", "description": "Directory to search in"},
			},
			"required": []string{"pattern"},
		},
	}
}

func grepToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        "Grep",
		Description: "Search file contents using ripgrep.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern":      map[string]any{"type": "string", "description": "Regular expression pattern to search for"},
				"path":         map[string]any{"type": "string", "description": "File or directory to search in"},
				"glob":         map[string]any{"type": "string", "description": "Glob pattern to filter files"},
				"output_mode":  map[string]any{"type": "string", "description": "content | files_with_matches | count"},
				"-i":           map[string]any{"type": "boolean", "description": "Case insensitive"},
				"head_limit":   map[string]any{"type": "integer", "description": "Max results"},
				"multiline":    map[string]any{"type": "boolean", "description": "Multiline mode"},
			},
			"required": []string{"pattern"},
		},
	}
}
