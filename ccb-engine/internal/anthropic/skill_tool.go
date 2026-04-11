package anthropic

import (
	"encoding/json"
	"os"
	"strings"

	"goc/agents/builtin"
	"goc/toolpool"
	"goc/types"
)

// SkillToolName matches src/tools/SkillTool/constants.ts SKILL_TOOL_NAME.
const SkillToolName = "Skill"

// skillToolDescriptionPrompt matches getPrompt() in src/tools/SkillTool/prompt.ts (must stay in sync with goc/commands/skill_listing_messages.go SkillToolDescriptionPrompt).
const skillToolDescriptionPrompt = `Execute a skill within the main conversation

When users ask you to perform tasks, check if any of the available skills match. Skills provide specialized capabilities and domain knowledge.

When users reference a "slash command" or "/<something>" (e.g., "/commit", "/review-pr"), they are referring to a skill. Use this tool to invoke it.

How to invoke:
- Use this tool with the skill name and optional arguments
- Examples:
  - ` + "`skill: \"pdf\"`" + ` - invoke the pdf skill
  - ` + "`skill: \"commit\", args: \"-m 'Fix bug'\"`" + ` - invoke with arguments
  - ` + "`skill: \"review-pr\", args: \"123\"`" + ` - invoke with arguments
  - ` + "`skill: \"ms-office-suite:pdf\"`" + ` - invoke using fully qualified name

Important:
- Available skills are listed in system-reminder messages in the conversation
- When a skill matches the user's request, this is a BLOCKING REQUIREMENT: invoke the relevant Skill tool BEFORE generating any other response about the task
- NEVER mention a skill without actually calling this tool
- Do not invoke a skill that is already running
- Do not use this tool for built-in CLI commands (like /help, /clear, etc.)
- If you see a <command-name> tag in the current conversation turn, the skill has ALREADY been loaded - follow the instructions directly instead of calling this tool again
`

// SkillToolDefinition matches the Skill tool registered for the Messages API (zod inputSchema in SkillTool.ts).
func SkillToolDefinition() ToolDefinition {
	return ToolDefinition{
		Name:        SkillToolName,
		Description: skillToolDescriptionPrompt,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"skill": map[string]any{
					"type":        "string",
					"description": `The skill name. E.g., "commit", "review-pr", or "pdf"`,
				},
				"args": map[string]any{
					"type":        "string",
					"description": "Optional arguments for the skill",
				},
			},
			"required": []string{"skill"},
		},
	}
}

// GouDemoDefaultTools is Skill + echo_stub (stub keeps engine wiring tests familiar).
func GouDemoDefaultTools() []ToolDefinition {
	out := make([]ToolDefinition, 0, 2)
	out = append(out, SkillToolDefinition())
	out = append(out, DefaultStubTools()...)
	return out
}

// GouDemoDefaultToolsJSON marshals [GouDemoDefaultTools].
func GouDemoDefaultToolsJSON() (json.RawMessage, error) {
	return json.Marshal(GouDemoDefaultTools())
}

// GouParityToolsJSON returns the same tools[] shape as gou-demo with GOU_DEMO_USE_EMBEDDED_TOOLS_API=1:
// embedded commands/data/tools_api.json (TS export) + deny/REPL filtering + echo_stub for localturn wiring.
// Use [GouParityToolList] only for name/schema stubs in tests and ParityToolRunner dispatch — not for API descriptions.
func GouParityToolsJSON() (json.RawMessage, error) {
	assembled, err := toolpool.AssembleToolPoolFromEmbedded(types.EmptyToolPermissionContextData(), nil)
	if err != nil {
		return nil, err
	}
	assembled = toolpool.PatchAgentToolDescriptionWithBuiltins(assembled, builtin.GetBuiltInAgents(builtin.ConfigFromEnv(), builtin.GuideContext{}))
	if n := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME")); n != "" {
		ds, errD := toolDefinitionsToSpecs([]ToolDefinition{DiscoverSkillsToolDefinition(n)})
		if errD != nil {
			return nil, errD
		}
		assembled = toolpool.UniqByName(append(ds, assembled...))
	}
	stubs, err := toolDefinitionsToSpecs(DefaultStubTools())
	if err != nil {
		return nil, err
	}
	assembled = toolpool.UniqByName(append(assembled, stubs...))
	return toolpool.MarshalToolsAPIDocumentDefinitions(assembled)
}

func toolDefinitionsToSpecs(defs []ToolDefinition) ([]types.ToolSpec, error) {
	out := make([]types.ToolSpec, 0, len(defs))
	for _, d := range defs {
		raw, err := json.Marshal(d.InputSchema)
		if err != nil {
			return nil, err
		}
		out = append(out, types.ToolSpec{
			Name:            d.Name,
			Description:     d.Description,
			InputJSONSchema: raw,
		})
	}
	return out, nil
}
