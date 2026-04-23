package toolpool

import (
	"encoding/json"
	"os"
	"strings"

	"goc/types"
)

// DiscoverSkillsToolSpecFromEnv returns a model tool spec when CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME is set.
// The tool name is taken from that env var; input schema matches the historical DiscoverSkills ToolDefinition.
func DiscoverSkillsToolSpecFromEnv() (types.ToolSpec, bool) {
	name := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME"))
	if name == "" {
		return types.ToolSpec{}, false
	}
	schema, err := json.Marshal(discoverSkillsInputSchemaObject())
	if err != nil {
		return types.ToolSpec{}, false
	}
	return types.ToolSpec{
		Name:            name,
		Description:     "Search and discover skills relevant to the current task when surfaced skills are insufficient.",
		InputJSONSchema: schema,
	}, true
}

func discoverSkillsInputSchemaObject() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"description": map[string]any{
				"type":        "string",
				"description": "Specific description of what you are doing or need skills for",
			},
		},
		"required": []string{"description"},
	}
}
